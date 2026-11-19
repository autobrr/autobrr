// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build windows || darwin || linux

package main

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/autobrr/autobrr/distrib/windows/icons"
	"github.com/autobrr/autobrr/internal/application"
	"github.com/autobrr/autobrr/internal/meta"

	"github.com/gogpu/systray"
)

func newTray(app *application.App, quit context.CancelFunc, updater *selfUpdater, restart func(), updatedFrom string) *systray.SystemTray {
	tray := systray.New()

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

		if updatedFrom != "" {
			tray.ShowNotification("autobrr", "autobrr was updated from "+updatedFrom+" to "+meta.GetVersion())
		}
	}()

	menu.Add("Open Config Folder", func() {
		if err := openFolder(app.Config().ConfigPath); err != nil {
			app.Log().Error().Err(err).Msg("could not open config folder")
		}
	})

	if runtime.GOOS == "windows" {
		addStartOnLogin(menu, app)
	}

	addUpdateMenu(menu, app, tray, updater, restart)

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

func addStartOnLogin(menu *systray.Menu, app *application.App) {
	exe, err := os.Executable()
	if err != nil {
		app.Log().Error().Err(err).Msg("could not find executable for start on login")
		return
	}

	configDir, err := filepath.Abs(app.Config().ConfigPath)
	if err != nil {
		app.Log().Error().Err(err).Msg("could not resolve config directory for start on login")
		return
	}

	command := loginCommand(exe, configDir)

	enabled, err := startsOnLogin("autobrr", command)
	if err != nil {
		app.Log().Error().Err(err).Msg("could not read start on login setting")
	}

	var item *systray.MenuItem
	item = menu.AddCheckbox("Start on login", enabled, func() {
		want := !item.IsChecked()

		if err := setStartOnLogin("autobrr", command, want); err != nil {
			app.Log().Error().Err(err).Bool("enabled", want).Msg("could not change start on login")
			return
		}

		item.SetChecked(want)
	})
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
