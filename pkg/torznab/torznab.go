// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
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
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"
)

type Client struct {
	http *http.Client

	Host   string
	ApiKey string

	UseBasicAuth bool
	BasicAuth    BasicAuth

	Capabilities *Caps

	Log *log.Logger
}

func (c *Client) WithHTTPClient(client *http.Client) {
	c.http = client
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

func NewClient(config Config) *Client {
	httpClient := &http.Client{
		Timeout:   config.Timeout,
		Transport: sharedhttp.Transport,
	}

	c := &Client{
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

func (c *Client) get(ctx context.Context, params url.Values) (*Feed, error) {
	if c.ApiKey != "" {
		params.Add("apikey", url.QueryEscape(c.ApiKey))
	}

	u, err := url.Parse(c.Host)
	if err != nil {
		return nil, err
	}

	u.Path = strings.TrimSuffix(u.Path, "/")
	u.RawQuery = params.Encode()
	reqUrl := u.String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not build request")
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
		return nil, errors.Wrap(err, "could not make request. %+v", req)
	}

	defer sharedhttp.DrainAndClose(resp)

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, errors.Wrap(err, "could not dump response")
	}

	c.Log.Printf("torznab get feed response dump: %q", dump)

	switch resp.StatusCode {
	case http.StatusOK:
		break
	case http.StatusUnauthorized:
		return nil, errors.New("unauthorized")
	case http.StatusNotFound:
		return nil, errors.New("not found, make sure the path is correct")
	default:
		return nil, errors.New("unexpected status code: %d", resp.StatusCode)
	}

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, resp.Body); err != nil {
		return nil, errors.Wrap(err, "torznab.io.Copy")
	}

	var response Feed
	if err := xml.Unmarshal(buf.Bytes(), &response); err != nil {
		return nil, errors.Wrap(err, "torznab: could not decode feed")
	}

	response.Raw = buf.String()

	return &response, nil
}

func (c *Client) FetchFeed(ctx context.Context) (*Feed, error) {
	if c.Capabilities == nil {
		caps, err := c.getCaps(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get caps for feed")
		}

		if caps == nil {
			return nil, errors.New("could not get caps for feed")
		}

		c.Capabilities = caps
	}

	params := url.Values{}
	params.Set("t", "search")
	params.Set("extended", "1")
	params.Add("limit", "50")
	if c.Capabilities != nil && c.Capabilities.Limits.Max > 0 {
		params.Set("limit", strconv.Itoa(c.Capabilities.Limits.Max))
	}

	res, err := c.get(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "could not get feed")
	}

	for _, item := range res.Channel.Items {
		item.MapCategories(c.Capabilities.Categories.Categories)
	}

	return res, nil
}

func (c *Client) getCaps(ctx context.Context) (*Caps, error) {
	params := url.Values{}
	params.Set("t", "caps")

	if c.ApiKey != "" {
		params.Add("apikey", url.QueryEscape(c.ApiKey))
	}

	u, err := url.Parse(c.Host)
	if err != nil {
		return nil, err
	}

	u.Path = strings.TrimSuffix(u.Path, "/")
	u.RawQuery = params.Encode()
	reqUrl := u.String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not build request")
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
		return nil, errors.Wrap(err, "could not make request. %+v", req)
	}

	defer sharedhttp.DrainAndClose(resp)

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, errors.Wrap(err, "could not dump response")
	}

	c.Log.Printf("torznab get caps response dump: %q", dump)

	switch resp.StatusCode {
	case http.StatusOK:
		break
	case http.StatusUnauthorized:
		return nil, errors.New("unauthorized")
	case http.StatusNotFound:
		return nil, errors.New("not found, make sure the path is correct")
	default:
		return nil, errors.New("unexpected status code: %d", resp.StatusCode)
	}

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, resp.Body); err != nil {
		return nil, errors.Wrap(err, "torznab.io.Copy")
	}

	var response Caps
	if err := xml.Unmarshal(buf.Bytes(), &response); err != nil {
		return nil, errors.Wrap(err, "torznab: could not decode feed")
	}

	return &response, nil
}

func (c *Client) FetchCaps(ctx context.Context) (*Caps, error) {
	res, err := c.getCaps(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get caps for feed")
	}

	return res, nil
}

func (c *Client) GetCaps() *Caps {
	return c.Capabilities
}

func (c *Client) Search(ctx context.Context, query string, categories []int) (*SearchResponse, error) {
	params := url.Values{}
	params.Set("t", "search")
	if query != "" {
		params.Add("q", query)
	}
	params.Set("extended", "1")
	params.Add("limit", "50")
	if c.Capabilities != nil && c.Capabilities.Limits.Max > 0 {
		params.Set("limit", strconv.Itoa(c.Capabilities.Limits.Max))
	}

	for _, cat := range categories {
		params.Add("cat", strconv.Itoa(cat))
	}

	res, err := c.get(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "could not search feed")
	}

	resp := &SearchResponse{
		Title: res.Channel.Title,
		Items: res.Channel.Items,
		Raw:   res.Raw,
	}

	return resp, nil
}
