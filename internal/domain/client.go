// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/autobrr/autobrr/pkg/errors"
)

type DownloadClientRepo interface {
	List(ctx context.Context) ([]DownloadClient, error)
	FindByID(ctx context.Context, id int32) (*DownloadClient, error)
	Store(ctx context.Context, client *DownloadClient) error
	Update(ctx context.Context, client *DownloadClient) error
	Delete(ctx context.Context, clientID int32) error
}

type DownloadClient struct {
	ID            int32                  `json:"id"`
	Name          string                 `json:"name"`
	Type          DownloadClientType     `json:"type"`
	Enabled       bool                   `json:"enabled"`
	Host          string                 `json:"host"`
	Port          int                    `json:"port"`
	TLS           bool                   `json:"tls"`
	TLSSkipVerify bool                   `json:"tls_skip_verify"`
	Username      string                 `json:"username"`
	Password      string                 `json:"password"`
	Settings      DownloadClientSettings `json:"settings,omitempty"`

	// cached http client
	Client any `json:"-"`
}

func (c DownloadClient) MarshalJSON() ([]byte, error) {
	type Alias DownloadClient
	return json.Marshal(&struct {
		*Alias
		Password string `json:"password"`
	}{
		Password: RedactString(c.Password),
		Alias:    (*Alias)(&c),
	})
}

func (c *DownloadClient) UnmarshalJSON(data []byte) error {
	type Alias DownloadClient
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// If the password appears to be redacted, don't overwrite the existing value
	if isRedactedValue(c.Password) {
		// Keep the original password by not updating it
		return nil
	}

	return nil
}

type DownloadClientSettings struct {
	APIKey                   string              `json:"apikey,omitempty"`
	Basic                    BasicAuth           `json:"basic,omitempty"` // Deprecated: Use Auth instead
	Rules                    DownloadClientRules `json:"rules,omitempty"`
	ExternalDownloadClientId int                 `json:"external_download_client_id,omitempty"`
	ExternalDownloadClient   string              `json:"external_download_client,omitempty"`
	Auth                     DownloadClientAuth  `json:"auth,omitempty"`
}

// MarshalJSON Custom method to translate Basic into Auth without including Basic in JSON output
func (dcs *DownloadClientSettings) MarshalJSON() ([]byte, error) {
	// Ensuring Auth is updated with Basic info before marshaling if Basic is set
	if dcs.Basic.Username != "" || dcs.Basic.Password != "" {
		dcs.Auth = DownloadClientAuth{
			Enabled:  dcs.Basic.Auth,
			Type:     DownloadClientAuthTypeBasic,
			Username: dcs.Basic.Username,
			Password: RedactString(dcs.Basic.Password),
		}
	}

	type Alias DownloadClientSettings
	return json.Marshal(&struct {
		*Alias
		APIKey string `json:"apikey,omitempty"`
	}{
		APIKey: RedactString(dcs.APIKey),
		Alias:  (*Alias)(dcs),
	})
}

// UnmarshalJSON Custom method to translate Basic into Auth
func (dcs *DownloadClientSettings) UnmarshalJSON(data []byte) error {
	type Alias DownloadClientSettings
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dcs),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// If Basic fields are not empty, populate Auth fields accordingly
	if aux.Basic.Username != "" || aux.Basic.Password != "" {
		dcs.Auth = DownloadClientAuth{
			Enabled:  aux.Basic.Auth,
			Type:     DownloadClientAuthTypeBasic,
			Username: aux.Basic.Username,
			Password: aux.Basic.Password,
		}
	}

	if isRedactedValue(aux.APIKey) {
		return nil
	}

	return nil
}

type DownloadClientAuthType string

const (
	DownloadClientAuthTypeNone   = "NONE"
	DownloadClientAuthTypeBasic  = "BASIC_AUTH"
	DownloadClientAuthTypeDigest = "DIGEST_AUTH"
)

type DownloadClientAuth struct {
	Enabled  bool                   `json:"enabled,omitempty"`
	Type     DownloadClientAuthType `json:"type,omitempty"`
	Username string                 `json:"username,omitempty"`
	Password string                 `json:"password,omitempty"`
}

func (d DownloadClientAuth) MarshalJSON() ([]byte, error) {
	type Alias DownloadClientAuth
	return json.Marshal(&struct {
		*Alias
		Password string `json:"password,omitempty"`
	}{
		Password: RedactString(d.Password),
		Alias:    (*Alias)(&d),
	})
}

