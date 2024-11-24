// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"database/sql"
	"path"
)

func dataSourceName(configPath string, name string) string {
	if configPath != "" {
		return path.Join(configPath, name)
	}

	return name
}

func toNullString(s string) sql.Null[string] {
	return sql.Null[string]{
		V:     s,
		Valid: s != "",
	}
}

func toNullInt32(s int32) sql.NullInt32 {
	return sql.NullInt32{
		Int32: s,
		Valid: s != 0,
	}
}

func toNullInt64(s int64) sql.NullInt64 {
	return sql.NullInt64{
		Int64: s,
		Valid: s != 0,
	}
}

func toNullFloat64(s float64) sql.NullFloat64 {
	return sql.NullFloat64{
		Float64: s,
		Valid:   s != 0,
	}
}
