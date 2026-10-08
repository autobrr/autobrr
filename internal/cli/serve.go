// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package cli

import (
	"context"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/autobrr/autobrr/internal/application"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/spf13/cobra"
)

// pgoDuration is how long a --pgo run records before shutting itself down.
// The release workflow relies on the process exiting on its own.
const pgoDuration = 5 * time.Second

func CommandServe() *cobra.Command {
	var command = &cobra.Command{
		Use:     "serve",
		Short:   "Start the service",
		Example: "  autobrr serve\n  autobrr serve --config=/config",
	}

	command.RunE = func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		profilePath, _ := cmd.Flags().GetString("pgo")

		ctx, cancel := ShutdownContext(cmd.Context())
		defer cancel()

		if profilePath != "" {
			stopProfile, err := startCPUProfile(profilePath)
			if err != nil {
				return err
			}

			defer stopProfile()

			var cancelPGO context.CancelFunc
			ctx, cancelPGO = context.WithTimeout(ctx, pgoDuration)
			defer cancelPGO()
		}

		return application.New(configPath).Run(ctx)
	}

	return command
}

// ShutdownContext is cancelled by the first termination signal. A second
// signal is left to the runtime and kills the process, so a hung shutdown can
// still be interrupted.
func ShutdownContext(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)

	context.AfterFunc(ctx, stop)

	return ctx, stop
}

func startCPUProfile(path string) (func(), error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not create cpu profile: %s", path)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()

		return nil, errors.Wrap(err, "could not start cpu profile")
	}

	return func() {
		pprof.StopCPUProfile()
		f.Close()
	}, nil
}
