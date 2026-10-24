package feed

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTorznabJob_processItems(t *testing.T) {
	const (
		magnetURI  = "magnet:?xt=urn:btih:deadbeef"
		proxyURL   = "http://jackett:9117/dl/mock/?jackett_apikey=key&file=Some.Release"
		torrentURL = "https://fake-feed.com/download/00000.torrent"
	)

	torrentEnclosure := &torznab.Enclosure{URL: torrentURL, Type: "application/x-bittorrent"}
	magnetEnclosure := &torznab.Enclosure{URL: magnetURI, Type: "application/x-bittorrent"}

	tests := []struct {
		name            string
		settings        *domain.FeedSettingsJSON
		item            torznab.FeedItem
		wantDownloadURL string
		wantMagnetURI   string
	}{
		{
			name:            "no feed settings keeps the torrent url",
			item:            torznab.FeedItem{Link: torrentURL, Enclosure: torrentEnclosure},
			wantDownloadURL: torrentURL,
		},
		{
			name:            "torrent type ignores the magneturl attr",
			settings:        &domain.FeedSettingsJSON{DownloadType: domain.FeedDownloadTypeTorrent},
			item:            torznab.FeedItem{Link: torrentURL, Enclosure: torrentEnclosure, MagnetURI: magnetURI},
			wantDownloadURL: torrentURL,
		},
		{
			name:          "magnet in the link",
			settings:      &domain.FeedSettingsJSON{DownloadType: domain.FeedDownloadTypeMagnet},
			item:          torznab.FeedItem{Link: magnetURI},
			wantMagnetURI: magnetURI,
		},
		{
			name:          "magnet in the enclosure",
			settings:      &domain.FeedSettingsJSON{DownloadType: domain.FeedDownloadTypeMagnet},
			item:          torznab.FeedItem{Link: proxyURL, Enclosure: magnetEnclosure},
			wantMagnetURI: magnetURI,
		},
		{
			name:            "magneturl attr wins over the proxy link",
			settings:        &domain.FeedSettingsJSON{DownloadType: domain.FeedDownloadTypeMagnet},
			item:            torznab.FeedItem{Link: proxyURL, MagnetURI: magnetURI},
			wantDownloadURL: proxyURL,
			wantMagnetURI:   magnetURI,
		},
		{
			name:            "proxy link is kept for ResolveMagnetURI to follow",
			settings:        &domain.FeedSettingsJSON{DownloadType: domain.FeedDownloadTypeMagnet},
			item:            torznab.FeedItem{Link: proxyURL},
			wantDownloadURL: proxyURL,
			wantMagnetURI:   proxyURL,
		},
		{
			name:          "magnet never stays in the download url",
			settings:      &domain.FeedSettingsJSON{DownloadType: domain.FeedDownloadTypeTorrent},
			item:          torznab.FeedItem{Link: proxyURL, Enclosure: magnetEnclosure},
			wantMagnetURI: magnetURI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &TorznabJob{
				Log: zerolog.New(io.Discard),
				Feed: &domain.Feed{
					Indexer:  domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed"},
					Settings: tt.settings,
				},
			}

			tt.item.Title = "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP"

			releases, err := j.processItems([]torznab.FeedItem{tt.item})
			require.NoError(t, err)
			require.Len(t, releases, 1)

			assert.Equal(t, tt.wantDownloadURL, releases[0].DownloadURL, "download url")
			assert.Equal(t, tt.wantMagnetURI, releases[0].MagnetURI, "magnet uri")
		})
	}
}

func TestTorznabJob_processItems_metadata(t *testing.T) {
	pubDate := time.Date(2026, time.September, 24, 5, 58, 24, 0, time.UTC)

	j := &TorznabJob{
		Log: zerolog.New(io.Discard),
		Feed: &domain.Feed{
			Indexer: domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed"},
		},
	}

	item := torznab.FeedItem{
		Title:   "That Show S01 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-NOGROUP",
		TvdbId:  "12345",
		ImdbId:  "tt1234567",
		TmdbId:  "54321",
		PubDate: torznab.Time{Time: pubDate},
	}

	releases, err := j.processItems([]torznab.FeedItem{item})
	require.NoError(t, err)
	require.Len(t, releases, 1)

	assert.Equal(t, 12345, releases[0].MetaTVDB, "tvdb id")
	assert.Equal(t, "tt1234567", releases[0].MetaIMDB, "imdb id")
	assert.Equal(t, 54321, releases[0].MetaTMDB, "tmdb id")
	assert.Equal(t, pubDate, releases[0].PublishDate, "publish date")
}

