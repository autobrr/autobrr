package database

import (
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func getDbs() []string {
	return []string{"sqlite", "postgres"}
}

func setupDatabaseForTest(t *testing.T, dbType string) *DB {
	err := os.Setenv("IS_TEST_ENV", "true")
	if err != nil {
		t.Fatalf("Could not set env variable: %v", err)
		return nil
	}

	cfg := &domain.Config{
		LogLevel:         "INFO",
		DatabaseType:     dbType,
		PostgresHost:     "localhost",
		PostgresPort:     5437,
		PostgresDatabase: "autobrr",
		PostgresUser:     "testdb",
		PostgresPass:     "testdb",
		PostgresSSLMode:  "disable",
	}

	// Init a new logger
	log := logger.New(cfg)

	// Initialize a new DB connection
	db, err := NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Could not create database: %v", err)
	}

	// Open the database connection
	if err := db.Open(); err != nil {
		t.Fatalf("Could not open db connection: %v", err)
	}

	return db
}

func setupLoggerForTest() logger.Logger {
	cfg := &domain.Config{
		LogLevel: "INFO",
	}
	log := logger.New(cfg)

	return log
}

func TestPingDatabase(t *testing.T) {
	// Setup database
	for _, dbType := range getDbs() {
		db := setupDatabaseForTest(t, dbType)
		defer db.Close()

		// Call the Ping method
		err := db.Ping()

		assert.NoError(t, err, "Database should be reachable")
	}
}
