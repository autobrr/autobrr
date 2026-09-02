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
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/sharedhttp"
	"github.com/rs/zerolog"

	"github.com/pkg/errors"
)

type PlaintextProcessor struct {
	processorBase
	httpClient *http.Client
}

func NewPlaintextProcessor(log zerolog.Logger, list *domain.List) *PlaintextProcessor {
	return &PlaintextProcessor{
		log:  log.With().Str("type", "plaintext").Str("list", list.Name).Logger(),
		list: list,
		httpClient: &http.Client{
			Timeout:   time.Second * 15,
			Transport: sharedhttp.Transport,
		},
	}
}

func (p *PlaintextProcessor) Process(ctx context.Context) (*domain.FilterUpdate, error) {

	if p.list.URL == "" {
		return nil, errors.New("no URL provided for plaintext")
	}

	p.log.Debug().Str("url", p.list.URL).Msg("fetching titles")

	// Parse the URL to determine if it's a file or HTTP scheme
	parsedURL, err := url.Parse(p.list.URL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse URL: %s", p.list.URL)
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

		p.log.Debug().Str("path", filePath).Msg("reading from file")

		body, err = os.ReadFile(filePath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read file: %s", filePath)
		}

	case "http", "https":
		// Use HTTP client for http:// or https:// URLs
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.list.URL, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "could not make new request for URL: %s", p.list.URL)
		}

		for _, header := range p.list.Headers {
			parts := strings.Split(header, "=")
			if len(parts) != 2 {
				continue
			}
			req.Header.Set(parts[0], parts[1])
		}

		//setUserAgent(req)

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to fetch titles from URL: %s", p.list.URL)
		}
		defer sharedhttp.DrainAndClose(resp)

		switch resp.StatusCode {
		case http.StatusOK:
			break
		default:
			return nil, errors.Errorf("failed to fetch list from URL: %s", p.list.URL)
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "text/plain") {
			return nil, errors.Errorf("unexpected content type for URL: %s expected text/plain got %s", p.list.URL, contentType)
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read response body from URL: %s", p.list.URL)
		}

	default:
		return nil, errors.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	filter, err := p.process(body)
	if err != nil {
		return nil, err
	}

	return filter, nil
}

func (p *PlaintextProcessor) process(body []byte) (*domain.FilterUpdate, error) {
	//ts := NewTitleSet()
	//ts.matchReleases = p.list.MatchRelease

	var titles []string
	for titleLine := range strings.SplitSeq(string(body), "\n") {
		title := strings.TrimSpace(titleLine)
		if title == "" {
			continue
		}
		if p.list.SkipCleanSanitize {
			titles = append(titles, title) // Add title as-is
		} else {
			titles = append(titles, processTitle(title, p.list.MatchRelease)...) // Existing logic
		}
	}

	if len(titles) == 0 {
		p.log.Debug().Msg("no titles found to update list")
		return nil, nil
	}

	joinedTitles := strings.Join(titles, ",")

	p.log.Trace().Str("titles", joinedTitles).Int("count", len(titles)).Msg("found titles")

	filter := domain.FilterUpdate{Shows: &joinedTitles}

	if p.list.MatchRelease {
		filter.Shows = new("")
		filter.MatchReleases = &joinedTitles
	}

	return &filter, nil
}
