// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"bytes"
	"strings"
	"text/template"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/Masterminds/sprig/v3"
)

type Macro struct {
	TorrentName         string
	TorrentPathName     string
	TorrentHash         string
	TorrentID           string
	TorrentUrl          string
	TorrentDataRawBytes []byte
	MagnetURI           string
	GroupID             string
	Indexer             string
	Title               string
	Category            string
	Categories          []string
	Resolution          string
	Source              string
	HDR                 string
	FilterName          string
	Size                uint64
	Season              int
	Episode             int
	Year                int
	CurrentYear         int
	CurrentMonth        int
	CurrentDay          int
	CurrentHour         int
	CurrentMinute       int
	CurrentSecond       int
}

func NewMacro(release Release) Macro {
	currentTime := time.Now()

	ma := Macro{
		TorrentName:         release.TorrentName,
		TorrentUrl:          release.TorrentURL,
		TorrentPathName:     release.TorrentTmpFile,
		TorrentDataRawBytes: release.TorrentDataRawBytes,
		TorrentHash:         release.TorrentHash,
		TorrentID:           release.TorrentID,
		MagnetURI:           release.MagnetURI,
		GroupID:             release.GroupID,
		Indexer:             release.Indexer,
		Title:               release.Title,
		Category:            release.Category,
		Categories:          release.Categories,
		Resolution:          release.Resolution,
		Source:              release.Source,
		HDR:                 strings.Join(release.HDR, ", "),
		FilterName:          release.FilterName,
		Size:                release.Size,
		Season:              release.Season,
		Episode:             release.Episode,
		Year:                release.Year,
		CurrentYear:         currentTime.Year(),
		CurrentMonth:        int(currentTime.Month()),
		CurrentDay:          currentTime.Day(),
		CurrentHour:         currentTime.Hour(),
		CurrentMinute:       currentTime.Minute(),
		CurrentSecond:       currentTime.Second(),
	}

	return ma
}

// Parse takes a string and replaces valid vars
func (m Macro) Parse(text string) (string, error) {
	if text == "" {
		return "", nil
	}

	// setup template
	tmpl, err := template.New("macro").Funcs(sprig.TxtFuncMap()).Parse(text)
	if err != nil {
		return "", errors.Wrap(err, "could parse macro template")
	}

	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, m)
	if err != nil {
		return "", errors.Wrap(err, "could not parse macro")
	}

	return tpl.String(), nil
}

// MustParse takes a string and replaces valid vars
func (m Macro) MustParse(text string) string {
	if text == "" {
		return ""
	}

	// setup template
	tmpl, err := template.New("macro").Funcs(sprig.TxtFuncMap()).Parse(text)
	if err != nil {
		return ""
	}

	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, m)
	if err != nil {
		return ""
	}

	return tpl.String()
}
