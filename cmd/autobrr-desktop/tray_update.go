// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build windows || darwin || linux

package main

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/autobrr/autobrr/internal/application"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/internal/meta"
	"github.com/autobrr/autobrr/pkg/errors"
	pkgversion "github.com/autobrr/autobrr/pkg/version"

	"github.com/gogpu/systray"
)

const (
	autoUpdateSetting = "AutoUpdate"

	// a release waits this long before it installs on its own, so a broken one
	// can be pulled or fixed first; "Install update" does not wait
	autoUpdateDelay    = 24 * time.Hour
	autoUpdateInterval = time.Hour

	updatedFromFlag = "--updated-from"
)

// updateMenu owns the tray's update items and the automatic updates.
type updateMenu struct {
	app     *application.App
	tray    *systray.SystemTray
	updater *selfUpdater
	restart func()

	// nil unless canSelfUpdate
	installItem *systray.MenuItem

	// counts update notifications, so a manual check can tell whether the
	// event it triggered already notified
	notices atomic.Int64

	// held from the start of an install until the restart, so a manual and an
	// automatic install never run at once
	installing atomic.Bool

	// turning automatic updates on checks right away instead of within the hour
	autoWake chan struct{}
}

// canSelfUpdate is true where the release ships the desktop zip that the
// updater installs.
func canSelfUpdate() bool {
	return runtime.GOOS == "windows" && runtime.GOARCH == "amd64"
}

func addUpdateMenu(menu *systray.Menu, app *application.App, tray *systray.SystemTray, updater *selfUpdater, restart func()) {
	m := &updateMenu{app: app, tray: tray, updater: updater, restart: restart, autoWake: make(chan struct{}, 1)}

	app.OnAppUpdate(func(_ context.Context, event events.AppUpdateEvent) error {
		m.notify(event.NewVersion)
		return nil
	})

	// a check before startup finishes would emit the release before the
	// notification service listens, and each release is emitted only once
	checkItem := menu.Add("Check for updates", func() {
		// off the tray thread, which also handles logoff, so a slow server
		// cannot hold up the menu or a clean shutdown
		go m.check()
	})
	checkItem.SetDisabled(true)

	go func() {
		<-app.Ready()
		checkItem.SetDisabled(false)
	}()

	if !canSelfUpdate() {
		return
	}

	m.installItem = menu.Add("Install update", func() {
		m.installItem.SetDisabled(true)

		// off the tray thread, so the menu keeps working during the download
		go func() {
			if !m.installConfirmed() {
				m.installItem.SetDisabled(false)
			}
		}()
	})
	m.installItem.SetDisabled(true)

	auto, err := desktopSetting(autoUpdateSetting)
	if err != nil {
		app.Log().Error().Err(err).Msg("could not read automatic update setting")
	}

	var autoItem *systray.MenuItem
	autoItem = menu.AddCheckbox("Install updates automatically", auto, func() {
		want := !autoItem.IsChecked()

		if err := setDesktopSetting(autoUpdateSetting, want); err != nil {
			app.Log().Error().Err(err).Bool("enabled", want).Msg("could not change automatic update setting")
			return
		}

		autoItem.SetChecked(want)

		if want {
			select {
			case m.autoWake <- struct{}{}:
			default:
			}
		}
	})

	go m.autoUpdate(autoItem)
}

func (m *updateMenu) notify(tag string) {
	m.notices.Add(1)

	if m.installItem == nil {
		m.tray.ShowNotification("autobrr", "New update available: "+tag)
		return
	}

	m.installItem.SetLabel("Install update " + tag)
	m.installItem.SetDisabled(false)
	m.tray.ShowNotification("autobrr", "New update available: "+tag+". Install it from the tray menu.")
}

func (m *updateMenu) check() {
	notices := m.notices.Load()

	rel, err := m.app.CheckUpdateAvailable(context.Background())
	if err != nil {
		m.app.Log().Error().Err(err).Msg("could not check for updates")
		m.tray.ShowNotification("autobrr", "Could not check for updates")
		return
	}

	if rel == nil {
		m.tray.ShowNotification("autobrr", "autobrr is up to date")
		return
	}

	// the event bus is synchronous, so a release found by this check has
	// already notified through OnAppUpdate
	if m.notices.Load() == notices {
		m.notify(rel.TagName)
	}
}

