// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	"sort"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/arr"
	"github.com/autobrr/autobrr/pkg/arr/whisparr"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

func (s *Service) whisparr(ctx context.Context, list *domain.List) error {
	l := s.log.With().Str("list", list.Name).Str("type", "whisparr").Int("client", list.ClientID).Logger()

	l.Debug().Msg("gathering titles")

	titles, err := s.processWhisparr(ctx, list, &l)
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

	filterUpdate := domain.FilterUpdate{Shows: &joinedTitles}

	if list.MatchRelease {
		filterUpdate.Shows = &nullString
		filterUpdate.MatchReleases = &joinedTitles
	}

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

func (s *Service) processWhisparr(ctx context.Context, list *domain.List, logger *zerolog.Logger) ([]string, error) {
	downloadClient, err := s.downloadClientSvc.GetClient(ctx, int32(list.ClientID)) // #nosec G115
	if err != nil {
		return nil, errors.Wrap(err, "could not get client with id %d", list.ClientID)
	}

	if !downloadClient.Enabled {
		return nil, errors.New("client %s %s not enabled", downloadClient.Type, downloadClient.Name)
	}

	client, ok := downloadClient.Client.(*whisparr.Client)
	if !ok {
		return nil, errors.New("client %s %s is not a whisparr client", downloadClient.Type, downloadClient.Name)
	}

	// Test rejects a version mismatch, so a v2 instance configured as v3 is
	// reported as such instead of failing with a 404 on the item endpoint below.
	status, err := client.Test(ctx)
	if err != nil {
		return nil, err
	}

	logger.Debug().Str("version", status.Version).Msg("connected to whisparr")

	var tags []*arr.Tag
	if len(list.TagsExclude) > 0 || len(list.TagsInclude) > 0 {
		t, err := client.GetTags(ctx)
		if err != nil {
			logger.Debug().Err(err).Msg("could not get tags")
		}
		tags = t
	}

	titleSet := make(map[string]struct{})
	addTitle := func(title string) {
		for _, t := range processTitle(title, list.MatchRelease) {
			titleSet[t] = struct{}{}
		}
	}

	var total, processedTitles int

	switch downloadClient.Type {
	case domain.DownloadClientTypeWhisparr:
		series, err := client.GetAllSeries(ctx)
		if err != nil {
			return nil, err
		}

		logger.Debug().Int("count", len(series)).Msg("found series to process")

		total = len(series)

		for _, show := range series {
			if !list.ShouldProcessItem(show.Monitored) {
				continue
			}

			if excludedByTags(list, tags, show.Tags) {
				continue
			}

			processedTitles++

			addTitle(show.Title)
		}

	case domain.DownloadClientTypeWhisparrV3:
		movies, err := client.GetMovies(ctx)
		if err != nil {
			return nil, err
		}

		logger.Debug().Int("count", len(movies)).Msg("found movies to process")

		total = len(movies)

		for _, movie := range movies {
			if !list.ShouldProcessItem(movie.Monitored) {
				continue
			}

			if excludedByTags(list, tags, movie.Tags) {
				continue
			}

			processedTitles++

			addTitle(movie.Title)

			if list.IncludeAlternateTitles {
				for _, title := range movie.AlternateTitles {
					addTitle(title.Title)
				}
			}
		}

	default:
		return nil, errors.New("client %s %s is not a whisparr client", downloadClient.Type, downloadClient.Name)
	}

	uniqueTitles := make([]string, 0, len(titleSet))
	for title := range titleSet {
		uniqueTitles = append(uniqueTitles, title)
	}

	sort.Strings(uniqueTitles)
	logger.Debug().Int("total", total).Int("processed", processedTitles).Int("created", len(uniqueTitles)).Msg("processed items")

	return uniqueTitles, nil
}

// excludedByTags reports whether the tags of an item exclude it from the list.
func excludedByTags(list *domain.List, tags []*arr.Tag, itemTags []int) bool {
	if len(list.TagsInclude) > 0 {
		if len(itemTags) == 0 {
			return true
		}

		if !containsTag(tags, itemTags, list.TagsInclude) {
			return true
		}
	}

	if len(list.TagsExclude) > 0 && containsTag(tags, itemTags, list.TagsExclude) {
		return true
	}

	return false
}
