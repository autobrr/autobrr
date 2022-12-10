package domain

import (
	"context"
	"github.com/autobrr/autobrr/pkg/errors"
)

type ActionRepo interface {
	Store(ctx context.Context, action Action) (*Action, error)
	StoreFilterActions(ctx context.Context, actions []*Action, filterID int64) ([]*Action, error)
	DeleteByFilterID(ctx context.Context, filterID int) error
	FindByFilterID(ctx context.Context, filterID int) ([]*Action, error)
	List(ctx context.Context) ([]Action, error)
	Delete(actionID int) error
	ToggleEnabled(actionID int) error
}

type Action struct {
	ID                    int                 `json:"id"`
	Name                  string              `json:"name"`
	Type                  ActionType          `json:"type"`
	Enabled               bool                `json:"enabled"`
	ExecCmd               string              `json:"exec_cmd,omitempty"`
	ExecArgs              string              `json:"exec_args,omitempty"`
	WatchFolder           string              `json:"watch_folder,omitempty"`
	Category              string              `json:"category,omitempty"`
	Tags                  string              `json:"tags,omitempty"`
	Label                 string              `json:"label,omitempty"`
	SavePath              string              `json:"save_path,omitempty"`
	Paused                bool                `json:"paused,omitempty"`
	IgnoreRules           bool                `json:"ignore_rules,omitempty"`
	SkipHashCheck         bool                `json:"skip_hash_check,omitempty"`
	ContentLayout         ActionContentLayout `json:"content_layout,omitempty"`
	LimitUploadSpeed      int64               `json:"limit_upload_speed,omitempty"`
	LimitDownloadSpeed    int64               `json:"limit_download_speed,omitempty"`
	LimitRatio            float64             `json:"limit_ratio,omitempty"`
	LimitSeedTime         int64               `json:"limit_seed_time,omitempty"`
	ReAnnounceSkip        bool                `json:"reannounce_skip,omitempty"`
	ReAnnounceDelete      bool                `json:"reannounce_delete,omitempty"`
	ReAnnounceInterval    int64               `json:"reannounce_interval,omitempty"`
	ReAnnounceMaxAttempts int64               `json:"reannounce_max_attempts,omitempty"`
	WebhookHost           string              `json:"webhook_host,omitempty"`
	WebhookType           string              `json:"webhook_type,omitempty"`
	WebhookMethod         string              `json:"webhook_method,omitempty"`
	WebhookData           string              `json:"webhook_data,omitempty"`
	WebhookHeaders        []string            `json:"webhook_headers,omitempty"`
	FilterID              int                 `json:"filter_id,omitempty"`
	ClientID              int32               `json:"client_id,omitempty"`
	Client                DownloadClient      `json:"client,omitempty"`
}

// ParseMacros parse all macros on action
func (a *Action) ParseMacros(release Release) error {
	var err error

	m := NewMacro(release)

	a.ExecArgs, err = m.Parse(a.ExecArgs)
	a.WatchFolder, err = m.Parse(a.WatchFolder)
	a.Category, err = m.Parse(a.Category)
	a.Tags, err = m.Parse(a.Tags)
	a.Label, err = m.Parse(a.Label)
	a.SavePath, err = m.Parse(a.SavePath)
	a.WebhookData, err = m.Parse(a.WebhookData)

	if err != nil {
		return errors.Wrap(err, "could not parse macros for action: %v", a.Name)
	}

	return nil
}

type ActionType string

const (
	ActionTypeTest         ActionType = "TEST"
	ActionTypeExec         ActionType = "EXEC"
	ActionTypeQbittorrent  ActionType = "QBITTORRENT"
	ActionTypeDelugeV1     ActionType = "DELUGE_V1"
	ActionTypeDelugeV2     ActionType = "DELUGE_V2"
	ActionTypeRTorrent     ActionType = "RTORRENT"
	ActionTypeTransmission ActionType = "TRANSMISSION"
	ActionTypeWatchFolder  ActionType = "WATCH_FOLDER"
	ActionTypeWebhook      ActionType = "WEBHOOK"
	ActionTypeRadarr       ActionType = "RADARR"
	ActionTypeSonarr       ActionType = "SONARR"
	ActionTypeLidarr       ActionType = "LIDARR"
	ActionTypeWhisparr     ActionType = "WHISPARR"
	ActionTypeReadarr      ActionType = "READARR"
)

type ActionContentLayout string

const (
	ActionContentLayoutOriginal        ActionContentLayout = "ORIGINAL"
	ActionContentLayoutSubfolderNone   ActionContentLayout = "SUBFOLDER_NONE"
	ActionContentLayoutSubfolderCreate ActionContentLayout = "SUBFOLDER_CREATE"
)
