// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package arr

type Tag struct {
	Label string
	ID    int
}

type Link struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

type Image struct {
	CoverType string `json:"coverType"`
	URL       string `json:"url"`
	RemoteURL string `json:"remoteUrl,omitempty"`
	Extension string `json:"extension,omitempty"`
}

type Ratings struct {
	Votes      int64   `json:"votes"`
	Value      float64 `json:"value"`
	Popularity float64 `json:"popularity,omitempty"`
}

type Value struct {
	Name string `json:"name"`
	ID   int64  `json:"id"`
}

// BaseQuality is a base quality profile.
type BaseQuality struct {
	Name       string `json:"name"`
	Source     string `json:"source,omitempty"`
	Modifier   string `json:"modifier,omitempty"`
	ID         int64  `json:"id"`
	Resolution int    `json:"resolution,omitempty"`
}

// Quality is a download quality profile attached to a movie, book, track or series.
// It may contain 1 or more profiles.
// Sonarr nor Readarr use Name or ID in this struct.
type Quality struct {
	Quality  *BaseQuality     `json:"quality,omitempty"`
	Revision *QualityRevision `json:"revision,omitempty"` // Not sure which app had this....
	Name     string           `json:"name,omitempty"`
	Items    []*Quality       `json:"items,omitempty"`
	ID       int              `json:"id,omitempty"`
	Allowed  bool             `json:"allowed"`
}

// QualityRevision is probably used in Sonarr.
type QualityRevision struct {
	Version  int64 `json:"version"`
	Real     int64 `json:"real"`
	IsRepack bool  `json:"isRepack,omitempty"`
}
