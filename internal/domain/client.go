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

	// cached http client
	Client   any
	Settings DownloadClientSettings `json:"settings,omitempty"`

	Name          string             `json:"name"`
	Type          DownloadClientType `json:"type"`
	Host          string             `json:"host"`
	Username      string             `json:"username"`
	Password      string             `json:"password"`
	Port          int                `json:"port"`
	ID            int32              `json:"id"`
	Enabled       bool               `json:"enabled"`
	TLS           bool               `json:"tls"`
	TLSSkipVerify bool               `json:"tls_skip_verify"`
}

type DownloadClientSettings struct {
	Auth                     DownloadClientAuth  `json:"auth,omitempty"`
	Basic                    BasicAuth           `json:"basic,omitempty"` // Deprecated: Use Auth instead
	APIKey                   string              `json:"apikey,omitempty"`
	ExternalDownloadClient   string              `json:"external_download_client,omitempty"`
	Rules                    DownloadClientRules `json:"rules,omitempty"`
	ExternalDownloadClientId int                 `json:"external_download_client_id,omitempty"`
}

// MarshalJSON Custom method to translate Basic into Auth without including Basic in JSON output
func (dcs *DownloadClientSettings) MarshalJSON() ([]byte, error) {
	// Ensuring Auth is updated with Basic info before marshaling if Basic is set
	if dcs.Basic.Username != "" || dcs.Basic.Password != "" {
		dcs.Auth = DownloadClientAuth{
			Enabled:  dcs.Basic.Auth,
			Type:     DownloadClientAuthTypeBasic,
			Username: dcs.Basic.Username,
			Password: dcs.Basic.Password,
		}
	}

	type Alias DownloadClientSettings
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(dcs),
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

	return nil
}

type DownloadClientAuthType string

const (
	DownloadClientAuthTypeNone   = "NONE"
	DownloadClientAuthTypeBasic  = "BASIC_AUTH"
	DownloadClientAuthTypeDigest = "DIGEST_AUTH"
)

type DownloadClientAuth struct {
	Type     DownloadClientAuthType `json:"type,omitempty"`
	Username string                 `json:"username,omitempty"`
	Password string                 `json:"password,omitempty"`
	Enabled  bool                   `json:"enabled,omitempty"`
}

type DownloadClientRules struct {
	IgnoreSlowTorrentsCondition IgnoreSlowTorrentsCondition `json:"ignore_slow_torrents_condition,omitempty"`
	MaxActiveDownloads          int                         `json:"max_active_downloads"`
	DownloadSpeedThreshold      int64                       `json:"download_speed_threshold"`
	UploadSpeedThreshold        int64                       `json:"upload_speed_threshold"`
	Enabled                     bool                        `json:"enabled"`
	IgnoreSlowTorrents          bool                        `json:"ignore_slow_torrents"`
}

type BasicAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Auth     bool   `json:"auth,omitempty"`
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
	Label string `json:"label"`
	ID    int    `json:"id"`
}
