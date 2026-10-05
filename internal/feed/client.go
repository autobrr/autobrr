// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"cmp"
	"context"
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/proxy"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/mmcdole/gofeed"
	"golang.org/x/net/publicsuffix"
)

const defaultTimeout = 60 * time.Second

// newHTTPClient builds the client every feed request goes through, so proxy, TLS and timeout
// settings apply the same way to refresh, test and caps.
func newHTTPClient(f *domain.Feed) (*http.Client, error) {
	timeout := defaultTimeout
	if f.Timeout > 0 {
		timeout = time.Duration(f.Timeout) * time.Second
	}

	if f.UseProxy && f.Proxy != nil {
		client, err := proxy.GetProxiedHTTPClient(f.Proxy)
		if err != nil {
			return nil, errors.Wrap(err, "could not get proxy client")
		}

		if f.TLSSkipVerify {
			if t, ok := client.Transport.(*http.Transport); ok {
				t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			}
		}

		client.Timeout = timeout

		return client, nil
	}

	transport := sharedhttp.Transport
	if f.TLSSkipVerify {
		transport = sharedhttp.TransportTLSInsecure
	}

	return &http.Client{Timeout: timeout, Transport: transport}, nil
}

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
