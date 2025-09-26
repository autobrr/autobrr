package migrations

import (
	"database/sql"

	migrator "github.com/autobrr/autobrr/pkg/migrator/sqlite"
	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func PostgresMigrations(db *sql.DB) *migrator.Migrator {
	migrate := migrator.NewMigrate(
		db,
		migrator.WithEmbedFS(SchemaMigrationsPostgres),
		migrator.WithSchemaFile("current_schema_postgres.sql"),
		migrator.WithLogger(zstdlog.NewStdLoggerWithLevel(log.With().Str("module", "database-migrations").Logger(), zerolog.InfoLevel)),
	)

	migrate.AddFileMigration("0_base_schema_postgres.sql")

	return migrate
}
