// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/stringutils"
	"github.com/autobrr/autobrr/pkg/arr"
	"github.com/autobrr/autobrr/pkg/arr/sportarr"
	"github.com/rs/zerolog"
)

type SportarrProcessor struct {
	processorBase
	client *sportarr.Client
}

func NewSportarrProcessor(log zerolog.Logger, list *domain.List, client *sportarr.Client) *SportarrProcessor {
	return &SportarrProcessor{
		log:    log,
		list:   list,
		client: client,
	}
}

func (p *SportarrProcessor) Process(ctx context.Context) (*domain.FilterUpdate, error) {
	titles, err := p.client.GetAllLeagues(ctx)
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

	filter, err := p.process(titles, tags)
	if err != nil {
		return nil, err
	}

	return filter, nil
}

func (p *SportarrProcessor) process(leagues []sportarr.League, tags []arr.Tag) (*domain.FilterUpdate, error) {
	p.log.Debug().Int("count", len(leagues)).Msg("found leagues to process")

	ts := NewTitleSet()
	ts.matchReleases = p.list.MatchRelease
	var processedTitles int

	for _, league := range leagues {
		if !p.list.ShouldProcessItem(league.Monitored) {
			continue
		}

		if excludedByTags(p.list, tags, league.Tags) {
			continue
		}

		processedTitles++

		ts.AddTitle(league.Name)
	}

	p.log.Debug().Int("total", len(leagues)).Int("processed", processedTitles).Int("created", ts.Len()).Msg("processed items")

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
