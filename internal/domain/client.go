package domain

import (
	"context"
	"fmt"
	"net/url"

	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/autobrr/go-qbittorrent"
)

type DownloadClientRepo interface {
	List(ctx context.Context) ([]DownloadClient, error)
	FindByID(ctx context.Context, id int32) (*DownloadClient, error)
	Store(ctx context.Context, client DownloadClient) (*DownloadClient, error)
	Update(ctx context.Context, client DownloadClient) (*DownloadClient, error)
	Delete(ctx context.Context, clientID int) error
}

type DownloadClient struct {
	ID            int                    `json:"id"`
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
}

type DownloadClientCached struct {
	Dc  *DownloadClient
	Qbt *qbittorrent.Client
}

type DownloadClientSettings struct {
	APIKey string              `json:"apikey,omitempty"`
	Basic  BasicAuth           `json:"basic,omitempty"`
	Rules  DownloadClientRules `json:"rules,omitempty"`
}

type DownloadClientRules struct {
	Enabled                bool  `json:"enabled"`
	MaxActiveDownloads     int   `json:"max_active_downloads"`
	IgnoreSlowTorrents     bool  `json:"ignore_slow_torrents"`
	DownloadSpeedThreshold int64 `json:"download_speed_threshold"`
	UploadSpeedThreshold   int64 `json:"upload_speed_threshold"`
}

type BasicAuth struct {
	Auth     bool   `json:"auth,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type DownloadClientType string

const (
	DownloadClientTypeQbittorrent  DownloadClientType = "QBITTORRENT"
	DownloadClientTypeDelugeV1     DownloadClientType = "DELUGE_V1"
	DownloadClientTypeDelugeV2     DownloadClientType = "DELUGE_V2"
	DownloadClientTypeRTorrent     DownloadClientType = "RTORRENT"
	DownloadClientTypeTransmission DownloadClientType = "TRANSMISSION"
	DownloadClientTypeRadarr       DownloadClientType = "RADARR"
	DownloadClientTypeSonarr       DownloadClientType = "SONARR"
	DownloadClientTypeLidarr       DownloadClientType = "LIDARR"
	DownloadClientTypeWhisparr     DownloadClientType = "WHISPARR"
	DownloadClientTypeReadarr      DownloadClientType = "READARR"
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

func (c DownloadClient) BuildLegacyHost() string {
	if c.Type == DownloadClientTypeQbittorrent {
		return c.qbitBuildLegacyHost()
	}
	return ""
}

// qbitBuildLegacyHost exists to support older configs
func (c DownloadClient) qbitBuildLegacyHost() string {
	// parse url
	u, _ := url.Parse(c.Host)

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
	return u.String()
}
