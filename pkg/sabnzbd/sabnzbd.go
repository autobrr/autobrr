// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package sabnzbd

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"
)

type Client struct {
	log *log.Logger

	http   *http.Client
	addr   string
	apiKey string

	basicUser string
	basicPass string
}

type Options struct {
	Log    *log.Logger
	Addr   string
	ApiKey string

	BasicUser string
	BasicPass string
}

func New(opts Options) *Client {
	c := &Client{
		addr:      opts.Addr,
		apiKey:    opts.ApiKey,
		basicUser: opts.BasicUser,
		basicPass: opts.BasicPass,
		log:       log.New(io.Discard, "", log.LstdFlags),
		http: &http.Client{
			Timeout:   time.Second * 60,
			Transport: sharedhttp.Transport,
		},
	}

	if opts.Log != nil {
		c.log = opts.Log
	}

	return c
}

func (c *Client) AddFromUrl(ctx context.Context, r AddNzbRequest) (*AddFileResponse, error) {
	v := url.Values{}
	v.Set("mode", "addurl")
	v.Set("name", r.Url)
	v.Set("output", "json")
	v.Set("apikey", c.apiKey)
	v.Set("cat", "*")

	if r.Category != "" {
		v.Set("cat", r.Category)
	}

	addr, err := url.JoinPath(c.addr, "/api")
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	u.RawQuery = v.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	if c.basicUser != "" && c.basicPass != "" {
		req.SetBasicAuth(c.basicUser, c.basicPass)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body := bufio.NewReader(res.Body)
	if _, err := body.Peek(1); err != nil && err != bufio.ErrBufferFull {
		return nil, errors.Wrap(err, "could not read body")
	}

	var data AddFileResponse
	if err := json.NewDecoder(body).Decode(&data); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal body")
	}

	return &data, nil
}

func (c *Client) Version(ctx context.Context) (*VersionResponse, error) {
	v := url.Values{}
	v.Set("mode", "version")
	v.Set("output", "json")
	v.Set("apikey", c.apiKey)

	addr, err := url.JoinPath(c.addr, "/api")
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	u.RawQuery = v.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	if c.basicUser != "" && c.basicPass != "" {
		req.SetBasicAuth(c.basicUser, c.basicPass)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body := bufio.NewReader(res.Body)
	if _, err := body.Peek(1); err != nil && err != bufio.ErrBufferFull {
		return nil, errors.Wrap(err, "could not read body")
	}

	var data VersionResponse
	if err := json.NewDecoder(body).Decode(&data); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal body")
	}

	return &data, nil
}

type VersionResponse struct {
	Version string `json:"version"`
}

type AddFileResponse struct {
	ApiError
	NzoIDs []string `json:"nzo_ids"`
}

type ApiError struct {
	ErrorMsg string `json:"error,omitempty"`
}

type AddNzbRequest struct {
	Url      string
	Category string
}
