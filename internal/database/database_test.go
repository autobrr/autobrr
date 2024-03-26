// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/stretchr/testify/assert"
)

func getDbs() []string {
	return []string{"sqlite", "postgres"}
}

var testDBs map[string]*DB

func setupPostgresForTest() *DB {
	dbtype := "postgres"
	if d, ok := testDBs[dbtype]; ok {
		return d
	}

	cfg := &domain.Config{
		LogLevel:         "INFO",
		DatabaseType:     dbtype,
		PostgresHost:     "localhost",
		PostgresPort:     5437,
		PostgresDatabase: "autobrr",
		PostgresUser:     "testdb",
		PostgresPass:     "testdb",
		PostgresSSLMode:  "disable",
	}

	// Init a new logger
	logr := logger.New(cfg)

	logr.With().Str("type", "postgres").Logger()

	// Initialize a new DB connection
	db, err := NewDB(cfg, logr)
	if err != nil {
		log.Fatalf("Could not create database: %q", err)
	}

	// Open the database connection
	if db.handler, err = sql.Open("postgres", db.DSN); err != nil {
		log.Fatalf("could not open postgres connection: %q", err)
	}

	if err = db.handler.Ping(); err != nil {
		log.Fatalf("could not ping postgres database: %q", err)
	}

	// drop tables before migrate to always have a clean state
	if _, err := db.handler.Exec(`
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;

-- Restore default permissions
GRANT ALL ON SCHEMA public TO testdb;
GRANT ALL ON SCHEMA public TO public;
`); err != nil {
		log.Fatalf("Could not drop database: %q", err)
	}

	// migrate db
	if err = db.migratePostgres(); err != nil {
		log.Fatalf("Could not migrate postgres database: %q", err)
	}

	testDBs[dbtype] = db

	return db
}

func setupSqliteForTest() *DB {
	dbtype := "sqlite"

	if d, ok := testDBs[dbtype]; ok {
		return d
	}

	cfg := &domain.Config{
		LogLevel:     "INFO",
		DatabaseType: dbtype,
	}

	// Init a new logger
	logr := logger.New(cfg)

	// Initialize a new DB connection
	db, err := NewDB(cfg, logr)
	if err != nil {
		log.Fatalf("Could not create database: %v", err)
	}

	// Open the database connection
	if err := db.Open(); err != nil {
		log.Fatalf("Could not open db connection: %v", err)
	}

	testDBs[dbtype] = db

	return db
}

func setupLoggerForTest() logger.Logger {
	cfg := &domain.Config{
		LogLevel: "ERROR",
	}
	log := logger.New(cfg)

	return log
}

func TestPingDatabase(t *testing.T) {
	// Setup database
	for _, db := range testDBs {

		// Call the Ping method
		err := db.Ping()

		assert.NoError(t, err, "Database should be reachable")
	}
}

func TestMain(m *testing.M) {
	if err := os.Setenv("IS_TEST_ENV", "true"); err != nil {
		log.Fatalf("Could not set env variable: %v", err)
	}

	testDBs = make(map[string]*DB)

	fmt.Println("setup")

	setupPostgresForTest()
	setupSqliteForTest()

	fmt.Println("running tests")

	//Run tests
	code := m.Run()

	fmt.Println("teardown")

	for _, d := range testDBs {
		if err := d.Close(); err != nil {
			log.Fatalf("Could not close db connection: %v", err)
		}
	}

	if err := os.Setenv("IS_TEST_ENV", "false"); err != nil {
		log.Fatalf("Could not set env variable: %v", err)
	}

	os.Exit(code)
}
