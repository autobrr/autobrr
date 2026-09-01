// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/arr/lidarr"
	"github.com/rs/zerolog"
)

type LidarrProcessor struct {
	processorBase
	client *lidarr.Client
}

func NewLidarrProcessor(log zerolog.Logger, list *domain.List, client *lidarr.Client) *LidarrProcessor {
	return &LidarrProcessor{
		log:    log,
		list:   list,
		client: client,
	}
}

func (p *LidarrProcessor) Process(ctx context.Context) (*domain.FilterUpdate, error) {
	data, err := p.client.GetAlbums(ctx, 0)
	if err != nil {
		return nil, err
	}

	filter, err := p.process(ctx, data)
	if err != nil {
		return nil, err
	}

	return filter, nil
}

func (p *LidarrProcessor) process(ctx context.Context, albums []lidarr.Album) (*domain.FilterUpdate, error) {
	p.log.Debug().Int("count", len(albums)).Msg("found albums to process")

	var titles []string
	var artists []string
	seenArtists := make(map[string]struct{})

	for _, album := range albums {
		if !p.list.ShouldProcessItem(album.Monitored) {
			continue
		}

		//if len(list.TagsInclude) > 0 {
		//	if len(album.Tags) == 0 {
		//		continue
		//	}
		//	if !containsTag(tags, album.Tags, list.TagsInclude) {
		//		continue
		//	}
		//}
		//
		//if len(list.TagsExclude) > 0 {
		//	if containsTag(tags, album.Tags, list.TagsExclude) {
		//		continue
		//	}
		//}

		// Fetch the artist details
		artist, err := p.client.GetArtistByID(ctx, album.ArtistID)
		if err != nil {
			p.log.Error().Err(err).Str("title", album.Title).Msg("error fetching artist details for album")
			continue // Skip this album if there's an error fetching the artist
		}

		if artist.Monitored {
			processedTitles := processTitle(album.Title, p.list.MatchRelease)
			titles = append(titles, processedTitles...)

			// Debug logging
			p.log.Debug().Str("artist", artist.ArtistName).Msg("processing artist")

			if _, exists := seenArtists[artist.ArtistName]; !exists {
				artists = append(artists, artist.ArtistName)
				seenArtists[artist.ArtistName] = struct{}{}
				p.log.Debug().Str("artist", artist.ArtistName).Msg("added artist") // Log when an artist is added
			}
		}
	}

	//sort.Strings(titles)
	p.log.Debug().Int("total", len(titles)).Int("processed", len(titles)).Int("created", len(artists)).Msg("processed items")

	p.log.Debug().Int("count", len(titles)).Msg("got filter titles")

	// Process titles
	var processedTitles []string
	for _, title := range titles {
		processedTitles = append(processedTitles, processTitle(title, p.list.MatchRelease)...)
	}

	if len(processedTitles) == 0 {
		p.log.Debug().Str("list", p.list.Name).Msg("no titles found to update")
		return nil, nil
	}

	// Update filter based on MatchRelease
	var filter domain.FilterUpdate

	if p.list.MatchRelease {
		joinedTitles := strings.Join(processedTitles, ",")
		if len(joinedTitles) == 0 {
			return nil, nil
		}

		p.log.Trace().Str("titles", joinedTitles).Int("count", len(processedTitles)).Msg("found titles")

		filter.MatchReleases = &joinedTitles
	} else {
		// Process artists only if MatchRelease is false
		var processedArtists []string
		for _, artist := range artists {
			processedArtists = append(processedArtists, processTitle(artist, p.list.MatchRelease)...)
		}

		joinedTitles := strings.Join(processedTitles, ",")

		p.log.Trace().Str("albums", joinedTitles).Int("count", len(processedTitles)).Msg("found titles")

		joinedArtists := strings.Join(processedArtists, ",")
		if len(joinedTitles) == 0 && len(joinedArtists) == 0 {
			return nil, nil
		}

		p.log.Trace().Str("artists", joinedArtists).Int("count", len(processedArtists)).Msg("found titles")

		filter.Albums = &joinedTitles
		filter.Artists = &joinedArtists
	}

	return &filter, nil
}
