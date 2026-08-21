// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/pkg/errors"
)

func (s *Service) plaintext(ctx context.Context, list *domain.List) error {
	l := s.log.With().Str("type", "plaintext").Str("list", list.Name).Logger()

	if list.URL == "" {
		return errors.New("no URL provided for plaintext")
	}

	l.Debug().Str("url", list.URL).Msg("fetching titles")

	// Parse the URL to determine if it's a file or HTTP scheme
	parsedURL, err := url.Parse(list.URL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse URL: %s", list.URL)
	}

	var body []byte

	// Handle different URL schemes
	switch parsedURL.Scheme {
	case "file":
		// Read from filesystem for file:// URLs
		filePath := parsedURL.Path

		if runtime.GOOS == "windows" {
			// On Windows, remove leading slash from path if needed
			if len(filePath) > 0 && filePath[0] == '/' && len(parsedURL.Host) > 0 {
				filePath = parsedURL.Host + filePath
			} else if len(filePath) > 0 && filePath[0] == '/' {
				filePath = filePath[1:]
			}
		}

		l.Debug().Str("path", filePath).Msg("reading from file")

		body, err = os.ReadFile(filePath)
		if err != nil {
			return errors.Wrapf(err, "failed to read file: %s", filePath)
		}

	case "http", "https":
		// Use HTTP client for http:// or https:// URLs
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
			return errors.Wrapf(err, "failed to fetch titles from URL: %s with status code: %d", list.URL, resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "text/plain") {
			return errors.Errorf("unexpected content type for URL: %s expected text/plain got %s", list.URL, contentType)
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrapf(err, "failed to read response body from URL: %s", list.URL)
		}

	default:
		return errors.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	var titles []string
	titleLines := strings.Split(string(body), "\n")
	for _, titleLine := range titleLines {
		title := strings.TrimSpace(titleLine)
		if title == "" {
			continue
		}
		if list.SkipCleanSanitize {
			titles = append(titles, title) // Add title as-is
		} else {
			titles = append(titles, processTitle(title, list.MatchRelease)...) // Existing logic
		}
	}

	if len(titles) == 0 {
		l.Debug().Msg("no titles found to update list")
		return nil
	}

	joinedTitles := strings.Join(titles, ",")

	l.Trace().Str("titles", joinedTitles).Int("count", len(titles)).Msg("found titles")

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
