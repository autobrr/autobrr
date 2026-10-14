// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package cli

import (
	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// openDatabase opens the database configured by --config. The caller closes it.
func openDatabase(cmd *cobra.Command) (*database.DB, zerolog.Logger, error) {
	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		return nil, zerolog.Nop(), errors.New("--config required")
	}

	cfg := config.New(configPath)

	l := logger.New(cfg.Config, nil)

	db, err := database.NewDB(cfg.Config, l)
	if err != nil {
		return nil, l, errors.Wrap(err, "could not init database")
	}

	if err := db.Open(); err != nil {
		return nil, l, errors.Wrap(err, "could not open database")
	}

	return db, l, nil
}
