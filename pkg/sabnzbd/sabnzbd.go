package sabnzbd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	addr   string
	apiKey string

	basicUser string
	basicPass string

	log *log.Logger

	Http *http.Client
}

type Options struct {
	Addr   string
	ApiKey string

	BasicUser string
	BasicPass string

	Log *log.Logger
}

func New(opts Options) *Client {
	c := &Client{
		addr:      opts.Addr,
		apiKey:    opts.ApiKey,
		basicUser: opts.BasicUser,
		basicPass: opts.BasicPass,
		log:       log.New(io.Discard, "", log.LstdFlags),
		Http: &http.Client{
			Timeout: time.Second * 60,
		},
	}

	if opts.Log != nil {
		c.log = opts.Log
	}

	return c
}

func (c *Client) AddFromUrl(ctx context.Context, link string) (*AddFileResponse, error) {
	v := url.Values{}
	v.Set("mode", "addurl")
	v.Set("name", link)
	v.Set("output", "json")
	v.Set("apikey", c.apiKey)

	addr, err := url.JoinPath(c.addr, "/api")
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	u.RawQuery = v.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	if c.basicUser != "" && c.basicPass != "" {
		req.SetBasicAuth(c.basicUser, c.basicPass)
	}

	res, err := c.Http.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	fmt.Print(body)

	var data AddFileResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func (c *Client) Version(ctx context.Context) (*VersionResponse, error) {
	v := url.Values{}
	v.Set("mode", "version")
	v.Set("output", "json")
	v.Set("apikey", c.apiKey)

	addr, err := url.JoinPath(c.addr, "/api")
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	u.RawQuery = v.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	if c.basicUser != "" && c.basicPass != "" {
		req.SetBasicAuth(c.basicUser, c.basicPass)
	}

	res, err := c.Http.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var data VersionResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

type VersionResponse struct {
	Version string `json:"version"`
}

type AddFileResponse struct {
	NzoIDs []string `json:"nzo_ids"`
	ApiError
}

type ApiError struct {
	ErrorMsg string `json:"error,omitempty"`
}
