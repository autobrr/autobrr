// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/list/provider/steam"
	"github.com/rs/zerolog"
)

type SteamProcessor struct {
	processorBase
	client *steam.Client
}

func NewSteamProcessor(log zerolog.Logger, list *domain.List) *SteamProcessor {
	return &SteamProcessor{
		log:    log,
		list:   list,
		client: steam.NewClient(log, list.Name, list.URL, list.Headers...),
	}
}

func (p *SteamProcessor) Process(ctx context.Context) (*domain.FilterUpdate, error) {
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

func (p *SteamProcessor) process(data *steam.ListResponse) (*domain.FilterUpdate, error) {
	var titles []string
	for _, item := range *data {
		titles = append(titles, item.Name)
	}

	filterTitles := []string{}
	for _, title := range titles {
		filterTitles = append(filterTitles, processTitle(title, p.list.MatchRelease)...)
	}

	if len(filterTitles) == 0 {
		p.log.Debug().Msg("no titles found to update list")
		return nil, nil
	}

	joinedTitles := strings.Join(filterTitles, ",")

	p.log.Trace().Str("titles", joinedTitles).Int("count", len(filterTitles)).Msg("found titles")

	filterUpdate := domain.FilterUpdate{MatchReleases: &joinedTitles}

	return &filterUpdate, nil
}
