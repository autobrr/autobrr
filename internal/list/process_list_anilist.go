// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/list/provider/anilist"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type AnilistProcessor struct {
	processorBase
	client *anilist.Client
}

func NewAnilistProcessor(log zerolog.Logger, list *domain.List) *AnilistProcessor {
	return &AnilistProcessor{
		log:    log,
		list:   list,
		client: anilist.NewClient(log, list.Name, list.URL, list.Headers...),
	}
}

func (p *AnilistProcessor) Process(ctx context.Context) (*domain.FilterUpdate, error) {
	data, err := p.client.GetList(ctx, p.list.URL)
	if err != nil {
		return nil, errors.Wrap(err, "could not get anilist list")
	}
	filter, err := p.process(data)
	if err != nil {
		return nil, errors.Wrap(err, "could not process anilist list")
	}
	return filter, nil
}

func (p *AnilistProcessor) process(data []anilist.Item) (*domain.FilterUpdate, error) {
	ts := NewTitleSet()
	ts.matchReleases = p.list.MatchRelease

	for _, item := range data {
		for _, title := range []string{item.English, item.Romaji} {
			ts.AddTitle(title)
		}

		for _, synonym := range item.Synonyms {
			ts.AddTitle(synonym)
		}
	}

	if ts.Len() == 0 {
		p.log.Debug().Msg("no titles found to update list")
		return nil, nil
	}

	joinedTitles := ts.FilterString()

	p.log.Trace().Str("titles", joinedTitles).Int("count", ts.Len()).Msg("found titles")

	filter := domain.FilterUpdate{Shows: &joinedTitles}

	if p.list.MatchRelease {
		filter.Shows = new("")
		filter.MatchReleases = &joinedTitles
	}

	return &filter, nil
}
