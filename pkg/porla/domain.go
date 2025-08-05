// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package porla

type SysVersions struct {
	Porla SysVersionsPorla `json:"porla"`
}

type SysVersionsPorla struct {
	Commitish string `json:"commitish"`
	Version   string `json:"version"`
}

type TorrentsAddReq struct {
	DownloadLimit *int64  `json:"download_limit,omitempty"`
	SavePath      string  `json:"save_path,omitempty"`
	Ti            string  `json:"ti,omitempty"`
	MagnetUri     string  `json:"magnet_uri,omitempty"`
	UploadLimit   *int64  `json:"upload_limit,omitempty"`
	Preset        *string `json:"preset,omitempty"`
}

type TorrentsAddRes struct {
}

type TorrentsListReq struct {
	Filters *TorrentsListFilters `json:"filters"`
}

type TorrentsListFilters struct {
	Query string `json:"query"`
}

type TorrentsListRes struct {
	Page          int       `json:"page"`
	PageSize      int       `json:"page_size"`
	TorrentsTotal int       `json:"torrents_total"`
	Torrents      []Torrent `json:"torrents"`
}

type Torrent struct {
	DownloadRate  int      `json:"download_rate"`
	UploadRate    int      `json:"upload_rate"`
	InfoHash      []string `json:"info_hash"`
	ListPeers     int      `json:"list_peers"`
	ListSeeds     int      `json:"list_seeds"`
	Name          string   `json:"name"`
	NumPeers      int      `json:"num_peers"`
	NumSeeds      int      `json:"num_seeds"`
	Progress      float64  `json:"progress"`
	QueuePosition int      `json:"queue_position"`
	SavePath      string   `json:"save_path"`
	Size          int      `json:"size"`
	Total         int      `json:"total"`
	TotalDone     int      `json:"total_done"`
}
