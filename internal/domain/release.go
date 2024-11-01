// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/avast/retry-go"
	"github.com/dustin/go-humanize"
	"github.com/moistari/rls"
	"golang.org/x/net/publicsuffix"
)

type ReleaseRepo interface {
	Store(ctx context.Context, release *Release) error
	Find(ctx context.Context, params ReleaseQueryParams) (*FindReleasesResponse, error)
	Get(ctx context.Context, req *GetReleaseRequest) (*Release, error)
	GetIndexerOptions(ctx context.Context) ([]string, error)
	Stats(ctx context.Context) (*ReleaseStats, error)
	Delete(ctx context.Context, req *DeleteReleaseRequest) error
	CheckSmartEpisodeCanDownload(ctx context.Context, p *SmartEpisodeParams) (bool, error)
	UpdateBaseURL(ctx context.Context, indexer string, oldBaseURL, newBaseURL string) error

	GetActionStatus(ctx context.Context, req *GetReleaseActionStatusRequest) (*ReleaseActionStatus, error)
	StoreReleaseActionStatus(ctx context.Context, status *ReleaseActionStatus) error

	StoreDuplicateProfile(ctx context.Context, profile *DuplicateReleaseProfile) error
	FindDuplicateReleaseProfiles(ctx context.Context) ([]*DuplicateReleaseProfile, error)
	DeleteReleaseProfileDuplicate(ctx context.Context, id int64) error
	CheckIsDuplicateRelease(ctx context.Context, profile *DuplicateReleaseProfile, release *Release) (bool, error)
}

type Release struct {
	ID                          int64                 `json:"id"`
	FilterStatus                ReleaseFilterStatus   `json:"filter_status"`
	Rejections                  []string              `json:"rejections"`
	Indexer                     IndexerMinimal        `json:"indexer"`
	FilterName                  string                `json:"filter"`
	Protocol                    ReleaseProtocol       `json:"protocol"`
	Implementation              ReleaseImplementation `json:"implementation"` // irc, rss, api
	Timestamp                   time.Time             `json:"timestamp"`
	Type                        rls.Type              `json:"type"` // rls.Type
	InfoURL                     string                `json:"info_url"`
	DownloadURL                 string                `json:"download_url"`
	MagnetURI                   string                `json:"-"`
	GroupID                     string                `json:"group_id"`
	TorrentID                   string                `json:"torrent_id"`
	TorrentTmpFile              string                `json:"-"`
	TorrentDataRawBytes         []byte                `json:"-"`
	TorrentHash                 string                `json:"-"`
	TorrentName                 string                `json:"name"`            // full release name
	NormalizedHash              string                `json:"normalized_hash"` // normalized torrent name and md5 hashed
	Size                        uint64                `json:"size"`
	Title                       string                `json:"title"`     // Parsed title
	SubTitle                    string                `json:"sub_title"` // Parsed secondary title for shows e.g. episode name
	Description                 string                `json:"-"`
	Category                    string                `json:"category"`
	Categories                  []string              `json:"categories,omitempty"`
	Season                      int                   `json:"season"`
	Episode                     int                   `json:"episode"`
	Year                        int                   `json:"year"`
	Month                       int                   `json:"month"`
	Day                         int                   `json:"day"`
	Resolution                  string                `json:"resolution"`
	Source                      string                `json:"source"`
	Codec                       []string              `json:"codec"`
	Container                   string                `json:"container"`
	HDR                         []string              `json:"hdr"`
	Audio                       []string              `json:"-"`
	AudioChannels               string                `json:"-"`
	AudioFormat                 string                `json:"-"`
	Bitrate                     string                `json:"-"`
	Group                       string                `json:"group"`
	Region                      string                `json:"-"`
	Language                    []string              `json:"-"`
	Proper                      bool                  `json:"proper"`
	Repack                      bool                  `json:"repack"`
	Website                     string                `json:"website"`
	Hybrid                      bool                  `json:"hybrid"`
	Edition                     []string              `json:"edition"`
	Cut                         []string              `json:"cut"`
	MediaProcessing             string                `json:"media_processing"` // Remux, Encode, Untouched
	Artists                     string                `json:"-"`
	LogScore                    int                   `json:"-"`
	HasCue                      bool                  `json:"-"`
	HasLog                      bool                  `json:"-"`
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
	Seeders                     int                   `json:"-"`
	Leechers                    int                   `json:"-"`
	AdditionalSizeCheckRequired bool                  `json:"-"`
	FilterID                    int                   `json:"-"`
	Filter                      *Filter               `json:"-"`
	ActionStatus                []ReleaseActionStatus `json:"action_status"`
}

// Hash return md5 hashed normalized release name
func (r *Release) Hash() string {
	normalized := rls.MustNormalize(r.TorrentName)
	h := md5.Sum([]byte(normalized))
	str := hex.EncodeToString(h[:])
	return str
}

func (r *Release) Raw(s string) rls.Release {
	return rls.ParseString(s)
}

func (r *Release) ParseType(s string) {
	r.Type = rls.ParseType(s)
}

