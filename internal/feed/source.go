// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/proxy"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/newznab"
	"github.com/autobrr/autobrr/pkg/sharedhttp"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/rs/zerolog"
)

const defaultTimeout = 60 * time.Second

// source fetches one feed format and maps its items to releases; everything format
// independent lives in refreshJob.
type source interface {
	fetch(ctx context.Context) (*fetchResult, error)
}

// capsSource is implemented by the formats that publish capabilities.
type capsSource interface {
	caps(ctx context.Context) (*domain.FeedCapabilities, error)
}

type fetchResult struct {
	// raw is stored as the feed's last run data
	raw     string
	entries []entry
}

type entry struct {
	key     string
	title   string
	pubDate time.Time
	release func() *domain.Release
}

// newSource is the only way to reach a feed over the network, so every request carries the
// feed's proxy, TLS and timeout settings.
func (s *Service) newSource(ctx context.Context, f *domain.Feed, log zerolog.Logger) (source, error) {
	if err := s.loadProxy(ctx, f); err != nil {
		return nil, err
	}

	client, err := newHTTPClient(f)
	if err != nil {
		return nil, err
	}

	if f.UseProxy && f.Proxy != nil {
		log.Debug().Str("proxy", f.Proxy.Name).Msg("using proxy for feed")
	}

	switch f.Type {
	case string(domain.FeedTypeTorznab):
		c := torznab.NewClient(torznab.Config{Host: f.URL, ApiKey: f.ApiKey, Log: log})
		c.WithHTTPClient(client)

		return &torznabSource{log: log, feed: f, client: c}, nil

	case string(domain.FeedTypeNewznab):
		c := newznab.NewClient(newznab.Config{Host: f.URL, ApiKey: f.ApiKey, Log: log})
		c.WithHTTPClient(client)

		return &newznabSource{log: log, feed: f, client: c}, nil

	case string(domain.FeedTypeRSS):
		return &rssSource{log: log, feed: f, client: client}, nil

	default:
		return nil, errors.New("unsupported feed type: %s", f.Type)
	}
}

// loadProxy attaches the feed's proxy when it is set to use an enabled one.
func (s *Service) loadProxy(ctx context.Context, f *domain.Feed) error {
	if !f.UseProxy {
		return nil
	}

	proxyConf, err := s.proxySvc.FindByID(ctx, f.ProxyID)
	if err != nil {
		return errors.Wrap(err, "could not find proxy for indexer feed")
	}

	if proxyConf.Enabled {
		f.Proxy = proxyConf
	}

	return nil
}

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
