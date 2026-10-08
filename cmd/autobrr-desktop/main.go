// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build windows || darwin || linux

package main

import (
	"context"
	"os"
	"runtime"
	_ "time/tzdata"

	"github.com/autobrr/autobrr/internal/application"
	"github.com/autobrr/autobrr/internal/cli"
	"github.com/autobrr/autobrr/internal/meta"

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

	app := application.New(configPath)

	tray := newTray(app, cancel)

	errc := make(chan error, 1)
	go func() {
		errc <- app.Run(ctx)
		tray.Remove()
	}()

	if err := tray.Run(); err != nil {
		app.Log().Error().Err(err).Msg("tray stopped unexpectedly")
	}

	cancel()

	return <-errc
}
