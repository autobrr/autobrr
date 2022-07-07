package domain

import (
	"context"
	"crypto/tls"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/dustin/go-humanize"
	"github.com/moistari/rls"
	"golang.org/x/net/publicsuffix"
)

type ReleaseRepo interface {
	Store(ctx context.Context, release *Release) (*Release, error)
	Find(ctx context.Context, params ReleaseQueryParams) (res []*Release, nextCursor int64, count int64, err error)
	FindRecent(ctx context.Context) ([]*Release, error)
	GetIndexerOptions(ctx context.Context) ([]string, error)
	GetActionStatusByReleaseID(ctx context.Context, releaseID int64) ([]ReleaseActionStatus, error)
	Stats(ctx context.Context) (*ReleaseStats, error)
	StoreReleaseActionStatus(ctx context.Context, actionStatus *ReleaseActionStatus) error
	Delete(ctx context.Context) error
}

type Release struct {
	ID                          int64                 `json:"id"`
	FilterStatus                ReleaseFilterStatus   `json:"filter_status"`
	Rejections                  []string              `json:"rejections"`
	Indexer                     string                `json:"indexer"`
	FilterName                  string                `json:"filter"`
	Protocol                    ReleaseProtocol       `json:"protocol"`
	Implementation              ReleaseImplementation `json:"implementation"` // irc, rss, api
	Timestamp                   time.Time             `json:"timestamp"`
	GroupID                     string                `json:"group_id"`
	TorrentID                   string                `json:"torrent_id"`
	TorrentURL                  string                `json:"-"`
	TorrentTmpFile              string                `json:"-"`
	TorrentHash                 string                `json:"-"`
	TorrentName                 string                `json:"torrent_name"` // full release name
	Size                        uint64                `json:"size"`
	Title                       string                `json:"title"` // Parsed title
	Category                    string                `json:"category"`
	Season                      int                   `json:"season"`
	Episode                     int                   `json:"episode"`
	Year                        int                   `json:"year"`
	Resolution                  string                `json:"resolution"`
	Source                      string                `json:"source"`
	Codec                       []string              `json:"codec"`
	Container                   string                `json:"container"`
	HDR                         []string              `json:"hdr"`
	Audio                       []string              `json:"-"`
	AudioChannels               string                `json:"-"`
	Group                       string                `json:"group"`
	Region                      string                `json:"-"`
	Language                    string                `json:"-"`
	Proper                      bool                  `json:"proper"`
	Repack                      bool                  `json:"repack"`
	Website                     string                `json:"website"`
	Artists                     string                `json:"-"`
	Type                        string                `json:"type"` // Album,Single,EP
	LogScore                    int                   `json:"-"`
	IsScene                     bool                  `json:"-"`
	Origin                      string                `json:"origin"` // P2P, Internal
	Tags                        []string              `json:"-"`
	ReleaseTags                 string                `json:"-"`
	Freeleech                   bool                  `json:"-"`
	FreeleechPercent            int                   `json:"-"`
	Bonus                       []string              `json:"-"`
	Uploader                    string                `json:"uploader"`
	PreTime                     string                `json:"pre_time"`
	Other                       []string              `json:"-"`
	RawCookie                   string                `json:"-"`
	AdditionalSizeCheckRequired bool                  `json:"-"`
	FilterID                    int                   `json:"-"`
	Filter                      *Filter               `json:"-"`
	ActionStatus                []ReleaseActionStatus `json:"action_status"`
}

type ReleaseActionStatus struct {
	ID         int64             `json:"id"`
	Status     ReleasePushStatus `json:"status"`
	Action     string            `json:"action"`
	Type       ActionType        `json:"type"`
	Client     string            `json:"client"`
	Filter     string            `json:"filter"`
	Rejections []string          `json:"rejections"`
	Timestamp  time.Time         `json:"timestamp"`
	ReleaseID  int64             `json:"-"`
}

