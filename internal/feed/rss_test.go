package feed

import (
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/release"

	"github.com/mmcdole/gofeed"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestRSSJob_processItem(t *testing.T) {
	now := time.Now()
	nowMinusTime := time.Now().Add(time.Duration(-3000) * time.Second)

	type fields struct {
		Feed              *domain.Feed
		Name              string
		IndexerIdentifier string
		Log               zerolog.Logger
		URL               string
		Repo              domain.FeedCacheRepo
		ReleaseSvc        release.Service
		attempts          int
		errors            []error
		JobID             int
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
				},
				Name:              "test feed",
				IndexerIdentifier: "mock-feed",
				Log:               zerolog.Logger{},
				URL:               "https://fake-feed.com/rss",
				Repo:              nil,
				ReleaseSvc:        nil,
				attempts:          0,
				errors:            nil,
				JobID:             0,
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
			want: &domain.Release{ID: 0, FilterStatus: "PENDING", Rejections: []string{}, Indexer: "mock-feed", FilterName: "", Protocol: "torrent", Implementation: "RSS", Timestamp: now, GroupID: "", TorrentID: "", TorrentURL: "https://fake-feed.com/details.php?id=00000&hit=1", TorrentTmpFile: "", TorrentDataRawBytes: []uint8(nil), TorrentHash: "", TorrentName: "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP", Size: 0x0, Title: "Some Release Title", Category: "", Season: 0, Episode: 0, Year: 2022, Resolution: "720p", Source: "WEB", Codec: []string{"H.264"}, Container: "", HDR: []string(nil), Audio: []string(nil), AudioChannels: "", Group: "GROUP", Region: "", Language: nil, Proper: false, Repack: false, Website: "", Artists: "", Type: "", LogScore: 0, IsScene: false, Origin: "", Tags: []string{}, ReleaseTags: "", Freeleech: false, FreeleechPercent: 0, Bonus: []string(nil), Uploader: "", PreTime: "", Other: []string(nil), RawCookie: "", AdditionalSizeCheckRequired: false, FilterID: 0, Filter: (*domain.Filter)(nil), ActionStatus: []domain.ReleaseActionStatus(nil)},
		},
		{
			name: "with_baseurl",
			fields: fields{
				Feed: &domain.Feed{
					MaxAge: 3600,
				},
				Name:              "test feed",
				IndexerIdentifier: "mock-feed",
				Log:               zerolog.Logger{},
				URL:               "https://fake-feed.com/rss",
				Repo:              nil,
				ReleaseSvc:        nil,
				attempts:          0,
				errors:            nil,
				JobID:             0,
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
			want: &domain.Release{ID: 0, FilterStatus: "PENDING", Rejections: []string{}, Indexer: "mock-feed", FilterName: "", Protocol: "torrent", Implementation: "RSS", Timestamp: now, GroupID: "", TorrentID: "", TorrentURL: "https://fake-feed.com/details.php?id=00000&hit=1", TorrentTmpFile: "", TorrentDataRawBytes: []uint8(nil), TorrentHash: "", TorrentName: "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP", Size: 0x0, Title: "Some Release Title", Category: "", Season: 0, Episode: 0, Year: 2022, Resolution: "720p", Source: "WEB", Codec: []string{"H.264"}, Container: "", HDR: []string(nil), Audio: []string(nil), AudioChannels: "", Group: "GROUP", Region: "", Language: nil, Proper: false, Repack: false, Website: "", Artists: "", Type: "", LogScore: 0, IsScene: false, Origin: "", Tags: []string{}, ReleaseTags: "", Freeleech: false, FreeleechPercent: 0, Bonus: []string(nil), Uploader: "", PreTime: "", Other: []string(nil), RawCookie: "", AdditionalSizeCheckRequired: false, FilterID: 0, Filter: (*domain.Filter)(nil), ActionStatus: []domain.ReleaseActionStatus(nil)},
		},
		{
			name: "time_parse",
			fields: fields{
				Feed: &domain.Feed{
					MaxAge: 360,
				},
				Name:              "test feed",
				IndexerIdentifier: "mock-feed",
				Log:               zerolog.Logger{},
				URL:               "https://fake-feed.com/rss",
				Repo:              nil,
				ReleaseSvc:        nil,
				attempts:          0,
				errors:            nil,
				JobID:             0,
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
			want: &domain.Release{ID: 0, FilterStatus: "PENDING", Rejections: []string{}, Indexer: "mock-feed", FilterName: "", Protocol: "torrent", Implementation: "RSS", Timestamp: now, GroupID: "", TorrentID: "", TorrentURL: "https://fake-feed.com/details.php?id=00000&hit=1", TorrentTmpFile: "", TorrentDataRawBytes: []uint8(nil), TorrentHash: "", TorrentName: "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP", Size: 0x0, Title: "Some Release Title", Category: "", Season: 0, Episode: 0, Year: 2022, Resolution: "720p", Source: "WEB", Codec: []string{"H.264"}, Container: "", HDR: []string(nil), Audio: []string(nil), AudioChannels: "", Group: "GROUP", Region: "", Language: nil, Proper: false, Repack: false, Website: "", Artists: "", Type: "", LogScore: 0, IsScene: false, Origin: "", Tags: []string{}, ReleaseTags: "", Freeleech: false, FreeleechPercent: 0, Bonus: []string(nil), Uploader: "", PreTime: "", Other: []string(nil), RawCookie: "", AdditionalSizeCheckRequired: false, FilterID: 0, Filter: (*domain.Filter)(nil), ActionStatus: []domain.ReleaseActionStatus(nil)},
		},
		{
			name: "time_parse",
			fields: fields{
				Feed: &domain.Feed{
					MaxAge: 360,
				},
				Name:              "test feed",
				IndexerIdentifier: "mock-feed",
				Log:               zerolog.Logger{},
				URL:               "https://fake-feed.com/rss",
				Repo:              nil,
				ReleaseSvc:        nil,
				attempts:          0,
				errors:            nil,
				JobID:             0,
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &RSSJob{
				Feed:              tt.fields.Feed,
				Name:              tt.fields.Name,
				IndexerIdentifier: tt.fields.IndexerIdentifier,
				Log:               tt.fields.Log,
				URL:               tt.fields.URL,
				CacheRepo:         tt.fields.Repo,
				ReleaseSvc:        tt.fields.ReleaseSvc,
				attempts:          tt.fields.attempts,
				errors:            tt.fields.errors,
				JobID:             tt.fields.JobID,
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
