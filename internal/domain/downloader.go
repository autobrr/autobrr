// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/autobrr/autobrr/pkg/errors"
)

type Downloader struct {
	ID            int32              `json:"id"`
	Name          string             `json:"name"`
	Type          DownloaderType     `json:"type"`
	Enabled       bool               `json:"enabled"`
	Host          string             `json:"host"`
	Port          int                `json:"port"`
	TLS           bool               `json:"tls"`
	TLSSkipVerify bool               `json:"tls_skip_verify"`
	Username      string             `json:"username"`
	Password      string             `json:"password"`
	Settings      DownloaderSettings `json:"settings,omitempty"`
}

func (c Downloader) MarshalJSON() ([]byte, error) {
	redactedSettings := DownloaderSettings{
		APIKey:                   RedactString(c.Settings.APIKey),
		Rules:                    c.Settings.Rules,
		ExternalDownloadClientId: c.Settings.ExternalDownloadClientId,
		ExternalDownloadClient:   c.Settings.ExternalDownloadClient,
		Auth: DownloaderAuth{
			Enabled:  c.Settings.Auth.Enabled,
			Type:     c.Settings.Auth.Type,
			Username: c.Settings.Auth.Username,
			Password: RedactString(c.Settings.Auth.Password),
		},
		Basic: BasicAuth{
			Auth:     c.Settings.Basic.Auth,
			Username: c.Settings.Basic.Username,
			Password: RedactString(c.Settings.Basic.Password),
		},
	}

	type Alias Downloader
	return json.Marshal(&struct {
		*Alias
		Password string             `json:"password"`
		Settings DownloaderSettings `json:"settings"`
	}{
		Password: RedactString(c.Password),
		Settings: redactedSettings,
		Alias:    (*Alias)(&c),
	})
}

type DownloaderSettings struct {
	APIKey                   string          `json:"apikey,omitempty"`
	Basic                    BasicAuth       `json:"basic,omitempty"` // Deprecated: Use Auth instead
	Rules                    DownloaderRules `json:"rules,omitempty"`
	ExternalDownloadClientId int             `json:"external_download_client_id,omitempty"`
	ExternalDownloadClient   string          `json:"external_download_client,omitempty"`
	Auth                     DownloaderAuth  `json:"auth,omitempty"`
}

// MarshalJSON Custom method to translate Basic into Auth without including Basic in JSON output
func (dcs *DownloaderSettings) MarshalJSON() ([]byte, error) {
	// Ensuring Auth is updated with Basic info before marshaling if Basic is set
	if dcs.Basic.Username != "" || dcs.Basic.Password != "" {
		dcs.Auth = DownloaderAuth{
			Enabled:  dcs.Basic.Auth,
			Type:     DownloaderAuthTypeBasic,
			Username: dcs.Basic.Username,
			Password: dcs.Basic.Password,
		}
	}

	type Alias DownloaderSettings
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(dcs),
	})
}

// UnmarshalJSON Custom method to translate Basic into Auth
func (dcs *DownloaderSettings) UnmarshalJSON(data []byte) error {
	type Alias DownloaderSettings
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
		dcs.Auth = DownloaderAuth{
			Enabled:  aux.Basic.Auth,
			Type:     DownloaderAuthTypeBasic,
			Username: aux.Basic.Username,
			Password: aux.Basic.Password,
		}
	}

	return nil
}

type DownloaderAuthType string

const (
	DownloaderAuthTypeNone   DownloaderAuthType = "NONE"
	DownloaderAuthTypeBasic  DownloaderAuthType = "BASIC_AUTH"
	DownloaderAuthTypeDigest DownloaderAuthType = "DIGEST_AUTH"
)

type DownloaderAuth struct {
	Enabled  bool               `json:"enabled,omitempty"`
	Type     DownloaderAuthType `json:"type,omitempty"`
	Username string             `json:"username,omitempty"`
	Password string             `json:"password,omitempty"`
}

//func (d DownloaderAuth) MarshalJSON() ([]byte, error) {
//	type Alias DownloaderAuth
//	return json.Marshal(&struct {
//		*Alias
//		Password string `json:"password,omitempty"`
//	}{
//		Password: RedactString(d.Password),
//		Alias:    (*Alias)(&d),
//	})
//}

