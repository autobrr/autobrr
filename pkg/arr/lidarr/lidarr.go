// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package lidarr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

type Config struct {
	Hostname string
	APIKey   string

	// basic auth username and password
	BasicAuth bool
	Username  string
	Password  string

	TLSSkipVerify bool

	Log zerolog.Logger
}

type ClientInterface interface {
	Test(ctx context.Context) (*SystemStatusResponse, error)
	Push(ctx context.Context, release Release) ([]string, error)
}

type Client struct {
	config Config
	http   *http.Client

	log zerolog.Logger
}

// New create new lidarr Client
func New(config Config) *Client {
	transport := sharedhttp.Transport
	if config.TLSSkipVerify {
		transport = sharedhttp.TransportTLSInsecure
	}

	httpClient := &http.Client{
		Timeout:   time.Second * 120,
		Transport: transport,
	}

	return &Client{
		config: config,
		http:   httpClient,
		log:    config.Log,
	}
}

// logger prefers the request-scoped logger from ctx, which carries trace_id
// and other correlation fields, over the static client logger.
func (c *Client) logger(ctx context.Context) *zerolog.Logger {
	if l := zerolog.Ctx(ctx); l.GetLevel() != zerolog.Disabled {
		return l
	}
	return &c.log
}

func (c *Client) Test(ctx context.Context) (*SystemStatusResponse, error) {
	status, res, err := c.get(ctx, "system/status")
	if err != nil {
		return nil, errors.Wrap(err, "lidarr Client get error")
	}

	if status == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	}

	c.logger(ctx).Trace().Int("status", status).Str("response", string(res)).Msg("lidarr system/status response")

	response := SystemStatusResponse{}
	err = json.Unmarshal(res, &response)
	if err != nil {
		return nil, errors.Wrap(err, "lidarr Client error json unmarshal")
	}

	return &response, nil
}

func (c *Client) Push(ctx context.Context, release Release) ([]string, error) {
	status, res, err := c.postBody(ctx, "release/push", release)
	if err != nil {
		return nil, errors.Wrap(err, "lidarr Client post error")
	}

	c.logger(ctx).Trace().Int("status", status).Str("response", string(res)).Msg("lidarr release/push response")

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

	pushResponse := PushResponse{}
	if err = json.Unmarshal(res, &pushResponse); err != nil {
		return nil, errors.Wrap(err, "lidarr Client error json unmarshal")
	}

	// log and return if rejected
	if pushResponse.Rejected {
		c.logger(ctx).Debug().Strs("rejections", pushResponse.Rejections).Msg("lidarr release/push rejected")
		return pushResponse.Rejections, nil
	}

	return nil, nil
}

func (c *Client) GetAlbums(ctx context.Context, mbID int64) ([]Album, error) {
	params := make(url.Values)
	if mbID != 0 {
		params.Set("ForeignAlbumId", strconv.FormatInt(mbID, 10))
	}

	data := make([]Album, 0)
	err := c.getJSON(ctx, "album", params, &data)
	if err != nil {
		return nil, errors.Wrap(err, "could not get tags")
	}

	return data, nil
}

func (c *Client) GetArtistByID(ctx context.Context, artistID int64) (*Artist, error) {
	var data Artist
	err := c.getJSON(ctx, "artist/"+strconv.FormatInt(artistID, 10), nil, &data)
	if err != nil {
		return nil, errors.Wrap(err, "could not get tags")
	}

	return &data, nil
}
