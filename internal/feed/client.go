// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"cmp"
	"context"
	"net/http"
	"net/http/cookiejar"

	"github.com/mmcdole/gofeed"
	"golang.org/x/net/publicsuffix"
)

type RSSParser struct {
	parser    *gofeed.Parser
	http      *http.Client
	cookie    string
	userAgent string
}

// NewFeedParser wraps the gofeed.Parser using our own http client for full control
func NewFeedParser(client *http.Client, cookie string, userAgent string) *RSSParser {
	// a copy so every parser starts with its own cookie jar
	httpClient := *client
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	httpClient.Jar = jar

	parser := gofeed.NewParser()
	parser.Client = &httpClient

	return &RSSParser{
		parser:    parser,
		http:      &httpClient,
		cookie:    cookie,
		userAgent: userAgent,
	}
}

func (c *RSSParser) ParseURLWithContext(ctx context.Context, feedURL string) (feed *gofeed.Feed, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", cmp.Or(c.userAgent, "Gofeed/1.0"))

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