// installConfirmed asks before it installs the latest release and restarts. It
// fetches the release again rather than trusting the one that enabled the menu
// item, which may have been replaced since. It reports whether the restart was
// requested.
func (m *updateMenu) installConfirmed() bool {
	if !m.installing.CompareAndSwap(false, true) {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	rel, err := m.app.CheckUpdateAvailable(ctx)
	if err != nil {
		m.installing.Store(false)
		m.app.Log().Error().Err(err).Msg("could not check for updates")
		showError("autobrr", fmt.Sprintf("Could not check for updates: %v", err))
		return false
	}

	if rel == nil {
		m.installing.Store(false)
		m.tray.ShowNotification("autobrr", "autobrr is up to date")
		return false
	}

	if !askYesNo("autobrr", fmt.Sprintf("Update autobrr from %s to %s?\n\nautobrr restarts to finish the update.", meta.GetVersion(), rel.TagName)) {
		m.installing.Store(false)
		return false
	}

	m.tray.ShowNotification("autobrr", "Downloading "+rel.TagName)

	if err := m.install(ctx, rel); err != nil {
		if !errors.Is(err, errShuttingDown) {
			showError("autobrr", fmt.Sprintf("Could not install %s: %v\n\nYou can download it from %s", rel.TagName, err, rel.HtmlURL))
		}
		return false
	}

	return true
}

// autoUpdate installs a release once it is autoUpdateDelay old, while the
// setting is on and update checks are enabled in the config. It runs until the
// process exits. A release that fails to install is not tried again on its
// own, so a broken asset is not downloaded every hour; "Install update" still
// offers it.
func (m *updateMenu) autoUpdate(autoItem *systray.MenuItem) {
	<-m.app.Ready()

	var failedTag string

	ticker := time.NewTicker(autoUpdateInterval)
	defer ticker.Stop()

	for first := true; ; first = false {
		if !first {
			select {
			case <-ticker.C:
			case <-m.autoWake:
			}
		}

		if !autoItem.IsChecked() || !m.app.Config().CheckForUpdates {
			continue
		}

		rel, err := m.app.CheckUpdateAvailable(context.Background())
		if err != nil {
			m.app.Log().Error().Err(err).Msg("could not check for updates")
			continue
		}

		// checked again, since the box may have been unticked during the request
		if rel == nil || rel.TagName == failedTag || !autoUpdateDue(rel, time.Now()) || !autoItem.IsChecked() {
			continue
		}

		if !m.installing.CompareAndSwap(false, true) {
			continue
		}

		m.tray.ShowNotification("autobrr", "Updating to "+rel.TagName+". autobrr restarts in a moment.")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		err = m.install(ctx, rel)
		cancel()

		if errors.Is(err, errShuttingDown) {
			return
		}

		if err != nil {
			failedTag = rel.TagName
			m.tray.ShowNotification("autobrr", "Automatic update to "+rel.TagName+" failed. See the log, or use Install update.")
			continue
		}

		return
	}
}

// install expects m.installing to be held. It releases it on failure and keeps
// it on success, since the restart follows.
func (m *updateMenu) install(ctx context.Context, rel *pkgversion.Release) error {
	if err := m.updater.install(ctx, rel); err != nil {
		m.installing.Store(false)
		m.app.Log().Error().Err(err).Str("version", rel.TagName).Msg("could not install update")
		return err
	}

	m.app.Log().Info().Str("version", rel.TagName).Msg("update installed, restarting")

	m.restart()

	return nil
}

func autoUpdateDue(rel *pkgversion.Release, now time.Time) bool {
	return !rel.PublishedAt.IsZero() && now.Sub(rel.PublishedAt) >= autoUpdateDelay
}

// restartArgs passes the version that was replaced to the new process, so it
// can tell the user it was updated. It drops the flag from an earlier update.
func restartArgs(args []string, from string) []string {
	out := make([]string, 0, len(args)+1)
	for _, arg := range args {
		if !strings.HasPrefix(arg, updatedFromFlag+"=") {
			out = append(out, arg)
		}
	}

	return append(out, updatedFromFlag+"="+from)
}
