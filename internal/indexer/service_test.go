// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package indexer

import (
	"context"
	"maps"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/asaskevich/EventBus"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type stubIndexerRepo struct {
	current *domain.Indexer
	updated *domain.Indexer
	deleted bool
	toggled bool
}

func (r *stubIndexerRepo) Store(_ context.Context, indexer domain.Indexer) (*domain.Indexer, error) {
	return &indexer, nil
}

func (r *stubIndexerRepo) Update(_ context.Context, indexer *domain.Indexer) error {
	r.updated = indexer
	return nil
}

func (r *stubIndexerRepo) List(_ context.Context) ([]domain.Indexer, error) { return nil, nil }

func (r *stubIndexerRepo) Delete(_ context.Context, _ int) error {
	r.deleted = true
	return nil
}

func (r *stubIndexerRepo) DeleteArchived(_ context.Context, _ int) error { return nil }

func (r *stubIndexerRepo) FindByFilterID(_ context.Context, _ int) ([]domain.Indexer, error) {
	return nil, nil
}

func (r *stubIndexerRepo) FindByID(_ context.Context, _ int) (*domain.Indexer, error) {
	indexer := *r.current
	settings := make(map[string]string, len(r.current.Settings))
	maps.Copy(settings, r.current.Settings)
	indexer.Settings = settings

	return &indexer, nil
}

func (r *stubIndexerRepo) GetBy(_ context.Context, _ domain.GetIndexerRequest) (*domain.Indexer, error) {
	return nil, nil
}

func (r *stubIndexerRepo) ToggleEnabled(_ context.Context, _ int, _ bool) error {
	r.toggled = true
	return nil
}

func (r *stubIndexerRepo) ReconcileDeprecations(_ context.Context, _ []domain.IndexerDeprecation, _ map[string]struct{}) error {
	return nil
}

func (r *stubIndexerRepo) ListDeprecations(_ context.Context) ([]domain.IndexerDeprecation, error) {
	return nil, nil
}

func TestServiceUpdate_SecretSettings(t *testing.T) {
	t.Parallel()

	current := &domain.Indexer{
		ID:             1,
		Name:           "Test RSS",
		Identifier:     "rss-test-rss",
		Implementation: "rss",
		Enabled:        true,
		Settings:       map[string]string{"api_key": "secret123", "host": "https://example.org"},
	}

	tests := []struct {
		name        string
		settings    map[string]string
		wantErr     string
		wantUpdated map[string]string
	}{
		{
			name:     "omitting a saved secret is rejected",
			settings: map[string]string{},
			wantErr:  "update omits saved secret setting 'api_key'",
		},
		{
			name:     "nil settings are rejected",
			settings: nil,
			wantErr:  "update omits saved secret setting 'api_key'",
		},
		{
			name:        "redacted placeholder restores the saved value",
			settings:    map[string]string{"api_key": domain.RedactedStr, "host": "https://example.org"},
			wantUpdated: map[string]string{"api_key": "secret123", "host": "https://example.org"},
		},
		{
			name:        "explicit new value replaces the saved one",
			settings:    map[string]string{"api_key": "rotated", "host": "https://example.org"},
			wantUpdated: map[string]string{"api_key": "rotated", "host": "https://example.org"},
		},
		{
			name:        "non-secret settings may be omitted",
			settings:    map[string]string{"api_key": domain.RedactedStr},
			wantUpdated: map[string]string{"api_key": "secret123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &stubIndexerRepo{current: current}

			svc := NewService(zerolog.Nop(), nil, EventBus.New(), repo, nil, nil)
			svc.mappedDefinitions[current.Identifier] = &domain.IndexerDefinition{Identifier: current.Identifier, Implementation: "rss"}

			update := &domain.Indexer{ID: 1, Name: "Test RSS", Identifier: current.Identifier, Implementation: "rss", Enabled: true, Settings: tt.settings}

			err := svc.Update(t.Context(), update)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, repo.updated, "a rejected update must not persist")
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, repo.updated)
			assert.Equal(t, tt.wantUpdated, repo.updated.Settings)
		})
	}
}

func TestServiceUpdate_EmptySavedSecretMayBeOmitted(t *testing.T) {
	t.Parallel()

	repo := &stubIndexerRepo{current: &domain.Indexer{
		ID:             1,
		Name:           "Test RSS",
		Identifier:     "rss-test-rss",
		Implementation: "rss",
		Settings:       map[string]string{"api_key": ""},
	}}

	svc := NewService(zerolog.Nop(), nil, EventBus.New(), repo, nil, nil)
	svc.mappedDefinitions["rss-test-rss"] = &domain.IndexerDefinition{Identifier: "rss-test-rss", Implementation: "rss"}

	err := svc.Update(t.Context(), &domain.Indexer{ID: 1, Name: "Test RSS", Identifier: "rss-test-rss", Implementation: "rss", Settings: map[string]string{}})

	assert.NoError(t, err)
	assert.NotNil(t, repo.updated)
}

func TestServiceRejectsArchivedMutations(t *testing.T) {
	tests := []struct {
		name string
		run  func(*Service) error
	}{
		{
			name: "update",
			run: func(service *Service) error {
				return service.Update(t.Context(), &domain.Indexer{ID: 1})
			},
		},
		{
			name: "delete",
			run: func(service *Service) error {
				return service.Delete(t.Context(), 1)
			},
		},
		{
			name: "toggle",
			run: func(service *Service) error {
				return service.ToggleEnabled(t.Context(), 1, false)
			},
		},
		{
			name: "api test",
			run: func(service *Service) error {
				return service.TestApi(t.Context(), domain.IndexerTestApiRequest{IndexerId: 1})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &stubIndexerRepo{current: &domain.Indexer{ID: 1, Archived: true, Settings: map[string]string{}}}
			service := NewService(zerolog.Nop(), nil, EventBus.New(), repo, nil, nil)

			assert.ErrorIs(t, tt.run(service), domain.ErrIndexerArchived)
			assert.Nil(t, repo.updated)
			assert.False(t, repo.deleted)
			assert.False(t, repo.toggled)
		})
	}
}
