package feed

import (
	"encoding/xml"
	"sort"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

type RSSResponse struct {
	Channel struct {
		Items []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string   `xml:"title,omitempty"`
	PubDate     Time     `xml:"pub_date,omitempty"`
	GUID        string   `xml:"guid,omitempty"`
	Comments    string   `xml:"comments"`
	Size        string   `xml:"size"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	Category    []string `xml:"category,omitempty"`
}

// Time credits: https://github.com/mrobinsn/go-newznab/blob/cd89d9c56447859fa1298dc9a0053c92c45ac7ef/newznab/structs.go#L150
type Time struct {
	time.Time
}

func (t *Time) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if err := e.EncodeToken(start); err != nil {
		return errors.Wrap(err, "failed to encode xml token")
	}
	if err := e.EncodeToken(xml.CharData([]byte(t.UTC().Format(time.RFC1123Z)))); err != nil {
		return errors.Wrap(err, "failed to encode xml token")
	}
	if err := e.EncodeToken(xml.EndElement{Name: start.Name}); err != nil {
		return errors.Wrap(err, "failed to encode xml token")
	}
	return nil
}

func (t *Time) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var raw string

	err := d.DecodeElement(&raw, &start)
	if err != nil {
		return errors.Wrap(err, "could not decode element")
	}

	date, err := time.Parse(time.RFC1123Z, raw)
	if err != nil {
		return errors.Wrap(err, "could not parse date")
	}

	*t = Time{date}
	return nil
}

type RSSJob struct {
	Name              string
	IndexerIdentifier string
	Log               zerolog.Logger
	URL               string
	Repo              domain.FeedCacheRepo
	ReleaseSvc        release.Service

	attempts int
	errors   []error

	JobID int
}

func NewRSSJob(name string, indexerIdentifier string, log zerolog.Logger, url string, repo domain.FeedCacheRepo, releaseSvc release.Service) *RSSJob {
	return &RSSJob{
		Name:              name,
		IndexerIdentifier: indexerIdentifier,
		Log:               log,
		URL:               url,
		Repo:              repo,
		ReleaseSvc:        releaseSvc,
	}
}

func (j *RSSJob) Run() {
	err := j.process()
	if err != nil {
		j.Log.Err(err).Int("attempts", j.attempts).Msg("rss feed process error")

		j.errors = append(j.errors, err)
	}

	j.attempts = 0
	j.errors = j.errors[:0]

	return
}

func (j *RSSJob) process() error {
	// TODO getFeed

	// get feed
	items, err := j.getFeed()
	if err != nil {
		j.Log.Error().Err(err).Msgf("error fetching rss feed items")
		return errors.Wrap(err, "error getting rss feed items")
	}

	j.Log.Debug().Msgf("found (%d) new items to process", len(items))

	if len(items) == 0 {
		return nil
	}

	releases := make([]*domain.Release, 0)

	for _, item := range items {
		rls := domain.NewRelease(j.IndexerIdentifier)

		rls.TorrentName = item.Title
		rls.TorrentURL = item.Link
		rls.Implementation = domain.ReleaseImplementationRSS
		rls.Indexer = j.IndexerIdentifier

		// parse size bytes string
		rls.ParseSizeBytesString(item.Size)

		rls.ParseString(item.Title)

		releases = append(releases, rls)
	}

	// process all new releases
	go j.ReleaseSvc.ProcessMultiple(releases)

	return nil
}

func (j *RSSJob) getFeed() ([]RSSItem, error) {
	// get feed

	feedItems := make([]RSSItem, 0)
	//feedItems, err := j.Client.GetFeed()
	//if err != nil {
	//	j.Log.Error().Err(err).Msgf("error fetching rss feed items")
	//	return nil, errors.Wrap(err, "error fetching rss feed items")
	//}

	j.Log.Debug().Msgf("refreshing rss feed: %v, found (%d) items", j.Name, len(feedItems))

	items := make([]RSSItem, 0)
	if len(feedItems) == 0 {
		return items, nil
	}

	sort.SliceStable(feedItems, func(i, j int) bool {
		return feedItems[i].PubDate.After(feedItems[j].PubDate.Time)
	})

	for _, i := range feedItems {
		if i.GUID == "" {
			continue
		}

		exists, err := j.Repo.Exists(j.Name, i.GUID)
		if err != nil {
			j.Log.Error().Err(err).Msg("could not check if item exists")
			continue
		}
		if exists {
			j.Log.Trace().Msgf("cache item exists, skipping release: %v", i.Title)
			continue
		}

		// set ttl to 1 month
		ttl := time.Now().AddDate(0, 1, 0)

		if err := j.Repo.Put(j.Name, i.GUID, []byte(i.Title), ttl); err != nil {
			j.Log.Error().Stack().Err(err).Str("guid", i.GUID).Msg("cache.Put: error storing item in cache")
			continue
		}

		// only append if we successfully added to cache
		items = append(items, i)
	}

	// send to filters
	return items, nil
}
