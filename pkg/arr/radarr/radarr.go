// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package radarr

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/arr"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"
)

type Config struct {
	Hostname string
	APIKey   string

	// basic auth username and password
	BasicAuth bool
	Username  string
	Password  string

	Log *log.Logger
}

type ClientInterface interface {
	Test(ctx context.Context) (*SystemStatusResponse, error)
	Push(ctx context.Context, release Release) ([]string, error)
}

type Client struct {
	config Config
	http   *http.Client

	Log *log.Logger
}

func New(config Config) *Client {
	httpClient := &http.Client{
		Timeout:   time.Second * 120,
		Transport: sharedhttp.Transport,
	}

	c := &Client{
		config: config,
		http:   httpClient,
		Log:    log.New(io.Discard, "", log.LstdFlags),
	}

	if config.Log != nil {
		c.Log = config.Log
	}

	return c
}

func (c *Client) Test(ctx context.Context) (*SystemStatusResponse, error) {
	status, res, err := c.get(ctx, "system/status")
	if err != nil {
		return nil, errors.Wrap(err, "radarr error running test")
	}

	if status == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	}

	response := SystemStatusResponse{}
	if err = json.Unmarshal(res, &response); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	c.Log.Printf("radarr system/status status: (%v) response: %v\n", status, string(res))

	return &response, nil
}

func (c *Client) Push(ctx context.Context, release Release) ([]string, error) {
	status, res, err := c.postBody(ctx, "release/push", release)
	if err != nil {
		return nil, errors.Wrap(err, "error push release")
	}

	c.Log.Printf("radarr release/push status: (%v) response: %v\n", status, string(res))

	if status == http.StatusBadRequest {
		badRequestResponses := make([]*BadRequestResponse, 0)
		if err = json.Unmarshal(res, &badRequestResponses); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal data")
		}

		rejections := []string{}
		for _, response := range badRequestResponses {
			rejections = append(rejections, response.String())
		}

		return rejections, nil
	}

	pushResponse := make([]PushResponse, 0)
	if err = json.Unmarshal(res, &pushResponse); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	// log and return if rejected
	if pushResponse[0].Rejected {
		rejections := strings.Join(pushResponse[0].Rejections, ", ")

		c.Log.Printf("radarr release/push rejected %v reasons: %q\n", release.Title, rejections)
		return pushResponse[0].Rejections, nil
	}

	// success true
	return nil, nil
}

func (c *Client) GetMovies(ctx context.Context, tmdbID int64) ([]Movie, error) {
	params := make(url.Values)
	if tmdbID != 0 {
		params.Set("tmdbId", strconv.FormatInt(tmdbID, 10))
	}

	data := make([]Movie, 0)
	err := c.getJSON(ctx, "movie", params, &data)
	if err != nil {
		return nil, errors.Wrap(err, "could not get tags")
	}

	return data, nil
}

func (c *Client) GetTags(ctx context.Context) ([]*arr.Tag, error) {
	data := make([]*arr.Tag, 0)
	err := c.getJSON(ctx, "tag", nil, &data)
	if err != nil {
		return nil, errors.Wrap(err, "could not get tags")
	}

	return data, nil
}
