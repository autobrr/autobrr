// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/list/provider/trakt"
	"github.com/autobrr/autobrr/internal/stringutils"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type TraktProcessor struct {
	processorBase
	client *trakt.Client
}

func NewTraktProcessor(log zerolog.Logger, list *domain.List) *TraktProcessor {
	return &TraktProcessor{
		log:    log,
		list:   list,
		client: trakt.NewClient(log, list.Name, list.URL, list.APIKey, list.Headers...),
	}
}

func (p *TraktProcessor) Process(ctx context.Context) (*domain.FilterUpdate, error) {
	data, err := p.client.GetList(ctx, p.list.URL)
	if err != nil {
		return nil, errors.Wrap(err, "could not make new request for URL")
	}

	filter, err := p.process(data)
	if err != nil {
		return nil, err
	}

	return filter, nil
}

func (p *TraktProcessor) process(data []trakt.Item) (*domain.FilterUpdate, error) {
	ts := NewTitleSet()
	ts.matchReleases = p.list.MatchRelease

	for _, item := range data {
		if item.Title != "" {
			ts.AddTitle(item.Title)
		}
		if item.Movie.Title != "" {
			ts.AddTitle(item.Movie.Title)
		}
		if item.Show.Title != "" {
			ts.AddTitle(item.Show.Title)
		}
	}

	if ts.Len() == 0 {
		p.log.Debug().Msg("no titles found to update list")
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
