// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package lidarr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"
)

type Config struct {
	Hostname string
	APIKey   string

	// basic auth username and password
	BasicAuth bool
	Username  string
	Password  string

	Log *log.Logger
}

type Client interface {
	Test(ctx context.Context) (*SystemStatusResponse, error)
	Push(ctx context.Context, release Release) ([]string, error)
}

type client struct {
	config Config
	http   *http.Client

	Log *log.Logger
}

// New create new lidarr client
func New(config Config) Client {
	httpClient := &http.Client{
		Timeout:   time.Second * 120,
		Transport: sharedhttp.Transport,
	}

	c := &client{
		config: config,
		http:   httpClient,
		Log:    log.New(io.Discard, "", log.LstdFlags),
	}

	if config.Log != nil {
		c.Log = config.Log
	}

	return c
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

type BadRequestResponse struct {
	PropertyName   string `json:"propertyName"`
	ErrorMessage   string `json:"errorMessage"`
	ErrorCode      string `json:"errorCode"`
	AttemptedValue string `json:"attemptedValue"`
	Severity       string `json:"severity"`
}

func (r BadRequestResponse) String() string {
	return fmt.Sprintf("[%s: %s] %s: %s - got value: %s", r.Severity, r.ErrorCode, r.PropertyName, r.ErrorMessage, r.AttemptedValue)
}

type SystemStatusResponse struct {
	Version string `json:"version"`
}

func (c *client) Test(ctx context.Context) (*SystemStatusResponse, error) {
	status, res, err := c.get(ctx, "system/status")
	if err != nil {
		return nil, errors.Wrap(err, "lidarr client get error")
	}

	if status == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	}

	c.Log.Printf("lidarr system/status response status: %v body: %v", status, string(res))

	response := SystemStatusResponse{}
	err = json.Unmarshal(res, &response)
	if err != nil {
		return nil, errors.Wrap(err, "lidarr client error json unmarshal")
	}

	return &response, nil
}

func (c *client) Push(ctx context.Context, release Release) ([]string, error) {
	status, res, err := c.postBody(ctx, "release/push", release)
	if err != nil {
		return nil, errors.Wrap(err, "lidarr client post error")
	}

	c.Log.Printf("lidarr release/push response status: %v body: %v", status, string(res))

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
		return nil, errors.Wrap(err, "lidarr client error json unmarshal")
	}

	// log and return if rejected
	if pushResponse.Rejected {
		rejections := strings.Join(pushResponse.Rejections, ", ")

		c.Log.Printf("lidarr release/push rejected %v reasons: %q\n", release.Title, rejections)
		return pushResponse.Rejections, nil
	}

	return nil, nil
}
