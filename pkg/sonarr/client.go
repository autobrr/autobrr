package sonarr

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"path"

	"github.com/rs/zerolog/log"
)

func (c *client) get(endpoint string) (*http.Response, error) {
	u, err := url.Parse(c.config.Hostname)
	u.Path = path.Join(u.Path, "/api/v3/", endpoint)
	reqUrl := u.String()

	req, err := http.NewRequest(http.MethodGet, reqUrl, http.NoBody)
	if err != nil {
		log.Error().Err(err).Msgf("sonarr client request error : %v", reqUrl)
		return nil, err
	}

	if c.config.BasicAuth {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	req.Header.Add("X-Api-Key", c.config.APIKey)
	req.Header.Set("User-Agent", "autobrr")

	res, err := c.http.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("sonarr client request error : %v", reqUrl)
		return nil, err
	}

	if res.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	}

	return res, nil
}

func (c *client) post(endpoint string, data interface{}) (*http.Response, error) {
	u, err := url.Parse(c.config.Hostname)
	u.Path = path.Join(u.Path, "/api/v3/", endpoint)
	reqUrl := u.String()

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Error().Err(err).Msgf("sonarr client could not marshal data: %v", reqUrl)
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, reqUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error().Err(err).Msgf("sonarr client request error: %v", reqUrl)
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
		log.Error().Err(err).Msgf("sonarr client request error: %v", reqUrl)
		return nil, err
	}

	// validate response
	if res.StatusCode == http.StatusUnauthorized {
		log.Error().Err(err).Msgf("sonarr client bad request: %v", reqUrl)
		return nil, errors.New("unauthorized: bad credentials")
	} else if res.StatusCode != http.StatusOK {
		log.Error().Err(err).Msgf("sonarr client request error: %v", reqUrl)
		return nil, errors.New("sonarr: bad request")
	}

	// return raw response and let the caller handle json unmarshal of body
	return res, nil
}
