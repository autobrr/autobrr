// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
	"strings"

	"github.com/autobrr/autobrr/pkg/errors"
)

type ActionRepo interface {
	Store(ctx context.Context, action Action) (*Action, error)
	StoreFilterActions(ctx context.Context, filterID int64, actions []*Action) ([]*Action, error)
	FindByFilterID(ctx context.Context, filterID int, active *bool, withClient bool) ([]*Action, error)
	List(ctx context.Context) ([]Action, error)
	Get(ctx context.Context, req *GetActionRequest) (*Action, error)
	Delete(ctx context.Context, req *DeleteActionRequest) error
	DeleteByFilterID(ctx context.Context, filterID int) error
	ToggleEnabled(actionID int) error
}

type Action struct {
	ID                       int                 `json:"id"`
	Name                     string              `json:"name"`
	Type                     ActionType          `json:"type"`
	Enabled                  bool                `json:"enabled"`
	ExecCmd                  string              `json:"exec_cmd,omitempty"`
	ExecArgs                 string              `json:"exec_args,omitempty"`
	WatchFolder              string              `json:"watch_folder,omitempty"`
	Category                 string              `json:"category,omitempty"`
	Tags                     string              `json:"tags,omitempty"`
	Label                    string              `json:"label,omitempty"`
	SavePath                 string              `json:"save_path,omitempty"`
	Paused                   bool                `json:"paused,omitempty"`
	IgnoreRules              bool                `json:"ignore_rules,omitempty"`
	FirstLastPiecePrio       bool                `json:"first_last_piece_prio,omitempty"`
	SkipHashCheck            bool                `json:"skip_hash_check,omitempty"`
	ContentLayout            ActionContentLayout `json:"content_layout,omitempty"`
	LimitUploadSpeed         int64               `json:"limit_upload_speed,omitempty"`
	LimitDownloadSpeed       int64               `json:"limit_download_speed,omitempty"`
	LimitRatio               float64             `json:"limit_ratio,omitempty"`
	LimitSeedTime            int64               `json:"limit_seed_time,omitempty"`
	PriorityLayout           PriorityLayout      `json:"priority,omitempty"`
	ReAnnounceSkip           bool                `json:"reannounce_skip,omitempty"`
	ReAnnounceDelete         bool                `json:"reannounce_delete,omitempty"`
	ReAnnounceInterval       int64               `json:"reannounce_interval,omitempty"`
	ReAnnounceMaxAttempts    int64               `json:"reannounce_max_attempts,omitempty"`
	WebhookHost              string              `json:"webhook_host,omitempty"`
	WebhookType              string              `json:"webhook_type,omitempty"`
	WebhookMethod            string              `json:"webhook_method,omitempty"`
	WebhookData              string              `json:"webhook_data,omitempty"`
	WebhookHeaders           []string            `json:"webhook_headers,omitempty"`
	ExternalDownloadClientID int32               `json:"external_download_client_id,omitempty"`
	ExternalDownloadClient   string              `json:"external_download_client,omitempty"`
	FilterID                 int                 `json:"filter_id,omitempty"`
	ClientID                 int32               `json:"client_id,omitempty"`
	Client                   *DownloadClient     `json:"client,omitempty"`
}

// CheckMacrosNeedTorrentTmpFile check if macros needs torrent downloaded
func (a *Action) CheckMacrosNeedTorrentTmpFile(release *Release) bool {
	if release.TorrentTmpFile == "" &&
		(strings.Contains(a.ExecArgs, "TorrentPathName") ||
			strings.Contains(a.ExecArgs, "TorrentDataRawBytes") ||
			strings.Contains(a.ExecArgs, "TorrentHash") ||
			strings.Contains(a.WebhookData, "TorrentPathName") ||
			strings.Contains(a.WebhookData, "TorrentDataRawBytes") ||
			strings.Contains(a.WebhookData, "TorrentHash") ||
			strings.Contains(a.SavePath, "TorrentPathName") ||
			strings.Contains(a.SavePath, "TorrentHash") ||
			a.Type == ActionTypeWatchFolder) {
		return true
	}

	return false
}

func (a *Action) CheckMacrosNeedRawDataBytes(release *Release) bool {
	// if webhook data contains TorrentDataRawBytes, lets read the file into bytes we can then use in the macro
	if len(release.TorrentDataRawBytes) == 0 &&
		(strings.Contains(a.ExecArgs, "TorrentDataRawBytes") || strings.Contains(a.WebhookData, "TorrentDataRawBytes") ||
			a.Type == ActionTypeWatchFolder) {
		return true
	}

	return false
}

// ParseMacros parse all macros on action
func (a *Action) ParseMacros(release *Release) error {
	var err error

	m := NewMacro(*release)

	a.ExecArgs, err = m.Parse(a.ExecArgs)
	if err != nil {
		return errors.Wrap(err, "could not parse exec args")
	}

	a.WatchFolder, err = m.Parse(a.WatchFolder)
	if err != nil {
		return errors.Wrap(err, "could not parse watch folder")
	}

	a.Category, err = m.Parse(a.Category)
	if err != nil {
		return errors.Wrap(err, "could not parse category")
	}

	a.Tags, err = m.Parse(a.Tags)
	if err != nil {
		return errors.Wrap(err, "could not parse tags")
	}

	a.Label, err = m.Parse(a.Label)
	if err != nil {
		return errors.Wrap(err, "could not parse label")
	}
	a.SavePath, err = m.Parse(a.SavePath)
	if err != nil {
		return errors.Wrap(err, "could not parse save_path")
	}
	a.WebhookData, err = m.Parse(a.WebhookData)
	if err != nil {
		return errors.Wrap(err, "could not parse webhook_data")
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
	ActionTypePorla        ActionType = "PORLA"
	ActionTypeWatchFolder  ActionType = "WATCH_FOLDER"
	ActionTypeWebhook      ActionType = "WEBHOOK"
	ActionTypeRadarr       ActionType = "RADARR"
	ActionTypeSonarr       ActionType = "SONARR"
	ActionTypeLidarr       ActionType = "LIDARR"
	ActionTypeWhisparr     ActionType = "WHISPARR"
	ActionTypeReadarr      ActionType = "READARR"
	ActionTypeSabnzbd      ActionType = "SABNZBD"
)

type ActionContentLayout string

const (
	ActionContentLayoutOriginal        ActionContentLayout = "ORIGINAL"
	ActionContentLayoutSubfolderNone   ActionContentLayout = "SUBFOLDER_NONE"
	ActionContentLayoutSubfolderCreate ActionContentLayout = "SUBFOLDER_CREATE"
)

type PriorityLayout string

const (
	PriorityLayoutMax     PriorityLayout = "MAX"
	PriorityLayoutMin     PriorityLayout = "MIN"
	PriorityLayoutDefault PriorityLayout = ""
)

type GetActionRequest struct {
	Id int
}

type DeleteActionRequest struct {
	ActionId int
}
