package domain

import "context"

type DownloadClientRepo interface {
	//FindByActionID(actionID int) ([]DownloadClient, error)
	List(ctx context.Context) ([]DownloadClient, error)
	FindByID(ctx context.Context, id int32) (*DownloadClient, error)
	Store(ctx context.Context, client DownloadClient) (*DownloadClient, error)
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
}

type BasicAuth struct {
	Auth     bool   `json:"auth,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type DownloadClientType string

const (
	DownloadClientTypeQbittorrent DownloadClientType = "QBITTORRENT"
	DownloadClientTypeDelugeV1    DownloadClientType = "DELUGE_V1"
	DownloadClientTypeDelugeV2    DownloadClientType = "DELUGE_V2"
	DownloadClientTypeRadarr      DownloadClientType = "RADARR"
	DownloadClientTypeSonarr      DownloadClientType = "SONARR"
	DownloadClientTypeLidarr      DownloadClientType = "LIDARR"
)
