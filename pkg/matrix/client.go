// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

// Package matrix is a minimal Matrix client-server API client covering what an
// announce listener needs: OAuth 2.0 device login, joining rooms and /sync.
// It does not support end-to-end encryption.
package matrix

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

// ErrCodeUnknownToken is returned by the homeserver for expired or revoked access tokens.
const ErrCodeUnknownToken = "M_UNKNOWN_TOKEN"

// Error is a standard Matrix error response.
type Error struct {
	StatusCode int
	ErrCode    string `json:"errcode"`
	Message    string `json:"error"`
	SoftLogout bool   `json:"soft_logout"`
	RetryAfter time.Duration
}

func (e *Error) Error() string {
	if e.ErrCode == "" {
		return "matrix: unexpected status " + strconv.Itoa(e.StatusCode)
	}

	return "matrix: " + strconv.Itoa(e.StatusCode) + " " + e.ErrCode + ": " + e.Message
}

// IsUnauthorized reports whether the access token was rejected.
func (e *Error) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized || e.ErrCode == ErrCodeUnknownToken
}

// Client talks to a single homeserver with one access token.
type Client struct {
	homeserver string
	http       *http.Client

	m     sync.RWMutex
	token string
}

// NewClient returns a client for the homeserver base URL, e.g. https://matrix.example.org.
func NewClient(homeserver string, httpClient *http.Client) *Client {
	return &Client{
		homeserver: strings.TrimRight(homeserver, "/"),
		http:       httpClient,
	}
}

// SetAccessToken replaces the token sent with authenticated requests.
func (c *Client) SetAccessToken(token string) {
	c.m.Lock()
	c.token = token
	c.m.Unlock()
}

func (c *Client) accessToken() string {
	c.m.RLock()
	defer c.m.RUnlock()

	return c.token
}

// WhoamiResponse identifies the owner of an access token.
type WhoamiResponse struct {
	UserID   string `json:"user_id"`
	DeviceID string `json:"device_id"`
}

// Whoami returns the user and device the access token belongs to.
func (c *Client) Whoami(ctx context.Context) (*WhoamiResponse, error) {
	var resp WhoamiResponse
	if err := c.do(ctx, http.MethodGet, "/_matrix/client/v3/account/whoami", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// JoinRoom joins a room by id or alias and returns the room id. Joining a room
// the user is already in succeeds.
func (c *Client) JoinRoom(ctx context.Context, roomIDOrAlias string) (string, error) {
	var resp struct {
		RoomID string `json:"room_id"`
	}

	if err := c.do(ctx, http.MethodPost, "/_matrix/client/v3/join/"+url.PathEscape(roomIDOrAlias), nil, struct{}{}, &resp); err != nil {
		return "", err
	}

	if resp.RoomID == "" {
		return "", errors.New("join response for %s has no room_id", roomIDOrAlias)
	}

	return resp.RoomID, nil
}

// AuthMetadata returns the OAuth 2.0 server metadata the homeserver delegates authentication to.
func (c *Client) AuthMetadata(ctx context.Context) (*AuthMetadata, error) {
	var resp AuthMetadata
	if err := c.do(ctx, http.MethodGet, "/_matrix/client/v1/auth_metadata", nil, nil, &resp); err != nil {
		return nil, err
	}

	if resp.DeviceAuthorizationEndpoint == "" || resp.TokenEndpoint == "" {
		return nil, errors.New("homeserver authorization server does not support the device authorization grant")
	}

	return &resp, nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	u := c.homeserver + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return errors.Wrap(err, "could not marshal request body")
		}

		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return errors.Wrap(err, "could not build request")
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if token := c.accessToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decodeError(resp)
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return errors.Wrap(err, "could not decode response")
	}

	return nil
}

func decodeError(resp *http.Response) error {
	e := &Error{StatusCode: resp.StatusCode}

	var body struct {
		ErrCode      string `json:"errcode"`
		Message      string `json:"error"`
		SoftLogout   bool   `json:"soft_logout"`
		RetryAfterMS int64  `json:"retry_after_ms"`
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&body); err == nil {
		e.ErrCode = body.ErrCode
		e.Message = body.Message
		e.SoftLogout = body.SoftLogout
		e.RetryAfter = time.Duration(body.RetryAfterMS) * time.Millisecond
	}

	if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && seconds > 0 {
		e.RetryAfter = time.Duration(seconds) * time.Second
	}

	return e
}
