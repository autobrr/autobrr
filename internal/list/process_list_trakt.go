// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/pkg/errors"
)

var (
	traktSmartListWebRegex  = regexp.MustCompile(`(?i)^(?:https?://)?(?:app\.|www\.)?trakt\.tv/lists/smart/view/([^/?#]+)`)
	traktUserListWebRegex   = regexp.MustCompile(`(?i)^(?:https?://)?(?:app\.|www\.)?trakt\.tv/users/([^/]+)/lists/([^/?#]+)`)
	traktUserWatchlistRegex = regexp.MustCompile(`(?i)^(?:https?://)?(?:app\.|www\.)?trakt\.tv/users/([^/]+)/watchlist`)
)

func transformTraktURL(rawURL string) string {
	if matches := traktSmartListWebRegex.FindStringSubmatch(rawURL); len(matches) == 2 {
		return fmt.Sprintf("https://api.trakt.tv/smart-lists/%s/items", matches[1])
	}

	if matches := traktUserListWebRegex.FindStringSubmatch(rawURL); len(matches) == 3 {
		return fmt.Sprintf("https://api.trakt.tv/users/%s/lists/%s/items", matches[1], matches[2])
	}

	if matches := traktUserWatchlistRegex.FindStringSubmatch(rawURL); len(matches) == 2 {
		return fmt.Sprintf("https://api.trakt.tv/users/%s/watchlist/items", matches[1])
	}

	return rawURL
}

func (s *Service) trakt(ctx context.Context, list *domain.List) error {
	l := s.log.With().Str("type", "trakt").Str("list", list.Name).Logger()

	if list.URL == "" {
		return errors.Errorf("no URL provided for trakt: %s", list.Name)
	}

	reqURL := transformTraktURL(list.URL)

	l.Debug().Str("url", reqURL).Msg("fetching titles")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return errors.Wrapf(err, "could not make new request for URL: %s", reqURL)
	}

	req.Header.Set("trakt-api-version", "2")

	if list.APIKey != "" {
		req.Header.Set("trakt-api-key", list.APIKey)
	}

	list.SetRequestHeaders(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch titles from URL: %s", reqURL)
	}
	defer sharedhttp.DrainAndClose(resp)

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("failed to fetch titles from URL: %s", reqURL)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return errors.Errorf("invalid content type for URL: %s, content type should be application/json", reqURL)
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
		return errors.Wrapf(err, "failed to decode JSON data from URL: %s", reqURL)
	}

	var titles []string
	for _, item := range data {
		if item.Title != "" {
			titles = append(titles, item.Title)
		}
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
		l.Debug().Msg("no titles found to update list")
		return nil
	}

	joinedTitles := strings.Join(filterTitles, ",")

	l.Trace().Str("titles", joinedTitles).Int("count", len(filterTitles)).Msg("found titles")

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

		l.Debug().Int("filter_id", filter.ID).Msg("successfully updated filter")
	}

	return nil
}
