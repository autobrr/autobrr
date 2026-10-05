// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"strconv"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/newznab"

	"github.com/rs/zerolog"
)

type newznabSource struct {
	log    zerolog.Logger
	feed   *domain.Feed
	client *newznab.Client
}

func (s *newznabSource) fetch(ctx context.Context) (*fetchResult, error) {
	feed, err := s.client.Search(ctx, "", s.feed.Categories)
	if err != nil {
		return nil, errors.Wrap(err, "error fetching feed items")
	}

	res := &fetchResult{raw: feed.Raw, entries: make([]entry, 0, len(feed.Items))}

	for _, item := range feed.Items {
		if item.GUID == "" {
			s.log.Error().Str("title", item.Title).Msg("item missing GUID")
			continue
		}

		res.entries = append(res.entries, entry{
			key:     item.GUID,
			title:   item.Title,
			pubDate: item.PubDate.Time,
			release: func() *domain.Release { return s.toRelease(item) },
		})
	}

	return res, nil
}

func (s *newznabSource) toRelease(item *newznab.FeedItem) *domain.Release {
	rls := domain.NewRelease(s.feed.Indexer)
	rls.Implementation = domain.ReleaseImplementationNewznab
	rls.Protocol = domain.ReleaseProtocolNzb

	rls.TorrentName = item.Title
	rls.InfoURL = item.GUID

	rls.ParseString(item.Title)

	rls.MetaIMDB = item.ImdbId
	if item.TmdbId != "" {
		if tmdbId, err := strconv.Atoi(item.TmdbId); err == nil {
			rls.MetaTMDB = tmdbId
		}
	}

	rls.Size = item.Size

	if item.Enclosure != nil && item.Enclosure.Type == "application/x-nzb" {
		rls.DownloadURL = item.Enclosure.Url
		if rls.Size == 0 && item.Enclosure.Length > item.Size {
			rls.Size = item.Enclosure.Length
		}
	}

	if len(item.Categories) == 1 {
		rls.Category = item.Categories[0].Name
	}

	// map newznab categories ID and Name into rls.Categories
	// so we can filter on both ID and Name
	for _, category := range item.Categories {
		rls.Categories = append(rls.Categories, []string{category.Name, strconv.Itoa(category.ID)}...)
	}

	return rls
}
