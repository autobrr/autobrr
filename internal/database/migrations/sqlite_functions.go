// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package migrations

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/hex"

	"github.com/autobrr/autobrr/pkg/errors"

	"modernc.org/sqlite"
)

// SQLite has no built-in hash functions, and migrations that hash stored secrets need one. Registering on the
// driver makes it available to every connection opened afterwards, which init guarantees.
func init() {
	if err := sqlite.RegisterDeterministicScalarFunction("sha256_hex", 1, sha256Hex); err != nil {
		panic(err)
	}
}

func sha256Hex(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
	var data []byte

	switch v := args[0].(type) {
	case nil:
		return nil, nil
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return nil, errors.New("sha256_hex: unsupported argument type %T", v)
	}

	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:]), nil
}
