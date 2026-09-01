// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/list/provider/metacritic"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type MetacriticProcessor struct {
	processorBase
	client *metacritic.Client
}

func NewMetacriticProcessor(log zerolog.Logger, list *domain.List) *MetacriticProcessor {
	return &MetacriticProcessor{
		log:    log,
		list:   list,
		client: metacritic.NewClient(log, list.Name, list.URL, list.Headers...),
	}
}

func (p *MetacriticProcessor) Process(ctx context.Context) (*domain.FilterUpdate, error) {
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

func (p *MetacriticProcessor) process(data *metacritic.ListResponse) (*domain.FilterUpdate, error) {
	// process artist + album variants and append to releases
	ts := NewTitleSet()
	ts.matchReleases = p.list.MatchRelease

	for _, alb := range data.Albums {
		artist := processTitle(alb.Artist, true)
		album := processTitle(alb.Title, true)

		for _, artistVar := range artist {
			for _, albumVar := range album {
				title := artistVar + albumVar
				ts.AddTitle(title)
			}
		}
	}

	joinedTitles := ts.FilterString()

	filterUpdate := &domain.FilterUpdate{
		Albums:        new(""),
		Artists:       new(""),
		MatchReleases: &joinedTitles,
	}

	return filterUpdate, nil
}
