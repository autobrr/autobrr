// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"bytes"
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

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/avast/retry-go"
	"github.com/dustin/go-humanize"
	"github.com/moistari/rls"
	"golang.org/x/net/publicsuffix"
)

type ReleaseRepo interface {
	Store(ctx context.Context, release *Release) error
	Find(ctx context.Context, params ReleaseQueryParams) (res []*Release, nextCursor int64, count int64, err error)
	FindRecent(ctx context.Context) ([]*Release, error)
	Get(ctx context.Context, req *GetReleaseRequest) (*Release, error)
	GetIndexerOptions(ctx context.Context) ([]string, error)
	Stats(ctx context.Context) (*ReleaseStats, error)
	Delete(ctx context.Context, req *DeleteReleaseRequest) error
	CanDownloadShow(ctx context.Context, title string, season int, episode int) (bool, error)

	GetActionStatus(ctx context.Context, req *GetReleaseActionStatusRequest) (*ReleaseActionStatus, error)
	StoreReleaseActionStatus(ctx context.Context, status *ReleaseActionStatus) error
}

type Release struct {
	ID                             int64                 `json:"id"`
	FilterStatus                   ReleaseFilterStatus   `json:"filter_status"`
	Rejections                     []string              `json:"rejections"`
	Indexer                        string                `json:"indexer"`
	FilterName                     string                `json:"filter"`
	Protocol                       ReleaseProtocol       `json:"protocol"`
	Implementation                 ReleaseImplementation `json:"implementation"` // irc, rss, api
	Timestamp                      time.Time             `json:"timestamp"`
	InfoURL                        string                `json:"info_url"`
	DownloadURL                    string                `json:"download_url"`
	MagnetURI                      string                `json:"-"`
	GroupID                        string                `json:"group_id"`
	TorrentID                      string                `json:"torrent_id"`
	TorrentTmpFile                 string                `json:"-"`
	TorrentDataRawBytes            []byte                `json:"-"`
	TorrentHash                    string                `json:"-"`
	TorrentName                    string                `json:"torrent_name"` // full release name
	Size                           uint64                `json:"size"`
	Title                          string                `json:"title"` // Parsed title
	Description                    string                `json:"-"`
	RecordLabel                    string                `json:"record_label"`
	Category                       string                `json:"category"`
	Categories                     []string              `json:"categories,omitempty"`
	Season                         int                   `json:"season"`
	Episode                        int                   `json:"episode"`
	Year                           int                   `json:"year"`
	Resolution                     string                `json:"resolution"`
	Source                         string                `json:"source"`
	Codec                          []string              `json:"codec"`
	Container                      string                `json:"container"`
	HDR                            []string              `json:"hdr"`
	Audio                          []string              `json:"-"`
	AudioChannels                  string                `json:"-"`
	Group                          string                `json:"group"`
	Region                         string                `json:"-"`
	Language                       []string              `json:"-"`
	Proper                         bool                  `json:"proper"`
	Repack                         bool                  `json:"repack"`
	Website                        string                `json:"website"`
	Artists                        string                `json:"-"`
	Type                           string                `json:"type"` // Album,Single,EP
	LogScore                       int                   `json:"-"`
	Origin                         string                `json:"origin"` // P2P, Internal
	Tags                           []string              `json:"-"`
	ReleaseTags                    string                `json:"-"`
	Freeleech                      bool                  `json:"-"`
	FreeleechPercent               int                   `json:"-"`
	Bonus                          []string              `json:"-"`
	Uploader                       string                `json:"uploader"`
	PreTime                        string                `json:"pre_time"`
	Other                          []string              `json:"-"`
	RawCookie                      string                `json:"-"`
	AdditionalDetailsCheckRequired bool                  `json:"-"`
	FilterID                       int                   `json:"-"`
	Filter                         *Filter               `json:"-"`
	ActionStatus                   []ReleaseActionStatus `json:"action_status"`
}

type ReleaseActionStatus struct {
	ID         int64             `json:"id"`
	Status     ReleasePushStatus `json:"status"`
	Action     string            `json:"action"`
	ActionID   int64             `json:"action_id"`
	Type       ActionType        `json:"type"`
	Client     string            `json:"client"`
	Filter     string            `json:"filter"`
	FilterID   int64             `json:"filter_id"`
	Rejections []string          `json:"rejections"`
	ReleaseID  int64             `json:"release_id"`
	Timestamp  time.Time         `json:"timestamp"`
}

