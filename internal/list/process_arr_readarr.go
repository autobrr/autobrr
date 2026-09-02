// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/stringutils"
	"github.com/autobrr/autobrr/pkg/arr/readarr"
	"github.com/rs/zerolog"
)

type ReadarrProcessor struct {
	processorBase
	client *readarr.Client
}

func NewReadarrProcessor(log zerolog.Logger, list *domain.List, client *readarr.Client) *ReadarrProcessor {
	return &ReadarrProcessor{
		log:    log,
		list:   list,
		client: client,
	}
}

func (p *ReadarrProcessor) Process(ctx context.Context) (*domain.FilterUpdate, error) {
	books, err := p.client.GetBooks(ctx, "")
	if err != nil {
		return nil, err
	}

	filter, err := p.process(books)
	if err != nil {
		return nil, err
	}

	return filter, nil
}

func (p *ReadarrProcessor) process(books []readarr.Book) (*domain.FilterUpdate, error) {
	p.log.Debug().Int("count", len(books)).Msg("found books to process")

	ts := NewTitleSet()
	ts.matchReleases = p.list.MatchRelease
	var processedTitles int

	for _, book := range books {
		if !p.list.ShouldProcessItem(book.Monitored) {
			continue
		}

		//if excludedByTags(p.list, tags, book.Tags) {
		//	continue
		//}

		processedTitles++

		ts.AddTitle(book.Title)
	}

	p.log.Debug().Int("total", len(books)).Int("processed", processedTitles).Int("created", ts.Len()).Msg("processed items")

	if ts.Len() == 0 {
		p.log.Debug().Str("list", p.list.Name).Msg("no titles found to update")
		return nil, nil
	}

	joinedTitles := ts.FilterString()

	p.log.Trace().Str("titles", stringutils.TruncateStr(joinedTitles, 1024)).Int("count", ts.Len()).Msg("found titles")

	filter := domain.FilterUpdate{MatchReleases: &joinedTitles}

	return &filter, nil
}
