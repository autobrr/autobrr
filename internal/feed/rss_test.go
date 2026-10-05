// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/dustin/go-humanize"
	"github.com/mmcdole/gofeed"
	"github.com/moistari/rls"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRSSSource_toRelease(t *testing.T) {
	t.Parallel()
	now := time.Now()

	type fields struct {
		Feed *domain.Feed
		Name string
		Log  zerolog.Logger
		URL  string
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
				Name: "test feed",
				Log:  zerolog.Logger{},
				URL:  "https://fake-feed.com/rss",
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
				Indexer:                         domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed", IdentifierExternal: "Mock Indexer"},
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
				Name: "test feed",
				Log:  zerolog.Logger{},
				URL:  "https://fake-feed.com/rss",
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
				Indexer:                         domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed", IdentifierExternal: "Mock Indexer"},
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
				Name: "test feed",
				Log:  zerolog.Logger{},
				URL:  "https://fake-feed.com/rss",
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
				Indexer:                         domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed", IdentifierExternal: "Mock Indexer"},
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
				Name: "Magnet feed",
				Log:  zerolog.Logger{},
				URL:  "https://fake-feed.com/rss",
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
				Indexer:                         domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed", IdentifierExternal: "Mock Indexer"},
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
		{
			name: "unicode_escaped_url_chars",
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
				Name: "test feed",
				Log:  zerolog.Logger{},
				URL:  "https://fake-feed.com/rss",
			},
			args: args{item: &gofeed.Item{
				Title: "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP",
				Description: `Category: Example
 Size: 1.49 GB`,
				Link: "https://fake-feed.com\u002fdownload.php\u003fid\u003d00000\u0026hit\u003d1\u0026type\u003dtorrent\u0026name\u003dSome%20Movie%20Title\u0026hash\u003dabc123\u0040group\u003aname\u0023section\u0025test\u002bextra",
				GUID: "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP",
			}},
			want: &domain.Release{
				ID:                              0,
				FilterStatus:                    "PENDING",
				Rejections:                      []string{},
				Indexer:                         domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed", IdentifierExternal: "Mock Indexer"},
				FilterName:                      "",
				Protocol:                        "torrent",
				Implementation:                  "RSS",
				AnnounceType:                    domain.AnnounceTypeNew,
				Timestamp:                       now,
				GroupID:                         "",
				TorrentID:                       "",
				DownloadURL:                     "https://fake-feed.com/download.php?id=00000&hit=1&type=torrent&name=Some%20Movie%20Title&hash=abc123@group:name#section%test+extra",
				TorrentTmpFile:                  "",
				TorrentDataRawBytes:             []uint8(nil),
				TorrentHash:                     "",
				TorrentName:                     "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP",
				NormalizedHash:                  "edfbe552ccde335f34b801e15930bc35",
				Size:                            1490000000,
				Title:                           "Some Release Title",
				Description:                     "Category: Example\n Size: 1.49 GB",
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
			tt.fields.Feed.URL = tt.fields.URL
			src := &rssSource{log: tt.fields.Log, feed: tt.fields.Feed}

			got := src.toRelease(tt.args.item)
			if got != nil {
				got.Timestamp = now // override to match
				got.TraceID = ""    // random per release, override to match
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRSSSource_toReleaseMagnet(t *testing.T) {
	t.Parallel()

	const (
		magnetURI = "magnet:?xt=urn:btih:deadbeef"
		detailURL = "https://fake-feed.com/details.php?id=00000"
	)

	magnetEnclosure := []*gofeed.Enclosure{{URL: magnetURI, Length: "1", Type: "application/x-bittorrent"}}

	tests := []struct {
		name            string
		downloadType    domain.FeedDownloadType
		item            *gofeed.Item
		wantDownloadURL string
		wantMagnetURI   string
	}{
		{
			name:            "magnet type keeps the detail link as fallback",
			downloadType:    domain.FeedDownloadTypeMagnet,
			item:            &gofeed.Item{Link: detailURL, Enclosures: magnetEnclosure},
			wantDownloadURL: detailURL,
			wantMagnetURI:   magnetURI,
		},
		{
			name:          "magnet in the link is not mangled into a relative url",
			downloadType:  domain.FeedDownloadTypeMagnet,
			item:          &gofeed.Item{Link: magnetURI},
			wantMagnetURI: magnetURI,
		},
		{
			name:          "magnet in the link is rescued without the magnet download type",
			item:          &gofeed.Item{Link: magnetURI},
			wantMagnetURI: magnetURI,
		},
		{
			name:          "magnet in the enclosure is rescued without the magnet download type",
			item:          &gofeed.Item{Enclosures: magnetEnclosure},
			wantMagnetURI: magnetURI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &rssSource{
				log: zerolog.Nop(),
				feed: &domain.Feed{
					URL:      "https://fake-feed.com/rss",
					Indexer:  domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed"},
					Settings: &domain.FeedSettingsJSON{DownloadType: tt.downloadType},
				},
			}

			tt.item.Title = "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP"

			got := src.toRelease(tt.item)
			require.NotNil(t, got)

			assert.Equal(t, tt.wantDownloadURL, got.DownloadURL, "download url")
			assert.Equal(t, tt.wantMagnetURI, got.MagnetURI, "magnet uri")
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			wantBytes, err := humanize.ParseBytes(tt.want)
			require.NoError(t, err)

			r := &domain.Release{}
			readSizeFromDescription(tt.str, r)
			assert.Equal(t, wantBytes, r.Size)
		})
	}
}

func TestRSSSource_fetch(t *testing.T) {
	const response = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
<title>Mock Feed</title>
<item>
<title>Item.With.Guid</title>
<link>https://fake-feed.com/download/1</link>
<guid>guid-1</guid>
<pubDate>Mon, 05 Oct 2026 10:00:00 +0000</pubDate>
</item>
<item>
<title>Item.Without.Guid</title>
<link>https://fake-feed.com/download/2</link>
</item>
<item>
<title>Item.With.Only.Title</title>
</item>
</channel>
</rss>`

	var gotCookie, gotUserAgent string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		gotUserAgent = r.Header.Get("User-Agent")

		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(response))
	}))
	defer srv.Close()

	f := &domain.Feed{
		Type:      string(domain.FeedTypeRSS),
		URL:       srv.URL,
		Cookie:    "uid=1; pass=secret",
		UserAgent: "autobrr-test",
		Indexer:   domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed"},
	}

	src, err := (&Service{}).newSource(t.Context(), f, zerolog.Nop())
	require.NoError(t, err)

	res, err := src.fetch(t.Context())
	require.NoError(t, err)

	assert.Equal(t, "uid=1; pass=secret", gotCookie)
	assert.Equal(t, "autobrr-test", gotUserAgent)
	assert.Contains(t, res.raw, "Item.With.Guid")

	require.Len(t, res.entries, 3)

	keys := make([]string, 0, len(res.entries))
	for _, e := range res.entries {
		keys = append(keys, e.key)
	}
	assert.Equal(t, []string{"guid-1", "https://fake-feed.com/download/2", "Item.With.Only.Title"}, keys, "guid, then link, then title")

	assert.Equal(t, time.Date(2026, time.October, 5, 10, 0, 0, 0, time.UTC), res.entries[0].pubDate.UTC())
	assert.True(t, res.entries[1].pubDate.IsZero(), "missing pub date stays zero so max age skips it")

	rls := res.entries[0].release()
	assert.Equal(t, domain.ReleaseImplementationRSS, rls.Implementation)
	assert.Equal(t, "https://fake-feed.com/download/1", rls.DownloadURL)
	assert.Equal(t, "uid=1; pass=secret", rls.RawCookie)
}
