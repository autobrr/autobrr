// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/stringutils"
	"github.com/autobrr/autobrr/pkg/arr"
	"github.com/autobrr/autobrr/pkg/arr/sonarr"
	"github.com/rs/zerolog"
)

type SonarrProcessor struct {
	processorBase
	client *sonarr.Client
}

func NewSonarrProcessor(log zerolog.Logger, list *domain.List, client *sonarr.Client) *SonarrProcessor {
	return &SonarrProcessor{
		log:    log,
		list:   list,
		client: client,
	}
}

func (p *SonarrProcessor) Process(ctx context.Context) (*domain.FilterUpdate, error) {
	movies, err := p.client.GetSeries(ctx, 0)
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

func (p *SonarrProcessor) process(shows []sonarr.Series, tags []arr.Tag) (*domain.FilterUpdate, error) {
	p.log.Debug().Int("count", len(shows)).Msg("found shows to process")

	ts := NewTitleSet()
	ts.matchReleases = p.list.MatchRelease
	var processedTitles int

	for _, show := range shows {
		if !p.list.ShouldProcessItem(show.Monitored) {
			continue
		}

		if excludedByTags(p.list, tags, show.Tags) {
			continue
		}

		processedTitles++

		ts.AddTitle(show.Title)
		if p.list.IncludeAlternateTitles {
			for _, title := range show.AlternateTitles {
				ts.AddTitle(title.Title)
			}
		}
	}

	p.log.Debug().Int("total", len(shows)).Int("processed", processedTitles).Int("created", ts.Len()).Msg("processed items")

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
