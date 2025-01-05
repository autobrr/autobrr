// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/arr/lidarr"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

func (s *service) lidarr(ctx context.Context, list *domain.List) error {
	l := s.log.With().Str("list", list.Name).Str("type", "lidarr").Int("client", list.ClientID).Logger()

	l.Debug().Msgf("gathering titles...")

	titles, artists, err := s.processLidarr(ctx, list, &l)
	if err != nil {
		return err
	}

	l.Debug().Msgf("got %d filter titles", len(titles))

	// Process titles
	var processedTitles []string
	for _, title := range titles {
		processedTitles = append(processedTitles, processTitle(title, list.MatchRelease)...)
	}

	if len(processedTitles) == 0 {
		l.Debug().Msgf("no titles found to update for list: %v", list.Name)
		return nil
	}

	// Update filter based on MatchRelease
	var f domain.FilterUpdate

	if list.MatchRelease {
		joinedTitles := strings.Join(processedTitles, ",")
		if len(joinedTitles) == 0 {
			return nil
		}

		l.Trace().Str("titles", joinedTitles).Msgf("found %d titles", len(joinedTitles))

		f.MatchReleases = &joinedTitles
	} else {
		// Process artists only if MatchRelease is false
		var processedArtists []string
		for _, artist := range artists {
			processedArtists = append(processedArtists, processTitle(artist, list.MatchRelease)...)
		}

		joinedTitles := strings.Join(processedTitles, ",")

		l.Trace().Str("albums", joinedTitles).Msgf("found %d titles", len(joinedTitles))

		joinedArtists := strings.Join(processedArtists, ",")
		if len(joinedTitles) == 0 && len(joinedArtists) == 0 {
			return nil
		}

		l.Trace().Str("artists", joinedArtists).Msgf("found %d titles", len(joinedArtists))

		f.Albums = &joinedTitles
		f.Artists = &joinedArtists
	}

	//joinedTitles := strings.Join(titles, ",")
	//
	//l.Trace().Msgf("%v", joinedTitles)
	//
	//if len(joinedTitles) == 0 {
	//	return nil
	//}

	for _, filter := range list.Filters {
		l.Debug().Msgf("updating filter: %v", filter.ID)

		f.ID = filter.ID

		if err := s.filterSvc.UpdatePartial(ctx, f); err != nil {
			return errors.Wrap(err, "error updating filter: %v", filter.ID)
		}

		l.Debug().Msgf("successfully updated filter: %v", filter.ID)
	}

	return nil
}

func (s *service) processLidarr(ctx context.Context, list *domain.List, logger *zerolog.Logger) ([]string, []string, error) {
	downloadClient, err := s.downloadClientSvc.GetClient(ctx, int32(list.ClientID))
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get client with id %d", list.ClientID)
	}

	if !downloadClient.Enabled {
		return nil, nil, errors.New("client %s %s not enabled", downloadClient.Type, downloadClient.Name)
	}

	client := downloadClient.Client.(*lidarr.Client)

	//var tags []*arr.Tag
	//if len(list.TagsExclude) > 0 || len(list.TagsInclude) > 0 {
	//	t, err := client.GetTags(ctx)
	//	if err != nil {
	//		logger.Debug().Msg("could not get tags")
	//	}
	//	tags = t
	//}

	albums, err := client.GetAlbums(ctx, 0)
	if err != nil {
		return nil, nil, err
	}

	logger.Debug().Msgf("found %d albums to process", len(albums))

	var titles []string
	var artists []string
	seenArtists := make(map[string]struct{})

	for _, album := range albums {
		if !list.ShouldProcessItem(album.Monitored) {
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
		artist, err := client.GetArtistByID(ctx, album.ArtistID)
		if err != nil {
			logger.Error().Err(err).Msgf("Error fetching artist details for album: %v", album.Title)
			continue // Skip this album if there's an error fetching the artist
		}

		if artist.Monitored {
			processedTitles := processTitle(album.Title, list.MatchRelease)
			titles = append(titles, processedTitles...)

			// Debug logging
			logger.Debug().Msgf("Processing artist: %s", artist.ArtistName)

			if _, exists := seenArtists[artist.ArtistName]; !exists {
				artists = append(artists, artist.ArtistName)
				seenArtists[artist.ArtistName] = struct{}{}
				logger.Debug().Msgf("Added artist: %s", artist.ArtistName) // Log when an artist is added
			}
		}
	}

	//sort.Strings(titles)
	logger.Debug().Msgf("Processed %d monitored albums with monitored artists, created %d titles, found %d unique artists", len(titles), len(titles), len(artists))

	return titles, artists, nil
}
