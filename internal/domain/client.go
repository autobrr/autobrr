package domain

type DownloadClientRepo interface {
	//FindByActionID(actionID int) ([]DownloadClient, error)
	List() ([]DownloadClient, error)
	FindByID(id int32) (*DownloadClient, error)
	Store(client DownloadClient) (*DownloadClient, error)
	Delete(clientID int) error
}

type DownloadClient struct {
	ID       int                `json:"id"`
	Name     string             `json:"name"`
	Type     DownloadClientType `json:"type"`
	Enabled  bool               `json:"enabled"`
	Host     string             `json:"host"`
	Port     int                `json:"port"`
	SSL      bool               `json:"ssl"`
	Username string             `json:"username"`
	Password string             `json:"password"`
}

type DownloadClientType string

const (
	DownloadClientTypeQbittorrent DownloadClientType = "QBITTORRENT"
	DownloadClientTypeDeluge      DownloadClientType = "DELUGE"
)
