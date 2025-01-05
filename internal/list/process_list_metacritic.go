// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/pkg/errors"
)

func (s *service) metacritic(ctx context.Context, list *domain.List) error {
	l := s.log.With().Str("type", "metacritic").Str("list", list.Name).Logger()

	if list.URL == "" {
		return errors.New("no URL provided for metacritic")
	}

	l.Debug().Msgf("fetching titles from %s", list.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, list.URL, nil)
	if err != nil {
		return errors.Wrapf(err, "could not make new request for URL: %s", list.URL)
	}

	list.SetRequestHeaders(req)

	//setUserAgent(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch titles from URL: %s", list.URL)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return errors.Errorf("No endpoint found at %v. (404 Not Found)", list.URL)
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrapf(err, "failed to fetch titles from URL: %s", list.URL)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return errors.Wrapf(err, "unexpected content type for URL: %s expected application/json got %s", list.URL, contentType)
	}

	var data struct {
		Title  string `json:"title"`
		Albums []struct {
			Artist string `json:"artist"`
			Title  string `json:"title"`
		} `json:"albums"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return errors.Wrapf(err, "failed to decode JSON data from URL: %s", list.URL)
	}

	var titles []string
	var artists []string

	for _, album := range data.Albums {
		titles = append(titles, album.Title)
		artists = append(artists, album.Artist)
	}

	// Deduplicate artists
	uniqueArtists := []string{}
	seenArtists := map[string]struct{}{}
	for _, artist := range artists {
		if _, ok := seenArtists[artist]; !ok {
			uniqueArtists = append(uniqueArtists, artist)
			seenArtists[artist] = struct{}{}
		}
	}

	filterTitles := []string{}
	for _, title := range titles {
		filterTitles = append(filterTitles, processTitle(title, list.MatchRelease)...)
	}

	filterArtists := []string{}
	for _, artist := range uniqueArtists {
		filterArtists = append(filterArtists, processTitle(artist, list.MatchRelease)...)
	}

	if len(filterTitles) == 0 && len(filterArtists) == 0 {
		l.Debug().Msgf("no titles found to update filter: %v", list.Name)
		return nil
	}

	joinedArtists := strings.Join(filterArtists, ",")
	joinedTitles := strings.Join(filterTitles, ",")

	l.Trace().Str("albums", joinedTitles).Msgf("found %d album titles", len(joinedTitles))
	l.Trace().Str("artists", joinedTitles).Msgf("found %d artit titles", len(joinedArtists))

	filterUpdate := domain.FilterUpdate{Albums: &joinedTitles, Artists: &joinedArtists}

	if list.MatchRelease {
		filterUpdate.Albums = &nullString
		filterUpdate.Artists = &nullString
		filterUpdate.MatchReleases = &joinedTitles
	}

	for _, filter := range list.Filters {
		l.Debug().Msgf("updating filter: %v", filter.ID)

		filterUpdate.ID = filter.ID

		if err := s.filterSvc.UpdatePartial(ctx, filterUpdate); err != nil {
			return errors.Wrapf(err, "error updating filter: %v", filter.ID)
		}

		l.Debug().Msgf("successfully updated filter: %v", filter.ID)
	}

	return nil
}
