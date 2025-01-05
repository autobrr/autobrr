// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/pkg/errors"
)

func (s *service) trakt(ctx context.Context, list *domain.List) error {
	l := s.log.With().Str("type", "trakt").Str("list", list.Name).Logger()

	if list.URL == "" {
		errMsg := "no URL provided for steam"
		l.Error().Msg(errMsg)
		return fmt.Errorf(errMsg)
	}

	l.Debug().Msgf("fetching titles from %s", list.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, list.URL, nil)
	if err != nil {
		l.Error().Err(err).Msg("could not make new request")
		return err
	}

	req.Header.Set("trakt-api-version", "2")

	if list.APIKey != "" {
		req.Header.Set("trakt-api-key", list.APIKey)
	}

	list.SetRequestHeaders(req)

	//setUserAgent(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		l.Error().Err(err).Msgf("failed to fetch titles from URL: %s", list.URL)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		l.Error().Msgf("failed to fetch titles from URL: %s", list.URL)
		return fmt.Errorf("failed to fetch titles from URL: %s", list.URL)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		errMsg := fmt.Sprintf("invalid content type for URL: %s, content type should be application/json", list.URL)
		return fmt.Errorf(errMsg)
	}

	var data []struct {
		Title string `json:"title"`
		Movie struct {
			Title string `json:"title"`
		} `json:"movie"`
		Show struct {
			Title string `json:"title"`
		} `json:"show"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		l.Error().Err(err).Msgf("failed to decode JSON data from URL: %s", list.URL)
		return err
	}

	var titles []string
	for _, item := range data {
		titles = append(titles, item.Title)
		if item.Movie.Title != "" {
			titles = append(titles, item.Movie.Title)
		}
		if item.Show.Title != "" {
			titles = append(titles, item.Show.Title)
		}
	}

	filterTitles := []string{}
	for _, title := range titles {
		filterTitles = append(filterTitles, processTitle(title, list.MatchRelease)...)
	}

	if len(filterTitles) == 0 {
		l.Debug().Msgf("no titles found to update for list: %v", list.Name)
		return nil
	}

	joinedTitles := strings.Join(filterTitles, ",")

	l.Trace().Str("titles", joinedTitles).Msgf("found %d titles", len(joinedTitles))

	filterUpdate := domain.FilterUpdate{Shows: &joinedTitles}

	if list.MatchRelease {
		filterUpdate.Shows = &nullString
		filterUpdate.MatchReleases = &joinedTitles
	}

	for _, filter := range list.Filters {
		filterUpdate.ID = filter.ID

		if err := s.filterSvc.UpdatePartial(ctx, filterUpdate); err != nil {
			return errors.Wrapf(err, "error updating filter: %v", filter.ID)
		}

		l.Debug().Msgf("successfully updated filter: %v", filter.ID)
	}

	return nil
}