func (r *Release) IsTypeVideo() bool {
	return r.Type.Is(rls.Movie, rls.Series, rls.Episode)
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
	OlderThan       int
	Indexers        []string
	ReleaseStatuses []string
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

type FindReleasesResponse struct {
	Data       []*Release `json:"data"`
	TotalCount uint64     `json:"count"`
	NextCursor int64      `json:"next_cursor"`
}

type ReleaseActionRetryReq struct {
	ReleaseId      int
	ActionStatusId int
	ActionId       int
}

type ReleaseProcessReq struct {
	IndexerIdentifier     string   `json:"indexer_identifier"`
	IndexerImplementation string   `json:"indexer_implementation"`
	AnnounceLines         []string `json:"announce_lines"`
}

type GetReleaseRequest struct {
	Id int
}

type GetReleaseActionStatusRequest struct {
	Id int
}

func NewRelease(indexer IndexerMinimal) *Release {
	r := &Release{
		Indexer:        indexer,
		FilterStatus:   ReleaseStatusFilterPending,
		Rejections:     []string{},
		Protocol:       ReleaseProtocolTorrent,
		Implementation: ReleaseImplementationIRC,
		Timestamp:      time.Now(),
		Tags:           []string{},
		Language:       []string{},
		Edition:        []string{},
		Cut:            []string{},
		Other:          []string{},
		Size:           0,
	}

	return r
}

func (r *Release) ParseString(title string) {
	rel := rls.ParseString(title)

	r.Type = rel.Type

	r.TorrentName = title
	r.NormalizedHash = r.Hash()

	r.Source = rel.Source
	r.Resolution = rel.Resolution
	r.Region = rel.Region

	if rel.Language != nil {
		r.Language = rel.Language
	}

	r.Audio = rel.Audio
	r.AudioChannels = rel.Channels
	r.Codec = rel.Codec
	r.Container = rel.Container
	r.HDR = rel.HDR
	r.Artists = rel.Artist

	if rel.Other != nil {
		r.Other = rel.Other
	}

	r.Proper = slices.Contains(r.Other, "PROPER")
	r.Repack = slices.Contains(r.Other, "REPACK") || slices.Contains(r.Other, "REREPACK")
	r.Hybrid = slices.Contains(r.Other, "HYBRiD")

	// TODO default to Encode and set Untouched for discs
	if slices.Contains(r.Other, "REMUX") {
		r.MediaProcessing = "REMUX"
	}

	if r.Title == "" {
		r.Title = rel.Title
	}
	r.SubTitle = rel.Subtitle

	if r.Season == 0 {
		r.Season = rel.Series
	}
	if r.Episode == 0 {
		r.Episode = rel.Episode
	}

	if r.Year == 0 {
		r.Year = rel.Year
	}
	if r.Month == 0 {
		r.Month = rel.Month
	}
	if r.Day == 0 {
		r.Day = rel.Day
	}

	if r.Group == "" {
		r.Group = rel.Group
	}

	if r.Website == "" {
		r.Website = rel.Collection
	}

	if rel.Cut != nil {
		r.Cut = rel.Cut
	}

	if rel.Edition != nil {
		r.Edition = rel.Edition
	}

	r.ParseReleaseTagsString(r.ReleaseTags)
}

func (r *Release) ParseReleaseTagsString(tags string) {
	if tags == "" {
		return
	}

	cleanTags := CleanReleaseTags(tags)
	t := ParseReleaseTagString(cleanTags)

	if len(t.Audio) > 0 {
		//r.Audio = getUniqueTags(r.Audio, t.Audio)
		r.Audio = t.Audio
	}

	if t.AudioBitrate != "" {
		r.Bitrate = t.AudioBitrate
	}

	if t.AudioFormat != "" {
		r.AudioFormat = t.AudioFormat
	}

	if r.AudioChannels == "" && t.Channels != "" {
		r.AudioChannels = t.Channels
	}

	if t.HasLog {
		r.HasLog = true

		if t.LogScore > 0 {
			r.LogScore = t.LogScore
		}
	}

	if t.HasCue {
		r.HasCue = true
	}

	if len(t.Bonus) > 0 {
		if sliceContainsSlice([]string{"Freeleech", "Freeleech!"}, t.Bonus) {
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
}

// ParseSizeBytesString If there are parsing errors, then it keeps the original (or default size 0)
// Otherwise, it will update the size only if the new size is bigger than the previous one.
func (r *Release) ParseSizeBytesString(size string) {
	s, err := humanize.ParseBytes(size)
	if err == nil && s > r.Size {
		r.Size = s
	}
}

func (r *Release) OpenTorrentFile() error {
	tmpFile, err := os.ReadFile(r.TorrentTmpFile)
	if err != nil {
		return errors.Wrap(err, "could not read torrent file: %v", r.TorrentTmpFile)
	}

	r.TorrentDataRawBytes = tmpFile

	return nil
}

// AudioString takes r.Audio and r.AudioChannels and returns a string like "DDP Atmos 5.1"
func (r *Release) AudioString() string {
	var audio []string

	audio = append(audio, r.Audio...)
	audio = append(audio, r.AudioChannels)

	if len(audio) > 0 {
		return strings.Join(audio, " ")
	}

	return ""
}

func (r *Release) DownloadTorrentFileCtx(ctx context.Context) error {
	return r.downloadTorrentFile(ctx)
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.DownloadURL, nil)
	if err != nil {
		return errors.Wrap(err, "error downloading file")
	}

	req.Header.Set("User-Agent", "autobrr")

	client := http.Client{
		Timeout:   time.Second * 60,
		Transport: sharedhttp.TransportTLSInsecure,
	}

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

	tmpFilePattern := "autobrr-"
	tmpDir := os.TempDir()

	// Create tmp file
	tmpFile, err := os.CreateTemp(tmpDir, tmpFilePattern)
	if err != nil {
		// inverse the err check to make it a bit cleaner
		if !errors.Is(err, os.ErrNotExist) {
			return errors.Wrap(err, "error creating tmp file")
		}

		if mkdirErr := os.MkdirAll(tmpDir, os.ModePerm); mkdirErr != nil {
			return errors.Wrap(mkdirErr, "could not create TMP dir: %s", tmpDir)
		}

		tmpFile, err = os.CreateTemp(tmpDir, tmpFilePattern)
		if err != nil {
			return errors.Wrap(err, "error creating tmp file in: %s", tmpDir)
		}
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
		//	return retry.Unrecoverable(errors.New("redirect encountered for torrent (%s) file (%s) - status code: %d - check indexer keys for %s", r.TorrentName, r.DownloadURL, resp.StatusCode, r.Indexer.Name))

		case http.StatusUnauthorized, http.StatusForbidden:
			return retry.Unrecoverable(errors.New("unrecoverable error downloading torrent (%s) file (%s) - status code: %d - check indexer keys for %s", r.TorrentName, r.DownloadURL, resp.StatusCode, r.Indexer.Name))

		case http.StatusMethodNotAllowed:
			return retry.Unrecoverable(errors.New("unrecoverable error downloading torrent (%s) file (%s) from '%s' - status code: %d. Check if the request method is correct", r.TorrentName, r.DownloadURL, r.Indexer.Name, resp.StatusCode))
		case http.StatusNotFound:
			return errors.New("torrent %s not found on %s (%d) - retrying", r.TorrentName, r.Indexer.Name, resp.StatusCode)

		case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return errors.New("server error (%d) encountered while downloading torrent (%s) file (%s) from '%s' - retrying", resp.StatusCode, r.TorrentName, r.DownloadURL, r.Indexer.Name)

		case http.StatusInternalServerError:
			return errors.New("server error (%d) encountered while downloading torrent (%s) file (%s) - check indexer keys for %s", resp.StatusCode, r.TorrentName, r.DownloadURL, r.Indexer.Name)

		default:
			return retry.Unrecoverable(errors.New("unexpected status code %d: check indexer keys for %s", resp.StatusCode, r.Indexer.Name))
		}

		resetTmpFile := func() {
			tmpFile.Seek(0, io.SeekStart)
			tmpFile.Truncate(0)
		}

		// Read the body into bytes
		bodyBytes, err := io.ReadAll(bufio.NewReader(resp.Body))
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
				return errors.Wrap(err, "metainfo unexpected content type, got HTML expected a bencoded torrent. check indexer keys for %s - %s", r.Indexer.Name, r.TorrentName)
			}

			return retry.Unrecoverable(errors.Wrap(err, "metainfo unexpected content type. check indexer keys for %s - %s", r.Indexer.Name, r.TorrentName))
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
	if r.TorrentTmpFile == "" {
		return
	}

	os.Remove(r.TorrentTmpFile)
	r.TorrentTmpFile = ""
}

// HasMagnetUri check uf MagnetURI is set and valid or empty
func (r *Release) HasMagnetUri() bool {
	if r.MagnetURI != "" && strings.HasPrefix(r.MagnetURI, MagnetURIPrefix) {
		return true
	}
	return false
}

const MagnetURIPrefix = "magnet:?"

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
		fl := StringEqualFoldMulti(freeleech, "1", "free", "freeleech", "freeleech!", "yes", "VIP")
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

type DuplicateReleaseProfile struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Protocol     bool   `json:"protocol"`
	ReleaseName  bool   `json:"release_name"`
	Exact        bool   `json:"exact"`
	Title        bool   `json:"title"`
	SubTitle     bool   `json:"sub_title"`
	Year         bool   `json:"year"`
	Month        bool   `json:"month"`
	Day          bool   `json:"day"`
	Source       bool   `json:"source"`
	Resolution   bool   `json:"resolution"`
	Codec        bool   `json:"codec"`
	Container    bool   `json:"container"`
	DynamicRange bool   `json:"dynamic_range"`
	Audio        bool   `json:"audio"`
	Group        bool   `json:"group"`
	Season       bool   `json:"season"`
	Episode      bool   `json:"episode"`
	Website      bool   `json:"website"`
	Proper       bool   `json:"proper"`
	Repack       bool   `json:"repack"`
	Edition      bool   `json:"edition"`
	Language     bool   `json:"language"`
}
