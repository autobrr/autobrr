// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"time"

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
	//store cookies in jar
	jarOptions := &cookiejar.Options{PublicSuffixList: publicsuffix.List}
	jar, _ := cookiejar.New(jarOptions)

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := &http.Client{
		Timeout:   time.Second * 60,
		Transport: customTransport,
		Jar:       jar,
	}

	c := &RSSParser{
		parser: gofeed.NewParser(),
		http:   httpClient,
		cookie: cookie,
	}

	c.http.Timeout = timeout

	return c
}

func (c *RSSParser) ParseURLWithContext(ctx context.Context, feedURL string) (feed *gofeed.Feed, err error) {
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
