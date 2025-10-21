// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package newznab

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
	"github.com/autobrr/autobrr/pkg/sharedhttp"
)

const DefaultTimeout = 60

type Client interface {
	GetFeed(ctx context.Context) (*Feed, error)
	GetCaps(ctx context.Context) (*Caps, error)
	Caps() *Caps
	WithHTTPClient(client *http.Client)
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

func (c *client) WithHTTPClient(client *http.Client) {
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

func NewClient(config Config) Client {
	httpClient := &http.Client{
		Timeout:   time.Second * DefaultTimeout,
		Transport: sharedhttp.Transport,
	}

	if config.Timeout > 0 {
		httpClient.Timeout = time.Second * config.Timeout
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

func (c *client) get(ctx context.Context, endpoint string, queryParams map[string]string) (int, *Feed, error) {
	params := url.Values{}
	params.Set("t", "search")

	for k, v := range queryParams {
		params.Add(k, v)
	}

	if c.ApiKey != "" {
		params.Add("apikey", c.ApiKey)
	}

	u, err := url.Parse(c.Host)
	if err != nil {
		return 0, nil, err
	}
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

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not make request. %+v", req)
	}

	defer sharedhttp.DrainAndClose(resp)

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not dump response")
	}

	c.Log.Printf("newznab get feed response dump: %q", dump)

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, resp.Body); err != nil {
		return resp.StatusCode, nil, errors.Wrap(err, "newznab.io.Copy")
	}

	var response Feed
	if err := xml.Unmarshal(buf.Bytes(), &response); err != nil {
		return resp.StatusCode, nil, errors.Wrap(err, "newznab: could not decode feed")
	}

	response.Raw = buf.String()

	return resp.StatusCode, &response, nil
}

func (c *client) getData(ctx context.Context, endpoint string, queryParams map[string]string) (*http.Response, error) {
	u, err := url.Parse(c.Host)
	if err != nil {
		return nil, errors.Wrap(err, "could not build request")
	}
	u.Path = strings.TrimSuffix(u.Path, "/")

	qp, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, errors.Wrap(err, "could not build request")
	}

	if c.ApiKey != "" {
		qp.Add("apikey", c.ApiKey)
	}

	for k, v := range queryParams {
		if qp.Has("t") {
			continue
		}
		qp.Add(k, v)
	}

	u.RawQuery = qp.Encode()
	reqUrl := u.String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not build request")
	}

	if c.UseBasicAuth {
		req.SetBasicAuth(c.BasicAuth.Username, c.BasicAuth.Password)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return resp, errors.Wrap(err, "could not make request. %+v", req)
	}

	return resp, nil
}

func (c *client) GetFeed(ctx context.Context) (*Feed, error) {

	p := map[string]string{"t": "search"}

	resp, err := c.getData(ctx, "", p)
	if err != nil {
		return nil, errors.Wrap(err, "could not get feed")
	}

	defer sharedhttp.DrainAndClose(resp)

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, errors.Wrap(err, "could not dump response")
	}

	c.Log.Printf("newznab get feed response dump: %q", dump)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("could not get feed")
	}

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, resp.Body); err != nil {
		return nil, errors.Wrap(err, "newznab.io.Copy")
	}

	var response Feed
	if err := xml.Unmarshal(buf.Bytes(), &response); err != nil {
		return nil, errors.Wrap(err, "newznab: could not decode feed")
	}

	response.Raw = buf.String()

	if c.Capabilities != nil {
		for _, item := range response.Channel.Items {
			item.MapCustomCategoriesFromAttr(c.Capabilities.Categories.Categories)
		}
	} else {
		for _, item := range response.Channel.Items {
			item.MapCategoriesFromAttr()
		}
	}

	return &response, nil
}

func (c *client) GetFeedAndCaps(ctx context.Context) (*Feed, error) {
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

	p := map[string]string{"t": "search"}

	status, res, err := c.get(ctx, "", p)
	if err != nil {
		return nil, errors.Wrap(err, "could not get feed")
	}

	if status != http.StatusOK {
		return nil, errors.New("could not get feed")
	}

	for _, item := range res.Channel.Items {
		item.MapCustomCategoriesFromAttr(c.Capabilities.Categories.Categories)
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
	if err != nil {
		return 0, nil, err
	}
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

	defer sharedhttp.DrainAndClose(resp)

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not dump response")
	}

	c.Log.Printf("newznab get caps response dump: %q", dump)

	if resp.StatusCode == http.StatusUnauthorized {
		return resp.StatusCode, nil, errors.New("unauthorized")
	} else if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, nil, errors.New("bad status: %d", resp.StatusCode)
	}

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, resp.Body); err != nil {
		return resp.StatusCode, nil, errors.Wrap(err, "newznab.io.Copy")
	}

	var response Caps
	if err := xml.Unmarshal(buf.Bytes(), &response); err != nil {
		return resp.StatusCode, nil, errors.Wrap(err, "newznab: could not decode feed")
	}

	return resp.StatusCode, &response, nil
}

func (c *client) GetCaps(ctx context.Context) (*Caps, error) {
	status, res, err := c.getCaps(ctx, "?t=caps", nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not get caps for feed")
	}

	if status != http.StatusOK {
		return nil, errors.Wrap(err, "could not get caps for feed")
	}

	return res, nil
}

func (c *client) Caps() *Caps {
	return c.Capabilities
}

//func (c *client) Search(ctx context.Context, query string) ([]FeedItem, error) {
//	v := url.Values{}
//	v.Add("q", query)
//	params := v.Encode()
//
//	status, res, err := c.get(ctx, "&t=search&"+params, nil)
//	if err != nil {
//		return nil, errors.Wrap(err, "could not search feed")
//	}
//
//	if status != http.StatusOK {
//		return nil, errors.New("could not search feed")
//	}
//
//	return res.Channel.Items, nil
//}
