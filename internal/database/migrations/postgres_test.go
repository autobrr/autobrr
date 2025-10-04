// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package migrations_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/database/migrations"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/migrator"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/stretchr/testify/require"
)

// runMigrationTestPostgres executes a pluggable migration test
func runMigrationTestPostgres(t *testing.T, testCase MigrationTestCase) {
	db, cleanup := setupTestPostgresDB(t)
	defer cleanup()

	migrate := migrations.PostgresMigrations(db.Handler)

	err := migrate.InitVersionTable()
	require.NoError(t, err)

	// Run initial schema setup (all migrations up to the target migration - 1)
	m := migrate.GetUpTo(testCase.MigrationsUntilName)

	err = migrate.RunMigrations(m)
	require.NoError(t, err)

	// Insert test data
	if testCase.SetupData != nil {
		err = testCase.SetupData(db.Handler)
		require.NoError(t, err, "Failed to setup test data")
	}

	// Get the migration to run
	currentMigration, err := migrate.Get(testCase.MigrationToRun)
	require.NoError(t, err, "Failed to get migration")

	// Run the specific migration being tested
	err = migrate.RunMigrations([]*migrator.Migration{currentMigration})
	require.NoError(t, err, "Failed to run target migration")

	// Validate the results
	if testCase.ValidateResult != nil {
		testCase.ValidateResult(db.Handler, t)
	}
}

func setupPGTestDB() (*database.DB, func(), error) {
	var (
		dbUsername = "postgres"
		dbPassword = "postgres"
		dbName     = "autobrr"
		dbPort     = 9876
	)

	pgLogger := &bytes.Buffer{}
	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Username(dbUsername).
		Password(dbPassword).
		Database(dbName).
		Port(uint32(dbPort)).
		Version(embeddedpostgres.V17).
		//RuntimePath("/tmp").
		//BinaryRepositoryURL("https://repo.local/central.proxy").
		StartTimeout(45 * time.Second).
		StartParameters(map[string]string{"max_connections": "200"}).
		Logger(pgLogger))

	err := postgres.Start()
	if err != nil {
		//t.Error(err)
		return nil, nil, err
	}

	//dsn := "postgres://postgres:postgres@localhost:9876/autobrr?sslmode=disable"
	dsn := fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?sslmode=disable", dbUsername, dbPassword, dbPort, dbName)

	cfg := &domain.Config{
		DatabaseType:        "postgres",
		DatabaseDSN:         dsn,
		DatabaseAutoMigrate: false,
	}

	log := logger.New(&domain.Config{LogLevel: "ERROR", LogPath: ""})
	db, err := database.NewDB(cfg, log)
	if err != nil {
		return nil, nil, err
	}

	err = db.Open()
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		_ = db.Close()
		_ = postgres.Stop()
	}

	return db, cleanup, nil
}

func setupTestPostgresDB(t *testing.T) (*database.DB, func()) {
	dsn := "postgres://postgres:postgres@localhost:9876/autobrr?sslmode=disable"

	cfg := &domain.Config{
		DatabaseType:        "postgres",
		DatabaseDSN:         dsn,
		DatabaseAutoMigrate: false,
	}

	log := logger.New(&domain.Config{LogLevel: "ERROR", LogPath: ""})
	db, err := database.NewDB(cfg, log)
	require.NoError(t, err)

	err = db.Open()
	require.NoError(t, err)

	cleanup := func() {
		_ = db.Close()
		//_ = os.Remove(dbPath)
	}

	return db, cleanup
}

// Test full migration sequence
func TestFullMigrationSequencePostgres(t *testing.T) {
	//db, cleanup := setupTestPostgresDB(t)
	db, cleanup, err := setupPGTestDB()
	defer cleanup()
	require.NoError(t, err)

	// This will run all migrations
	migrate := migrations.PostgresMigrations(db.Handler)

	err = migrate.Migrate()
	require.NoError(t, err)

	//// Verify current schema version
	//var version int
	//err = db.Handler.QueryRow("PRAGMA user_version").Scan(&version)
	//require.NoError(t, err)
	//
	//expectedVersion := len(database.sqliteMigrations)
	//assert.Equal(t, expectedVersion, version, "Should be at latest migration version")
}

//func TestMain(m *testing.M) {
//	pgDb, cleanup, err := setupPGTestDB()
//	//defer cleanup()
//	if err != nil {
//		fmt.Println(err)
//		os.Exit(1)
//	}
//
//	code := m.Run()
//
//	cleanup()
//
//	os.Exit(code)
//}