type DownloadTorrentFileResponse struct {
	MetaInfo    *metainfo.MetaInfo
	TmpFileName string
}

type ReleaseStats struct {
	TotalCount          int64 `json:"total_count"`
	FilteredCount       int64 `json:"filtered_count"`
	FilterRejectedCount int64 `json:"filter_rejected_count"`
	PushApprovedCount   int64 `json:"push_approved_count"`
	PushRejectedCount   int64 `json:"push_rejected_count"`
}

type ReleasePushStatus string

const (
	ReleasePushStatusApproved ReleasePushStatus = "PUSH_APPROVED"
	ReleasePushStatusRejected ReleasePushStatus = "PUSH_REJECTED"
	ReleasePushStatusErr      ReleasePushStatus = "PUSH_ERROR"

	//ReleasePushStatusPending  ReleasePushStatus = "PENDING" // Initial status
)

func (r ReleasePushStatus) String() string {
	switch r {
	case ReleasePushStatusApproved:
		return "Approved"
	case ReleasePushStatusRejected:
		return "Rejected"
	case ReleasePushStatusErr:
		return "Error"
	default:
		return "Unknown"
	}
}

type ReleaseFilterStatus string

const (
	ReleaseStatusFilterApproved ReleaseFilterStatus = "FILTER_APPROVED"
	ReleaseStatusFilterPending  ReleaseFilterStatus = "PENDING"

	//ReleaseStatusFilterRejected ReleaseFilterStatus = "FILTER_REJECTED"
)

type ReleaseProtocol string

const (
	ReleaseProtocolTorrent ReleaseProtocol = "torrent"
)

type ReleaseImplementation string

const (
	ReleaseImplementationIRC     ReleaseImplementation = "IRC"
	ReleaseImplementationTorznab ReleaseImplementation = "TORZNAB"
)

type ReleaseQueryParams struct {
	Limit   uint64
	Offset  uint64
	Cursor  uint64
	Sort    map[string]string
	Filters struct {
		Indexers   []string
		PushStatus string
	}
	Search string
}

func NewRelease(indexer string) *Release {
	r := &Release{
		Indexer:        indexer,
		FilterStatus:   ReleaseStatusFilterPending,
		Rejections:     []string{},
		Protocol:       ReleaseProtocolTorrent,
		Implementation: ReleaseImplementationIRC,
		Timestamp:      time.Now(),
		Tags:           []string{},
	}

	return r
}

func (r *Release) ParseString(title string) {
	rel := rls.ParseString(title)

	r.TorrentName = title
	r.Title = rel.Title
	r.Source = rel.Source
	r.Resolution = rel.Resolution
	r.Season = rel.Series
	r.Episode = rel.Episode
	r.Region = rel.Region
	r.Audio = rel.Audio
	r.AudioChannels = rel.Channels
	r.Codec = rel.Codec
	r.Container = rel.Container
	r.HDR = rel.HDR
	r.Other = rel.Other
	r.Artists = rel.Artist

	if r.Year == 0 {
		r.Year = rel.Year
	}

	if r.Group == "" {
		r.Group = rel.Group
	}

	r.ParseReleaseTagsString(r.ReleaseTags)

	return
}

func (r *Release) ParseReleaseTagsString(tags string) {
	// trim delimiters and closest space
	re := regexp.MustCompile(`\| |/ |, `)
	cleanTags := re.ReplaceAllString(tags, "")

	t := ParseReleaseTagString(cleanTags)

	if len(t.Audio) > 0 {
		r.Audio = append(r.Audio, t.Audio...)
	}
	if len(t.Bonus) > 0 {
		if sliceContainsSlice([]string{"Freeleech"}, t.Bonus) {
			r.Freeleech = true
		}
		// TODO handle percent and other types

		r.Bonus = append(r.Bonus, t.Bonus...)
	}
	if len(t.Codec) > 0 {
		r.Codec = append(r.Codec, t.Codec)
	}
	if len(t.Other) > 0 {
		r.Other = append(r.Other, t.Other...)
	}

	if r.Container == "" && t.Container != "" {
		r.Container = t.Container
	}
	if r.Resolution == "" && t.Resolution != "" {
		r.Resolution = t.Resolution
	}
	if r.Source == "" && t.Source != "" {
		r.Source = t.Source
	}
	if r.AudioChannels == "" && t.Channels != "" {
		r.AudioChannels = t.Channels
	}

	return
}

