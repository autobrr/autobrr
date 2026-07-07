// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package indexer

import (
	"time"

	"github.com/autobrr/autobrr/internal/domain"
)

// Deprecations is the canonical, embedded source of truth for indexers that have been
// removed/retired from the shipped definitions.
//
// To sunset an indexer:
//  1. delete internal/indexer/definitions/<identifier>.yaml
//  2. append one entry below
//
// On every boot, reconcileDeprecations() projects this list into the database:
// it upserts the metadata into the indexer_deprecation table and flips the archived
// flag on any matching (orphaned) indexer row. There is no per-removal SQL migration.
//
// Notes:
//   - Identifier must match the removed definition's identifier (and the value stored in
//     release.indexer / indexer.identifier).
//   - Name is the friendly display name; it powers name resolution even for users who
//     hard-deleted the indexer row before upgrading (COALESCE(i.name, d.name, r.indexer)).
//   - AliasOf is display-only. True renames (where the tracker lives on under a new
//     identifier) are handled by a rename migration instead (see migration 92,
//     rotorrent -> seedcore); use AliasOf only to hint at a successor for the UI.
//   - DeprecatedAt is stamped in code (a fixed date), never time.Now(), so reconcile is
//     deterministic across restarts.
var Deprecations = []domain.IndexerDeprecation{
	{
		Identifier:   "fnp",
		Name:         "FearNoPeer",
		Reason:       "Tracker shut down",
		IssueURL:     "https://github.com/autobrr/autobrr/pull/2453",
		DeprecatedAt: time.Date(2026, time.May, 11, 0, 0, 0, 0, time.UTC),
	},
	{
		Identifier:   "ianon",
		Name:         "iAnon",
		Reason:       "Tracker shut down",
		IssueURL:     "https://github.com/autobrr/autobrr/pull/2221",
		DeprecatedAt: time.Date(2025, time.October, 15, 0, 0, 0, 0, time.UTC),
	},
	{
		Identifier:   "lillesky",
		Name:         "LilleSky",
		Reason:       "Tracker shut down",
		IssueURL:     "https://github.com/autobrr/autobrr/pull/1735",
		DeprecatedAt: time.Date(2024, time.September, 21, 0, 0, 0, 0, time.UTC),
	},
	{
		Identifier:   "lusthive",
		Name:         "LustHive",
		Reason:       "Tracker shut down",
		IssueURL:     "https://github.com/autobrr/autobrr/pull/2007",
		DeprecatedAt: time.Date(2025, time.March, 23, 0, 0, 0, 0, time.UTC),
	},
	{
		Identifier:   "oppaitime",
		Name:         "OppaiTime",
		Reason:       "Tracker shut down",
		IssueURL:     "https://github.com/autobrr/autobrr/pull/631",
		DeprecatedAt: time.Date(2023, time.January, 8, 0, 0, 0, 0, time.UTC),
	},
	{
		Identifier:   "polishsource",
		Name:         "PolishSource",
		Reason:       "Tracker shut down",
		IssueURL:     "https://github.com/autobrr/autobrr/pull/1943",
		DeprecatedAt: time.Date(2025, time.January, 18, 0, 0, 0, 0, time.UTC),
	},
	{
		Identifier:   "ptn",
		Name:         "Piratethenet",
		Reason:       "Tracker shut down",
		IssueURL:     "https://github.com/autobrr/autobrr/pull/1185",
		DeprecatedAt: time.Date(2023, time.October, 16, 0, 0, 0, 0, time.UTC),
	},
	{
		Identifier:   "stt",
		Name:         "SkipTheTrailers",
		Reason:       "Tracker shut down",
		IssueURL:     "https://github.com/autobrr/autobrr/pull/1708",
		DeprecatedAt: time.Date(2024, time.September, 4, 0, 0, 0, 0, time.UTC),
	},
	{
		Identifier:   "tfm",
		Name:         "ToonsForMe",
		Reason:       "Tracker shut down",
		IssueURL:     "https://github.com/autobrr/autobrr/pull/1407",
		DeprecatedAt: time.Date(2024, time.February, 14, 0, 0, 0, 0, time.UTC),
	},
	{
		Identifier:   "torrentdb",
		Name:         "TorrentDB",
		Reason:       "Tracker shut down",
		IssueURL:     "https://github.com/autobrr/autobrr/pull/626",
		DeprecatedAt: time.Date(2023, time.January, 6, 0, 0, 0, 0, time.UTC),
	},
	{
		Identifier:   "torrentseeds",
		Name:         "TorrentSeeds",
		Reason:       "Tracker shut down",
		IssueURL:     "https://github.com/autobrr/autobrr/pull/2040",
		DeprecatedAt: time.Date(2025, time.April, 22, 0, 0, 0, 0, time.UTC),
	},
	{
		Identifier:   "torrentseeds-music",
		Name:         "TorrentSeeds Music",
		Reason:       "Tracker shut down",
		IssueURL:     "https://github.com/autobrr/autobrr/pull/2040",
		DeprecatedAt: time.Date(2025, time.April, 22, 0, 0, 0, 0, time.UTC),
	},
	{
		Identifier:   "tsc",
		Name:         "TorrentSectorCrew",
		Reason:       "Tracker shut down",
		IssueURL:     "https://github.com/autobrr/autobrr/pull/1917",
		DeprecatedAt: time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
	},
	{
		Identifier:   "uhdbits",
		Name:         "UHDBits",
		Reason:       "Tracker shut down",
		IssueURL:     "https://github.com/autobrr/autobrr/pull/2361",
		DeprecatedAt: time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
	},
}
