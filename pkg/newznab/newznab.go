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
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"
)

const DefaultTimeout = 60

type Client struct {
	http *http.Client

	Host   string
	ApiKey string

	UseBasicAuth bool
	BasicAuth    BasicAuth

	Capabilities *Caps

	Log   *log.Logger
	Debug bool
}

func (c *Client) WithHTTPClient(client *http.Client) {
	c.http = client
}

func (c *Client) WithDebug() {
	c.Debug = true
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
		Timeout:   time.Second * DefaultTimeout,
		Transport: sharedhttp.Transport,
	}

	if config.Timeout > 0 {
		httpClient.Timeout = time.Second * config.Timeout
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

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "could not make request. %+v", req)
	}

	defer sharedhttp.DrainAndClose(resp)

	if c.Debug {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return nil, errors.Wrap(err, "could not dump response")
		}

		c.Log.Printf("newznab get feed response dump: %q", dump)
	}

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
		return nil, errors.Wrap(err, "newznab.io.Copy")
	}

	var response Feed
	if err := xml.Unmarshal(buf.Bytes(), &response); err != nil {
		return nil, errors.Wrap(err, "newznab: could not decode feed")
	}

	response.Raw = buf.String()

	return &response, nil
}

func (c *Client) getData(ctx context.Context, params url.Values) (*http.Response, error) {
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
		qp.Add("apikey", url.QueryEscape(c.ApiKey))
	}

	for k, v := range params {
		for _, vv := range v {
			qp.Add(k, vv)
		}
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

func (c *Client) GetFeed(ctx context.Context) (*Feed, error) {
	if err := c.getAndSetCaps(ctx); err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("t", "search")
	params.Set("extended", "1")
	params.Add("limit", "50")
	if c.Capabilities != nil && c.Capabilities.Limits.Max > 0 {
		params.Set("limit", strconv.Itoa(c.Capabilities.Limits.Max))
	}

	resp, err := c.getData(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "could not get feed")
	}

	defer sharedhttp.DrainAndClose(resp)

	if c.Debug {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return nil, errors.Wrap(err, "could not dump response")
		}

		c.Log.Printf("newznab get feed response dump: %q", dump)
	}

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

	if c.Debug {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return nil, errors.Wrap(err, "could not dump response")
		}

		c.Log.Printf("newznab get caps response dump: %q", dump)
	}

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
		return nil, errors.Wrap(err, "newznab.io.Copy")
	}

	var response Caps
	if err := xml.Unmarshal(buf.Bytes(), &response); err != nil {
		return nil, errors.Wrap(err, "newznab: could not decode feed")
	}

	return &response, nil
}

func (c *Client) GetCaps(ctx context.Context) (*Caps, error) {
	res, err := c.getCaps(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get caps for feed")
	}

	return res, nil
}

func (c *Client) getAndSetCaps(ctx context.Context) error {
	if c.Capabilities == nil {
		caps, err := c.getCaps(ctx)
		if err != nil {
			return errors.Wrap(err, "could not get caps for feed")
		}

		if caps == nil {
			return errors.New("could not get caps for feed")
		}

		c.Capabilities = caps
	}

	return nil
}

func (c *Client) Caps() *Caps {
	return c.Capabilities
}

func (c *Client) Search(ctx context.Context, query string, categories []int) (*SearchResponse, error) {
	if err := c.getAndSetCaps(ctx); err != nil {
		return nil, err
	}

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

	if c.Capabilities != nil {
		for _, item := range res.Channel.Items {
			item.MapCustomCategoriesFromAttr(c.Capabilities.Categories.Categories)
		}
	} else {
		for _, item := range res.Channel.Items {
			item.MapCategoriesFromAttr()
		}
	}

	resp := &SearchResponse{
		Title: res.Channel.Title,
		Items: res.Channel.Items,
		Raw:   res.Raw,
	}

	return resp, nil
}
