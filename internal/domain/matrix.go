// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

// MatrixAuthMethod is how a Matrix network obtained its access token.
type MatrixAuthMethod string

const (
	// MatrixAuthMethodOAuth is an OAuth 2.0 device login with a refreshable token.
	MatrixAuthMethodOAuth MatrixAuthMethod = "OAUTH"
	// MatrixAuthMethodToken is a pasted access token that cannot be refreshed.
	MatrixAuthMethodToken MatrixAuthMethod = "TOKEN"
)

func (m MatrixAuthMethod) String() string {
	return string(m)
}

// IsValid reports whether the method is one of the supported values.
func (m MatrixAuthMethod) IsValid() bool {
	switch m {
	case MatrixAuthMethodOAuth, MatrixAuthMethodToken:
		return true
	default:
		return false
	}
}

// MatrixNetwork is a Matrix account on one homeserver and the announce rooms it listens to.
type MatrixNetwork struct {
	ID         int64            `json:"id"`
	Name       string           `json:"name"`
	Enabled    bool             `json:"enabled"`
	Homeserver string           `json:"homeserver"`
	AuthMethod MatrixAuthMethod `json:"auth_method"`
	// UserID and DeviceID are filled from whoami after login.
	UserID   string `json:"user_id"`
	DeviceID string `json:"device_id"`

	AccessToken        string     `json:"access_token"`
	RefreshToken       string     `json:"-"`
	TokenExpiresAt     *time.Time `json:"-"`
	OAuthClientID      string     `json:"-"`
	OAuthTokenEndpoint string     `json:"-"`

	Rooms  []MatrixRoom         `json:"rooms"`
	Health *MatrixNetworkHealth `json:"health,omitempty"`
}

func (n MatrixNetwork) MarshalJSON() ([]byte, error) {
	type Alias MatrixNetwork
	return json.Marshal(&struct {
		*Alias
		AccessToken string `json:"access_token"`
	}{
		AccessToken: RedactString(n.AccessToken),
		Alias:       (*Alias)(&n),
	})
}

// HasCredentials reports whether the network has an access token to connect with.
func (n MatrixNetwork) HasCredentials() bool {
	return n.AccessToken != ""
}

// CanRefresh reports whether an expired access token can be renewed without a new login.
func (n MatrixNetwork) CanRefresh() bool {
	return n.AuthMethod == MatrixAuthMethodOAuth && n.RefreshToken != "" && n.OAuthClientID != "" && n.OAuthTokenEndpoint != ""
}

// Validate normalizes the homeserver URL and room names and rejects incomplete networks.
func (n *MatrixNetwork) Validate() error {
	if strings.TrimSpace(n.Name) == "" {
		return errors.New("network name must not be empty")
	}

	homeserver, err := NormalizeMatrixHomeserver(n.Homeserver)
	if err != nil {
		return err
	}

	n.Homeserver = homeserver

	if n.AuthMethod != "" && !n.AuthMethod.IsValid() {
		return errors.New("invalid matrix auth method: %q", n.AuthMethod)
	}

	for i := range n.Rooms {
		if err := n.Rooms[i].Validate(); err != nil {
			return err
		}
	}

	return nil
}

// NormalizeMatrixHomeserver returns the homeserver base URL as scheme://host[:port],
// defaulting to https when no scheme is given.
func NormalizeMatrixHomeserver(homeserver string) (string, error) {
	homeserver = strings.TrimSpace(homeserver)
	if homeserver == "" {
		return "", errors.New("homeserver must not be empty")
	}

	if !strings.Contains(homeserver, "://") {
		homeserver = "https://" + homeserver
	}

	u, err := url.Parse(homeserver)
	if err != nil {
		return "", errors.Wrap(err, "invalid homeserver url: '%s'", homeserver)
	}

	if u.Scheme != "https" && u.Scheme != "http" {
		return "", errors.New("invalid homeserver url scheme: '%s'", u.Scheme)
	}

	if u.Host == "" {
		return "", errors.New("invalid homeserver url: '%s'", homeserver)
	}

	if u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("homeserver url must not contain credentials, a query or a fragment: '%s'", homeserver)
	}

	return u.Scheme + "://" + strings.ToLower(u.Host) + strings.TrimRight(u.Path, "/"), nil
}

