// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package domain

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/moistari/rls"
	"github.com/rs/zerolog"
)

var trackerLessTestTorrent = `d7:comment19:This is just a test10:created by12:Johnny Bravo13:creation datei1430648794e8:encoding5:UTF-84:infod6:lengthi1128e4:name12:testfile.bin12:piece lengthi32768e6:pieces20:Õˆë	=‘UŒäiÎ^æ °Eâ?ÇÒe5:nodesl35:udp://tracker.openbittorrent.com:8035:udp://tracker.openbittorrent.com:80ee`

func TestRelease_DownloadTorrentFile(t *testing.T) {
	// disable logger
	zerolog.SetGlobalLevel(zerolog.Disabled)

	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()

	mux.HandleFunc("/files/valid_torrent_as_html", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html")
		payload, _ := os.ReadFile("testdata/archlinux-2011.08.19-netinstall-i686.iso.torrent")
		w.Write(payload)
	})

	mux.HandleFunc("/files/invalid_torrent_as_html", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html")
		payload := []byte("<html><head></head><body>This is not the torrent you are looking for</body></html>")
		w.Write(payload)
	})

	mux.HandleFunc("/index.html", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html")
		payload := []byte("<html><head></head><body>This is not the torrent you are looking for</body></html>")
		w.Write(payload)
	})

	mux.HandleFunc("/plaintext", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain")
		payload := []byte("This is not a valid torrent file.")
		w.Write(payload)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "401") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
			return
		}
		if strings.Contains(r.RequestURI, "403") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("forbidden"))
			return
		}
		if strings.Contains(r.RequestURI, "404") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("not found"))
			return
		}
		if strings.Contains(r.RequestURI, "405") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("method not allowed"))
			return
		}

		if strings.Contains(r.RequestURI, "file.torrent") {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/x-bittorrent")
			payload, _ := os.ReadFile("testdata/archlinux-2011.08.19-netinstall-i686.iso.torrent")
			w.Write(payload)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	})

	type fields struct {
		ID                          int64
		FilterStatus                ReleaseFilterStatus
		Rejections                  []string
		Indexer                     IndexerMinimal
		FilterName                  string
		Protocol                    ReleaseProtocol
		Implementation              ReleaseImplementation
		Timestamp                   time.Time
		GroupID                     string
		TorrentID                   string
		DownloadURL                 string
		TorrentTmpFile              string
		TorrentDataRawBytes         []byte
		TorrentHash                 string
		TorrentName                 string
		Size                        uint64
		Title                       string
		Category                    string
		Categories                  []string
		Season                      int
		Episode                     int
		Year                        int
		Resolution                  string
		Source                      string
		Codec                       []string
		Container                   string
		HDR                         []string
		Audio                       []string
		AudioChannels               string
		Group                       string
		Region                      string
		Language                    []string
		Proper                      bool
		Repack                      bool
		Website                     string
		Artists                     string
		Type                        rls.Type
		LogScore                    int
		Origin                      string
		Tags                        []string
		ReleaseTags                 string
		Freeleech                   bool
		FreeleechPercent            int
		Bonus                       []string
		Uploader                    string
		PreTime                     string
		Other                       []string
		RawCookie                   string
		AdditionalSizeCheckRequired bool
		FilterID                    int
		Filter                      *Filter
		ActionStatus                []ReleaseActionStatus
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "401",
			fields: fields{
				Indexer: IndexerMinimal{
					ID:         0,
					Name:       "Mock Indexer",
					Identifier: "mock-indexer",
				},
				TorrentName: "Test.Release-GROUP",
				DownloadURL: fmt.Sprintf("%s/%d", ts.URL, 401),
				Protocol:    ReleaseProtocolTorrent,
			},
			wantErr: true,
		},
		{
			name: "403",
			fields: fields{
				Indexer: IndexerMinimal{
					ID:         0,
					Name:       "Mock Indexer",
					Identifier: "mock-indexer",
				},
				TorrentName: "Test.Release-GROUP",
				DownloadURL: fmt.Sprintf("%s/%d", ts.URL, 403),
				Protocol:    ReleaseProtocolTorrent,
			},
			wantErr: true,
		},
		{
			name: "500",
			fields: fields{
				Indexer: IndexerMinimal{
					ID:         0,
					Name:       "Mock Indexer",
					Identifier: "mock-indexer",
				},
				TorrentName: "Test.Release-GROUP",
				DownloadURL: fmt.Sprintf("%s/%d", ts.URL, 500),
				Protocol:    ReleaseProtocolTorrent,
			},
			wantErr: true,
		},
		{
			name: "ok",
			fields: fields{
				Indexer: IndexerMinimal{
					ID:         0,
					Name:       "Mock Indexer",
					Identifier: "mock-indexer",
				},
				TorrentName: "Test.Release-GROUP",
				DownloadURL: fmt.Sprintf("%s/%s", ts.URL, "file.torrent"),
				Protocol:    ReleaseProtocolTorrent,
			},
			wantErr: false,
		},
		{
			name: "valid_torrent_with_text-html_header",
			fields: fields{
				Indexer: IndexerMinimal{
					ID:         0,
					Name:       "Mock Indexer",
					Identifier: "mock-indexer",
				},
				TorrentName: "Test.Release-GROUP",
				DownloadURL: fmt.Sprintf("%s/files/%s", ts.URL, "valid_torrent_as_html"),
				Protocol:    ReleaseProtocolTorrent,
			},
			wantErr: false,
		},
		{
			name: "invalid_torrent_with_text-html_header",
			fields: fields{
				Indexer: IndexerMinimal{
					ID:         0,
					Name:       "Mock Indexer",
					Identifier: "mock-indexer",
				},
				TorrentName: "Test.Release-GROUP",
				DownloadURL: fmt.Sprintf("%s/files/%s", ts.URL, "invalid_torrent_as_html"),
				Protocol:    ReleaseProtocolTorrent,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Release{
				ID:                          tt.fields.ID,
				FilterStatus:                tt.fields.FilterStatus,
				Rejections:                  tt.fields.Rejections,
				Indexer:                     tt.fields.Indexer,
				FilterName:                  tt.fields.FilterName,
				Protocol:                    tt.fields.Protocol,
				Implementation:              tt.fields.Implementation,
				Timestamp:                   tt.fields.Timestamp,
				GroupID:                     tt.fields.GroupID,
				TorrentID:                   tt.fields.TorrentID,
				DownloadURL:                 tt.fields.DownloadURL,
				TorrentTmpFile:              tt.fields.TorrentTmpFile,
				TorrentDataRawBytes:         tt.fields.TorrentDataRawBytes,
				TorrentHash:                 tt.fields.TorrentHash,
				TorrentName:                 tt.fields.TorrentName,
				Size:                        tt.fields.Size,
				Title:                       tt.fields.Title,
				Category:                    tt.fields.Category,
				Categories:                  tt.fields.Categories,
				Season:                      tt.fields.Season,
				Episode:                     tt.fields.Episode,
				Year:                        tt.fields.Year,
				Resolution:                  tt.fields.Resolution,
				Source:                      tt.fields.Source,
				Codec:                       tt.fields.Codec,
				Container:                   tt.fields.Container,
				HDR:                         tt.fields.HDR,
				Audio:                       tt.fields.Audio,
				AudioChannels:               tt.fields.AudioChannels,
				Group:                       tt.fields.Group,
				Region:                      tt.fields.Region,
				Language:                    tt.fields.Language,
				Proper:                      tt.fields.Proper,
				Repack:                      tt.fields.Repack,
				Website:                     tt.fields.Website,
				Artists:                     tt.fields.Artists,
				Type:                        tt.fields.Type,
				LogScore:                    tt.fields.LogScore,
				Origin:                      tt.fields.Origin,
				Tags:                        tt.fields.Tags,
				ReleaseTags:                 tt.fields.ReleaseTags,
				Freeleech:                   tt.fields.Freeleech,
				FreeleechPercent:            tt.fields.FreeleechPercent,
				Bonus:                       tt.fields.Bonus,
				Uploader:                    tt.fields.Uploader,
				PreTime:                     tt.fields.PreTime,
				Other:                       tt.fields.Other,
				RawCookie:                   tt.fields.RawCookie,
				AdditionalSizeCheckRequired: tt.fields.AdditionalSizeCheckRequired,
				FilterID:                    tt.fields.FilterID,
				Filter:                      tt.fields.Filter,
				ActionStatus:                tt.fields.ActionStatus,
			}
			err := r.DownloadTorrentFileCtx(context.Background())
			if err == nil && tt.wantErr {
				fmt.Println("error")
			}

		})
	}
}
