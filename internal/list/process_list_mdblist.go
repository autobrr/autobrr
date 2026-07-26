// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/pkg/errors"
)

func (s *Service) mdblist(ctx context.Context, list *domain.List) error {
	l := s.log.With().Str("type", "mdblist").Str("list", list.Name).Logger()

	if list.URL == "" {
		return errors.New("no URL provided for Mdblist")
	}

	//var titles []string

	//green := color.New(color.FgGreen).SprintFunc()
	l.Debug().Str("url", list.URL).Msg("fetching titles")

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
	defer sharedhttp.DrainAndClose(resp)

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("failed to fetch titles from URL: %s", list.URL)
	}

	var data []struct {
		Title       string `json:"title"`
		ReleaseYear int    `json:"release_year"`
		MediaType   string `json:"mediatype"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return errors.Wrapf(err, "failed to decode JSON data from URL: %s", list.URL)
	}

	var filterTitles []string
	for _, item := range data {
		title := item.Title
		if list.IncludeYear && list.MatchRelease && item.ReleaseYear > 0 && item.MediaType == "movie" {
			title = title + "*" + strconv.Itoa(item.ReleaseYear) + "*"
		}
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
		l.Debug().Int("filter_id", filter.ID).Msg("updating filter")

		filterUpdate.ID = filter.ID

		if err := s.filterSvc.UpdatePartial(ctx, filterUpdate); err != nil {
			return errors.Wrapf(err, "error updating filter: %v", filter.ID)
		}

		l.Debug().Int("filter_id", filter.ID).Msg("successfully updated filter")
	}

	return nil
}
