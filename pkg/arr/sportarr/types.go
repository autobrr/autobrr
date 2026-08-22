// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package sportarr

import (
	"fmt"
	"strings"
)

type ReleasePushRequest struct {
	Title            string `json:"title"`
	InfoUrl          string `json:"infoUrl,omitempty"`
	DownloadUrl      string `json:"downloadUrl,omitempty"`
	MagnetUrl        string `json:"magnetUrl,omitempty"`
	Size             uint64 `json:"size"`
	Indexer          string `json:"indexer"`
	DownloadProtocol string `json:"downloadProtocol"`
	Protocol         string `json:"protocol"`
	PublishDate      string `json:"publishDate"`
	IndexerFlags     int    `json:"indexerFlags,omitempty"`
}

type ReleasePushResponse struct {
	Approved     bool     `json:"approved"`
	Rejected     bool     `json:"rejected"`
	TempRejected bool     `json:"temporarilyRejected"`
	Rejections   []string `json:"rejections"`
}

type BadRequestResponse struct {
	PropertyName   string `json:"propertyName"`
	ErrorMessage   string `json:"errorMessage"`
	ErrorCode      string `json:"errorCode"`
	AttemptedValue string `json:"attemptedValue"`
	Severity       string `json:"severity"`
}

func (r *BadRequestResponse) String() string {
	return fmt.Sprintf("[%s: %s] %s: %s - got value: %s", r.Severity, r.ErrorCode, r.PropertyName, r.ErrorMessage, r.AttemptedValue)
}

type SystemStatusResponse struct {
	Version string `json:"version"`
}

// League is a Sportarr library entry, the equivalent of a Sonarr series:
// events (games, races, cards) belong to a league the way episodes belong
// to a series.
type League struct {
	ID        int64  `json:"id"`
	Name      string `json:"name,omitempty"`
	Sport     string `json:"sport,omitempty"`
	Monitored bool   `json:"monitored"`
	Tags      []int  `json:"tags,omitempty"`
}

type IndexerFlags int

const (
	IndexerFlagFreeleech    IndexerFlags = 1
	IndexerFlagHalfleech    IndexerFlags = 2
	IndexerFlagDoubleUpload IndexerFlags = 4
	IndexerFlagInternal     IndexerFlags = 8
	IndexerFlagScene        IndexerFlags = 16
	IndexerFlagFreeleech75  IndexerFlags = 32
	IndexerFlagFreeleech25  IndexerFlags = 64
	IndexerFlagNuked        IndexerFlags = 128
	IndexerFlagSubtitles    IndexerFlags = 256
)

type ReleaseMeta struct {
	FreeleechPercent int
	Origin           string // "scene" or "internal"
	Nuked            bool
	HasSubtitles     bool
	DoubleUpload     bool
}

// BuildIndexerFlags maps release metadata onto the flag bitmask Sportarr's
// release/push endpoint understands (the same mask Sonarr uses).
func BuildIndexerFlags(m ReleaseMeta) IndexerFlags {
	var flags IndexerFlags
	switch m.FreeleechPercent {
	case 100:
		flags |= IndexerFlagFreeleech
	case 75:
		flags |= IndexerFlagFreeleech75
	case 50:
		flags |= IndexerFlagHalfleech
	case 25:
		flags |= IndexerFlagFreeleech25
	}
	switch strings.ToLower(strings.TrimSpace(m.Origin)) {
	case "internal":
		flags |= IndexerFlagInternal
	case "scene":
		flags |= IndexerFlagScene
	}
	if m.Nuked {
		flags |= IndexerFlagNuked
	}
	if m.HasSubtitles {
		flags |= IndexerFlagSubtitles
	}
	if m.DoubleUpload {
		flags |= IndexerFlagDoubleUpload
	}
	return flags
}
