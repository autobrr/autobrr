// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package indexer

import (
	"bytes"
	"context"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// fakeIndexerRepo records the reconcile side effects; unused methods are no-op stubs.
type fakeIndexerRepo struct {
	list       []domain.Indexer
	upserted   []string
	archived   []string
	unarchived []string
}

func (f *fakeIndexerRepo) UpsertDeprecation(_ context.Context, d domain.IndexerDeprecation) error {
	f.upserted = append(f.upserted, d.Identifier)
	return nil
}
func (f *fakeIndexerRepo) ArchiveByIdentifier(_ context.Context, identifier string) error {
	f.archived = append(f.archived, identifier)
	return nil
}
func (f *fakeIndexerRepo) UnarchiveByIdentifier(_ context.Context, identifier string) error {
	f.unarchived = append(f.unarchived, identifier)
	return nil
}
func (f *fakeIndexerRepo) List(_ context.Context) ([]domain.Indexer, error) { return f.list, nil }
func (f *fakeIndexerRepo) ListDeprecations(_ context.Context) ([]domain.IndexerDeprecation, error) {
	return nil, nil
}

func (f *fakeIndexerRepo) Store(_ context.Context, i domain.Indexer) (*domain.Indexer, error) {
	return &i, nil
}
func (f *fakeIndexerRepo) Update(_ context.Context, i domain.Indexer) (*domain.Indexer, error) {
	return &i, nil
}
func (f *fakeIndexerRepo) Delete(_ context.Context, _ int) error { return nil }
func (f *fakeIndexerRepo) FindByFilterID(_ context.Context, _ int) ([]domain.Indexer, error) {
	return nil, nil
}
func (f *fakeIndexerRepo) FindByID(_ context.Context, _ int) (*domain.Indexer, error) {
	return nil, nil
}
func (f *fakeIndexerRepo) GetBy(_ context.Context, _ domain.GetIndexerRequest) (*domain.Indexer, error) {
	return nil, nil
}
func (f *fakeIndexerRepo) ToggleEnabled(_ context.Context, _ int, _ bool) error { return nil }

func TestReconcileDeprecations(t *testing.T) {
	fake := &fakeIndexerRepo{
		list: []domain.Indexer{
			{Identifier: "fnp"},           // orphan + in registry  -> archive
			{Identifier: "customreadded"}, // in mappedDefinitions   -> un-archive (revived)
			{Identifier: "activetracker"}, // live + not in registry -> untouched
			{Identifier: "forgotten"},     // orphan + not registered -> tripwire (no DB write)
		},
	}

	s := &Service{
		log:  zerolog.Nop(),
		repo: fake,
		// a re-added custom definition and a normal live indexer are present as definitions
		mappedDefinitions: map[string]*domain.IndexerDefinition{
			"activetracker": {Identifier: "activetracker"},
			"customreadded": {Identifier: "customreadded"},
		},
	}

	deprecations := []domain.IndexerDeprecation{
		{Identifier: "fnp", Name: "FearNoPeer"},
		{Identifier: "customreadded", Name: "Custom Re-Added"},
	}

	require.NoError(t, s.reconcileDeprecations(context.Background(), deprecations))

	// every registry entry has its metadata upserted
	assert.ElementsMatch(t, []string{"fnp", "customreadded"}, fake.upserted)
	// only the orphan (no live definition) gets archived
	assert.Equal(t, []string{"fnp"}, fake.archived)
	// the re-added one (present in mappedDefinitions) is un-archived, never archived
	assert.Equal(t, []string{"customreadded"}, fake.unarchived)
	assert.NotContains(t, fake.archived, "customreadded", "a re-added custom definition must not be archived")
	assert.NotContains(t, fake.archived, "activetracker")
}

// TestDeprecationsRegistry guards the embedded registry against the two mistakes that are easy
// to make when sunsetting an indexer: (1) registering an identifier that still has a live
// shipped definition (which would wrongly flag a working tracker as deprecated - e.g. "stc"
// was removed then re-added as skipthecommercials.yaml), and (2) malformed/duplicate entries.
func TestDeprecationsRegistry(t *testing.T) {
	// collect the identifier of every currently-shipped definition
	live := make(map[string]string) // identifier -> file name
	entries, err := fs.ReadDir(Definitions, "definitions")
	require.NoError(t, err)

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		data, err := fs.ReadFile(Definitions, "definitions/"+entry.Name())
		require.NoError(t, err)

		// minimal decode (no KnownFields) so it is tolerant of v1/v2 schemas
		var d struct {
			Identifier string `yaml:"identifier"`
		}
		require.NoErrorf(t, yaml.NewDecoder(bytes.NewReader(data)).Decode(&d), "decode %s", entry.Name())

		if d.Identifier != "" {
			live[d.Identifier] = entry.Name()
		}
	}
	require.NotEmpty(t, live, "expected embedded indexer definitions to be present")

	seen := make(map[string]struct{}, len(Deprecations))
	for _, dep := range Deprecations {
		if file, ok := live[dep.Identifier]; ok {
			t.Errorf("deprecation %q still has a live definition (%s) - remove it from the registry or it will mark a working tracker as deprecated", dep.Identifier, file)
		}

		assert.NotEmpty(t, dep.Identifier, "deprecation entry with empty Identifier")
		assert.NotEmptyf(t, dep.Name, "deprecation %q has empty Name", dep.Identifier)
		assert.NotEmptyf(t, dep.Reason, "deprecation %q has empty Reason", dep.Identifier)
		assert.Falsef(t, dep.DeprecatedAt.IsZero(), "deprecation %q has zero DeprecatedAt", dep.Identifier)

		if _, dup := seen[dep.Identifier]; dup {
			t.Errorf("duplicate deprecation identifier %q", dep.Identifier)
		}
		seen[dep.Identifier] = struct{}{}
	}
}
