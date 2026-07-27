// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package aria2

import "strconv"

type Version struct {
	Version         string   `json:"version"`
	EnabledFeatures []string `json:"enabledFeatures"`
}

// Options are the per download options passed to addUri and addTorrent. aria2
// takes every option value as a string, including numbers and booleans.
type Options map[string]string

type Status struct {
	Gid             string `json:"gid"`
	Status          string `json:"status"`
	TotalLength     string `json:"totalLength"`
	CompletedLength string `json:"completedLength"`
}

// Downloading reports whether the download still has data left to fetch. A
// finished torrent stays active while it seeds, so active alone says nothing
// about how busy the daemon is.
func (s Status) Downloading() bool {
	total := parseInt(s.TotalLength)

	// a magnet has no size until its metadata has been fetched
	if total == 0 {
		return true
	}

	return parseInt(s.CompletedLength) < total
}

func parseInt(s string) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}

	return v
}
