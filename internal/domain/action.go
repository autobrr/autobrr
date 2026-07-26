// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"strings"

	"github.com/autobrr/autobrr/pkg/errors"
)

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
	DownloadPath             string              `json:"download_path,omitempty"`
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

// macroFields returns the action fields that get macro expanded and could
// reference the torrent file. Keep in sync with ParseMacros.
func (a *Action) macroFields() []string {
	return []string{a.ExecArgs, a.WatchFolder, a.SavePath, a.DownloadPath, a.WebhookData}
}

// CheckMacrosNeedTorrentTmpFile check if macros need the torrent written to disk.
// Only the path macros need a real file, everything else works off the raw bytes.
func (a *Action) CheckMacrosNeedTorrentTmpFile(release *Release) bool {
	if release.TorrentTmpFile != "" {
		return false
	}

	return containsAnyMacro(a.macroFields(), "TorrentPathName", "TorrentTmpFile")
}

// CheckMacrosNeedRawDataBytes check if macros need the torrent downloaded into memory.
// This is a superset of CheckMacrosNeedTorrentTmpFile since a tmp file can only be
// written once we hold the contents.
func (a *Action) CheckMacrosNeedRawDataBytes(release *Release) bool {
	if len(release.TorrentDataRawBytes) != 0 {
		return false
	}

	if a.Type == ActionTypeWatchFolder {
		return true
	}

	return containsAnyMacro(a.macroFields(), "TorrentPathName", "TorrentTmpFile", "TorrentDataRawBytes", "TorrentHash")
}

func containsAnyMacro(fields []string, macros ...string) bool {
	for _, field := range fields {
		if field == "" {
			continue
		}

		for _, macro := range macros {
			if strings.Contains(field, macro) {
				return true
			}
		}
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
	a.DownloadPath, err = m.Parse(a.DownloadPath)
	if err != nil {
		return errors.Wrap(err, "could not parse download_path")
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
	ActionTypeNzbget       ActionType = "NZBGET"
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
