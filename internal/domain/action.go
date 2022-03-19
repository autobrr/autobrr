package domain

import "context"

type ActionRepo interface {
	Store(ctx context.Context, action Action) (*Action, error)
	StoreFilterActions(ctx context.Context, actions []Action, filterID int64) ([]Action, error)
	DeleteByFilterID(ctx context.Context, filterID int) error
	FindByFilterID(ctx context.Context, filterID int) ([]Action, error)
	List() ([]Action, error)
	Delete(actionID int) error
	ToggleEnabled(actionID int) error
}

type Action struct {
	ID                 int        `json:"id"`
	Name               string     `json:"name"`
	Type               ActionType `json:"type"`
	Enabled            bool       `json:"enabled"`
	ExecCmd            string     `json:"exec_cmd,omitempty"`
	ExecArgs           string     `json:"exec_args,omitempty"`
	WatchFolder        string     `json:"watch_folder,omitempty"`
	Category           string     `json:"category,omitempty"`
	Tags               string     `json:"tags,omitempty"`
	Label              string     `json:"label,omitempty"`
	SavePath           string     `json:"save_path,omitempty"`
	Paused             bool       `json:"paused,omitempty"`
	IgnoreRules        bool       `json:"ignore_rules,omitempty"`
	LimitUploadSpeed   int64      `json:"limit_upload_speed,omitempty"`
	LimitDownloadSpeed int64      `json:"limit_download_speed,omitempty"`
	Host               string     `json:"host,omitempty"`
	Data               string     `json:"data,omitempty"`
	Headers            []string   `json:"headers,omitempty"`
	FilterID           int        `json:"filter_id,omitempty"`
	ClientID           int32      `json:"client_id,omitempty"`
}

type ActionType string

const (
	ActionTypeTest        ActionType = "TEST"
	ActionTypeExec        ActionType = "EXEC"
	ActionTypeQbittorrent ActionType = "QBITTORRENT"
	ActionTypeDelugeV1    ActionType = "DELUGE_V1"
	ActionTypeDelugeV2    ActionType = "DELUGE_V2"
	ActionTypeWatchFolder ActionType = "WATCH_FOLDER"
	ActionTypeWebhook     ActionType = "WEBHOOK"
	ActionTypeRadarr      ActionType = "RADARR"
	ActionTypeSonarr      ActionType = "SONARR"
	ActionTypeLidarr      ActionType = "LIDARR"
)
