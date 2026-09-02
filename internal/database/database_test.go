// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package database

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type testDB struct {
	db      *DB
	cleanup cleanupFunc
}

type cleanupFunc func()

var testDBs map[string]*testDB

// embeddedPGBinariesPath returns the directory the Postgres binaries are extracted to.
// embedded-postgres wipes its runtime path on every Start, so instances get their own
// while the binaries are extracted once here and reused. The path is package specific
// so a cold start can never have two test binaries extracting into the same directory.
func embeddedPGBinariesPath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}

	return filepath.Join(cacheDir, "autobrr", "embedded-postgres", "v17-database")
}

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}

// startEmbeddedPG boots an embedded Postgres on a random port and returns
// an open *database.DB plus a cleanup that stops the server. Used so the per-migration
// test in this file can boot Postgres once and reuse it across sub-tests.
func startEmbeddedPG() (*DB, cleanupFunc) {
	freePort, err := GetFreePort()
	if err != nil {
		log.Fatalf("Could not start embedded postgres: %q", err)
	}

	cfg := &domain.Config{
		LogLevel:            "ERROR",
		LogPath:             "",
		DatabaseType:        "postgres",
		DatabaseDSN:         "",
		PostgresHost:        "localhost",
		PostgresPort:        freePort,
		PostgresDatabase:    "autobrr",
		PostgresUser:        "testdb",
		PostgresPass:        "testdb",
		DatabaseAutoMigrate: false,
	}

	cfg.DatabaseDSN = fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?sslmode=disable", cfg.PostgresUser, cfg.PostgresPass, cfg.PostgresPort, cfg.PostgresDatabase)

	runtimePath, err := os.MkdirTemp("", "autobrr-pg-runtime-")
	if err != nil {
		log.Fatalf("Could not create embedded postgres runtime dir: %q", err)
	}

	pgLogger := &bytes.Buffer{}
	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Username(cfg.PostgresUser).
		Password(cfg.PostgresPass).
		Database(cfg.PostgresDatabase).
		Port(uint32(cfg.PostgresPort)).
		Version(embeddedpostgres.V17).
		BinariesPath(embeddedPGBinariesPath()).
		RuntimePath(runtimePath).
		StartTimeout(45 * time.Second).
		StartParameters(map[string]string{"max_connections": "200"}).
		Logger(pgLogger))
	if err := pg.Start(); err != nil {
		log.Fatalf("Could not start embedded postgres database: %q", err)
	}

	l := logger.New(cfg, nil)
	db, err := NewDB(cfg, l)
	if err != nil {
		log.Fatalf("Could not create database: %q", err)
	}

	if err := db.Open(); err != nil {
		log.Fatalf("Could not open database: %q", err)
	}

	// drop tables before migrate to always have a clean state
	if _, err := db.Handler.Exec(`
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

	cleanup := func() {
		_ = db.Close()
		if err := pg.Stop(); err != nil {
			log.Printf("failed to stop embedded postgres: %v\npg log:\n%s", err, pgLogger.String())
		}
		_ = os.RemoveAll(runtimePath)
	}

	testDBInstance := &testDB{
		db:      db,
		cleanup: cleanup,
	}

	testDBs[cfg.DatabaseType] = testDBInstance

	return db, cleanup
}

func setupSqliteForTest() (*DB, cleanupFunc) {
	cfg := &domain.Config{
		LogLevel:     "ERROR",
		DatabaseType: "sqlite",
		DatabaseDSN:  ":memory:",
	}

	// Init a new logger
	dbLogger := logger.New(cfg, nil)

	// Initialize a new DB connection
	db, err := NewDB(cfg, dbLogger)
	if err != nil {
		log.Fatalf("Could not create database: %v", err)
	}

	// Open the database connection
	if err := db.Open(); err != nil {
		log.Fatalf("Could not open db connection: %v", err)
	}

	// migrate db
	if err = db.migrateSQLite(); err != nil {
		log.Fatalf("Could not migrate postgres database: %q", err)
	}

	cleanup := func() {
		_ = db.Close()
	}

	testDBInstance := &testDB{
		db:      db,
		cleanup: cleanup,
	}

	testDBs[cfg.DatabaseType] = testDBInstance

	return db, cleanup
}

func setupLoggerForTest() zerolog.Logger {
	return logger.New(&domain.Config{LogLevel: "ERROR"}, nil)
}

func TestPingDatabase(t *testing.T) {
	// Setup database
	for dbType, testDb := range testDBs {
		db := testDb.db
		t.Run(fmt.Sprintf("Test_db_ping [%s]", dbType), func(t *testing.T) {
			err := db.Ping()
			assert.NoError(t, err, "Database should be reachable")
		})
	}
}

func TestMain(m *testing.M) {
	if err := os.Setenv("IS_TEST_ENV", "true"); err != nil {
		log.Fatalf("Could not set env variable: %v", err)
	}

	testDBs = make(map[string]*testDB)

	fmt.Println("setup")

	startEmbeddedPG()
	setupSqliteForTest()

	fmt.Println("running tests")

	//Run tests
	code := m.Run()

	fmt.Println("teardown")
	for _, d := range testDBs {
		d.cleanup()
	}

	if err := os.Setenv("IS_TEST_ENV", "false"); err != nil {
		log.Fatalf("Could not set env variable: %v", err)
	}

	os.Exit(code)
}