type DownloaderRules struct {
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

//func (b BasicAuth) MarshalJSON() ([]byte, error) {
//	type Alias BasicAuth
//	return json.Marshal(&struct {
//		*Alias
//		Password string `json:"password,omitempty"`
//	}{
//		Password: RedactString(b.Password),
//		Alias:    (*Alias)(&b),
//	})
//}

type IgnoreSlowTorrentsCondition string

const (
	IgnoreSlowTorrentsModeAlways     IgnoreSlowTorrentsCondition = "ALWAYS"
	IgnoreSlowTorrentsModeMaxReached IgnoreSlowTorrentsCondition = "MAX_DOWNLOADS_REACHED"
)

type DownloaderType string

const (
	DownloaderTypeQbittorrent  DownloaderType = "QBITTORRENT"
	DownloaderTypeDelugeV1     DownloaderType = "DELUGE_V1"
	DownloaderTypeDelugeV2     DownloaderType = "DELUGE_V2"
	DownloaderTypeRTorrent     DownloaderType = "RTORRENT"
	DownloaderTypeTransmission DownloaderType = "TRANSMISSION"
	DownloaderTypePorla        DownloaderType = "PORLA"
	DownloaderTypeAria2        DownloaderType = "ARIA2"
	DownloaderTypeRadarr       DownloaderType = "RADARR"
	DownloaderTypeSonarr       DownloaderType = "SONARR"
	DownloaderTypeLidarr       DownloaderType = "LIDARR"
	DownloaderTypeWhisparr     DownloaderType = "WHISPARR"
	DownloaderTypeWhisparrV3   DownloaderType = "WHISPARR_V3"
	DownloaderTypeReadarr      DownloaderType = "READARR"
	DownloaderTypeSportarr     DownloaderType = "SPORTARR"
	DownloaderTypeSabnzbd      DownloaderType = "SABNZBD"
	DownloaderTypeNzbget       DownloaderType = "NZBGET"
)

func (t DownloaderType) Valid() error {
	switch t {
	case DownloaderTypeQbittorrent,
		DownloaderTypeDelugeV1,
		DownloaderTypeDelugeV2,
		DownloaderTypeRTorrent,
		DownloaderTypeTransmission,
		DownloaderTypePorla,
		DownloaderTypeAria2,
		DownloaderTypeNzbget,
		DownloaderTypeSabnzbd,
		DownloaderTypeLidarr,
		DownloaderTypeRadarr,
		DownloaderTypeReadarr,
		DownloaderTypeSonarr,
		DownloaderTypeSportarr,
		DownloaderTypeWhisparr,
		DownloaderTypeWhisparrV3:
		return nil
	default:
		return errors.New("invalid download client type")
	}
}

// Validate basic validation of client
func (c Downloader) Validate() error {
	// basic validation of client
	if c.Host == "" {
		return errors.New("validation error: missing host")
	}

	if err := c.Type.Valid(); err != nil {
		return err
	}

	return nil
}

func (c Downloader) BuildLegacyHost() (string, error) {
	if c.Type == DownloaderTypeQbittorrent {
		return c.qbitBuildLegacyHost()
	}
	if c.Type == DownloaderTypeTransmission {
		return c.transmissionBuildLegacyHost()
	}
	return c.Host, nil
}

// qbitBuildLegacyHost exists to support older configs
func (c Downloader) qbitBuildLegacyHost() (string, error) {
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

// transmissionBuildLegacyHost builds the full Transmission RPC URL from host, port, and tls settings
func (c Downloader) transmissionBuildLegacyHost() (string, error) {
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

	// reset Path if it's the same as Host (means Host was just a hostname)
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

	// Ensure path ends with /rpc
	if !strings.HasSuffix(u.Path, "/rpc") {
		path, err := url.JoinPath(u.Path, "/rpc")
		if err != nil {
			return "", err
		}
		u.Path = path
	}

	// make into new string and return
	return u.String(), nil
}

type ArrTag struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}
