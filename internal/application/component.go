// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package application

import (
	"context"
	"slices"
	"time"
)

const shutdownTimeout = 30 * time.Second

// component is one startable part of the application. Components start in
// list order and stop in reverse, so a component may depend on anything
// listed before it.
type component struct {
	name string

	// optional components log a failed start and let the rest of the
	// application come up, matching the pre-refactor behavior for them.
	optional bool

	start func(ctx context.Context) error
	stop  func(ctx context.Context) error
}

// startComponents returns the components that started, so the caller can
// stop exactly those when a later one fails.
func (app *App) startComponents(ctx context.Context, components []component) ([]component, error) {
	started := make([]component, 0, len(components))

	for _, c := range components {
		if err := c.start(ctx); err != nil {
			if c.optional {
				app.log.Error().Err(err).Str("component", c.name).Msg("could not start component")
				continue
			}

			return started, err
		}

		started = append(started, c)
	}

	return started, nil
}

// stopComponents runs on a fresh context because the run context is already
// cancelled by the time shutdown starts.
func (app *App) stopComponents(components []component) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	for _, c := range slices.Backward(components) {
		if c.stop == nil {
			continue
		}

		app.log.Debug().Str("component", c.name).Msg("stopping component")

		if err := c.stop(ctx); err != nil {
			app.log.Error().Err(err).Str("component", c.name).Msg("could not stop component")
		}
	}
}

// serve runs a blocking serve loop in the background and hands an unexpected
// exit to onError.
func serve(fn func() error, onError func(error)) {
	go func() {
		if err := fn(); err != nil {
			onError(err)
		}
	}()
}
