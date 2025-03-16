// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
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
	AudioChannels             string
	AudioFormat               string
	Bitrate                   string
	Category                  string
	Container                 string
	Description               string
	DownloadUrl               string
	FilterName                string
	Group                     string
	GroupID                   string
	HDR                       string
	Implementation            string
	Indexer                   string
	IndexerIdentifier         string
	IndexerIdentifierExternal string
	IndexerName               string
	InfoUrl                   string
	MagnetURI                 string
	MetaIMDB                  string
	Origin                    string
	PreTime                   string
	Protocol                  string
	Region                    string
	Resolution                string
	SizeString                string
	SkipDuplicateProfileName  string
	Source                    string
	Tags                      string
	Title                     string
	TorrentHash               string
	TorrentID                 string
	TorrentName               string
	TorrentPathName           string
	TorrentUrl                string
	TorrentTmpFile            string
	Type                      string
	Uploader                  string
	RecordLabel               string
	Website                   string
	Audio                     []string
	Bonus                     []string
	Categories                []string
	Codec                     []string
	Language                  []string
	Other                     []string
	TorrentDataRawBytes       []byte
	CurrentDay                int
	CurrentHour               int
	CurrentMinute             int
	CurrentMonth              int
	CurrentSecond             int
	CurrentYear               int
	Episode                   int
	FilterID                  int
	FreeleechPercent          int
	Leechers                  int
	LogScore                  int
	Season                    int
	Seeders                   int
	Size                      uint64
	SkipDuplicateProfileID    int64
	Year                      int
	Month                     int
	Day                       int
	Freeleech                 bool
	HasCue                    bool
	HasLog                    bool
	IsDuplicate               bool
	Proper                    bool
	Repack                    bool
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
		Description:               release.Description,
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
		IsDuplicate:               release.IsDuplicate,
		Language:                  release.Language,
		Leechers:                  release.Leechers,
		LogScore:                  release.LogScore,
		MagnetURI:                 release.MagnetURI,
		MetaIMDB:                  release.MetaIMDB,
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
		SkipDuplicateProfileID:    release.SkipDuplicateProfileID,
		SkipDuplicateProfileName:  release.SkipDuplicateProfileName,
		Tags:                      strings.Join(release.Tags, ", "),
		Title:                     release.Title,
		TorrentDataRawBytes:       release.TorrentDataRawBytes,
		TorrentHash:               release.TorrentHash,
		TorrentID:                 release.TorrentID,
		TorrentName:               release.TorrentName,
		TorrentPathName:           release.TorrentTmpFile,
		TorrentUrl:                release.DownloadURL,
		TorrentTmpFile:            release.TorrentTmpFile,
		Type:                      release.Type.String(),
		Uploader:                  release.Uploader,
		RecordLabel:               release.RecordLabel,
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
