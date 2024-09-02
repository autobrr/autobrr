// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"bytes"
	"strings"
	"text/template"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/Masterminds/sprig/v3"
	"github.com/dustin/go-humanize"
)

type Macro struct {
	Artists                   string
	Audio                     []string
	AudioChannels             string
	AudioFormat               string
	Bitrate                   string
	Bonus                     []string
	Categories                []string
	Category                  string
	Codec                     []string
	Container                 string
	CurrentDay                int
	CurrentHour               int
	CurrentMinute             int
	CurrentMonth              int
	CurrentSecond             int
	CurrentYear               int
	DownloadUrl               string
	Episode                   int
	FilterID                  int
	FilterName                string
	Freeleech                 bool
	FreeleechPercent          int
	Group                     string
	GroupID                   string
	HDR                       string
	HasCue                    bool
	HasLog                    bool
	Implementation            string
	Indexer                   string
	IndexerIdentifier         string
	IndexerIdentifierExternal string
	IndexerName               string
	InfoUrl                   string
	Language                  []string
	Leechers                  int
	LogScore                  int
	MagnetURI                 string
	Origin                    string
	Other                     []string
	PreTime                   string
	Protocol                  string
	Proper                    bool
	Region                    string
	Repack                    bool
	Resolution                string
	Season                    int
	Seeders                   int
	Size                      uint64
	SizeString                string
	Source                    string
	Tags                      string
	Title                     string
	TorrentDataRawBytes       []byte
	TorrentHash               string
	TorrentID                 string
	TorrentName               string
	TorrentPathName           string
	TorrentUrl                string
	TorrentTmpFile            string
	Type                      string
	Uploader                  string
	Website                   string
	Year                      int
	Month                     int
	Day                       int
}

func NewMacro(release Release) Macro {
	currentTime := time.Now()

	ma := Macro{
		Artists:                   release.Artists,
		Audio:                     release.Audio,
		AudioChannels:             release.AudioChannels,
		AudioFormat:               release.AudioFormat,
		Bitrate:                   release.Bitrate,
		Bonus:                     release.Bonus,
		Categories:                release.Categories,
		Category:                  release.Category,
		Codec:                     release.Codec,
		Container:                 release.Container,
		CurrentDay:                currentTime.Day(),
		CurrentHour:               currentTime.Hour(),
		CurrentMinute:             currentTime.Minute(),
		CurrentMonth:              int(currentTime.Month()),
		CurrentSecond:             currentTime.Second(),
		CurrentYear:               currentTime.Year(),
		DownloadUrl:               release.DownloadURL,
		Episode:                   release.Episode,
		FilterID:                  release.FilterID,
		FilterName:                release.FilterName,
		Freeleech:                 release.Freeleech,
		FreeleechPercent:          release.FreeleechPercent,
		Group:                     release.Group,
		GroupID:                   release.GroupID,
		HDR:                       strings.Join(release.HDR, ", "),
		HasCue:                    release.HasCue,
		HasLog:                    release.HasLog,
		Implementation:            release.Implementation.String(),
		Indexer:                   release.Indexer.Identifier,
		IndexerIdentifier:         release.Indexer.Identifier,
		IndexerIdentifierExternal: release.Indexer.IdentifierExternal,
		IndexerName:               release.Indexer.Name,
		InfoUrl:                   release.InfoURL,
		Language:                  release.Language,
		Leechers:                  release.Leechers,
		LogScore:                  release.LogScore,
		MagnetURI:                 release.MagnetURI,
		Origin:                    release.Origin,
		Other:                     release.Other,
		PreTime:                   release.PreTime,
		Protocol:                  release.Protocol.String(),
		Proper:                    release.Proper,
		Region:                    release.Region,
		Repack:                    release.Repack,
		Resolution:                release.Resolution,
		Season:                    release.Season,
		Seeders:                   release.Seeders,
		Size:                      release.Size,
		SizeString:                humanize.Bytes(release.Size),
		Source:                    release.Source,
		Tags:                      strings.Join(release.Tags, ", "),
		Title:                     release.Title,
		TorrentDataRawBytes:       release.TorrentDataRawBytes,
		TorrentHash:               release.TorrentHash,
		TorrentID:                 release.TorrentID,
		TorrentName:               release.TorrentName,
		TorrentPathName:           release.TorrentTmpFile,
		TorrentUrl:                release.DownloadURL,
		TorrentTmpFile:            release.TorrentTmpFile,
		Type:                      release.Type,
		Uploader:                  release.Uploader,
		Website:                   release.Website,
		Year:                      release.Year,
		Month:                     release.Month,
		Day:                       release.Day,
	}

	return ma
}

// Parse takes a string and replaces valid vars
func (m Macro) Parse(text string) (string, error) {
	if text == "" {
		return "", nil
	}

	// TODO implement template cache

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