// MatrixRoom is a room a network joins and listens to for announces.
type MatrixRoom struct {
	ID      int64 `json:"id"`
	Enabled bool  `json:"enabled"`
	// Name is the room alias (#room:server) or room id (!id:server) from the indexer definition.
	Name string `json:"name"`
	// RoomID is the resolved room id, set after the first join.
	RoomID string            `json:"room_id"`
	Health *MatrixRoomHealth `json:"health,omitempty"`
}

// Validate rejects names that are not aliases or room ids. Names keep their case
// because room ids are case-sensitive.
func (r *MatrixRoom) Validate() error {
	r.Name = strings.TrimSpace(r.Name)

	if !IsMatrixRoomName(r.Name) {
		return errors.New("invalid matrix room: '%s' must be an alias like #room:server or a room id like !id:server", r.Name)
	}

	return nil
}

// IsMatrixRoomName reports whether name is a room alias or room id with a server part.
func IsMatrixRoomName(name string) bool {
	if len(name) < 4 || (name[0] != '#' && name[0] != '!') {
		return false
	}

	localpart, server, ok := strings.Cut(name[1:], ":")

	return ok && localpart != "" && server != ""
}

// MatrixNetworkHealth is the runtime state of a network's sync loop.
type MatrixNetworkHealth struct {
	Connected      bool       `json:"connected"`
	ConnectedSince *time.Time `json:"connected_since,omitempty"`
	Healthy        bool       `json:"healthy"`
	Error          string     `json:"error,omitempty"`
	// LoginRequired is set when the network has no login or its login was rejected.
	LoginRequired bool         `json:"login_required"`
	Login         *MatrixLogin `json:"login,omitempty"`
}

// MatrixRoomHealth is the runtime state of a joined room.
type MatrixRoomHealth struct {
	Joined       bool       `json:"joined"`
	JoinedSince  *time.Time `json:"joined_since,omitempty"`
	LastAnnounce *time.Time `json:"last_announce,omitempty"`
	Error        string     `json:"error,omitempty"`
}

// MatrixLoginState is the progress of a device login.
type MatrixLoginState string

const (
	MatrixLoginStatePending   MatrixLoginState = "PENDING"
	MatrixLoginStateSucceeded MatrixLoginState = "SUCCEEDED"
	MatrixLoginStateFailed    MatrixLoginState = "FAILED"
)

// MatrixLogin is a device login the user completes by entering UserCode at VerificationURI.
type MatrixLogin struct {
	State                   MatrixLoginState `json:"state"`
	UserCode                string           `json:"user_code,omitempty"`
	VerificationURI         string           `json:"verification_uri,omitempty"`
	VerificationURIComplete string           `json:"verification_uri_complete,omitempty"`
	ExpiresAt               time.Time        `json:"expires_at"`
	UserID                  string           `json:"user_id,omitempty"`
	Error                   string           `json:"error,omitempty"`
}

// MatrixTokens is the credential set a login or token refresh persists in one write.
type MatrixTokens struct {
	AuthMethod         MatrixAuthMethod
	UserID             string
	DeviceID           string
	AccessToken        string
	RefreshToken       string
	TokenExpiresAt     *time.Time
	OAuthClientID      string
	OAuthTokenEndpoint string
}

// MatrixSetTokenRequest stores a pasted access token on a network.
type MatrixSetTokenRequest struct {
	NetworkID   int64  `json:"-"`
	AccessToken string `json:"access_token"`
}

// MatrixManualProcessRequest feeds a message into a room's announce processor as if the announcer had sent it.
type MatrixManualProcessRequest struct {
	NetworkID int64  `json:"-"`
	Room      string `json:"room"`
	Message   string `json:"msg"`
}

// MatrixMessage is a room message kept in the room history and streamed to the UI.
type MatrixMessage struct {
	// ID is the Matrix event id, which lets the UI merge history and live messages without duplicates.
	ID      string    `json:"id"`
	Network int64     `json:"network"`
	Room    string    `json:"room"`
	Sender  string    `json:"sender"`
	Message string    `json:"msg"`
	Time    time.Time `json:"time"`
}

// MatrixHealthEvent is the health of a network and its rooms, streamed to the UI on every change.
type MatrixHealthEvent struct {
	Network int64                       `json:"network"`
	Health  MatrixNetworkHealth         `json:"health"`
	Rooms   map[string]MatrixRoomHealth `json:"rooms"`
}