func (r *Release) ParseSizeBytesString(size string) {
	s, err := humanize.ParseBytes(size)
	if err != nil {
		// log could not parse into bytes
		r.Size = 0
	}
	r.Size = s
}

func (r *Release) DownloadTorrentFile() error {
	if r.TorrentURL == "" {
		return errors.New("download_file: url can't be empty")
	} else if r.TorrentTmpFile != "" {
		// already downloaded
		return nil
	}

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return errors.Wrap(err, "could not create cookiejar")
	}

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{
		Transport: customTransport,
		Jar:       jar,
	}

	req, err := http.NewRequest("GET", r.TorrentURL, nil)
	if err != nil {
		return errors.Wrap(err, "error downloading file")
	}

	if r.RawCookie != "" {
		// set the cookie on the header instead of req.AddCookie
		// since we have a raw cookie like "uid=10; pass=000"
		req.Header.Set("Cookie", r.RawCookie)
	}

	// Get the data
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "error downloading file")
	}
	defer resp.Body.Close()

	// retry logic

	if resp.StatusCode != http.StatusOK {
		return errors.New("error downloading torrent (%v) file (%v) from '%v' - status code: %d", r.TorrentName, r.TorrentURL, r.Indexer, resp.StatusCode)
	}

	// Create tmp file
	tmpFile, err := os.CreateTemp("", "autobrr-")
	if err != nil {
		return errors.Wrap(err, "error creating tmp file")
	}
	defer tmpFile.Close()

	// Write the body to file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return errors.Wrap(err, "error writing downloaded file: %v", tmpFile.Name())
	}

	meta, err := metainfo.LoadFromFile(tmpFile.Name())
	if err != nil {
		return errors.Wrap(err, "metainfo could not load file contents: %v", tmpFile.Name())
	}

	torrentMetaInfo, err := meta.UnmarshalInfo()
	if err != nil {
		return errors.Wrap(err, "metainfo could not unmarshal info from torrent: %v", tmpFile.Name())
	}

	r.TorrentTmpFile = tmpFile.Name()
	r.TorrentHash = meta.HashInfoBytes().String()
	r.Size = uint64(torrentMetaInfo.TotalLength())

	// remove file if fail

	return nil
}

func (r *Release) addRejection(reason string) {
	r.Rejections = append(r.Rejections, reason)
}

func (r *Release) addRejectionF(format string, v ...interface{}) {
	r.Rejections = append(r.Rejections, fmt.Sprintf(format, v...))
}

// ResetRejections reset rejections between filter checks
func (r *Release) resetRejections() {
	r.Rejections = []string{}
}

func (r *Release) RejectionsString() string {
	if len(r.Rejections) > 0 {
		return strings.Join(r.Rejections, ", ")
	}
	return ""
}

