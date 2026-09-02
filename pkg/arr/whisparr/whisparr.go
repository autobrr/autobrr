// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package whisparr

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

// Whisparr major versions. V2 is the Sonarr based release that manages studios
// as series, V3 (Eros) is the Radarr based one that manages movies and scenes.
const (
	VersionV2 = 2
	VersionV3 = 3
)

type Config struct {
	Hostname string
	APIKey   string

	// Version is the expected Whisparr major version, VersionV2 or VersionV3.
	Version int

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

// New create new whisparr Client
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

// Test calls system/status and verifies the reported major version matches the
// configured one. The two Whisparr versions expose different endpoints, so a
// mismatch is reported here instead of failing later with a confusing 404.
func (c *Client) Test(ctx context.Context) (*SystemStatusResponse, error) {
	status, res, err := c.get(ctx, "system/status")
	if err != nil {
		return nil, errors.Wrap(err, "could not make Test")
	}

	if status == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	}

	c.logger(ctx).Trace().Int("status", status).Str("response", string(res)).Msg("whisparr system/status response")

	response := SystemStatusResponse{}
	if err = json.Unmarshal(res, &response); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	if err := c.checkVersion(response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) checkVersion(status SystemStatusResponse) error {
	if c.config.Version == 0 {
		return nil
	}

	major := status.MajorVersion()
	if major == 0 || major == c.config.Version {
		return nil
	}

	return errors.New("client is configured for Whisparr v%d but the server reports version %s, change the client type to Whisparr (v%d)", c.config.Version, status.Version, major)
}

func (c *Client) Push(ctx context.Context, release ReleasePushRequest) ([]string, error) {
	status, res, err := c.postBody(ctx, "release/push", release)
	if err != nil {
		return nil, errors.Wrap(err, "could not push release to whisparr")
	}

	c.logger(ctx).Trace().Int("status", status).Str("response", string(res)).Msg("whisparr release/push response")

	if status == http.StatusBadRequest {
		badRequestResponses := make([]*BadRequestResponse, 0)

		if err = json.Unmarshal(res, &badRequestResponses); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal data")
		}

		rejections := make([]string, 0, len(badRequestResponses))
		for _, response := range badRequestResponses {
			rejections = append(rejections, response.String())
		}

		for _, response := range badRequestResponses {
			if strings.EqualFold(response.PropertyName, "DownloadClient") || strings.EqualFold(response.PropertyName, "DownloadClientId") {
				return nil, errors.New("whisparr push failed due to invalid configuration: %s", strings.Join(rejections, "; "))
			}
		}

		return rejections, nil
	}

	pushResponse := make([]ReleasePushResponse, 0)
	if err = json.Unmarshal(res, &pushResponse); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	if len(pushResponse) == 0 {
		return nil, errors.New("whisparr push returned an empty response")
	}

	// rejected is false when every rejection is temporary, and a temporarily rejected release
	// waits in the pending queue instead of being grabbed, so both flags have to be reported
	if pushResponse[0].Rejected || pushResponse[0].TempRejected {
		c.logger(ctx).Debug().Bool("temporary", pushResponse[0].TempRejected).Strs("rejections", pushResponse[0].Rejections).Msg("whisparr release/push rejected")
		return pushResponse[0].Rejections, nil
	}

	// successful push
	return nil, nil
}

// GetAllSeries returns the studios of a Whisparr v2 instance. The endpoint does
// not exist on v3.
func (c *Client) GetAllSeries(ctx context.Context) ([]Series, error) {
	data := make([]Series, 0)
	if err := c.getJSON(ctx, "series", nil, &data); err != nil {
		return nil, errors.Wrap(err, "could not get series")
	}

	return data, nil
}

// GetMovies returns the movies and scenes of a Whisparr v3 instance. The
// endpoint does not exist on v2.
func (c *Client) GetMovies(ctx context.Context) ([]Movie, error) {
	data := make([]Movie, 0)
	if err := c.getJSON(ctx, "movie", nil, &data); err != nil {
		return nil, errors.Wrap(err, "could not get movies")
	}

	return data, nil
}

func (c *Client) GetTags(ctx context.Context) ([]arr.Tag, error) {
	data := make([]arr.Tag, 0)
	if err := c.getJSON(ctx, "tag", nil, &data); err != nil {
		return nil, errors.Wrap(err, "could not get tags")
	}

	return data, nil
}
