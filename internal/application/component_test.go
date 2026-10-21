// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package application

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// A shutdown requested while starting must not start the components after it,
// they would connect out only to be torn down again.
func TestStartComponents_StopsWhenContextEnds(t *testing.T) {
	app := &App{log: zerolog.Nop()}

	ctx, cancel := context.WithCancel(t.Context())

	var calls []string
	components := []component{
		{name: "first", start: func(context.Context) error {
			calls = append(calls, "first")
			cancel()

			return nil
		}},
		{name: "second", start: func(context.Context) error {
			calls = append(calls, "second")

			return nil
		}},
	}

	started, err := app.startComponents(ctx, components)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []string{"first"}, calls)
	assert.Len(t, started, 1)
}
