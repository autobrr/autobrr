package domain

type DownloadClientRepo interface {
	//FindByActionID(actionID int) ([]DownloadClient, error)
	List() ([]DownloadClient, error)
	FindByID(id int32) (*DownloadClient, error)
	Store(client DownloadClient) (*DownloadClient, error)
	Delete(clientID int) error
}

type DownloadClient struct {
	ID       int                    `json:"id"`
	Name     string                 `json:"name"`
	Type     DownloadClientType     `json:"type"`
	Enabled  bool                   `json:"enabled"`
	Host     string                 `json:"host"`
	Port     int                    `json:"port"`
	SSL      bool                   `json:"ssl"`
	Username string                 `json:"username"`
	Password string                 `json:"password"`
	Settings DownloadClientSettings `json:"settings,omitempty"`
}

type DownloadClientSettings struct {
	APIKey string    `json:"apikey,omitempty"`
	Basic  BasicAuth `json:"basic,omitempty"`
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
