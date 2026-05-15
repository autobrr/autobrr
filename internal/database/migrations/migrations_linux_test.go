// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package migrations_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/database/migrations"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateOldVersionsToLatest(t *testing.T) {
	// This test downloads an old version of autobrr (v1.30.0), initializes a database with it,
	// then runs migrations to the latest version to ensure backward compatibility.

	version := "1.30.0"
	arch := "linux_x86_64"

	// Create temp directory for test
	tempDir := t.TempDir()
	//binaryPath := tempDir + "/autobrr"
	ctlBinaryPath := tempDir + "/autobrrctl"
	archiveName := "autobrr_" + version + "_" + arch + ".tar.gz"
	archivePath := tempDir + "/" + archiveName

	// Download the old version from GitHub releases
	downloadURL := "https://github.com/autobrr/autobrr/releases/download/v" + version + "/" + archiveName
	t.Logf("Downloading %s from %s", archiveName, downloadURL)

	// Download using curl
	cmd := "curl -L -o " + archivePath + " " + downloadURL
	err := executeCommand(t, cmd)
	require.NoError(t, err, "Failed to download old version")

	// Extract the archive
	t.Logf("Extracting %s", archiveName)
	cmd = "tar -xzf " + archivePath + " -C " + tempDir
	err = executeCommand(t, cmd)
	require.NoError(t, err, "Failed to extract archive")

	// Make binary executable
	cmd = "chmod +x " + ctlBinaryPath
	err = executeCommand(t, cmd)
	require.NoError(t, err, "Failed to make binary executable")

	t.Run("Test with sqlite", func(t *testing.T) {
		workDir := tempDir + "/sqlite"
		err = executeCommand(t, fmt.Sprintf("mkdir %s", workDir))
		require.NoError(t, err, "Failed to create work directory")

		configPath := workDir + "/config.toml"
		dbPath := workDir + "/autobrr.db"

		// Create minimal config file
		// Use a random port to avoid conflicts
		configContent := `# Minimal config for testing
logLevel = "ERROR"
checkForUpdates = false
host = "127.0.0.1"
port = 0

[database]
type = "sqlite"
`
		err = writeFile(configPath, configContent)
		require.NoError(t, err, "Failed to write config file")

		// Run the old version to initialize the database (with timeout)
		// Old version uses the config directory for the database by default
		t.Logf("Running v%s to initialize database", version)
		pass := "SuperSecretTestPassword"
		cmdRun := fmt.Sprintf("echo %q | %s --config=%s create-user testuser", pass, ctlBinaryPath, workDir)
		err = executeCommand(t, cmdRun)
		require.NoError(t, err, "Failed to run old version")

		// List files in temp directory for debugging
		//_ = executeCommand(t, "ls -la "+workDir)

		// Verify database was created
		require.FileExists(t, dbPath, "Database file should have been created by old version")

		// Now open the database with the current code and run migrations
		t.Logf("Running migrations from v%s to latest", version)
		cfg := &domain.Config{
			DatabaseType:        "sqlite",
			DatabaseDSN:         dbPath,
			DatabaseAutoMigrate: false,
		}

		log := logger.New(&domain.Config{LogLevel: "ERROR", LogPath: ""})
		db, err := database.NewDB(cfg, log)
		require.NoError(t, err, "Failed to create database connection")

		err = db.Open()
		require.NoError(t, err, "Failed to open database")
		defer db.Close()

		var premigrateVer int
		err = db.Handler.QueryRow("PRAGMA user_version").Scan(&premigrateVer)
		require.NoError(t, err, "should have a number of applied migrations")

		// Run all migrations
		migrate := migrations.SQLiteMigrations(db.Handler, log.With().Logger())
		err = migrate.Migrate()
		require.NoError(t, err, "Failed to run migrations from old version to latest")

		applied, err := migrate.CountApplied()
		require.NoError(t, err, "should have a number of applied migrations")
		assert.Equal(t, migrate.TotalMigrations(), applied, "should have all migrations applied")

		var ver int
		err = db.Handler.QueryRow("PRAGMA user_version").Scan(&ver)
		assert.Equal(t, 0, ver, "should have reset user_version to 0")

		// Verify we can query basic tables
		var count int
		err = db.Handler.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
		require.NoError(t, err, "Should be able to query users table")
		assert.Equal(t, 1, count, "Should be one user in users table")

		t.Logf("Successfully migrated from v%s to latest version", version)
	})

	t.Run("Test with postgres", func(t *testing.T) {
		workDir := tempDir + "/postgres"
		err = executeCommand(t, fmt.Sprintf("mkdir %s", workDir))
		require.NoError(t, err, "Failed to create work directory")

		configPath := workDir + "/config.toml"

		var (
			dbUsername = "postgres"
			dbPassword = "postgres"
			dbName     = "autobrr"
			dbPort     = 9877
		)

		pgLogger := &bytes.Buffer{}
		postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
			Username(dbUsername).
			Password(dbPassword).
			Database(dbName).
			Port(uint32(dbPort)).
			Version(embeddedpostgres.V17).
			//RuntimePath("/tmp").
			StartTimeout(45 * time.Second).
			StartParameters(map[string]string{"max_connections": "200"}).
			Logger(pgLogger))

		err := postgres.Start()
		require.NoError(t, err, "Failed to start postgres")

		dsn := fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?sslmode=disable", dbUsername, dbPassword, dbPort, dbName)

		// Create a minimal config file
		configContent := fmt.Sprintf(`# Minimal config for testing
logLevel = "ERROR"
checkForUpdates = false
host = "127.0.0.1"
port = 0

databaseType = "postgres"
postgresHost = %q
postgresPort = %d
postgresUser = %q
postgresPass = %q
postgresDatabase = %q
`, "localhost", dbPort, dbUsername, dbPassword, dbName)
		err = writeFile(configPath, configContent)
		require.NoError(t, err, "Failed to write config file")

		// Run the old version to initialize the database (with timeout)
		// Old version uses the config directory for the database by default
		t.Logf("Running v%s to initialize database", version)
		pass := "SuperSecretTestPassword"
		cmdRun := fmt.Sprintf("echo %q | %s --config=%s create-user testuser", pass, ctlBinaryPath, workDir)
		err = executeCommand(t, cmdRun)
		require.NoError(t, err, "Failed to run old version")

		// Now open the database with the current code and run migrations
		t.Logf("Running migrations from v%s to latest", version)
		cfg := &domain.Config{
			DatabaseType:        "postgres",
			DatabaseDSN:         dsn,
			DatabaseAutoMigrate: false,
		}

		log := logger.New(&domain.Config{LogLevel: "ERROR", LogPath: ""})
		db, err := database.NewDB(cfg, log)
		require.NoError(t, err, "Failed to create database connection")

		err = db.Open()
		require.NoError(t, err, "Failed to open database")
		defer db.Close()

		var premigrateVer int
		err = db.Handler.QueryRow("SELECT version FROM schema_migrations WHERE id = 1").Scan(&premigrateVer)
		require.NoError(t, err, "should have a number of applied migrations")

		// Run all migrations
		migrate := migrations.PostgresMigrations(db.Handler, log.With().Logger())
		err = migrate.Migrate()
		require.NoError(t, err, "Failed to run migrations from old version to latest")

		applied, err := migrate.CountApplied()
		require.NoError(t, err, "should have a number of applied migrations")
		assert.Equal(t, migrate.TotalMigrations(), applied, "should have all migrations applied")

		var userCount int
		err = db.Handler.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
		require.NoError(t, err, "Should be able to query users table")
		assert.Equal(t, 1, userCount, "Should be one user in users table")

		t.Logf("Successfully migrated from v%s to latest version", version)

		defer t.Cleanup(func() {
			t.Logf("Stopping postgres")
			postgres.Stop()
		})
	})

}

// Helper function to execute shell commands
func executeCommand(t *testing.T, cmd string) error {
	t.Helper()
	var err error
	// Use sh -c to execute the command
	cmdExec := []string{"sh", "-c", cmd}
	out, execErr := exec.Command(cmdExec[0], cmdExec[1:]...).CombinedOutput()
	t.Logf("Command output: %s", string(out))
	if execErr != nil {
		err = execErr
	}
	return err
}

// Helper function to write content to a file
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
