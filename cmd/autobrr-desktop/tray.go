// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build windows || darwin || linux

package main

import (
	"context"
	"net"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/autobrr/autobrr/distrib/windows/icons"
	"github.com/autobrr/autobrr/internal/application"
	"github.com/autobrr/autobrr/internal/events"

	"github.com/gogpu/systray"
)

func newTray(app *application.App, quit context.CancelFunc) *systray.SystemTray {
	tray := systray.New()

	// counts update notifications, so a manual check can tell whether the
	// event it triggered already notified
	var updateNotices atomic.Int64

	notifyUpdate := func(tag string) {
		updateNotices.Add(1)
		tray.ShowNotification("autobrr", "New update available: "+tag)
	}

	app.OnAppUpdate(func(_ context.Context, event events.AppUpdateEvent) error {
		notifyUpdate(event.NewVersion)
		return nil
	})

	menu := systray.NewMenu()

	openItem := menu.Add("Open Web UI", func() {
		cfg := app.Config()
		if err := openBrowser(webURL(cfg.Host, cfg.Port, cfg.BaseURL)); err != nil {
			app.Log().Error().Err(err).Msg("could not open browser")
		}
	})
	openItem.SetDisabled(true)

	go func() {
		<-app.Ready()
		openItem.SetDisabled(false)
	}()

	menu.Add("Open Config Folder", func() {
		if err := openFolder(app.Config().ConfigPath); err != nil {
			app.Log().Error().Err(err).Msg("could not open config folder")
		}
	})

	menu.Add("Check for updates", func() {
		notices := updateNotices.Load()

		rel, err := app.CheckUpdateAvailable(context.Background())
		if err != nil {
			app.Log().Error().Err(err).Msg("could not check for updates")
			tray.ShowNotification("autobrr", "Could not check for updates")
			return
		}

		if rel == nil {
			tray.ShowNotification("autobrr", "autobrr is up to date")
			return
		}

		// the event bus is synchronous, so a release found by this check has
		// already notified through OnAppUpdate
		if updateNotices.Load() == notices {
			notifyUpdate(rel.TagName)
		}
	})

	menu.AddSeparator()

	menu.Add("Quit", quit)

	tray.SetIcon(icons.TrayLight).
		SetDarkModeIcon(icons.TrayDark).
		SetTooltip("autobrr").
		SetAppName("autobrr").
		SetMenu(menu)

	tray.Show()

	return tray
}

// webURL maps wildcard bind addresses to localhost, since a browser cannot
// open "all addresses". autobrr has no built-in TLS, so the scheme is http.
func webURL(host string, port int, baseURL string) string {
	switch host {
	case "", "0.0.0.0", "::":
		host = "localhost"
	}

	return "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + normalizeBasePath(baseURL) + "/"
}

// normalizeBasePath returns baseURL with a leading slash and no trailing
// slash, or an empty string for the root.
func normalizeBasePath(baseURL string) string {
	base := strings.Trim(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return ""
	}

	return "/" + base
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return start(exec.Command("rundll32", "url.dll,FileProtocolHandler", url))
	case "darwin":
		return start(exec.Command("open", url))
	default:
		return start(exec.Command("xdg-open", url))
	}
}

// openFolder resolves dir first, since a relative --config is relative to our
// working directory, not the file manager's.
func openFolder(dir string) error {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	case "windows":
		return start(exec.Command("explorer", dir))
	case "darwin":
		return start(exec.Command("open", dir))
	default:
		return start(exec.Command("xdg-open", dir))
	}
}

func start(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}

	// reap the child so it does not linger as a zombie
	go cmd.Wait()

	return nil
}
