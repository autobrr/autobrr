// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build windows || darwin || linux

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	_ "time/tzdata"

	"github.com/autobrr/autobrr/internal/application"
	"github.com/autobrr/autobrr/internal/cli"
	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/meta"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func init() {
	// the tray event loop must run on the main OS thread on macOS and Windows
	runtime.LockOSThread()

	// cobra refuses to run when started from Explorer unless this is cleared,
	// and the check runs inside Execute, before any RunE
	cobra.MousetrapHelpText = ""
}

func main() {
	meta.Set(version, commit, date)

	var configPath string

	root := &cobra.Command{
		Use:          "autobrr-desktop",
		Short:        "autobrr with a system tray icon",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), configPath)
		},
	}

	root.PersistentFlags().StringVar(&configPath, "config", "", "path to configuration directory")

	root.AddCommand(cli.CommandVersion())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// run keeps the tray on the calling goroutine and the application on another.
// Whichever ends first takes the other down: Quit cancels ctx, and a returning
// Run removes the tray, which unblocks tray.Run.
func run(ctx context.Context, configPath string) error {
	ctx, cancel := cli.ShutdownContext(ctx)
	defer cancel()

	app, err := newApp(configPath)
	if err != nil {
		showError("autobrr", err.Error())

		return err
	}

	tray := newTray(app, cancel)

	var runErr error
	done := make(chan struct{})
	go func() {
		runErr = app.Run(ctx)
		close(done)
		tray.Remove()
	}()

	if err := onSessionEnd(func() {
		cancel()
		<-done
	}); err != nil {
		app.Log().Error().Err(err).Msg("could not watch for session end")
	}

	if err := tray.Run(); err != nil {
		app.Log().Error().Err(err).Msg("tray stopped unexpectedly")
	}

	cancel()
	<-done

	if runErr != nil {
		showError("autobrr", fmt.Sprintf("autobrr stopped: %v\n\nLog file: %s", runErr, app.Config().LogPath))
	}

	return runErr
}

// newApp defaults to the OS config directory, because a desktop launch has no
// --config and its working directory is often read-only, and to a log file in
// it, because the GUI build has no console to log to.
func newApp(configPath string) (*application.App, error) {
	if configPath == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return nil, errors.Wrap(err, "could not find config directory")
		}

		configPath = filepath.Join(dir, "autobrr")
	}

	cfg := config.New(configPath)

	if cfg.Config.LogPath == "" {
		cfg.Config.LogPath = filepath.Join(configPath, "logs", "autobrr.log")
	}

	return application.NewFromConfig(cfg), nil
}
