// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package whisparr

import (
	"context"
	"encoding/json"
	"net/http"
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

type Client interface {
	Test(ctx context.Context) (*SystemStatusResponse, error)
	Push(ctx context.Context, release Release) ([]string, error)
}

type client struct {
	config Config
	http   *http.Client

	log zerolog.Logger
}

func New(config Config) Client {
	transport := sharedhttp.Transport
	if config.TLSSkipVerify {
		transport = sharedhttp.TransportTLSInsecure
	}

	httpClient := &http.Client{
		Timeout:   time.Second * 120,
		Transport: transport,
	}

	return &client{
		config: config,
		http:   httpClient,
		log:    config.Log,
	}
}

// logger prefers the request-scoped logger from ctx, which carries trace_id
// and other correlation fields, over the static client logger.
func (c *client) logger(ctx context.Context) *zerolog.Logger {
	if l := zerolog.Ctx(ctx); l.GetLevel() != zerolog.Disabled {
		return l
	}
	return &c.log
}

type Release struct {
	Title            string `json:"title"`
	InfoUrl          string `json:"infoUrl,omitempty"`
	DownloadUrl      string `json:"downloadUrl,omitempty"`
	MagnetUrl        string `json:"magnetUrl,omitempty"`
	Size             uint64 `json:"size"`
	Indexer          string `json:"indexer"`
	DownloadProtocol string `json:"downloadProtocol"`
	Protocol         string `json:"protocol"`
	PublishDate      string `json:"publishDate"`
	DownloadClientId int    `json:"downloadClientId,omitempty"`
	DownloadClient   string `json:"downloadClient,omitempty"`
}

type PushResponse struct {
	Approved     bool     `json:"approved"`
	Rejected     bool     `json:"rejected"`
	TempRejected bool     `json:"temporarilyRejected"`
	Rejections   []string `json:"rejections"`
}

type SystemStatusResponse struct {
	Version string `json:"version"`
}

func (c *client) Test(ctx context.Context) (*SystemStatusResponse, error) {
	res, err := c.get(ctx, "system/status")
	if err != nil {
		return nil, errors.Wrap(err, "could not test whisparr")
	}

	defer sharedhttp.DrainAndClose(res)

	response := SystemStatusResponse{}
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	c.logger(ctx).Trace().Int("status", res.StatusCode).Interface("response", response).Msg("whisparr system/status response")

	return &response, nil
}

func (c *client) Push(ctx context.Context, release Release) ([]string, error) {
	res, err := c.post(ctx, "release/push", release)
	if err != nil {
		return nil, errors.Wrap(err, "could not push release to whisparr: %+v", release)
	}

	if res == nil {
		return nil, nil
	}

	defer sharedhttp.DrainAndClose(res)

	pushResponse := make([]PushResponse, 0)
	err = json.NewDecoder(res.Body).Decode(&pushResponse)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	c.logger(ctx).Trace().Int("status", res.StatusCode).Interface("response", pushResponse).Msg("whisparr release/push response")

	// log and return if rejected
	if pushResponse[0].Rejected {
		c.logger(ctx).Debug().Strs("rejections", pushResponse[0].Rejections).Msg("whisparr release/push rejected")
		return pushResponse[0].Rejections, nil
	}

	// success true
	return nil, nil
}
