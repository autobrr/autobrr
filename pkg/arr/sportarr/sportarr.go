// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package sportarr

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/pkg/arr"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	goversion "github.com/hashicorp/go-version"
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
	Push(ctx context.Context, release ReleasePushRequest) ([]string, error)
}

type Client struct {
	config Config
	http   *http.Client

	log zerolog.Logger
}

// New create new sportarr Client
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
		return nil, errors.Wrap(err, "could not make Test")
	}

	if status == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	}

	c.logger(ctx).Trace().Int("status", status).Str("response", string(res)).Msg("sportarr system/status response")

	response := SystemStatusResponse{}
	if err = json.Unmarshal(res, &response); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	return &response, nil
}

func (c *Client) Push(ctx context.Context, release ReleasePushRequest) ([]string, error) {
	status, res, err := c.postBody(ctx, "release/push", release)
	if err != nil {
		return nil, errors.Wrap(err, "could not push release to sportarr")
	}

	c.logger(ctx).Trace().Int("status", status).Str("response", string(res)).Msg("sportarr release/push response")

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

	pushResponse := make([]ReleasePushResponse, 0)
	if err = json.Unmarshal(res, &pushResponse); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	if len(pushResponse) == 0 {
		return nil, errors.New("sportarr release/push returned an empty response")
	}

	// rejected is false when every rejection is temporary, and a temporarily rejected release
	// waits in the pending queue instead of being grabbed, so both flags have to be reported
	if pushResponse[0].Rejected || pushResponse[0].TempRejected {
		c.logger(ctx).Debug().Bool("temporary", pushResponse[0].TempRejected).Strs("rejections", pushResponse[0].Rejections).Msg("sportarr release/push rejected")
		return pushResponse[0].Rejections, nil
	}

	// successful push
	return nil, nil
}

func (c *Client) GetAllLeagues(ctx context.Context) ([]League, error) {
	data := make([]League, 0)
	err := c.getJSON(ctx, "leagues", nil, &data)
	if err != nil {
		return nil, errors.Wrap(err, "could not get leagues")
	}

	return data, nil
}

func (c *Client) GetTags(ctx context.Context) ([]arr.Tag, error) {
	data := make([]arr.Tag, 0)
	err := c.getJSON(ctx, "tag", nil, &data)
	if err != nil {
		return nil, errors.Wrap(err, "could not get tags")
	}

	return data, nil
}

// MinimumVersion is the first Sportarr release that serves the native
// application API this client talks to (/api/release/push).
const MinimumVersion = "4.0.1024"

var minimumVersion = goversion.Must(goversion.NewVersion(MinimumVersion))

// SupportsNativeAPI reports whether the version from system/status is at
// least MinimumVersion. Unparseable versions return false.
func (r SystemStatusResponse) SupportsNativeAPI() bool {
	v, err := goversion.NewVersion(r.Version)
	if err != nil {
		return false
	}

	return v.GreaterThanOrEqual(minimumVersion)
}
