// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/mmcdole/gofeed"
	"golang.org/x/net/publicsuffix"
)

type RSSParser struct {
	parser *gofeed.Parser
	http   *http.Client
	cookie string
}

// NewFeedParser wraps the gofeed.Parser using our own http client for full control
func NewFeedParser(timeout time.Duration, cookie string) *RSSParser {
	httpClient := &http.Client{
		Timeout:   time.Second * 60,
		Transport: sharedhttp.TransportTLSInsecure,
	}

	if cookie != "" {
		//store cookies in jar
		jarOptions := &cookiejar.Options{PublicSuffixList: publicsuffix.List}
		jar, _ := cookiejar.New(jarOptions)
		httpClient.Jar = jar
	}

	c := &RSSParser{
		parser: gofeed.NewParser(),
		http:   httpClient,
		cookie: cookie,
	}

	c.http.Timeout = timeout
	c.parser.Client = httpClient

	return c
}

func (c *RSSParser) WithHTTPClient(client *http.Client) {
	httpClient := client
	if client.Jar == nil {
		jarOptions := &cookiejar.Options{PublicSuffixList: publicsuffix.List}
		jar, _ := cookiejar.New(jarOptions)
		httpClient.Jar = jar
	}

	c.http = httpClient
	c.parser.Client = httpClient
}

func (c *RSSParser) ParseURLWithContext(ctx context.Context, feedURL string) (feed *gofeed.Feed, err error) {
	parsedURL, err := url.Parse(feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed URL: %w", err)
	}

	switch parsedURL.Scheme {
	case "file":
		return c.parseFile(parsedURL)
	case "http", "https":
		return c.parseHTTP(ctx, feedURL)
	default:
		return nil, fmt.Errorf("unsupported URL scheme: %q", parsedURL.Scheme)
	}
}

func (c *RSSParser) parseFile(parsedURL *url.URL) (*gofeed.Feed, error) {
	filePath := parsedURL.Path

	if runtime.GOOS == "windows" {
		// On Windows, remove leading slash from path if needed
		if len(filePath) > 0 && filePath[0] == '/' && len(parsedURL.Host) > 0 {
			filePath = parsedURL.Host + filePath
		} else if len(filePath) > 0 && filePath[0] == '/' {
			filePath = filePath[1:]
		}
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", filePath, err)
	}
	defer f.Close()

	return c.parser.Parse(f)
}

func (c *RSSParser) parseHTTP(ctx context.Context, feedURL string) (feed *gofeed.Feed, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Gofeed/1.0")

	if c.cookie != "" {
		// set raw cookie as header
		req.Header.Set("Cookie", c.cookie)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	if resp != nil {
		defer func() {
			ce := resp.Body.Close()
			if ce != nil {
				err = ce
			}
		}()
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, gofeed.HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
	}

	return c.parser.Parse(resp.Body)
}
