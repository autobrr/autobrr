// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package whisparr

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path"

	"github.com/autobrr/autobrr/pkg/errors"
)

func (c *client) get(ctx context.Context, endpoint string) (*http.Response, error) {
	u, err := url.Parse(c.config.Hostname)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse url: %s", c.config.Hostname)
	}

	u.Path = path.Join(u.Path, "/api/v3/", endpoint)
	reqUrl := u.String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, http.NoBody)
	if err != nil {
		return nil, errors.Wrap(err, "could not build request")
	}

	if c.config.BasicAuth {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	req.Header.Add("X-Api-Key", c.config.APIKey)
	req.Header.Set("User-Agent", "autobrr")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "could not make request: %+v", req)
	}

	if res.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	}

	return res, nil
}

func (c *client) post(ctx context.Context, endpoint string, data interface{}) (*http.Response, error) {
	u, err := url.Parse(c.config.Hostname)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse url: %s", c.config.Hostname)
	}

	u.Path = path.Join(u.Path, "/api/v3/", endpoint)
	reqUrl := u.String()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal data: %+v", data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, errors.Wrap(err, "could not build request")
	}

	if c.config.BasicAuth {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	req.Header.Add("X-Api-Key", c.config.APIKey)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("User-Agent", "autobrr")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "could not make request: %+v", req)
	}

	// validate response
	if res.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	} else if res.StatusCode != http.StatusOK {
		return nil, errors.New("whisparr: bad request")
	}

	// return raw response and let the caller handle json unmarshal of body
	return res, nil
}
