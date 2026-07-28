// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package sportarr

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/arr"
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

	// log and return if rejected
	if pushResponse[0].Rejected {
		c.logger(ctx).Debug().Strs("rejections", pushResponse[0].Rejections).Msg("sportarr release/push rejected")
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

func (c *Client) GetTags(ctx context.Context) ([]*arr.Tag, error) {
	data := make([]*arr.Tag, 0)
	err := c.getJSON(ctx, "tag", nil, &data)
	if err != nil {
		return nil, errors.Wrap(err, "could not get tags")
	}

	return data, nil
}

// MinimumVersion is the first Sportarr release that serves the native
// application API this client talks to (/api/release/push).
const MinimumVersion = "4.0.1024"

// SupportsNativeAPI reports whether the version from system/status is at
// least MinimumVersion. Unparseable versions return false.
func (r SystemStatusResponse) SupportsNativeAPI() bool {
	return compareVersions(r.Version, MinimumVersion) >= 0
}

// compareVersions compares dotted numeric versions segment by segment,
// treating missing or non-numeric segments as 0.
func compareVersions(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")

	segments := len(as)
	if len(bs) > segments {
		segments = len(bs)
	}

	for i := 0; i < segments; i++ {
		av, bv := 0, 0
		if i < len(as) {
			av = atoiSafe(as[i])
		}
		if i < len(bs) {
			bv = atoiSafe(bs[i])
		}
		if av != bv {
			if av > bv {
				return 1
			}
			return -1
		}
	}

	return 0
}

func atoiSafe(s string) int {
	v := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return v
		}
		v = v*10 + int(r-'0')
	}
	return v
}
