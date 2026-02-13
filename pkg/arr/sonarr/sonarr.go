// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package sonarr

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"log"

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

	TLSSkipVerify bool

	Log *log.Logger
}

type ClientInterface interface {
	Test(ctx context.Context) (*SystemStatusResponse, error)
	Push(ctx context.Context, release ReleasePushRequest) ([]string, error)
}

type Client struct {
	config Config
	http   *http.Client

	Log *log.Logger
}

// New create new sonarr Client
func New(config Config) *Client {
	transport := sharedhttp.Transport
	if config.TLSSkipVerify {
		transport = sharedhttp.TransportTLSInsecure
	}

	httpClient := &http.Client{
		Timeout:   time.Second * 120,
		Transport: transport,
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
		return nil, errors.Wrap(err, "could not make Test")
	}

	if status == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	}

	c.Log.Printf("sonarr system/status status: (%v) response: %v\n", status, string(res))

	response := SystemStatusResponse{}
	if err = json.Unmarshal(res, &response); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	return &response, nil
}

func (c *Client) Push(ctx context.Context, release ReleasePushRequest) ([]string, error) {
	status, res, err := c.postBody(ctx, "release/push", release)
	if err != nil {
		return nil, errors.Wrap(err, "could not push release to sonarr")
	}

	c.Log.Printf("sonarr release/push status: (%v) response: %v\n", status, string(res))

	if status == http.StatusBadRequest {
		badRequestResponses := make([]*BadRequestResponse, 0)

		if err = json.Unmarshal(res, &badRequestResponses); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal data")
		}

		for _, response := range badRequestResponses {
			if strings.EqualFold(response.PropertyName, "DownloadClient") || strings.EqualFold(response.PropertyName, "DownloadClientId") {
				rejections := make([]string, 0, len(badRequestResponses))
				for _, r := range badRequestResponses {
					rejections = append(rejections, r.String())
				}

				return nil, errors.New("sonarr push failed due to invalid configuration: %s", strings.Join(rejections, "; "))
			}
		}

		rejections := []string{}
		for _, response := range badRequestResponses {
			rejections = append(rejections, response.String())
		}

		return rejections, nil
	}

	pushResponse := make([]ReleasePushResponse, 0)
	if err = json.Unmarshal(res, &pushResponse); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	// log and return if rejected
	if pushResponse[0].Rejected {
		rejections := strings.Join(pushResponse[0].Rejections, ", ")

		c.Log.Printf("sonarr release/push rejected %v reasons: %q\n", release.Title, rejections)
		return pushResponse[0].Rejections, nil
	}

	// successful push
	return nil, nil
}

func (c *Client) GetAllSeries(ctx context.Context) ([]Series, error) {
	return c.GetSeries(ctx, 0)
}

func (c *Client) GetSeries(ctx context.Context, tvdbID int64) ([]Series, error) {
	params := make(url.Values)
	if tvdbID != 0 {
		params.Set("tvdbId", strconv.FormatInt(tvdbID, 10))
	}

	data := make([]Series, 0)
	err := c.getJSON(ctx, "series", params, &data)
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
