// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"os"
	_ "time/tzdata"

	"github.com/autobrr/autobrr/internal/cli"
	"github.com/autobrr/autobrr/internal/meta"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	meta.Set(version, commit, date)

	var configPath, profilePath string

	serve := cli.CommandServe()

	root := &cobra.Command{
		Use:   "autobrr",
		Short: "autobrr",
		Long:  "autobrr",
		// running without a command starts the service, kept for legacy reasons
		RunE: serve.RunE,
		// a failing server should not print the usage block
		SilenceUsage: true,
	}

	// persistent flags reach both `autobrr --config` and `autobrr serve --config`
	root.PersistentFlags().StringVar(&configPath, "config", "", "path to configuration directory")
	root.PersistentFlags().StringVar(&profilePath, "pgo", "", "internal build flag")

	// serve's own flags are shared by pointer, so the bare form parses into the
	// same variables serve does
	root.Flags().AddFlagSet(serve.Flags())

	root.AddCommand(serve)
	root.AddCommand(cli.CommandDb())
	root.AddCommand(cli.CommandFilter())
	root.AddCommand(cli.CommandUser())
	root.AddCommand(cli.CommandVersion())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
