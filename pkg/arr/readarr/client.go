// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package readarr

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/autobrr/autobrr/pkg/errors"
)

func (c *Client) get(ctx context.Context, endpoint string) (int, []byte, error) {
	u, err := url.Parse(c.config.Hostname)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not parse url: %s", c.config.Hostname)
	}

	u.Path = path.Join(u.Path, "/api/v1/", endpoint)
	reqUrl := u.String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, http.NoBody)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not build request")
	}

	if c.config.BasicAuth {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, errors.Wrap(err, "readarr.http.Do(req): %+v", req)
	}

	defer resp.Body.Close()

	if resp.Body == nil {
		return resp.StatusCode, nil, errors.New("response body is nil")
	}

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, resp.Body); err != nil {
		return resp.StatusCode, nil, errors.Wrap(err, "readarr.io.Copy")
	}

	return resp.StatusCode, buf.Bytes(), nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, params url.Values, data any) error {
	u, err := url.Parse(c.config.Hostname)
	if err != nil {
		return errors.Wrap(err, "could not parse url: %s", c.config.Hostname)
	}

	u.Path = path.Join(u.Path, "/api/v1/", endpoint)
	reqUrl := u.String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, http.NoBody)
	if err != nil {
		return errors.Wrap(err, "could not build request")
	}

	if c.config.BasicAuth {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	c.setHeaders(req)

	req.URL.RawQuery = params.Encode()

	resp, err := c.http.Do(req)
	if err != nil {
		return errors.Wrap(err, "readarr.http.Do(req): %+v", req)
	}

	defer resp.Body.Close()

	if resp.Body == nil {
		return errors.New("response body is nil")
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return errors.Wrap(err, "could not unmarshal data")
	}

	return nil
}

func (c *Client) post(ctx context.Context, endpoint string, data interface{}) (*http.Response, error) {
	u, err := url.Parse(c.config.Hostname)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse url: %s", c.config.Hostname)
	}

	u.Path = path.Join(u.Path, "/api/v1/", endpoint)
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
		return res, errors.New("unauthorized: bad credentials")
	} else if res.StatusCode != http.StatusOK {
		return res, errors.New("readarr: bad request")
	}

	// return raw response and let the caller handle json unmarshal of body
	return res, nil
}

func (c *Client) postBody(ctx context.Context, endpoint string, data interface{}) (int, []byte, error) {
	u, err := url.Parse(c.config.Hostname)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not parse url: %s", c.config.Hostname)
	}

	u.Path = path.Join(u.Path, "/api/v1/", endpoint)
	reqUrl := u.String()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not marshal data: %+v", data)
	}

	c.Log.Printf("readarr push JSON: %s\n", string(jsonData))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not build request")
	}

	if c.config.BasicAuth {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, errors.Wrap(err, "readarr.http.Do(req): %+v", req)
	}

	defer resp.Body.Close()

	if resp.Body == nil {
		return resp.StatusCode, nil, errors.New("response body is nil")
	}

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, resp.Body); err != nil {
		return resp.StatusCode, nil, errors.Wrap(err, "readarr.io.Copy")
	}

	if resp.StatusCode == http.StatusBadRequest {
		return resp.StatusCode, buf.Bytes(), nil
	} else if resp.StatusCode < 200 || resp.StatusCode > 401 {
		return resp.StatusCode, buf.Bytes(), errors.New("readarr: bad request: %v (status: %s): %s", resp.Request.RequestURI, resp.Status, buf.String())
	}

	return resp.StatusCode, buf.Bytes(), nil
}

func (c *Client) setHeaders(req *http.Request) {
	if req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("User-Agent", "autobrr")

	req.Header.Set("X-Api-Key", c.config.APIKey)
}
