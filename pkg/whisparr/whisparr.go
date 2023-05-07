// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package whisparr

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
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

func New(config Config) Client {

	httpClient := &http.Client{
		Timeout: time.Second * 30,
	}

	c := &client{
		config: config,
		http:   httpClient,
		Log:    config.Log,
	}

	if config.Log == nil {
		c.Log = log.New(io.Discard, "", log.LstdFlags)
	}

	return c
}

type Release struct {
	Title            string `json:"title"`
	DownloadUrl      string `json:"downloadUrl,omitempty"`
	MagnetUrl        string `json:"magnetUrl,omitempty"`
	Size             int64  `json:"size"`
	Indexer          string `json:"indexer"`
	DownloadProtocol string `json:"downloadProtocol"`
	Protocol         string `json:"protocol"`
	PublishDate      string `json:"publishDate"`
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

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not read body")
	}

	response := SystemStatusResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	c.Log.Printf("whisparr system/status status: (%v) response: %v\n", res.Status, string(body))

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

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not read body")
	}

	pushResponse := make([]PushResponse, 0)
	err = json.Unmarshal(body, &pushResponse)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	c.Log.Printf("whisparr release/push status: (%v) response: %v\n", res.Status, string(body))

	// log and return if rejected
	if pushResponse[0].Rejected {
		rejections := strings.Join(pushResponse[0].Rejections, ", ")

		c.Log.Printf("whisparr release/push rejected %v reasons: %q\n", release.Title, rejections)
		return pushResponse[0].Rejections, nil
	}

	// success true
	return nil, nil
}
