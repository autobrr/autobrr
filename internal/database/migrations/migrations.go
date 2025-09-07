package migrations

import "embed"

var (
	// go:embed *.sql
	SchemaMigrations embed.FS

	//go:embed sqlite/*.sql
	SchemaMigrationsSQLite embed.FS

	// go:embed postgres/*.sql
	SchemaMigrationsPostgres embed.FS
)
