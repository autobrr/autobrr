package radarr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/rs/zerolog/log"
)

func (c *client) get(endpoint string) (int, []byte, error) {
	u, err := url.Parse(c.config.Hostname)
	u.Path = path.Join(u.Path, "/api/v3/", endpoint)
	reqUrl := u.String()

	req, err := http.NewRequest(http.MethodGet, reqUrl, http.NoBody)
	if err != nil {
		log.Error().Err(err).Msgf("radarr client request error : %v", reqUrl)
		return 0, nil, err
	}

	if c.config.BasicAuth {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("radarr client.get request error: %v", reqUrl)
		return 0, nil, fmt.Errorf("radarr.http.Do(req): %w", err)
	}

	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, resp.Body); err != nil {
		return resp.StatusCode, nil, fmt.Errorf("radarr.io.Copy: %w", err)
	}

	return resp.StatusCode, buf.Bytes(), nil
}

func (c *client) post(endpoint string, data interface{}) (*http.Response, error) {
	u, err := url.Parse(c.config.Hostname)
	u.Path = path.Join(u.Path, "/api/v3/", endpoint)
	reqUrl := u.String()

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Error().Err(err).Msgf("radarr client could not marshal data: %v", reqUrl)
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, reqUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error().Err(err).Msgf("radarr client request error: %v", reqUrl)
		return nil, err
	}

	if c.config.BasicAuth {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	req.Header.Add("X-Api-Key", c.config.APIKey)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("User-Agent", "autobrr")

	res, err := c.http.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("radarr client request error: %v", reqUrl)
		return nil, err
	}

	// validate response
	if res.StatusCode == http.StatusUnauthorized {
		log.Error().Err(err).Msgf("radarr client bad request: %v", reqUrl)
		return nil, errors.New("unauthorized: bad credentials")
	} else if res.StatusCode == http.StatusBadRequest {
		log.Error().Err(err).Msgf("radarr client request error: %v", reqUrl)
		return nil, errors.New("radarr: bad request")
	} else if res.StatusCode != http.StatusOK {
		log.Error().Err(err).Msgf("radarr client request error: %v", reqUrl)
		return nil, errors.New("radarr: bad request")
	}

	// return raw response and let the caller handle json unmarshal of body
	return res, nil
}

func (c *client) postBody(endpoint string, data interface{}) (int, []byte, error) {
	u, err := url.Parse(c.config.Hostname)
	u.Path = path.Join(u.Path, "/api/v3/", endpoint)
	reqUrl := u.String()

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Error().Err(err).Msgf("radarr client could not marshal data: %v", reqUrl)
		return 0, nil, err
	}

	req, err := http.NewRequest(http.MethodPost, reqUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error().Err(err).Msgf("radarr client request error: %v", reqUrl)
		return 0, nil, err
	}

	if c.config.BasicAuth {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("radarr client request error: %v", reqUrl)
		return 0, nil, fmt.Errorf("radarr.http.Do(req): %w", err)
	}

	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, resp.Body); err != nil {
		return resp.StatusCode, nil, fmt.Errorf("radarr.io.Copy: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return resp.StatusCode, buf.Bytes(), fmt.Errorf("radarr: bad request: %v (status: %s): %s", resp.Request.RequestURI, resp.Status, buf.String())
	}

	return resp.StatusCode, buf.Bytes(), nil
}

func (c *client) setHeaders(req *http.Request) {
	if req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("User-Agent", "autobrr")

	req.Header.Set("X-Api-Key", c.config.APIKey)
}
