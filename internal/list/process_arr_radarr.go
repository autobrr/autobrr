// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/stringutils"
	"github.com/autobrr/autobrr/pkg/arr"
	"github.com/autobrr/autobrr/pkg/arr/radarr"

	"github.com/rs/zerolog"
)

type RadarrProcessor struct {
	processorBase
	client *radarr.Client
}

func NewRadarrProcessor(log zerolog.Logger, list *domain.List, client *radarr.Client) *RadarrProcessor {
	return &RadarrProcessor{
		log:    log,
		list:   list,
		client: client,
	}
}

func (p *RadarrProcessor) Process(ctx context.Context) (*domain.FilterUpdate, error) {
	movies, err := p.client.GetMovies(ctx, 0)
	if err != nil {
		return nil, err
	}

	var tags []arr.Tag
	if len(p.list.TagsExclude) > 0 || len(p.list.TagsInclude) > 0 {
		t, err := p.client.GetTags(ctx)
		if err != nil {
			p.log.Debug().Msg("could not get tags")
		}
		tags = t
	}

	filter, err := p.process(movies, tags)
	if err != nil {
		return nil, err
	}

	return filter, nil
}

func (p *RadarrProcessor) process(movies []radarr.Movie, tags []arr.Tag) (*domain.FilterUpdate, error) {
	p.log.Debug().Int("count", len(movies)).Msg("found movies to process")

	ts := NewTitleSet()
	ts.matchReleases = p.list.MatchRelease

	var processedTitles int

	for _, movie := range movies {
		if !p.list.ShouldProcessItem(movie.Monitored) {
			continue
		}

		if excludedByTags(p.list, tags, movie.Tags) {
			continue
		}

		processedTitles++

		// Taking the international title and the original title and appending them to the titles array.
		for _, title := range []string{movie.Title, movie.OriginalTitle} {
			ts.AddTitle(title)
		}

		if p.list.IncludeAlternateTitles {
			for _, title := range movie.AlternateTitles {
				ts.AddTitle(title.Title)
			}
		}
	}

	p.log.Debug().Int("total", len(movies)).Int("processed", processedTitles).Int("created", ts.Len()).Msg("processed items")

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