func (d *DownloadClientAuth) UnmarshalJSON(data []byte) error {
	type Alias DownloadClientAuth
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(d),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// If the password appears to be redacted, don't overwrite the existing value
	if isRedactedValue(d.Password) {
		// Keep the original password by not updating it
		return nil
	}

	return nil
}

type DownloadClientRules struct {
	Enabled                     bool                        `json:"enabled"`
	MaxActiveDownloads          int                         `json:"max_active_downloads"`
	IgnoreSlowTorrents          bool                        `json:"ignore_slow_torrents"`
	IgnoreSlowTorrentsCondition IgnoreSlowTorrentsCondition `json:"ignore_slow_torrents_condition,omitempty"`
	DownloadSpeedThreshold      int64                       `json:"download_speed_threshold"`
	UploadSpeedThreshold        int64                       `json:"upload_speed_threshold"`
}

type BasicAuth struct {
	Auth     bool   `json:"auth,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func (b BasicAuth) MarshalJSON() ([]byte, error) {
	type Alias BasicAuth
	return json.Marshal(&struct {
		*Alias
		Password string `json:"password,omitempty"`
	}{
		Password: RedactString(b.Password),
		Alias:    (*Alias)(&b),
	})
}

func (b *BasicAuth) UnmarshalJSON(data []byte) error {
	type Alias BasicAuth
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(b),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// If the password appears to be redacted, don't overwrite the existing value
	if isRedactedValue(b.Password) {
		// Keep the original password by not updating it
		return nil
	}

	return nil
}

type IgnoreSlowTorrentsCondition string

const (
	IgnoreSlowTorrentsModeAlways     IgnoreSlowTorrentsCondition = "ALWAYS"
	IgnoreSlowTorrentsModeMaxReached IgnoreSlowTorrentsCondition = "MAX_DOWNLOADS_REACHED"
)

type DownloadClientType string

const (
	DownloadClientTypeQbittorrent  DownloadClientType = "QBITTORRENT"
	DownloadClientTypeDelugeV1     DownloadClientType = "DELUGE_V1"
	DownloadClientTypeDelugeV2     DownloadClientType = "DELUGE_V2"
	DownloadClientTypeRTorrent     DownloadClientType = "RTORRENT"
	DownloadClientTypeTransmission DownloadClientType = "TRANSMISSION"
	DownloadClientTypePorla        DownloadClientType = "PORLA"
	DownloadClientTypeRadarr       DownloadClientType = "RADARR"
	DownloadClientTypeSonarr       DownloadClientType = "SONARR"
	DownloadClientTypeLidarr       DownloadClientType = "LIDARR"
	DownloadClientTypeWhisparr     DownloadClientType = "WHISPARR"
	DownloadClientTypeReadarr      DownloadClientType = "READARR"
	DownloadClientTypeSabnzbd      DownloadClientType = "SABNZBD"
)

// Validate basic validation of client
func (c DownloadClient) Validate() error {
	// basic validation of client
	if c.Host == "" {
		return errors.New("validation error: missing host")
	} else if c.Type == "" {
		return errors.New("validation error: missing type")
	}

	return nil
}

func (c DownloadClient) BuildLegacyHost() (string, error) {
	if c.Type == DownloadClientTypeQbittorrent {
		return c.qbitBuildLegacyHost()
	}
	return c.Host, nil
}

// qbitBuildLegacyHost exists to support older configs
func (c DownloadClient) qbitBuildLegacyHost() (string, error) {
	// parse url
	u, err := url.Parse(c.Host)
	if err != nil {
		return "", err
	}

	// reset Opaque
	u.Opaque = ""

	// set scheme
	scheme := "http"
	if c.TLS {
		scheme = "https"
	}
	u.Scheme = scheme

	// if host is empty lets use one from settings
	if u.Host == "" {
		u.Host = c.Host
	}

	// reset Path
	if u.Host == u.Path {
		u.Path = ""
	}

	// handle ports
	if c.Port > 0 {
		if c.Port == 80 || c.Port == 443 {
			// skip for regular http and https
		} else {
			u.Host = fmt.Sprintf("%v:%v", u.Host, c.Port)
		}
	}

	// make into new string and return
	return u.String(), nil
}

type ArrTag struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}
