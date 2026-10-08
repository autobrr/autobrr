// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package cli

import (
	"fmt"
	"strings"

	"github.com/autobrr/autobrr/internal/database/tools"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/spf13/cobra"
)

func CommandDb() *cobra.Command {
	var command = &cobra.Command{
		Use:   "db",
		Short: "Database maintenance commands",
	}

	command.AddCommand(CommandDbSeed())
	command.AddCommand(CommandDbReset())
	command.AddCommand(CommandDbConvert())

	return command
}

func CommandDbSeed() *cobra.Command {
	var command = &cobra.Command{
		Use:     "seed",
		Short:   "Seed the SQLite database",
		Example: "  autobrr db seed --db-path /path/to/autobrr.db --seed-db /path/to/seed",
	}

	var dbPath, seedPath string

	command.Flags().StringVar(&dbPath, "db-path", "", "path to the database file")
	command.Flags().StringVar(&seedPath, "seed-db", "", "path to SQL seed file")
	_ = command.MarkFlagRequired("db-path")
	_ = command.MarkFlagRequired("seed-db")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		if err := tools.NewSQLiteSeeder(dbPath, seedPath).Seed(); err != nil {
			return errors.Wrap(err, "could not seed database")
		}

		fmt.Println("Database seeding completed successfully!")

		return nil
	}

	return command
}

func CommandDbReset() *cobra.Command {
	var command = &cobra.Command{
		Use:     "reset",
		Short:   "Reset and reseed the SQLite database",
		Example: "  autobrr db reset --db-path /path/to/autobrr.db --seed-db /path/to/seed",
	}

	var dbPath, seedPath string

	command.Flags().StringVar(&dbPath, "db-path", "", "path to the database file")
	command.Flags().StringVar(&seedPath, "seed-db", "", "path to SQL seed file")
	_ = command.MarkFlagRequired("db-path")
	_ = command.MarkFlagRequired("seed-db")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		seeder := tools.NewSQLiteSeeder(dbPath, seedPath)

		if err := seeder.Reset(); err != nil {
			return errors.Wrap(err, "could not reset database")
		}

		if err := seeder.Seed(); err != nil {
			return errors.Wrap(err, "could not seed database")
		}

		fmt.Println("Database reset and reseed completed successfully!")

		return nil
	}

	return command
}

func CommandDbConvert() *cobra.Command {
	var command = &cobra.Command{
		Use:     "convert",
		Short:   "Convert a SQLite database to PostgreSQL",
		Example: "  autobrr db convert --sqlite-db /path/to/autobrr.db --postgres-url postgres://username:password@127.0.0.1:5432/autobrr",
	}

	var (
		sqlitePath    string
		postgresURL   string
		excludeTables string
		dryRun        bool
	)

	command.Flags().StringVar(&sqlitePath, "sqlite-db", "", "path to SQLite database file")
	command.Flags().StringVar(&postgresURL, "postgres-url", "", "DSN for PostgreSQL database: postgres://user:pass@host:port/db?sslmode=disable")
	command.Flags().StringVar(&excludeTables, "exclude-tables", "", "comma separated list of tables to exclude from conversion")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "dry run")
	_ = command.MarkFlagRequired("sqlite-db")
	_ = command.MarkFlagRequired("postgres-url")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		opts := tools.Opts{
			DryRun:        dryRun,
			ExcludeTables: strings.Split(excludeTables, ","),
		}

		l := logger.New(&domain.Config{LogLevel: "TRACE"}, nil)

		if err := tools.NewConverter(l, sqlitePath, postgresURL).Convert(cmd.Context(), opts); err != nil {
			return errors.Wrap(err, "database conversion failed")
		}

		return nil
	}

	return command
}
