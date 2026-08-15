// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package indexer

import (
	"context"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeIndexerRepo struct {
	list              []domain.Indexer
	deprecations      []domain.IndexerDeprecation
	activeIdentifiers map[string]struct{}
}

func (f *fakeIndexerRepo) ReconcileDeprecations(_ context.Context, deprecations []domain.IndexerDeprecation, activeIdentifiers map[string]struct{}) error {
	f.deprecations = deprecations
	f.activeIdentifiers = activeIdentifiers
	return nil
}
func (f *fakeIndexerRepo) List(_ context.Context) ([]domain.Indexer, error) { return f.list, nil }
func (f *fakeIndexerRepo) ListDeprecations(_ context.Context) ([]domain.IndexerDeprecation, error) {
	return nil, nil
}

func (f *fakeIndexerRepo) Store(_ context.Context, i domain.Indexer) (*domain.Indexer, error) {
	return &i, nil
}
func (f *fakeIndexerRepo) Update(_ context.Context, _ *domain.Indexer) error { return nil }
func (f *fakeIndexerRepo) Delete(_ context.Context, _ int) error             { return nil }
func (f *fakeIndexerRepo) DeleteArchived(_ context.Context, _ int) error     { return nil }
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
			{Identifier: "fnp"},
			{Identifier: "customreadded"},
			{Identifier: "activetracker"},
			{Identifier: "forgotten"},
		},
	}

	s := &Service{
		log:  zerolog.Nop(),
		repo: fake,
		definitions: map[string]domain.IndexerDefinition{
			"activetracker": {Identifier: "activetracker"},
			"customreadded": {Identifier: "customreadded"},
		},
	}

	deprecations := []domain.IndexerDeprecation{
		{Identifier: "fnp", Name: "FearNoPeer"},
		{Identifier: "customreadded", Name: "Custom Re-Added"},
	}

	require.NoError(t, s.reconcileDeprecations(context.Background(), deprecations))

	assert.Equal(t, deprecations, fake.deprecations)
	assert.Contains(t, fake.activeIdentifiers, "customreadded")
	assert.Contains(t, fake.activeIdentifiers, "activetracker")
	assert.NotContains(t, fake.activeIdentifiers, "fnp")
	assert.NotContains(t, fake.activeIdentifiers, "forgotten")
}

func TestDeprecatedIndexerDefinitions(t *testing.T) {
	deprecations, err := LoadDeprecatedIndexerDefinitions()
	require.NoError(t, err)
	require.NotEmpty(t, deprecations)

	service := &Service{definitions: map[string]domain.IndexerDefinition{}}
	require.NoError(t, service.LoadIndexerDefinitions())

	for _, deprecation := range deprecations {
		assert.NotContains(t, service.definitions, deprecation.Identifier, "deprecated indexer must not have an active bundled definition")
	}
}
