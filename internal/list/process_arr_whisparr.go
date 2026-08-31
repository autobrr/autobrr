// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/stringutils"
	"github.com/autobrr/autobrr/pkg/arr"
	"github.com/autobrr/autobrr/pkg/arr/whisparr"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

type WhisparrProcessor struct {
	processorBase
	client *whisparr.Client
}

func NewWhisparrProcessor(log zerolog.Logger, list *domain.List, client *whisparr.Client) *WhisparrProcessor {
	return &WhisparrProcessor{
		log:    log,
		list:   list,
		client: client,
	}
}

func (p *WhisparrProcessor) Process(ctx context.Context) (*domain.FilterUpdate, error) {
	// Test rejects a version mismatch, so a v2 instance configured as v3 is
	// reported as such instead of failing with a 404 on the item endpoint below.
	status, err := p.client.Test(ctx)
	if err != nil {
		return nil, err
	}

	p.log.Debug().Str("version", status.Version).Msg("connected to whisparr")

	var tags []arr.Tag
	if len(p.list.TagsExclude) > 0 || len(p.list.TagsInclude) > 0 {
		t, err := p.client.GetTags(ctx)
		if err != nil {
			p.log.Debug().Msg("could not get tags")
		}
		tags = t
	}

	ts := NewTitleSet()
	ts.matchReleases = p.list.MatchRelease

	var total, processedTitles int

	//switch p.arrVersion {
	//case domain.DownloadClientTypeWhisparr:
	switch p.list.Type {
	case domain.ListTypeWhisparr:

		series, err := p.client.GetAllSeries(ctx)
		if err != nil {
			return nil, err
		}

		p.log.Debug().Int("count", len(series)).Msg("found series to process")

		total = len(series)

		for _, show := range series {
			if !p.list.ShouldProcessItem(show.Monitored) {
				continue
			}

			if excludedByTags(p.list, tags, show.Tags) {
				continue
			}

			processedTitles++

			ts.AddTitle(show.Title)
		}

	//case domain.DownloadClientTypeWhisparrV3:
	case domain.ListTypeWhisparrV3:
		movies, err := p.client.GetMovies(ctx)
		if err != nil {
			return nil, err
		}

		p.log.Debug().Int("count", len(movies)).Msg("found movies to process")

		total = len(movies)

		for _, movie := range movies {
			if !p.list.ShouldProcessItem(movie.Monitored) {
				continue
			}

			if excludedByTags(p.list, tags, movie.Tags) {
				continue
			}

			processedTitles++

			ts.AddTitle(movie.Title)

			if p.list.IncludeAlternateTitles {
				for _, title := range movie.AlternateTitles {
					ts.AddTitle(title.Title)
				}
			}
		}

	default:
		return nil, errors.New("client %s is not a whisparr client", p.list.Type)
	}

	p.log.Debug().Int("total", total).Int("processed", processedTitles).Int("created", ts.Len()).Msg("processed items")

	if ts.Len() == 0 {
		p.log.Debug().Str("list", p.list.Name).Msg("no titles found to update")
		return nil, nil
	}

	joinedTitles := ts.FilterString()

	p.log.Trace().Str("titles", stringutils.TruncateStr(joinedTitles, 1024)).Int("count", ts.Len()).Msg("found titles")

	filter := domain.FilterUpdate{Shows: &joinedTitles}

	if p.list.MatchRelease {
		filter.Shows = new("")
		filter.MatchReleases = &joinedTitles
	}

	return &filter, nil
}

// excludedByTags reports whether the tags of an item exclude it from the list.
func excludedByTags(list *domain.List, tags []arr.Tag, itemTags []int) bool {
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
