// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	"sort"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/arr"
	"github.com/autobrr/autobrr/pkg/arr/sportarr"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

func (s *Service) sportarr(ctx context.Context, list *domain.List) error {
	l := s.log.With().Str("list", list.Name).Str("type", "sportarr").Int("client", list.ClientID).Logger()

	l.Debug().Msg("gathering titles")

	titles, err := s.processSportarr(ctx, list, &l)
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

func (s *Service) processSportarr(ctx context.Context, list *domain.List, logger *zerolog.Logger) ([]string, error) {
	downloadClient, err := s.downloadClientSvc.GetClient(ctx, int32(list.ClientID)) // #nosec G115
	if err != nil {
		return nil, errors.Wrap(err, "could not get client with id %d", list.ClientID)
	}

	if !downloadClient.Enabled {
		return nil, errors.New("client %s %s not enabled", downloadClient.Type, downloadClient.Name)
	}

	client, ok := downloadClient.Client.(*sportarr.Client)
	if !ok {
		return nil, errors.New("client %s %s is not a sportarr client", downloadClient.Type, downloadClient.Name)
	}

	var tags []*arr.Tag
	if len(list.TagsExclude) > 0 || len(list.TagsInclude) > 0 {
		t, err := client.GetTags(ctx)
		if err != nil {
			logger.Debug().Msg("could not get tags")
		}
		tags = t
	}

	leagues, err := client.GetAllLeagues(ctx)
	if err != nil {
		return nil, err
	}

	logger.Debug().Int("count", len(leagues)).Msg("found leagues to process")

	titleSet := make(map[string]struct{})
	var processedTitles int

	for _, league := range leagues {
		if !list.ShouldProcessItem(league.Monitored) {
			continue
		}

		if len(list.TagsInclude) > 0 {
			if len(league.Tags) == 0 {
				continue
			}
			if !containsTag(tags, league.Tags, list.TagsInclude) {
				continue
			}
		}

		if len(list.TagsExclude) > 0 {
			if containsTag(tags, league.Tags, list.TagsExclude) {
				continue
			}
		}

		processedTitles++

		titles := processTitle(league.Name, list.MatchRelease)
		for _, title := range titles {
			titleSet[title] = struct{}{}
		}
	}

	uniqueTitles := make([]string, 0, len(titleSet))
	for title := range titleSet {
		uniqueTitles = append(uniqueTitles, title)
	}

	sort.Strings(uniqueTitles)
	logger.Debug().Int("total", len(leagues)).Int("processed", processedTitles).Int("created", len(uniqueTitles)).Msg("processed items")

	return uniqueTitles, nil
}
