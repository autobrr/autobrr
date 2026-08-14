// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	"sort"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/arr/chaptarr"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

func (s *Service) chaptarr(ctx context.Context, list *domain.List) error {
	l := s.log.With().Str("list", list.Name).Str("type", "chaptarr").Int("client", list.ClientID).Logger()

	l.Debug().Msg("gathering titles")

	titles, err := s.processChaptarr(ctx, list, &l)
	if err != nil {
		return err
	}

	l.Debug().Int("count", len(titles)).Msg("got filter titles")

	if len(titles) == 0 {
		l.Debug().Str("list", list.Name).Msg("no titles found to update")
		return nil
	}

	joinedTitles := strings.Join(titles, ",")

	l.Trace().Str("titles", joinedTitles).Int("count", len(titles)).Msg("found titles")

	filterUpdate := domain.FilterUpdate{MatchReleases: &joinedTitles}

	for _, filter := range list.Filters {
		l.Debug().Int("filter_id", filter.ID).Msg("updating filter")

		filterUpdate.ID = filter.ID

		if err := s.filterSvc.UpdatePartial(ctx, filterUpdate); err != nil {
			return errors.Wrap(err, "error updating filter: %v", filter.ID)
		}

		l.Debug().Int("filter_id", filter.ID).Msg("successfully updated filter")
	}

	return nil
}

func (s *Service) processChaptarr(ctx context.Context, list *domain.List, logger *zerolog.Logger) ([]string, error) {
	downloadClient, err := s.downloadClientSvc.GetClient(ctx, int32(list.ClientID))
	if err != nil {
		return nil, errors.Wrap(err, "could not get client with id %d", list.ClientID)
	}

	if !downloadClient.Enabled {
		return nil, errors.New("client %s %s not enabled", downloadClient.Type, downloadClient.Name)
	}

	client, ok := downloadClient.Client.(*chaptarr.Client)
	if !ok {
		return nil, errors.New("client %s %s is not a chaptarr client", downloadClient.Type, downloadClient.Name)
	}

	books, err := client.GetBooks(ctx, "")
	if err != nil {
		return nil, err
	}

	logger.Debug().Int("count", len(books)).Msg("found books to process")

	// a book held as both an audiobook and an ebook is two rows in Chaptarr,
	// so the same title comes back twice
	titleSet := make(map[string]struct{})
	var processedTitles int

	for _, book := range books {
		if !list.ShouldProcessItem(book.Monitored) {
			continue
		}

		processedTitles++

		for _, title := range processTitle(book.Title, list.MatchRelease) {
			titleSet[title] = struct{}{}
		}
	}

	uniqueTitles := make([]string, 0, len(titleSet))
	for title := range titleSet {
		uniqueTitles = append(uniqueTitles, title)
	}

	sort.Strings(uniqueTitles)
	logger.Debug().Int("total", len(books)).Int("processed", processedTitles).Int("created", len(uniqueTitles)).Msg("processed items")

	return uniqueTitles, nil
}
