// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"database/sql"

	"github.com/autobrr/autobrr/pkg/errors"
)

var (
	ErrRecordNotFound                 = sql.ErrNoRows
	ErrUpdateFailed                   = errors.New("update failed")
	ErrDeleteFailed                   = errors.New("delete failed")
	ErrNoActiveFiltersFoundForIndexer = errors.New("no active filters found for indexer")
	ErrUnexpectedLine                 = errors.New("unexpected line")
	ErrIndexerNotFound                = errors.New("indexer not found")
	ErrIndexerArchived                = errors.New("indexer is archived")
	ErrIndexerNotArchived             = errors.New("indexer is not archived")
	ErrIndexerInUse                   = errors.New("indexer is still used by filters")
	ErrNotificationNotFound           = errors.New("notification not found")
	ErrIRCNetworkHandlerNotFound      = errors.New("could not find network handler")
)
