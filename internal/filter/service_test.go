// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package filter

import (
	"context"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type indexerSvcStub struct {
	indexers []domain.Indexer
}

func (s *indexerSvcStub) FindByFilterID(_ context.Context, _ int) ([]domain.Indexer, error) {
	return s.indexers, nil
}

func (s *indexerSvcStub) List(_ context.Context) ([]domain.Indexer, error) {
	return s.indexers, nil
}

// filterRepoStub answers the existence check; every other repo method panics
// through the embedded nil interface, so reaching persistence fails the test.
type filterRepoStub struct {
	filterRepo
	filter *domain.Filter
}

func (s *filterRepoStub) FindByID(_ context.Context, filterID int) (*domain.Filter, error) {
	if s.filter == nil || s.filter.ID != filterID {
		return nil, domain.ErrRecordNotFound
	}
	return s.filter, nil
}

func TestService_validateIndexers(t *testing.T) {
	svc := &Service{
		log:        zerolog.Nop(),
		indexerSvc: &indexerSvcStub{indexers: []domain.Indexer{{ID: 1}, {ID: 2}}},
	}

	t.Run("accepts existing indexers", func(t *testing.T) {
		assert.NoError(t, svc.validateIndexers(t.Context(), []domain.Indexer{{ID: 1}, {ID: 2}}))
	})

	t.Run("accepts empty selection", func(t *testing.T) {
		assert.NoError(t, svc.validateIndexers(t.Context(), nil))
	})

	t.Run("rejects unknown indexer", func(t *testing.T) {
		err := svc.validateIndexers(t.Context(), []domain.Indexer{{ID: 1}, {ID: 99}})
		assert.ErrorIs(t, err, domain.ErrIndexerNotFound)
	})
}

func TestService_Update_ValidatesIndexersBeforePersisting(t *testing.T) {
	svc := &Service{
		log:        zerolog.Nop(),
		repo:       &filterRepoStub{filter: &domain.Filter{ID: 1}},
		indexerSvc: &indexerSvcStub{indexers: []domain.Indexer{{ID: 1}}},
	}

	err := svc.Update(t.Context(), &domain.Filter{
		ID:       1,
		Name:     "filter",
		Indexers: []domain.Indexer{{ID: 99}},
	})
	assert.ErrorIs(t, err, domain.ErrIndexerNotFound)
}

func TestService_Update_MissingFilterWinsOverUnknownIndexer(t *testing.T) {
	svc := &Service{
		log:        zerolog.Nop(),
		repo:       &filterRepoStub{},
		indexerSvc: &indexerSvcStub{indexers: []domain.Indexer{{ID: 1}}},
	}

	err := svc.Update(t.Context(), &domain.Filter{
		ID:       99,
		Name:     "filter",
		Indexers: []domain.Indexer{{ID: 99}},
	})
	assert.ErrorIs(t, err, domain.ErrRecordNotFound)
}

func TestService_UpdatePartial_ValidatesIndexersBeforePersisting(t *testing.T) {
	svc := &Service{
		log:        zerolog.Nop(),
		repo:       &filterRepoStub{filter: &domain.Filter{ID: 1}},
		indexerSvc: &indexerSvcStub{indexers: []domain.Indexer{{ID: 1}}},
	}

	err := svc.UpdatePartial(t.Context(), domain.FilterUpdate{
		ID:       1,
		Indexers: []domain.Indexer{{ID: 99}},
	})
	assert.ErrorIs(t, err, domain.ErrIndexerNotFound)
}

func TestService_UpdatePartial_MissingFilterWinsOverUnknownIndexer(t *testing.T) {
	svc := &Service{
		log:        zerolog.Nop(),
		repo:       &filterRepoStub{},
		indexerSvc: &indexerSvcStub{indexers: []domain.Indexer{{ID: 1}}},
	}

	err := svc.UpdatePartial(t.Context(), domain.FilterUpdate{
		ID:       99,
		Indexers: []domain.Indexer{{ID: 99}},
	})
	assert.ErrorIs(t, err, domain.ErrRecordNotFound)
}
