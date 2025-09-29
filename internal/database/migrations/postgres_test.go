package migrations_test

import (
	"testing"

	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/database/migrations"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/migrator/sqlite"

	"github.com/stretchr/testify/require"
)

// runMigrationTestPostgres executes a pluggable migration test
func runMigrationTestPostgres(t *testing.T, testCase MigrationTestCase) {
	db, cleanup := setupTestPostgresDB(t)
	defer cleanup()

	migrate := migrations.PostgresMigrations(db.Handler)

	// Run initial schema setup (all migrations up to the target migration - 1)
	m := migrate.GetUpTo(testCase.MigrationsUntilName)

	err := migrate.RunMigrations(m)
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

func setupTestPostgresDB(t *testing.T) (*database.DB, func()) {
	dsn := "postgres://postgres:postgres@localhost:5432/autobrr?sslmode=disable"

	cfg := &domain.Config{
		DatabaseType: "postgres",
		DatabaseDSN:  dsn,
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
	db, cleanup := setupTestPostgresDB(t)
	defer cleanup()

	// This will run all migrations
	migrate := migrations.PostgresMigrations(db.Handler)

	err := migrate.Migrate()
	require.NoError(t, err)

	//// Verify current schema version
	//var version int
	//err = db.Handler.QueryRow("PRAGMA user_version").Scan(&version)
	//require.NoError(t, err)
	//
	//expectedVersion := len(database.sqliteMigrations)
	//assert.Equal(t, expectedVersion, version, "Should be at latest migration version")
}
