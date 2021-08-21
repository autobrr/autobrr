package radarr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

func (c *Client) get(endpoint string) (*http.Response, error) {
	reqUrl := fmt.Sprintf("%v/api/v3/%v", c.config.Hostname, endpoint)

	req, err := http.NewRequest(http.MethodGet, reqUrl, http.NoBody)
	if err != nil {
		log.Error().Err(err).Msgf("radarr client request error : %v", reqUrl)
		return nil, err
	}

	if c.config.BasicAuth {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	req.Header.Add("X-Api-Key", c.config.APIKey)

	res, err := c.http.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("radarr client request error : %v", reqUrl)
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Msgf("radarr client error reading body: %v", reqUrl)
		return nil, err
	}

	log.Debug().Msgf("body: %s", string(resBody))

	return res, nil
}

func (c *Client) post(endpoint string, data interface{}) error {
	reqUrl := fmt.Sprintf("%v/api/v3/%v", c.config.Hostname, endpoint)

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Error().Err(err).Msgf("radarr client could not marshal data: %v", reqUrl)
		return err
	}

	req, err := http.NewRequest(http.MethodPost, reqUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error().Err(err).Msgf("radarr client request error: %v", reqUrl)
		return err
	}

	if c.config.BasicAuth {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	req.Header.Add("X-Api-Key", c.config.APIKey)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	res, err := c.http.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("radarr client request error: %v", reqUrl)
		return err
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized {
		return errors.New("unauthorized: bad credentials")
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Msgf("radarr client error reading body: %v", reqUrl)
		return err
	}

	log.Debug().Msgf("body: %s", string(resBody))

	// TODO unmarshal response

	return nil
}
