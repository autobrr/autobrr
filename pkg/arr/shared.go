// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package arr

import (
	"encoding/json"
)

type Tag struct {
	ID    int
	Label string
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
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// BaseQuality is a base quality profile.
type BaseQuality struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Source     string `json:"source,omitempty"`
	Resolution int    `json:"resolution,omitempty"`
	Modifier   string `json:"modifier,omitempty"`
}

// Quality is a download quality profile attached to a movie, book, track or series.
// It may contain 1 or more profiles.
// Sonarr nor Readarr use Name or ID in this struct.
type Quality struct {
	Name     string           `json:"name,omitempty"`
	ID       int              `json:"id,omitempty"`
	Quality  *BaseQuality     `json:"quality,omitempty"`
	Items    []*Quality       `json:"items,omitempty"`
	Allowed  bool             `json:"allowed"`
	Revision *QualityRevision `json:"revision,omitempty"` // Not sure which app had this....
}

// QualityRevision is probably used in Sonarr.
type QualityRevision struct {
	Version  int64 `json:"version"`
	Real     int64 `json:"real"`
	IsRepack bool  `json:"isRepack,omitempty"`
}

// ErrorResponse is the single error object Servarr apps return for API exceptions, as opposed to the array of validation failures.
type ErrorResponse struct {
	Message     string `json:"message"`
	Description string `json:"description,omitempty"`
	Title       string `json:"title,omitempty"`
	Detail      string `json:"detail,omitempty"`
}

func (e *ErrorResponse) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Detail != "" {
		return e.Detail
	}
	if e.Title != "" {
		return e.Title
	}
	return e.Description
}

// ParseErrorResponse decodes body as a single error object, ok is false when it is not one.
func ParseErrorResponse(body []byte) (*ErrorResponse, bool) {
	var resp ErrorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, false
	}

	if resp.Error() == "" {
		return nil, false
	}

	return &resp, true
}