type DeleteReleaseRequest struct {
	OlderThan int
}

func NewReleaseActionStatus(action *Action, release *Release) *ReleaseActionStatus {
	s := &ReleaseActionStatus{
		ID:         0,
		Status:     ReleasePushStatusPending,
		Action:     action.Name,
		ActionID:   int64(action.ID),
		Type:       action.Type,
		Filter:     release.FilterName,
		FilterID:   int64(release.FilterID),
		Rejections: []string{},
		Timestamp:  time.Now(),
		ReleaseID:  release.ID,
	}

	if action.Client != nil {
		s.Client = action.Client.Name
	}

	return s
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
	PushErrorCount      int64 `json:"push_error_count"`
}

type ReleasePushStatus string

const (
	ReleasePushStatusPending  ReleasePushStatus = "PENDING" // Initial status
	ReleasePushStatusApproved ReleasePushStatus = "PUSH_APPROVED"
	ReleasePushStatusRejected ReleasePushStatus = "PUSH_REJECTED"
	ReleasePushStatusErr      ReleasePushStatus = "PUSH_ERROR"
)

func (r ReleasePushStatus) String() string {
	switch r {
	case ReleasePushStatusPending:
		return "Pending"
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

func ValidReleasePushStatus(s string) bool {
	switch s {
	case string(ReleasePushStatusPending):
		return true
	case string(ReleasePushStatusApproved):
		return true
	case string(ReleasePushStatusRejected):
		return true
	case string(ReleasePushStatusErr):
		return true
	default:
		return false
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
	ReleaseProtocolNzb     ReleaseProtocol = "usenet"
)

func (r ReleaseProtocol) String() string {
	switch r {
	case ReleaseProtocolTorrent:
		return "torrent"
	case ReleaseProtocolNzb:
		return "usenet"
	default:
		return "torrent"
	}
}

type ReleaseImplementation string

const (
	ReleaseImplementationIRC     ReleaseImplementation = "IRC"
	ReleaseImplementationTorznab ReleaseImplementation = "TORZNAB"
	ReleaseImplementationNewznab ReleaseImplementation = "NEWZNAB"
	ReleaseImplementationRSS     ReleaseImplementation = "RSS"
)

func (r ReleaseImplementation) String() string {
	switch r {
	case ReleaseImplementationIRC:
		return "IRC"
	case ReleaseImplementationTorznab:
		return "TORZNAB"
	case ReleaseImplementationNewznab:
		return "NEWZNAB"
	case ReleaseImplementationRSS:
		return "RSS"
	default:
		return "IRC"
	}
}

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

type ReleaseActionRetryReq struct {
	ReleaseId      int
	ActionStatusId int
	ActionId       int
}

type GetReleaseRequest struct {
	Id int
}

type GetReleaseActionStatusRequest struct {
	Id int
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
		Size:           0,
	}

	return r
}

func (r *Release) ParseString(title string) {
	rel := rls.ParseString(title)

	r.TorrentName = title
	r.Source = rel.Source
	r.Resolution = rel.Resolution
	r.Region = rel.Region
	r.Audio = rel.Audio
	r.AudioChannels = rel.Channels
	r.Codec = rel.Codec
	r.Container = rel.Container
	r.HDR = rel.HDR
	r.Other = rel.Other
	r.Artists = rel.Artist
	r.Language = rel.Language

	if r.Title == "" {
		r.Title = rel.Title
	}

	if r.Season == 0 {
		r.Season = rel.Series
	}
	if r.Episode == 0 {
		r.Episode = rel.Episode
	}

	if r.Year == 0 {
		r.Year = rel.Year
	}

	if r.Group == "" {
		r.Group = rel.Group
	}

	r.ParseReleaseTagsString(r.ReleaseTags)
}

var ErrUnrecoverableError = errors.New("unrecoverable error")

func (r *Release) ParseReleaseTagsString(tags string) {
	// trim delimiters and closest space
	re := regexp.MustCompile(`\| |/ |, `)
	cleanTags := re.ReplaceAllString(tags, "")

	t := ParseReleaseTagString(cleanTags)

	if len(t.Audio) > 0 {
		r.Audio = getUniqueTags(r.Audio, t.Audio)
	}

	if len(t.Bonus) > 0 {
		if sliceContainsSlice([]string{"Freeleech"}, t.Bonus) {
			r.Freeleech = true
		}
		// TODO handle percent and other types

		r.Bonus = append(r.Bonus, t.Bonus...)
	}
	if len(t.Codec) > 0 {
		r.Codec = getUniqueTags(r.Codec, append(make([]string, 0, 1), t.Codec))
	}
	if len(t.Other) > 0 {
		r.Other = getUniqueTags(r.Other, t.Other)
	}
	if r.Origin == "" && t.Origin != "" {
		r.Origin = t.Origin
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
}

// ParseSizeBytesString If there are parsing errors, then it keeps the original (or default size 0)
// Otherwise, it will update the size only if the new size is bigger than the previous one.
func (r *Release) ParseSizeBytesString(size string) {
	s, err := humanize.ParseBytes(size)
	if err == nil && s > r.Size {
		r.Size = s
	}
}

func (r *Release) DownloadTorrentFileCtx(ctx context.Context) error {
	return r.downloadTorrentFile(ctx)
}

func (r *Release) DownloadTorrentFile() error {
	return r.downloadTorrentFile(context.Background())
}

func (r *Release) downloadTorrentFile(ctx context.Context) error {
	if r.HasMagnetUri() {
		return errors.New("downloading magnet links is not supported: %s", r.MagnetURI)
	} else if r.Protocol != ReleaseProtocolTorrent {
		return errors.New("could not download file: protocol %s is not supported", r.Protocol)
	}

	if r.DownloadURL == "" {
		return errors.New("download_file: url can't be empty")
	} else if r.TorrentTmpFile != "" {
		// already downloaded
		return nil
	}

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{
		Transport: customTransport,
		Timeout:   time.Second * 45,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.DownloadURL, nil)
	if err != nil {
		return errors.Wrap(err, "error downloading file")
	}

	req.Header.Set("User-Agent", "autobrr")

	if r.RawCookie != "" {
		jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		if err != nil {
			return errors.Wrap(err, "could not create cookiejar")
		}
		client.Jar = jar

		// set the cookie on the header instead of req.AddCookie
		// since we have a raw cookie like "uid=10; pass=000"
		req.Header.Set("Cookie", r.RawCookie)
	}

	// Create tmp file
	tmpFile, err := os.CreateTemp("", "autobrr-")
	if err != nil {
		return errors.Wrap(err, "error creating tmp file")
	}
	defer tmpFile.Close()

	errFunc := retry.Do(func() error {
		// Get the data
		resp, err := client.Do(req)
		if err != nil {
			return errors.Wrap(err, "error downloading file")
		}
		defer resp.Body.Close()

		// Check server response
		switch resp.StatusCode {
		case http.StatusOK:
			// Continue processing the response
		//case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		//	// Handle redirect
		//	return retry.Unrecoverable(errors.New("redirect encountered for torrent (%s) file (%s) - status code: %d - check indexer keys for %s", r.TorrentName, r.DownloadURL, resp.StatusCode, r.Indexer))

		case http.StatusUnauthorized, http.StatusForbidden:
			return retry.Unrecoverable(errors.New("unrecoverable error downloading torrent (%s) file (%s) - status code: %d - check indexer keys for %s", r.TorrentName, r.DownloadURL, resp.StatusCode, r.Indexer))

		case http.StatusMethodNotAllowed:
			return retry.Unrecoverable(errors.New("unrecoverable error downloading torrent (%s) file (%s) from '%s' - status code: %d. Check if the request method is correct", r.TorrentName, r.DownloadURL, r.Indexer, resp.StatusCode))

		case http.StatusNotFound:
			return errors.New("torrent %s not found on %s (%d) - retrying", r.TorrentName, r.Indexer, resp.StatusCode)

		case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return errors.New("server error (%d) encountered while downloading torrent (%s) file (%s) from '%s' - retrying", resp.StatusCode, r.TorrentName, r.DownloadURL, r.Indexer)

		case http.StatusInternalServerError:
			return errors.New("server error (%d) encountered while downloading torrent (%s) file (%s) - check indexer keys for %s", resp.StatusCode, r.TorrentName, r.DownloadURL, r.Indexer)

		default:
			return retry.Unrecoverable(errors.New("unexpected status code %d: check indexer keys for %s", resp.StatusCode, r.Indexer))
		}

		resetTmpFile := func() {
			tmpFile.Seek(0, io.SeekStart)
			tmpFile.Truncate(0)
		}

		// Read the body into bytes
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "error reading response body")
		}

		// Create a new reader for bodyBytes
		bodyReader := bytes.NewReader(bodyBytes)

		// Try to decode as torrent file
		meta, err := metainfo.Load(bodyReader)
		if err != nil {
			resetTmpFile()

			// explicitly check for unexpected content type that match html
			var bse *bencode.SyntaxError
			if errors.As(err, &bse) {
				// regular error so we can retry if we receive html first run
				return errors.Wrap(err, "metainfo unexpected content type, got HTML expected a bencoded torrent. check indexer keys for %s - %s", r.Indexer, r.TorrentName)
			}

			return retry.Unrecoverable(errors.Wrap(err, "metainfo unexpected content type. check indexer keys for %s - %s", r.Indexer, r.TorrentName))
		}

		// Write the body to file
		if _, err := tmpFile.Write(bodyBytes); err != nil {
			resetTmpFile()
			return errors.Wrap(err, "error writing downloaded file: %s", tmpFile.Name())
		}

		torrentMetaInfo, err := meta.UnmarshalInfo()
		if err != nil {
			resetTmpFile()
			return retry.Unrecoverable(errors.Wrap(err, "metainfo could not unmarshal info from torrent: %s", tmpFile.Name()))
		}

		hashInfoBytes := meta.HashInfoBytes().Bytes()
		if len(hashInfoBytes) < 1 {
			resetTmpFile()
			return retry.Unrecoverable(errors.New("could not read infohash"))
		}

		r.TorrentTmpFile = tmpFile.Name()
		r.TorrentHash = meta.HashInfoBytes().String()
		r.Size = uint64(torrentMetaInfo.TotalLength())

		return nil
	},
		retry.Delay(time.Second*3),
		retry.Attempts(3),
		retry.MaxJitter(time.Second*1),
	)

	return errFunc
}

func (r *Release) CleanupTemporaryFiles() {
	if len(r.TorrentTmpFile) == 0 {
		return
	}

	os.Remove(r.TorrentTmpFile)
	r.TorrentTmpFile = ""
}

// HasMagnetUri check uf MagnetURI is set or empty
func (r *Release) HasMagnetUri() bool {
	return r.MagnetURI != ""
}

type magnetRoundTripper struct{}

func (rt *magnetRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Scheme == "magnet" {
		responseBody := r.URL.String()
		respReader := io.NopCloser(strings.NewReader(responseBody))

		resp := &http.Response{
			Status:        http.StatusText(http.StatusOK),
			StatusCode:    http.StatusOK,
			Body:          respReader,
			ContentLength: int64(len(responseBody)),
			Header: map[string][]string{
				"Content-Type": {"text/plain"},
				"Location":     {responseBody},
			},
			Proto:      "HTTP/2.0",
			ProtoMajor: 2,
		}

		return resp, nil
	}

	return http.DefaultTransport.RoundTrip(r)
}

func (r *Release) ResolveMagnetUri(ctx context.Context) error {
	if r.MagnetURI == "" {
		return nil
	} else if strings.HasPrefix(r.MagnetURI, "magnet:?") {
		return nil
	}

	client := http.Client{
		Transport: &magnetRoundTripper{},
		Timeout:   time.Second * 60,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.MagnetURI, nil)
	if err != nil {
		return errors.Wrap(err, "could not build request to resolve magnet uri")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")

	res, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "could not make request to resolve magnet uri")
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.New("unexpected status code: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return errors.Wrap(err, "could not read response body")
	}

	magnet := string(body)
	if magnet != "" {
		r.MagnetURI = magnet
	}

	return nil
}

func (r *Release) addRejection(reason string) {
	r.Rejections = append(r.Rejections, reason)
}

func (r *Release) AddRejectionF(format string, v ...interface{}) {
	r.addRejectionF(format, v...)
}

func (r *Release) addRejectionF(format string, v ...interface{}) {
	r.Rejections = append(r.Rejections, fmt.Sprintf(format, v...))
}

// ResetRejections reset rejections between filter checks
func (r *Release) resetRejections() {
	r.Rejections = []string{}
}

func (r *Release) RejectionsString(trim bool) string {
	if len(r.Rejections) > 0 {
		out := strings.Join(r.Rejections, ", ")
		if trim && len(out) > 1024 {
			out = out[:1024]
		}

		return out
	}
	return ""
}

// MapVars map vars from regex captures to fields on release
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
		fl := StringEqualFoldMulti(freeleech, "freeleech", "freeleech!", "yes", "1", "VIP")
		if fl {
			r.Freeleech = true
			// default to 100 and override if freeleechPercent is present in next function
			r.FreeleechPercent = 100
			r.Bonus = append(r.Bonus, "Freeleech")
		}
	}

	if freeleechPercent, err := getStringMapValue(varMap, "freeleechPercent"); err == nil {
		// special handling for BHD to map their freeleech into percent
		if def.Identifier == "beyondhd" {
			if freeleechPercent == "Capped FL" {
				freeleechPercent = "100%"
			} else if strings.Contains(freeleechPercent, "% FL") {
				freeleechPercent = strings.Replace(freeleechPercent, " FL", "", -1)
			}
		}

		// remove % and trim spaces
		freeleechPercent = strings.Replace(freeleechPercent, "%", "", -1)
		freeleechPercent = strings.Trim(freeleechPercent, " ")

		freeleechPercentInt, err := strconv.Atoi(freeleechPercent)
		if err != nil {
			//log.Debug().Msgf("bad freeleechPercent var: %v", year)
		}

		if freeleechPercentInt > 0 {
			r.Freeleech = true
			r.FreeleechPercent = freeleechPercentInt

			r.Bonus = append(r.Bonus, "Freeleech")

			switch freeleechPercentInt {
			case 25:
				r.Bonus = append(r.Bonus, "Freeleech25")
			case 50:
				r.Bonus = append(r.Bonus, "Freeleech50")
			case 75:
				r.Bonus = append(r.Bonus, "Freeleech75")
			case 100:
				r.Bonus = append(r.Bonus, "Freeleech100")
			}
		}
	}

	if uploader, err := getStringMapValue(varMap, "uploader"); err == nil {
		r.Uploader = uploader
	}

	if record_label, err := getStringMapValue(varMap, "record_label"); err == nil {
		r.RecordLabel = record_label
	}

	if torrentSize, err := getStringMapValue(varMap, "torrentSize"); err == nil {
		// handling for indexer who doesn't explicitly set which size unit is used like (AR)
		if def.IRC != nil && def.IRC.Parse != nil && def.IRC.Parse.ForceSizeUnit != "" {
			torrentSize = fmt.Sprintf("%s %s", torrentSize, def.IRC.Parse.ForceSizeUnit)
		}

		size, err := humanize.ParseBytes(torrentSize)
		if err != nil {
			// log could not parse into bytes
		}
		r.Size = size
	}

	if scene, err := getStringMapValue(varMap, "scene"); err == nil {
		if StringEqualFoldMulti(scene, "true", "yes", "1") {
			r.Origin = "SCENE"
		}
	}

	// set origin. P2P, SCENE, O-SCENE and Internal
	if origin, err := getStringMapValue(varMap, "origin"); err == nil {
		r.Origin = origin
	}

	if internal, err := getStringMapValue(varMap, "internal"); err == nil {
		if StringEqualFoldMulti(internal, "internal", "yes", "1") {
			r.Origin = "INTERNAL"
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
		tagsArr := []string{}
		s := strings.Split(tags, ",")
		for _, t := range s {
			tagsArr = append(tagsArr, strings.Trim(t, " "))
		}
		r.Tags = tagsArr
	}

	if title, err := getStringMapValue(varMap, "title"); err == nil {
		r.Title = title
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

	if episodeVal, err := getStringMapValue(varMap, "releaseEpisode"); err == nil {
		episode, _ := strconv.Atoi(episodeVal)
		r.Episode = episode
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

func StringEqualFoldMulti(s string, values ...string) bool {
	for _, value := range values {
		if strings.EqualFold(s, value) {
			return true
		}
	}
	return false
}

func getUniqueTags(target []string, source []string) []string {
	toAppend := make([]string, 0, len(source))

	for _, t := range source {
		found := false
		norm := rls.MustNormalize(t)

		for _, s := range target {
			if rls.MustNormalize(s) == norm {
				found = true
				break
			}
		}

		if !found {
			toAppend = append(toAppend, t)
		}
	}

	target = append(target, toAppend...)

	return target
}
