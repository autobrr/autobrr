// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/pkg/errors"
)

func (s *service) anilist(ctx context.Context, list *domain.List) error {
	l := s.log.With().Str("type", "anilist").Str("list", list.Name).Logger()

	if list.URL == "" {
		return errors.New("no URL provided for AniList")
	}

	l.Debug().Msgf("fetching titles from %s", list.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, list.URL, nil)
	if err != nil {
		return errors.Wrapf(err, "could not make new request for URL: %s", list.URL)
	}

	list.SetRequestHeaders(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch titles from URL: %s", list.URL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("failed to fetch titles from URL: %s", list.URL)
	}

	var data []struct {
		Romaji   string   `json:"romaji"`
		English  string   `json:"english"`
		Synonyms []string `json:"synonyms"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return errors.Wrapf(err, "failed to decode JSON data from URL: %s", list.URL)
	}

	// remove duplicates
	titleSet := make(map[string]struct{})
	for _, item := range data {
		titlesToProcess := make(map[string]struct{})
		titlesToProcess[item.Romaji] = struct{}{}
		titlesToProcess[item.English] = struct{}{}
		for _, synonym := range item.Synonyms {
			titlesToProcess[synonym] = struct{}{}
		}

		for title := range titlesToProcess {
			for _, processedTitle := range processTitle(title, list.MatchRelease) {
				titleSet[processedTitle] = struct{}{}
			}
		}
	}

	filterTitles := make([]string, 0, len(titleSet))
	for title := range titleSet {
		filterTitles = append(filterTitles, title)
	}

	if len(filterTitles) == 0 {
		l.Debug().Msgf("no titles found to update for list: %v", list.Name)
		return nil
	}

	sort.Strings(filterTitles)
	joinedTitles := strings.Join(filterTitles, ",")

	l.Trace().Str("titles", joinedTitles).Msgf("found %d titles", len(joinedTitles))

	filterUpdate := domain.FilterUpdate{Shows: &joinedTitles}

	if list.MatchRelease {
		filterUpdate.Shows = &nullString
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