func TestTorznabJob_RunE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryType := r.URL.Query().Get("t")
		switch queryType {
		case "search":
			payload, err := os.ReadFile("testdata/torznab/torznab_response.xml")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/xml")
			w.Write(payload)
			break

		case "caps":
			payload, err := os.ReadFile("testdata/torznab/caps_response.xml")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/xml")
			w.Write(payload)
			break
		}
	}))
	defer srv.Close()

	type fields struct {
		Feed       *domain.Feed
		Name       string
		Log        zerolog.Logger
		URL        string
		Client     *torznab.Client
		Repo       jobFeedRepo
		CacheRepo  jobFeedCacheRepo
		ReleaseSvc jobReleaseSvc
		attempts   int
		errors     []error
		JobID      int
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "test",
			fields: fields{
				Name: "test",
				Log:  zerolog.New(io.Discard),
				Feed: &domain.Feed{
					MaxAge: 0,
					Indexer: domain.IndexerMinimal{
						ID:                 0,
						Name:               "Mock Feed",
						Identifier:         "mock-feed",
						IdentifierExternal: "Mock Indexer",
					},
				},
				URL:        srv.URL,
				Client:     torznab.NewClient(torznab.Config{Host: srv.URL}),
				Repo:       &mockFeedRepo{},
				CacheRepo:  &mockFeedCacheRepo{},
				ReleaseSvc: &mockReleaseSvc{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &TorznabJob{
				Feed:       tt.fields.Feed,
				Name:       tt.fields.Name,
				Log:        tt.fields.Log,
				URL:        tt.fields.URL,
				Client:     tt.fields.Client,
				Repo:       tt.fields.Repo,
				CacheRepo:  tt.fields.CacheRepo,
				ReleaseSvc: tt.fields.ReleaseSvc,
				attempts:   tt.fields.attempts,
				errors:     tt.fields.errors,
				JobID:      tt.fields.JobID,
			}
			err := j.RunE(t.Context())
			assert.NoError(t, err)
		})
	}
}

type mockTorznabFeedClient struct {
	limit      int
	total      int
	pages      map[int][]torznab.FeedItem
	fallback   []torznab.FeedItem
	errOffsets map[int]error
	offsets    []int
}

func (m *mockTorznabFeedClient) WithHTTPClient(_ *http.Client) {}

func (m *mockTorznabFeedClient) Search(_ context.Context, _ string, _ []int, offset int) (*torznab.SearchResponse, error) {
	m.offsets = append(m.offsets, offset)

	if err := m.errOffsets[offset]; err != nil {
		return nil, err
	}

	items, ok := m.pages[offset]
	if !ok {
		items = m.fallback
	}

	respItems := make([]*torznab.FeedItem, 0, len(items))
	for i := range items {
		respItems = append(respItems, &items[i])
	}

	return &torznab.SearchResponse{Items: respItems, Limit: m.limit, Total: m.total}, nil
}

