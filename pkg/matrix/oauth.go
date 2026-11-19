// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package matrix

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

const grantTypeDeviceCode = "urn:ietf:params:oauth:grant-type:device_code"

// OAuth error codes returned while polling the token endpoint for a device grant.
const (
	OAuthErrAuthorizationPending = "authorization_pending"
	OAuthErrSlowDown             = "slow_down"
	OAuthErrAccessDenied         = "access_denied"
	OAuthErrExpiredToken         = "expired_token"
	OAuthErrInvalidGrant         = "invalid_grant"
)

// AuthMetadata is the subset of the RFC 8414 server metadata the device login uses.
type AuthMetadata struct {
	Issuer                      string `json:"issuer"`
	TokenEndpoint               string `json:"token_endpoint"`
	RegistrationEndpoint        string `json:"registration_endpoint"`
	DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`
	RevocationEndpoint          string `json:"revocation_endpoint"`
}

// OAuthError is an RFC 6749 error response.
type OAuthError struct {
	StatusCode  int
	Code        string `json:"error"`
	Description string `json:"error_description"`
}

func (e *OAuthError) Error() string {
	if e.Description != "" {
		return "oauth: " + e.Code + ": " + e.Description
	}

	return "oauth: " + e.Code
}

// ClientMetadata is the RFC 7591 registration request, with the fields the Matrix spec requires.
type ClientMetadata struct {
	ClientName              string   `json:"client_name"`
	ClientURI               string   `json:"client_uri"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	ApplicationType         string   `json:"application_type"`
}

// DefaultClientMetadata registers a public client that only uses the device and refresh grants.
func DefaultClientMetadata() ClientMetadata {
	return ClientMetadata{
		ClientName: "autobrr",
		ClientURI:  "https://autobrr.com/",
		GrantTypes: []string{grantTypeDeviceCode, "refresh_token"},
		// RFC 7591 defaults response_types to ["code"], which authorization servers reject
		// without the authorization_code grant
		ResponseTypes:           []string{},
		TokenEndpointAuthMethod: "none",
		ApplicationType:         "native",
	}
}

// DeviceAuthorization is the RFC 8628 device authorization response.
type DeviceAuthorization struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// Token is an access token response from the token endpoint.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

// ExpiresAt returns when the access token expires, or the zero time when it does not.
func (t Token) ExpiresAt(now time.Time) time.Time {
	if t.ExpiresIn <= 0 {
		return time.Time{}
	}

	return now.Add(time.Duration(t.ExpiresIn) * time.Second)
}

// OAuthClient runs the OAuth 2.0 flows against a homeserver's authorization server.
type OAuthClient struct {
	http *http.Client
}

// NewOAuthClient returns an OAuth client using httpClient.
func NewOAuthClient(httpClient *http.Client) *OAuthClient {
	return &OAuthClient{http: httpClient}
}

// RegisterClient dynamically registers a client and returns its client id.
func (c *OAuthClient) RegisterClient(ctx context.Context, metadata *AuthMetadata, client ClientMetadata) (string, error) {
	if metadata.RegistrationEndpoint == "" {
		return "", errors.New("authorization server does not support dynamic client registration")
	}

	data, err := json.Marshal(client)
	if err != nil {
		return "", errors.Wrap(err, "could not marshal client metadata")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, metadata.RegistrationEndpoint, bytes.NewReader(data))
	if err != nil {
		return "", errors.Wrap(err, "could not build request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", decodeOAuthError(resp)
	}

	var out struct {
		ClientID string `json:"client_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", errors.Wrap(err, "could not decode response")
	}

	if out.ClientID == "" {
		return "", errors.New("registration response has no client_id")
	}

	return out.ClientID, nil
}

// RequestDeviceAuthorization starts a device grant for a new Matrix device.
func (c *OAuthClient) RequestDeviceAuthorization(ctx context.Context, metadata *AuthMetadata, clientID, deviceID string) (*DeviceAuthorization, error) {
	form := url.Values{
		"client_id": {clientID},
		"scope":     {"urn:matrix:client:api:* urn:matrix:client:device:" + deviceID},
	}

	var out DeviceAuthorization
	if err := c.postForm(ctx, metadata.DeviceAuthorizationEndpoint, form, &out); err != nil {
		return nil, err
	}

	if out.DeviceCode == "" || out.UserCode == "" || out.VerificationURI == "" || out.ExpiresIn <= 0 {
		return nil, errors.New("incomplete device authorization response")
	}

	if out.Interval <= 0 {
		out.Interval = 5
	}

	return &out, nil
}

// PollDeviceToken makes one token request for a pending device grant. While the
// user has not approved yet it returns an *OAuthError with Code authorization_pending.
func (c *OAuthClient) PollDeviceToken(ctx context.Context, metadata *AuthMetadata, clientID, deviceCode string) (*Token, error) {
	form := url.Values{
		"grant_type":  {grantTypeDeviceCode},
		"device_code": {deviceCode},
		"client_id":   {clientID},
	}

	var out Token
	if err := c.postForm(ctx, metadata.TokenEndpoint, form, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// RefreshToken exchanges a refresh token for a new token pair. Refresh tokens may
// be single-use, so the returned refresh token must replace the old one.
func (c *OAuthClient) RefreshToken(ctx context.Context, tokenEndpoint, clientID, refreshToken string) (*Token, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}

	var out Token
	if err := c.postForm(ctx, tokenEndpoint, form, &out); err != nil {
		return nil, err
	}

	if out.RefreshToken == "" {
		out.RefreshToken = refreshToken
	}

	return &out, nil
}

func (c *OAuthClient) postForm(ctx context.Context, endpoint string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return errors.Wrap(err, "could not build request")
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decodeOAuthError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return errors.Wrap(err, "could not decode response")
	}

	return nil
}

func decodeOAuthError(resp *http.Response) error {
	e := &OAuthError{StatusCode: resp.StatusCode}

	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(e); err != nil || e.Code == "" {
		e.Code = "http_" + strconv.Itoa(resp.StatusCode)
	}

	return e
}

const deviceIDAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// NewDeviceID returns a random Matrix device id.
func NewDeviceID() string {
	b := make([]byte, 10)
	_, _ = rand.Read(b)

	for i := range b {
		b[i] = deviceIDAlphabet[int(b[i])%len(deviceIDAlphabet)]
	}

	return string(b)
}
