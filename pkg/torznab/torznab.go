// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package torznab

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

type Client interface {
	FetchFeed(ctx context.Context) (*Feed, error)
	FetchCaps(ctx context.Context) (*Caps, error)
	GetCaps() *Caps
}

type client struct {
	http *http.Client

	Host   string
	ApiKey string

	UseBasicAuth bool
	BasicAuth    BasicAuth

	Capabilities *Caps

	Log *log.Logger
}

type BasicAuth struct {
	Username string
	Password string
}

type Config struct {
	Host    string
	ApiKey  string
	Timeout time.Duration

	UseBasicAuth bool
	BasicAuth    BasicAuth

	Log *log.Logger
}

type Capabilities struct {
	Search     Searching
	Categories Categories
}

func NewClient(config Config) Client {
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	c := &client{
		http:   httpClient,
		Host:   config.Host,
		ApiKey: config.ApiKey,
		Log:    log.New(io.Discard, "", log.LstdFlags),
	}

	if config.Log != nil {
		c.Log = config.Log
	}

	return c
}

func (c *client) get(ctx context.Context, endpoint string, opts map[string]string) (int, *Feed, error) {
	params := url.Values{
		"t": {"search"},
	}

	if c.ApiKey != "" {
		params.Add("apikey", c.ApiKey)
	}

	u, err := url.Parse(c.Host)
	u.Path = strings.TrimSuffix(u.Path, "/")
	u.RawQuery = params.Encode()
	reqUrl := u.String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not build request")
	}

	if c.UseBasicAuth {
		req.SetBasicAuth(c.BasicAuth.Username, c.BasicAuth.Password)
	}

	// Jackett only supports api key via url param while Prowlarr does that and via header
	//if c.ApiKey != "" {
	//	req.Header.Add("X-API-Key", c.ApiKey)
	//}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not make request. %+v", req)
	}

	defer resp.Body.Close()

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not dump response")
	}

	c.Log.Printf("torznab get feed response dump: %q", dump)

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, resp.Body); err != nil {
		return resp.StatusCode, nil, errors.Wrap(err, "torznab.io.Copy")
	}

	var response Feed
	if err := xml.Unmarshal(buf.Bytes(), &response); err != nil {
		return resp.StatusCode, nil, errors.Wrap(err, "torznab: could not decode feed")
	}

	response.Raw = buf.String()

	return resp.StatusCode, &response, nil
}

func (c *client) FetchFeed(ctx context.Context) (*Feed, error) {
	if c.Capabilities == nil {
		status, caps, err := c.getCaps(ctx, "?t=caps", nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not get caps for feed")
		}

		if status != http.StatusOK {
			return nil, errors.Wrap(err, "could not get caps for feed")
		}

		c.Capabilities = caps
	}

	status, res, err := c.get(ctx, "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not get feed")
	}

	if status != http.StatusOK {
		return nil, errors.New("could not get feed")
	}

	for _, item := range res.Channel.Items {
		item.MapCategories(c.Capabilities.Categories.Categories)
	}

	return res, nil
}

func (c *client) getCaps(ctx context.Context, endpoint string, opts map[string]string) (int, *Caps, error) {
	params := url.Values{
		"t": {"caps"},
	}

	if c.ApiKey != "" {
		params.Add("apikey", c.ApiKey)
	}

	u, err := url.Parse(c.Host)
	u.Path = strings.TrimSuffix(u.Path, "/")
	u.RawQuery = params.Encode()
	reqUrl := u.String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not build request")
	}

	if c.UseBasicAuth {
		req.SetBasicAuth(c.BasicAuth.Username, c.BasicAuth.Password)
	}

	// Jackett only supports api key via url param while Prowlarr does that and via header
	//if c.ApiKey != "" {
	//	req.Header.Add("X-API-Key", c.ApiKey)
	//}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not make request. %+v", req)
	}

	defer resp.Body.Close()

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not dump response")
	}

	c.Log.Printf("torznab get caps response dump: %q", dump)

	if resp.StatusCode == http.StatusUnauthorized {
		return resp.StatusCode, nil, errors.New("unauthorized")
	} else if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, nil, errors.New("bad status: %d", resp.StatusCode)
	}

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, resp.Body); err != nil {
		return resp.StatusCode, nil, errors.Wrap(err, "torznab.io.Copy")
	}

	var response Caps
	if err := xml.Unmarshal(buf.Bytes(), &response); err != nil {
		return resp.StatusCode, nil, errors.Wrap(err, "torznab: could not decode feed")
	}

	return resp.StatusCode, &response, nil
}

func (c *client) FetchCaps(ctx context.Context) (*Caps, error) {

	status, res, err := c.getCaps(ctx, "?t=caps", nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not get caps for feed")
	}

	if status != http.StatusOK {
		return nil, errors.Wrap(err, "could not get caps for feed")
	}

	return res, nil
}

func (c *client) GetCaps() *Caps {
	return c.Capabilities
}

func (c *client) Search(ctx context.Context, query string) ([]*FeedItem, error) {
	v := url.Values{}
	v.Add("q", query)
	params := v.Encode()

	status, res, err := c.get(ctx, "&t=search&"+params, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not search feed")
	}

	if status != http.StatusOK {
		return nil, errors.New("could not search feed")
	}

	return res.Channel.Items, nil
}