func TestTorznabJob_getFeed_pagination(t *testing.T) {
	newItem := func(guid string) torznab.FeedItem {
		return torznab.FeedItem{GUID: guid, Title: "Title." + guid}
	}

	guids := func(items []torznab.FeedItem) []string {
		out := make([]string, 0, len(items))
		for _, item := range items {
			out = append(out, item.GUID)
		}
		return out
	}

	newJob := func(client torznabClient, cache *stubFeedCacheRepo, maxPages int) *TorznabJob {
		return &TorznabJob{
			Log: zerolog.New(io.Discard),
			Feed: &domain.Feed{
				Indexer:  domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed"},
				Settings: &domain.FeedSettingsJSON{MaxPages: maxPages},
				LastRun:  time.Now().Add(-time.Hour),
			},
			Client:    client,
			Repo:      &mockFeedRepo{},
			CacheRepo: cache,
		}
	}

	t.Run("stops at the cached boundary on page 2", func(t *testing.T) {
		client := &mockTorznabFeedClient{
			limit: 3,
			pages: map[int][]torznab.FeedItem{
				0: {newItem("new-1"), newItem("new-2"), newItem("new-3")},
				3: {newItem("new-4"), newItem("old-1"), newItem("old-2")},
			},
		}
		cache := &stubFeedCacheRepo{existing: map[string]bool{"old-1": true, "old-2": true}}

		items, err := newJob(client, cache, domain.MaxFeedPages).getFeed(t.Context())
		require.NoError(t, err)

		// oldest first: the second page's items precede the first page's
		assert.Equal(t, []string{"new-4", "new-3", "new-2", "new-1"}, guids(items))
		assert.Equal(t, []int{0, 3}, client.offsets)

		require.Len(t, cache.putCalls, 2)
		assert.Equal(t, []string{"new-1", "new-2", "new-3"}, cache.putCalls[0])
		assert.Equal(t, []string{"new-4"}, cache.putCalls[1])
	})

	t.Run("fully cached first page stops immediately", func(t *testing.T) {
		client := &mockTorznabFeedClient{
			limit: 3,
			pages: map[int][]torznab.FeedItem{
				0: {newItem("old-1"), newItem("old-2"), newItem("old-3")},
			},
		}
		cache := &stubFeedCacheRepo{existing: map[string]bool{"old-1": true, "old-2": true, "old-3": true}}

		items, err := newJob(client, cache, domain.MaxFeedPages).getFeed(t.Context())
		require.NoError(t, err)

		assert.Empty(t, items)
		assert.Equal(t, []int{0}, client.offsets)
	})

	t.Run("short page is the end of the feed", func(t *testing.T) {
		client := &mockTorznabFeedClient{
			limit: 3,
			pages: map[int][]torznab.FeedItem{
				0: {newItem("new-1"), newItem("new-2")},
			},
		}

		items, err := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, domain.MaxFeedPages).getFeed(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-2", "new-1"}, guids(items))
		assert.Equal(t, []int{0}, client.offsets)
	})

	t.Run("catch-up backfills until the short page", func(t *testing.T) {
		client := &mockTorznabFeedClient{
			limit: 3,
			pages: map[int][]torznab.FeedItem{
				0: {newItem("new-1"), newItem("new-2"), newItem("new-3")},
				3: {newItem("new-4"), newItem("new-5"), newItem("new-6")},
				6: {newItem("new-7")},
			},
		}

		items, err := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, domain.MaxFeedPages).getFeed(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-7", "new-6", "new-5", "new-4", "new-3", "new-2", "new-1"}, guids(items))
		assert.Equal(t, []int{0, 3, 6}, client.offsets)
	})

	t.Run("max pages caps the backfill", func(t *testing.T) {
		client := &mockTorznabFeedClient{
			limit: 3,
			pages: map[int][]torznab.FeedItem{
				0: {newItem("new-1"), newItem("new-2"), newItem("new-3")},
				3: {newItem("new-4"), newItem("new-5"), newItem("new-6")},
				6: {newItem("new-7"), newItem("new-8"), newItem("new-9")},
			},
		}

		items, err := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, 2).getFeed(t.Context())
		require.NoError(t, err)

		assert.Len(t, items, 6)
		assert.Equal(t, []int{0, 3}, client.offsets)
	})

	t.Run("indexer ignoring the offset stops after a repeated page", func(t *testing.T) {
		samePage := []torznab.FeedItem{newItem("new-1"), newItem("new-2"), newItem("new-3")}
		client := &mockTorznabFeedClient{limit: 3, fallback: samePage}

		items, err := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, domain.MaxFeedPages).getFeed(t.Context())
		require.NoError(t, err)

		// the repeated items are deduped within the run, no release twice
		assert.Equal(t, []string{"new-3", "new-2", "new-1"}, guids(items))
		assert.Equal(t, []int{0, 3}, client.offsets)
	})

	t.Run("repeated page with shifted order stops on zero new items", func(t *testing.T) {
		client := &mockTorznabFeedClient{
			limit: 3,
			pages: map[int][]torznab.FeedItem{
				0: {newItem("new-1"), newItem("new-2"), newItem("new-3")},
				3: {newItem("new-2"), newItem("new-1"), newItem("new-3")},
			},
		}

		items, err := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, domain.MaxFeedPages).getFeed(t.Context())
		require.NoError(t, err)

		// the repeated page starts with a different item, so only the zero new
		// items rule stops it from paginating on
		assert.Equal(t, []string{"new-3", "new-2", "new-1"}, guids(items))
		assert.Equal(t, []int{0, 3}, client.offsets)
	})

	t.Run("reported total paginates past short pages", func(t *testing.T) {
		client := &mockTorznabFeedClient{
			limit: 3,
			total: 7,
			pages: map[int][]torznab.FeedItem{
				0: {newItem("new-1"), newItem("new-2")},
				2: {newItem("new-3"), newItem("new-4")},
				4: {newItem("new-5"), newItem("new-6")},
				6: {newItem("new-7")},
			},
		}

		items, err := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, domain.MaxFeedPages).getFeed(t.Context())
		require.NoError(t, err)

		// every page is shorter than the requested limit, but the reported
		// total says there is more until the last page
		assert.Equal(t, []string{"new-7", "new-6", "new-5", "new-4", "new-3", "new-2", "new-1"}, guids(items))
		assert.Equal(t, []int{0, 2, 4, 6}, client.offsets)
	})

	t.Run("overlapping pages are deduped within the run", func(t *testing.T) {
		client := &mockTorznabFeedClient{
			limit: 3,
			pages: map[int][]torznab.FeedItem{
				0: {newItem("new-1"), newItem("new-2"), newItem("new-3")},
				3: {newItem("new-3"), newItem("new-4")},
			},
		}

		items, err := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, domain.MaxFeedPages).getFeed(t.Context())
		require.NoError(t, err)

		// new-3 shifted onto page 2 but must not be processed twice, and the
		// overlap must not count as the previous run's boundary
		assert.Equal(t, []string{"new-4", "new-3", "new-2", "new-1"}, guids(items))
		assert.Equal(t, []int{0, 3}, client.offsets)
	})

	t.Run("items without GUID are dropped", func(t *testing.T) {
		client := &mockTorznabFeedClient{
			limit: 3,
			pages: map[int][]torznab.FeedItem{
				0: {torznab.FeedItem{Title: "Title.no-guid"}, newItem("new-1")},
			},
		}

		items, err := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, domain.MaxFeedPages).getFeed(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-1"}, guids(items))
	})

	t.Run("cache write failure drops the page and stops paginating", func(t *testing.T) {
		client := &mockTorznabFeedClient{
			limit: 3,
			pages: map[int][]torznab.FeedItem{
				0: {newItem("new-1"), newItem("new-2"), newItem("new-3")},
				3: {newItem("new-4"), newItem("new-5"), newItem("new-6")},
			},
		}
		cache := &stubFeedCacheRepo{existing: map[string]bool{}, putErr: errors.New("disk full")}

		items, err := newJob(client, cache, domain.MaxFeedPages).getFeed(t.Context())
		require.NoError(t, err)

		// nothing was cached, so nothing may be processed this run
		assert.Empty(t, items)
		assert.Empty(t, cache.putCalls)
		assert.Equal(t, []int{0}, client.offsets)
	})

	twoFullPages := func() map[int][]torznab.FeedItem {
		return map[int][]torznab.FeedItem{
			0: {newItem("new-1"), newItem("new-2"), newItem("new-3")},
			3: {newItem("new-4"), newItem("new-5"), newItem("new-6")},
			6: {newItem("new-7")},
		}
	}

	t.Run("unset max pages fetches only the first page", func(t *testing.T) {
		client := &mockTorznabFeedClient{limit: 3, pages: twoFullPages()}

		_, err := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, 0).getFeed(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []int{0}, client.offsets)
	})

	t.Run("fetch error on page 2 keeps the cached first page", func(t *testing.T) {
		client := &mockTorznabFeedClient{
			limit:      3,
			pages:      twoFullPages(),
			errOffsets: map[int]error{3: errors.New("429 too many requests")},
		}
		cache := &stubFeedCacheRepo{existing: map[string]bool{}}

		items, err := newJob(client, cache, domain.MaxFeedPages).getFeed(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-3", "new-2", "new-1"}, guids(items))
		assert.Equal(t, []int{0, 3}, client.offsets)
	})

	t.Run("cache lookup error on page 2 keeps the cached first page", func(t *testing.T) {
		client := &mockTorznabFeedClient{limit: 3, pages: twoFullPages()}
		cache := &stubFeedCacheRepo{existing: map[string]bool{}, existingErrOnCall: 2}

		items, err := newJob(client, cache, domain.MaxFeedPages).getFeed(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-3", "new-2", "new-1"}, guids(items))
		assert.Equal(t, []int{0, 3}, client.offsets)
	})

	t.Run("cache write error on page 2 keeps the cached first page", func(t *testing.T) {
		client := &mockTorznabFeedClient{limit: 3, pages: twoFullPages()}
		cache := &stubFeedCacheRepo{existing: map[string]bool{}, putErrOnCall: 2}

		items, err := newJob(client, cache, domain.MaxFeedPages).getFeed(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-3", "new-2", "new-1"}, guids(items))
		assert.Len(t, cache.putCalls, 1)
		assert.Equal(t, []int{0, 3}, client.offsets)
	})

	t.Run("fetch error on page 1 fails the run", func(t *testing.T) {
		client := &mockTorznabFeedClient{limit: 3, errOffsets: map[int]error{0: errors.New("503")}}

		j := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, domain.MaxFeedPages)
		j.Feed.LastRun = time.Time{}

		_, err := j.getFeed(t.Context())
		require.Error(t, err)

		assert.False(t, j.hasFetched, "a failed first fetch must retry the single page next run")
	})

	t.Run("first run fetches only the first page", func(t *testing.T) {
		client := &mockTorznabFeedClient{limit: 3, pages: twoFullPages()}
		j := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, domain.MaxFeedPages)
		j.Feed.LastRun = time.Time{}

		items, err := j.getFeed(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-3", "new-2", "new-1"}, guids(items))
		assert.Equal(t, []int{0}, client.offsets)

		// the job's own run now marks the cache boundary, so the next run paginates
		client.pages = map[int][]torznab.FeedItem{
			0: {newItem("next-1"), newItem("next-2"), newItem("next-3")},
			3: {newItem("next-4"), newItem("new-1"), newItem("new-2")},
		}

		items, err = j.getFeed(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"next-4", "next-3", "next-2", "next-1"}, guids(items))
		assert.Equal(t, []int{0, 0, 3}, client.offsets)
	})

	t.Run("last run older than the cache ttl fetches only the first page", func(t *testing.T) {
		client := &mockTorznabFeedClient{limit: 3, pages: twoFullPages()}
		j := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, domain.MaxFeedPages)
		j.Feed.LastRun = time.Now().AddDate(0, 0, -40)

		_, err := j.getFeed(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []int{0}, client.offsets)
	})

	t.Run("last run within the cache ttl paginates", func(t *testing.T) {
		client := &mockTorznabFeedClient{limit: 3, pages: twoFullPages()}
		j := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, domain.MaxFeedPages)
		j.Feed.LastRun = time.Now().AddDate(0, 0, -3)

		_, err := j.getFeed(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []int{0, 3, 6}, client.offsets)
	})

	datedItem := func(guid string, pubDate time.Time) torznab.FeedItem {
		item := newItem(guid)
		item.PubDate = torznab.Time{Time: pubDate}
		return item
	}

	t.Run("max age stops after a page reaching older items", func(t *testing.T) {
		now := time.Now()
		client := &mockTorznabFeedClient{
			limit: 3,
			pages: map[int][]torznab.FeedItem{
				0: {datedItem("new-1", now), datedItem("new-2", now.Add(-time.Hour)), datedItem("new-3", now.Add(-7*24*time.Hour))},
				3: {datedItem("new-4", now.Add(-8*24*time.Hour))},
			},
		}
		j := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, domain.MaxFeedPages)
		j.Feed.MaxAge = 3600 * 24

		items, err := j.getFeed(t.Context())
		require.NoError(t, err)

		// the page's items still reach processItems, which applies the age filter
		assert.Len(t, items, 3)
		assert.Equal(t, []int{0}, client.offsets)
	})

	t.Run("max age reached on page 2 stops after page 2", func(t *testing.T) {
		now := time.Now()
		client := &mockTorznabFeedClient{
			limit: 2,
			pages: map[int][]torznab.FeedItem{
				0: {datedItem("new-1", now), datedItem("new-2", now.Add(-time.Hour))},
				2: {datedItem("new-3", now.Add(-2*time.Hour)), datedItem("new-4", now.Add(-48*time.Hour))},
				4: {datedItem("new-5", now.Add(-72*time.Hour)), datedItem("new-6", now.Add(-96*time.Hour))},
			},
		}
		j := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, domain.MaxFeedPages)
		j.Feed.MaxAge = 3600 * 24

		_, err := j.getFeed(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []int{0, 2}, client.offsets)
	})

	t.Run("max age disabled paginates past old items", func(t *testing.T) {
		old := time.Now().AddDate(0, 0, -30)
		client := &mockTorznabFeedClient{
			limit: 2,
			pages: map[int][]torznab.FeedItem{
				0: {datedItem("new-1", old), datedItem("new-2", old)},
				2: {datedItem("new-3", old)},
			},
		}

		_, err := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, domain.MaxFeedPages).getFeed(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []int{0, 2}, client.offsets)
	})

	t.Run("items without a valid pub date never reach max age", func(t *testing.T) {
		client := &mockTorznabFeedClient{
			limit: 2,
			pages: map[int][]torznab.FeedItem{
				0: {datedItem("new-1", time.Time{}), datedItem("new-2", time.Unix(0, 0))},
				2: {datedItem("new-3", time.Time{})},
			},
		}
		j := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, domain.MaxFeedPages)
		j.Feed.MaxAge = 3600

		_, err := j.getFeed(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []int{0, 2}, client.offsets)
	})
}