// MapVars better name
func (r *Release) MapVars(def *IndexerDefinition, varMap map[string]string) error {

	if torrentName, err := getStringMapValue(varMap, "torrentName"); err != nil {
		return errors.Wrap(err, "failed parsing required field")
	} else {
		r.TorrentName = html.UnescapeString(torrentName)
	}

	if torrentID, err := getStringMapValue(varMap, "torrentId"); err == nil {
		r.TorrentID = torrentID
	}

	if category, err := getStringMapValue(varMap, "category"); err == nil {
		r.Category = category
	}

	if freeleech, err := getStringMapValue(varMap, "freeleech"); err == nil {
		fl := strings.EqualFold(freeleech, "freeleech") || strings.EqualFold(freeleech, "yes") || strings.EqualFold(freeleech, "1")
		if fl {
			r.Freeleech = true
			r.Bonus = append(r.Bonus, "Freeleech")
		}
	}

	if freeleechPercent, err := getStringMapValue(varMap, "freeleechPercent"); err == nil {
		// remove % and trim spaces
		freeleechPercent = strings.Replace(freeleechPercent, "%", "", -1)
		freeleechPercent = strings.Trim(freeleechPercent, " ")

		freeleechPercentInt, err := strconv.Atoi(freeleechPercent)
		if err != nil {
			//log.Debug().Msgf("bad freeleechPercent var: %v", year)
		}

		r.Freeleech = true
		r.FreeleechPercent = freeleechPercentInt

		r.Bonus = append(r.Bonus, "Freeleech")

		switch freeleechPercentInt {
		case 25:
			r.Bonus = append(r.Bonus, "Freeleech25")
			break
		case 50:
			r.Bonus = append(r.Bonus, "Freeleech50")
			break
		case 75:
			r.Bonus = append(r.Bonus, "Freeleech75")
			break
		case 100:
			r.Bonus = append(r.Bonus, "Freeleech100")
			break
		}

	}

	if uploader, err := getStringMapValue(varMap, "uploader"); err == nil {
		r.Uploader = uploader
	}

	if torrentSize, err := getStringMapValue(varMap, "torrentSize"); err == nil {
		// handling for indexer who doesn't explicitly set which size unit is used like (AR)
		if def.Parse != nil && def.Parse.ForceSizeUnit != "" {
			torrentSize = fmt.Sprintf("%v %v", torrentSize, def.Parse.ForceSizeUnit)
		}

		size, err := humanize.ParseBytes(torrentSize)
		if err != nil {
			// log could not parse into bytes
		}
		r.Size = size
	}

	if scene, err := getStringMapValue(varMap, "scene"); err == nil {
		r.IsScene = strings.EqualFold(scene, "true") || strings.EqualFold(scene, "yes")
	}

	// set origin. P2P, SCENE, O-SCENE and Internal
	if origin, err := getStringMapValue(varMap, "origin"); err == nil {
		r.Origin = origin

		if r.IsScene {
			r.Origin = "SCENE"
		}
	}

	if yearVal, err := getStringMapValue(varMap, "year"); err == nil {
		year, err := strconv.Atoi(yearVal)
		if err != nil {
			//log.Debug().Msgf("bad year var: %v", year)
		}
		r.Year = year
	}

	if tags, err := getStringMapValue(varMap, "tags"); err == nil {
		tagArr := strings.Split(strings.ReplaceAll(tags, " ", ""), ",")
		r.Tags = tagArr
	}

	// handle releaseTags. Most of them are redundant but some are useful
	if releaseTags, err := getStringMapValue(varMap, "releaseTags"); err == nil {
		r.ReleaseTags = releaseTags
	}

	if resolution, err := getStringMapValue(varMap, "resolution"); err == nil {
		r.Resolution = resolution
	}

	if releaseGroup, err := getStringMapValue(varMap, "releaseGroup"); err == nil {
		r.Group = releaseGroup
	}

	return nil
}

func getStringMapValue(stringMap map[string]string, key string) (string, error) {
	lowerKey := strings.ToLower(key)

	// case-insensitive match
	for k, v := range stringMap {
		if strings.ToLower(k) == lowerKey {
			return v, nil
		}
	}

	return "", errors.New("key was not found in map: %q", lowerKey)
}

func SplitAny(s string, seps string) []string {
	splitter := func(r rune) bool {
		return strings.ContainsRune(seps, r)
	}
	return strings.FieldsFunc(s, splitter)
}
