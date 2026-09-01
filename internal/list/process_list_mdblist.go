// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	"fmt"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/list/provider/mdblist"
	"github.com/rs/zerolog"
)

type MDBListProcessor struct {
	processorBase
	client *mdblist.Client
}

func NewMDBListProcessor(log zerolog.Logger, list *domain.List) *MDBListProcessor {
	return &MDBListProcessor{
		log:    log,
		list:   list,
		client: mdblist.NewClient(log, list.Name, list.URL, list.Headers...),
	}
}

func (p *MDBListProcessor) Process(ctx context.Context) (*domain.FilterUpdate, error) {
	data, err := p.client.GetList(ctx, p.list.URL)
	if err != nil {
		return nil, err
	}

	filter, err := p.process(data)
	if err != nil {
		return nil, err
	}

	return filter, nil
}

func (p *MDBListProcessor) process(data []mdblist.Item) (*domain.FilterUpdate, error) {
	ts := NewTitleSet()
	ts.matchReleases = p.list.MatchRelease

	for _, item := range data {
		title := item.Title
		if p.list.IncludeYear && p.list.MatchRelease && item.ReleaseYear > 0 && item.MediaType == "movie" {
			title = fmt.Sprintf("%s*%d", title, item.ReleaseYear)
		}
		ts.AddTitle(title)
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
