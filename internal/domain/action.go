package domain

type ActionRepo interface {
	Store(action Action) (*Action, error)
	FindByFilterID(filterID int) ([]Action, error)
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
	FilterID           int        `json:"filter_id,omitempty"`
	ClientID           int32      `json:"client_id,omitempty"`
}

type ActionType string

const (
	ActionTypeTest        ActionType = "TEST"
	ActionTypeExec        ActionType = "EXEC"
	ActionTypeQbittorrent ActionType = "QBITTORRENT"
	ActionTypeDeluge      ActionType = "DELUGE"
	ActionTypeWatchFolder ActionType = "WATCH_FOLDER"
)
