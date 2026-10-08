// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build windows || darwin || linux

package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/autobrr/autobrr/internal/application"
	"github.com/autobrr/autobrr/internal/events"

	"github.com/gogpu/systray"
)

func newTray(app *application.App, quit context.CancelFunc) *systray.SystemTray {
	tray := systray.New()

	menu := systray.NewMenu()
	menu.Add("Open Web UI", func() {
		cfg := app.Config()
		if err := openBrowser(webURL(cfg.Host, cfg.Port, cfg.BaseURL)); err != nil {
			app.Log().Error().Err(err).Msg("could not open browser")
		}
	})

	menu.Add("Check for updates", func() {
		rel, err := app.CheckUpdateAvailable(context.Background())
		if err != nil {
			app.Log().Error().Err(err).Msg("could not check for updates")
			return
		}

		if rel == nil {
			tray.ShowNotification("autobrr", "autobrr is up to date")
			return
		}

		tray.ShowNotification("autobrr", "New update available: "+rel.TagName)
	})

	menu.AddSeparator()

	menu.Add("Quit", quit)

	app.OnAppUpdate(func(_ context.Context, event events.AppUpdateEvent) error {
		tray.ShowNotification("autobrr", "New update available: "+event.NewVersion)
		return nil
	})

	tray.SetIcon(generateIcon(22, color.RGBA{R: 0, G: 180, B: 80, A: 255})).
		SetDarkModeIcon(generateIcon(22, color.RGBA{R: 0, G: 230, B: 230, A: 255})).
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

func generateIcon(size int, c color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			if x == 0 || x == size-1 || y == 0 || y == size-1 {
				img.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
			} else {
				img.SetRGBA(x, y, c)
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)

	return buf.Bytes()
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// reap the child so it does not linger as a zombie
	go cmd.Wait()

	return nil
}
