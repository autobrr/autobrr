// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/release"

	"github.com/dustin/go-humanize"
	"github.com/mmcdole/gofeed"
	"github.com/moistari/rls"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestRSSJob_processItem(t *testing.T) {
	t.Parallel()
	now := time.Now()
	nowMinusTime := time.Now().Add(time.Duration(-3000) * time.Second)

	type fields struct {
		Feed       *domain.Feed
		Name       string
		Log        zerolog.Logger
		URL        string
		Repo       domain.FeedCacheRepo
		ReleaseSvc release.Service
		attempts   int
		errors     []error
		JobID      int
	}
	type args struct {
		item *gofeed.Item
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *domain.Release
	}{
		{
			name: "no_baseurl",
			fields: fields{
				Feed: &domain.Feed{
					MaxAge: 3600,
					Indexer: domain.IndexerMinimal{
						ID:                 0,
						Name:               "Mock Feed",
						Identifier:         "mock-feed",
						IdentifierExternal: "Mock Indexer",
					},
				},
				Name:       "test feed",
				Log:        zerolog.Logger{},
				URL:        "https://fake-feed.com/rss",
				Repo:       nil,
				ReleaseSvc: nil,
				attempts:   0,
				errors:     nil,
				JobID:      0,
			},
			args: args{item: &gofeed.Item{
				Title: "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP",
				Description: `Category: Example
 Size: 1.49 GB
 Status: 27 seeders and 1 leechers
 Speed: 772.16 kB/s
 Added: 2022-09-29 16:06:08
`,
				Link: "/details.php?id=00000&hit=1",
				GUID: "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP",
			}},
			want: &domain.Release{
				ID:                              0,
				FilterStatus:                    "PENDING",
				Rejections:                      []string{},
				Indexer:                         domain.IndexerMinimal{0, "Mock Feed", "mock-feed", "Mock Indexer"},
				FilterName:                      "",
				Protocol:                        "torrent",
				Implementation:                  "RSS",
				AnnounceType:                    domain.AnnounceTypeNew,
				Timestamp:                       now,
				GroupID:                         "",
				TorrentID:                       "",
				DownloadURL:                     "https://fake-feed.com/details.php?id=00000&hit=1",
				TorrentTmpFile:                  "",
				TorrentDataRawBytes:             []uint8(nil),
				TorrentHash:                     "",
				TorrentName:                     "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP",
				NormalizedHash:                  "edfbe552ccde335f34b801e15930bc35",
				Size:                            1490000000,
				Title:                           "Some Release Title",
				Description:                     "Category: Example\n Size: 1.49 GB\n Status: 27 seeders and 1 leechers\n Speed: 772.16 kB/s\n Added: 2022-09-29 16:06:08\n",
				Category:                        "",
				Season:                          0,
				Episode:                         0,
				Year:                            2022,
				Month:                           9,
				Day:                             22,
				Resolution:                      "720p",
				Source:                          "WEB",
				Codec:                           []string{"H.264"},
				Container:                       "",
				HDR:                             []string(nil),
				Audio:                           []string(nil),
				AudioChannels:                   "",
				Group:                           "GROUP",
				Region:                          "",
				Language:                        []string{},
				Proper:                          false,
				Repack:                          false,
				Edition:                         []string{},
				Cut:                             []string{},
				Website:                         "",
				Artists:                         "",
				Type:                            rls.Episode,
				LogScore:                        0,
				Origin:                          "",
				Tags:                            []string{},
				ReleaseTags:                     "",
				Freeleech:                       false,
				FreeleechPercent:                0,
				Bonus:                           []string(nil),
				Uploader:                        "",
				PreTime:                         "",
				Other:                           []string{},
				RawCookie:                       "",
				AdditionalSizeCheckRequired:     false,
				AdditionalUploaderCheckRequired: false,
				FilterID:                        0,
				Filter:                          (*domain.Filter)(nil),
				ActionStatus:                    []domain.ReleaseActionStatus(nil),
			},
		},
		{
			name: "with_baseurl",
			fields: fields{
				Feed: &domain.Feed{
					MaxAge: 3600,
					Indexer: domain.IndexerMinimal{
						ID:                 0,
						Name:               "Mock Feed",
						Identifier:         "mock-feed",
						IdentifierExternal: "Mock Indexer",
					},
				},
				Name:       "test feed",
				Log:        zerolog.Logger{},
				URL:        "https://fake-feed.com/rss",
				Repo:       nil,
				ReleaseSvc: nil,
				attempts:   0,
				errors:     nil,
				JobID:      0,
			},
			args: args{item: &gofeed.Item{
				Title: "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP",
				Description: `Category: Example
 Size: 1.49 GB
 Status: 27 seeders and 1 leechers
 Speed: 772.16 kB/s
 Added: 2022-09-29 16:06:08
`,
				Link: "https://fake-feed.com/details.php?id=00000&hit=1",
				GUID: "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP",
			}},
			want: &domain.Release{
				ID:                              0,
				FilterStatus:                    "PENDING",
				Rejections:                      []string{},
				Indexer:                         domain.IndexerMinimal{0, "Mock Feed", "mock-feed", "Mock Indexer"},
				FilterName:                      "",
				Protocol:                        "torrent",
				Implementation:                  "RSS",
				AnnounceType:                    domain.AnnounceTypeNew,
				Timestamp:                       now,
				GroupID:                         "",
				TorrentID:                       "",
				DownloadURL:                     "https://fake-feed.com/details.php?id=00000&hit=1",
				TorrentTmpFile:                  "",
				TorrentDataRawBytes:             []uint8(nil),
				TorrentHash:                     "",
				TorrentName:                     "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP",
				NormalizedHash:                  "edfbe552ccde335f34b801e15930bc35",
				Size:                            1490000000,
				Title:                           "Some Release Title",
				Description:                     "Category: Example\n Size: 1.49 GB\n Status: 27 seeders and 1 leechers\n Speed: 772.16 kB/s\n Added: 2022-09-29 16:06:08\n",
				Category:                        "",
				Season:                          0,
				Episode:                         0,
				Year:                            2022,
				Month:                           9,
				Day:                             22,
				Resolution:                      "720p",
				Source:                          "WEB",
				Codec:                           []string{"H.264"},
				Container:                       "",
				HDR:                             []string(nil),
				Audio:                           []string(nil),
				AudioChannels:                   "",
				Group:                           "GROUP",
				Region:                          "",
				Language:                        []string{},
				Proper:                          false,
				Repack:                          false,
				Edition:                         []string{},
				Cut:                             []string{},
				Website:                         "",
				Artists:                         "",
				Type:                            rls.Episode,
				LogScore:                        0,
				Origin:                          "",
				Tags:                            []string{},
				ReleaseTags:                     "",
				Freeleech:                       false,
				FreeleechPercent:                0,
				Bonus:                           []string(nil),
				Uploader:                        "",
				PreTime:                         "",
				Other:                           []string{},
				RawCookie:                       "",
				AdditionalSizeCheckRequired:     false,
				AdditionalUploaderCheckRequired: false,
				FilterID:                        0,
				Filter:                          (*domain.Filter)(nil),
				ActionStatus:                    []domain.ReleaseActionStatus(nil),
			},
		},
		{
			name: "time_parse",
			fields: fields{
				Feed: &domain.Feed{
					MaxAge: 360,
					Indexer: domain.IndexerMinimal{
						ID:                 0,
						Name:               "Mock Feed",
						Identifier:         "mock-feed",
						IdentifierExternal: "Mock Indexer",
					},
				},
				Name:       "test feed",
				Log:        zerolog.Logger{},
				URL:        "https://fake-feed.com/rss",
				Repo:       nil,
				ReleaseSvc: nil,
				attempts:   0,
				errors:     nil,
				JobID:      0,
			},
			args: args{item: &gofeed.Item{
				Title: "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP",
				Description: `Category: Example
 Size: 1.49 GB
 Status: 27 seeders and 1 leechers
 Speed: 772.16 kB/s
 Added: 2022-09-29 16:06:08
`,
				Link: "https://fake-feed.com/details.php?id=00000&hit=1",
				GUID: "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP",
				//PublishedParsed: &nowMinusTime,
			}},
			want: &domain.Release{
				ID:                              0,
				FilterStatus:                    "PENDING",
				Rejections:                      []string{},
				Indexer:                         domain.IndexerMinimal{0, "Mock Feed", "mock-feed", "Mock Indexer"},
				FilterName:                      "",
				Protocol:                        "torrent",
				Implementation:                  "RSS",
				AnnounceType:                    domain.AnnounceTypeNew,
				Timestamp:                       now,
				GroupID:                         "",
				TorrentID:                       "",
				DownloadURL:                     "https://fake-feed.com/details.php?id=00000&hit=1",
				TorrentTmpFile:                  "",
				TorrentDataRawBytes:             []uint8(nil),
				TorrentHash:                     "",
				TorrentName:                     "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP",
				NormalizedHash:                  "edfbe552ccde335f34b801e15930bc35",
				Size:                            1490000000,
				Title:                           "Some Release Title",
				Description:                     "Category: Example\n Size: 1.49 GB\n Status: 27 seeders and 1 leechers\n Speed: 772.16 kB/s\n Added: 2022-09-29 16:06:08\n",
				Category:                        "",
				Season:                          0,
				Episode:                         0,
				Year:                            2022,
				Month:                           9,
				Day:                             22,
				Resolution:                      "720p",
				Source:                          "WEB",
				Codec:                           []string{"H.264"},
				Container:                       "",
				HDR:                             []string(nil),
				Audio:                           []string(nil),
				AudioChannels:                   "",
				Group:                           "GROUP",
				Region:                          "",
				Language:                        []string{},
				Proper:                          false,
				Repack:                          false,
				Edition:                         []string{},
				Cut:                             []string{},
				Website:                         "",
				Artists:                         "",
				Type:                            rls.Episode,
				LogScore:                        0,
				Origin:                          "",
				Tags:                            []string{},
				ReleaseTags:                     "",
				Freeleech:                       false,
				FreeleechPercent:                0,
				Bonus:                           []string(nil),
				Uploader:                        "",
				PreTime:                         "",
				Other:                           []string{},
				RawCookie:                       "",
				AdditionalSizeCheckRequired:     false,
				AdditionalUploaderCheckRequired: false,
				FilterID:                        0,
				Filter:                          (*domain.Filter)(nil),
				ActionStatus:                    []domain.ReleaseActionStatus(nil),
			},
		},
		{
			name: "time_parse",
			fields: fields{
				Feed: &domain.Feed{
					MaxAge: 360,
				},
				Name:       "test feed",
				Log:        zerolog.Logger{},
				URL:        "https://fake-feed.com/rss",
				Repo:       nil,
				ReleaseSvc: nil,
				attempts:   0,
				errors:     nil,
				JobID:      0,
			},
			args: args{item: &gofeed.Item{
				Title: "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP",
				Description: `Category: Example
 Size: 1.49 GB
 Status: 27 seeders and 1 leechers
 Speed: 772.16 kB/s
 Added: 2022-09-29 16:06:08
`,
				Link:            "https://fake-feed.com/details.php?id=00000&hit=1",
				GUID:            "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP",
				PublishedParsed: &nowMinusTime,
			}},
			want: nil,
		},
		{
			name: "magnet",
			fields: fields{
				Feed: &domain.Feed{
					MaxAge: 3600,
					Indexer: domain.IndexerMinimal{
						ID:                 0,
						Name:               "Mock Feed",
						Identifier:         "mock-feed",
						IdentifierExternal: "Mock Indexer",
					},
					Settings: &domain.FeedSettingsJSON{DownloadType: domain.FeedDownloadTypeMagnet},
				},
				Name:       "Magnet feed",
				Log:        zerolog.Logger{},
				URL:        "https://fake-feed.com/rss",
				Repo:       nil,
				ReleaseSvc: nil,
				attempts:   0,
				errors:     nil,
				JobID:      0,
			},
			args: args{item: &gofeed.Item{
				Title:       "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP",
				Description: "Category: Example",
				Link:        "https://fake-feed.com/details.php?id=00000&hit=1",
				GUID:        "https://fake-feed.com/details.php?id=00000&hit=1",
				Enclosures: []*gofeed.Enclosure{
					{
						URL:    "magnet:?xt=this-not-a-valid-magnet",
						Length: "1",
						Type:   "application/x-bittorrent",
					},
				},
			}},
			want: &domain.Release{
				ID:                              0,
				FilterStatus:                    "PENDING",
				Rejections:                      []string{},
				Indexer:                         domain.IndexerMinimal{0, "Mock Feed", "mock-feed", "Mock Indexer"},
				FilterName:                      "",
				Protocol:                        "torrent",
				Implementation:                  "RSS",
				AnnounceType:                    domain.AnnounceTypeNew,
				Timestamp:                       now,
				GroupID:                         "",
				TorrentID:                       "",
				DownloadURL:                     "https://fake-feed.com/details.php?id=00000&hit=1",
				MagnetURI:                       "magnet:?xt=this-not-a-valid-magnet",
				TorrentTmpFile:                  "",
				TorrentDataRawBytes:             []uint8(nil),
				TorrentHash:                     "",
				TorrentName:                     "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP",
				NormalizedHash:                  "edfbe552ccde335f34b801e15930bc35",
				Size:                            0,
				Title:                           "Some Release Title",
				Description:                     "Category: Example",
				Category:                        "",
				Season:                          0,
				Episode:                         0,
				Year:                            2022,
				Month:                           9,
				Day:                             22,
				Resolution:                      "720p",
				Source:                          "WEB",
				Codec:                           []string{"H.264"},
				Container:                       "",
				HDR:                             []string(nil),
				Audio:                           []string(nil),
				AudioChannels:                   "",
				Group:                           "GROUP",
				Region:                          "",
				Language:                        []string{},
				Proper:                          false,
				Repack:                          false,
				Edition:                         []string{},
				Cut:                             []string{},
				Website:                         "",
				Artists:                         "",
				Type:                            rls.Episode,
				LogScore:                        0,
				Origin:                          "",
				Tags:                            []string{},
				ReleaseTags:                     "",
				Freeleech:                       false,
				FreeleechPercent:                0,
				Bonus:                           []string(nil),
				Uploader:                        "",
				PreTime:                         "",
				Other:                           []string{},
				RawCookie:                       "",
				AdditionalSizeCheckRequired:     false,
				AdditionalUploaderCheckRequired: false,
				FilterID:                        0,
				Filter:                          (*domain.Filter)(nil),
				ActionStatus:                    []domain.ReleaseActionStatus(nil),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &RSSJob{
				Feed:       tt.fields.Feed,
				Name:       tt.fields.Name,
				Log:        tt.fields.Log,
				URL:        tt.fields.URL,
				CacheRepo:  tt.fields.Repo,
				ReleaseSvc: tt.fields.ReleaseSvc,
				attempts:   tt.fields.attempts,
				errors:     tt.fields.errors,
				JobID:      tt.fields.JobID,
			}
			got := j.processItem(tt.args.item)
			if got != nil {
				got.Timestamp = now // override to match
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_isMaxAge(t *testing.T) {
	t.Parallel()
	type args struct {
		maxAge int
		item   time.Time
		now    time.Time
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "01",
			args: args{
				maxAge: 3600,
				item:   time.Now().Add(time.Duration(-500) * time.Second),
				now:    time.Now(),
			},
			want: true,
		},
		{
			name: "02",
			args: args{
				maxAge: 3600,
				item:   time.Now().Add(time.Duration(-5000) * time.Second),
				now:    time.Now(),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, isNewerThanMaxAge(tt.args.maxAge, tt.args.item, tt.args.now), "isNewerThanMaxAge(%v, %v, %v)", tt.args.maxAge, tt.args.item, tt.args.now)
		})
	}
}

func Test_readSizeFromDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		str  string
		want string
	}{
		{
			name: "with size in GB",
			str:  "Size: 12GB",
			want: "12GB",
		},
		{
			name: "with size in GB with space",
			str:  "Size: 12 GB",
			want: "12GB",
		},
		{
			name: "with size in GiB",
			str:  "Size: 12 GiB",
			want: "12GiB",
		},
		{
			name: "with size in MiB",
			str:  "Size: 537 MiB",
			want: "537MiB",
		},
		{
			name: "with HTML tags",
			str:  "<strong>Size</strong>: 20.48 GiB<br>",
			want: "20.48GiB",
		},
		{
			name: "with additional text",
			str:  "file.name-GROUP / 20.48 GiB / x265",
			want: "20.48GiB",
		},
		{
			name: "without size info",
			str:  "<strong>Uploaded</strong>: 38 minutes ago<br>",
			want: "0B",
		},
		{
			name: "multiple sizes",
			str:  "<strong>Uploaded</strong>: 38B minutes ago<br>Size: 32GB",
			want: "32GB",
		},
		{
			name: "upgrade size",
			str:  `<p> <strong>Name</strong>: One.S01E01.German.DL.DTS.1080p.BluRay.x265.10bit-Cats<br> <strong>Category</strong>: Anime Serien<br> <strong>Type</strong>: Encode<br> <strong>Resolution</strong>: 1080p<br> <strong>Size</strong>: 2.49 GiB<br> <strong>Uploaded</strong>: vor 3 Minuten<br> <strong>Seeders</strong>: 1 | <strong>Leechers</strong>: 7 | <strong>Completed</strong>: 0<br> <strong>Uploader</strong>: Hochgeladen von xxx <br> IMDB Link:<a href="https://anon.to?http://www.imdb.com/title/tt1" target="_blank">tt1</a><br> TMDB Link: <a href="https://anon.to?https://www.themoviedb.org/tv/1" target="_blank">1</a><br> </p>`,
			want: "2.49GiB",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			wantBytes, err := humanize.ParseBytes(tt.want)
			if err != nil {
				t.Fatalf("Failed to parse size string %q: %v", tt.want, err)
			}

			r := &domain.Release{}
			readSizeFromDescription(tt.str, r)
			if r.Size != wantBytes {
				t.Errorf("readSizeFromDescription(%q) got %v bytes, want %v bytes", tt.str, r.Size, wantBytes)
			}
		})
	}
}
