// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"database/sql"

	"github.com/autobrr/autobrr/pkg/errors"
)

var (
	ErrRecordNotFound = sql.ErrNoRows
	ErrUpdateFailed   = errors.New("update failed")
	ErrDeleteFailed   = errors.New("delete failed")
)
