// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package migrations

import "embed"

var (
	//go:embed sqlite/*.sql
	SchemaMigrationsSQLite embed.FS

	//go:embed postgres/*.sql
	SchemaMigrationsPostgres embed.FS
)
