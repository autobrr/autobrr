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

	"golang.org/x/net/publicsuffix"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/dustin/go-humanize"
	"github.com/moistari/rls"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type ReleaseRepo interface {
	Store(ctx context.Context, release *Release) (*Release, error)
	Find(ctx context.Context, params ReleaseQueryParams) (res []*Release, nextCursor int64, count int64, err error)
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
	Raw                         string                `json:"raw"`   // Raw release
	Clean                       string                `json:"clean"` // cleaned release name
	Title                       string                `json:"title"` // Parsed title
	Category                    string                `json:"category"`
	Season                      int                   `json:"season"`
	Episode                     int                   `json:"episode"`
	Year                        int                   `json:"year"`
	Resolution                  string                `json:"resolution"`
	Source                      string                `json:"source"` // CD, DVD, Vinyl, DAT, Cassette, WEB, Other
	Codec                       string                `json:"codec"`
	CodecArr                    []string              `json:"-"`
	Container                   string                `json:"container"`
	HDR                         string                `json:"hdr"`
	HDRArr                      []string              `json:"-"`
	Audio                       string                `json:"audio"`
	AudioArr                    []string              `json:"-"`
	AudioChannels               string                `json:"-"`
	Group                       string                `json:"group"`
	Region                      string                `json:"region"`
	Language                    string                `json:"language"`
	Edition                     string                `json:"edition"` // Extended, directors cut
	Unrated                     bool                  `json:"unrated"`
	Hybrid                      bool                  `json:"hybrid"`
	Proper                      bool                  `json:"proper"`
	Repack                      bool                  `json:"repack"`
	Website                     string                `json:"website"`
	ThreeD                      bool                  `json:"-"`
	Artists                     []string              `json:"artists"`
	Type                        string                `json:"type"`    // Album,Single,EP
	Format                      string                `json:"format"`  // music only
	Quality                     string                `json:"quality"` // quality
	LogScore                    int                   `json:"log_score"`
	HasLog                      bool                  `json:"has_log"`
	HasCue                      bool                  `json:"has_cue"`
	IsScene                     bool                  `json:"is_scene"`
	Origin                      string                `json:"origin"` // P2P, Internal
	Tags                        []string              `json:"tags"`
	ReleaseTags                 string                `json:"-"`
	ReleaseTagsArr              []string              `json:"-"`
	Freeleech                   bool                  `json:"freeleech"`
	FreeleechPercent            int                   `json:"freeleech_percent"`
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
	ReleasePushStatusPending  ReleasePushStatus = "PENDING" // Initial status
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
	ReleaseStatusFilterRejected ReleaseFilterStatus = "FILTER_REJECTED"
	ReleaseStatusFilterPending  ReleaseFilterStatus = "PENDING"
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

func NewRelease(indexer string, line string) (*Release, error) {
	r := &Release{
		Indexer:        indexer,
		Raw:            line,
		FilterStatus:   ReleaseStatusFilterPending,
		Rejections:     []string{},
		Protocol:       ReleaseProtocolTorrent,
		Implementation: ReleaseImplementationIRC,
		Timestamp:      time.Now(),
		Artists:        []string{},
		Tags:           []string{},
	}

	return r, nil
}

func (r *Release) ParseString(title string) error {
	rel := rls.ParseString(title)

	r.TorrentName = title
	r.Title = rel.Title
	r.Source = rel.Source
	r.Resolution = rel.Resolution
	r.Year = rel.Year
	r.Season = rel.Series
	r.Episode = rel.Episode
	r.Group = rel.Group
	r.Region = rel.Region
	//r.Language = rel.Language
	r.AudioArr = rel.Audio
	r.AudioChannels = rel.Channels
	r.CodecArr = rel.Codec
	r.Container = rel.Container
	r.HDRArr = rel.HDR
	r.Other = rel.Other
	r.Artists = []string{rel.Artist}

	r.ParseReleaseTagsString(r.ReleaseTags)

	return nil
}

func (r *Release) ParseReleaseTagsString(tags string) error {
	// trim delimiters and closest space
	re := regexp.MustCompile(`\| |\/ |, `)
	cleanTags := re.ReplaceAllString(tags, "")

	t := ParseReleaseTagString(cleanTags)

	r.AudioArr = append(r.AudioArr, t.Audio...)
	r.Bonus = append(r.Bonus, t.Bonus...)
	r.Other = append(r.Other, t.Other...)

	if r.Codec == "" && t.Codec != "" {
		r.Codec = t.Codec
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

	return nil
}

func (r *Release) Parse() error {
	var err error

	err = r.extractYear()
	err = r.extractSeason()
	err = r.extractEpisode()
	err = r.extractResolution()
	err = r.extractSource()
	err = r.extractCodec()
	err = r.extractContainer()
	err = r.extractHDR()
	err = r.extractAudio()
	err = r.extractGroup()
	err = r.extractRegion()
	err = r.extractLanguage()
	err = r.extractEdition()
	err = r.extractUnrated()
	err = r.extractHybrid()
	err = r.extractProper()
	err = r.extractRepack()
	err = r.extractWebsite()
	err = r.extractReleaseTags()

	r.Clean = cleanReleaseName(r.TorrentName)

	if err != nil {
		log.Trace().Msgf("could not parse release: %v", r.TorrentName)
		return err
	}

	return nil
}

func (r *Release) ParseSizeBytesString(size string) {
	s, err := humanize.ParseBytes(size)
	if err != nil {
		// log could not parse into bytes
		r.Size = 0
	}
	r.Size = s
}

func (r *Release) extractYear() error {
	if r.Year > 0 {
		return nil
	}

	y, err := findLastInt(r.TorrentName, `\b(((?:19[0-9]|20[0-9])[0-9]))\b`)
	if err != nil {
		return err
	}
	r.Year = y

	return nil
}

func (r *Release) extractSeason() error {
	s, err := findLastInt(r.TorrentName, `(?i)(?:S|Season\s*)(\d{1,3})`)
	if err != nil {
		return err
	}
	r.Season = s

	return nil
}

func (r *Release) extractEpisode() error {
	e, err := findLastInt(r.TorrentName, `(?i)[ex]([0-9]{2})(?:[^0-9]|$)`)
	if err != nil {
		return err
	}
	r.Episode = e

	return nil
}

func (r *Release) extractResolution() error {
	v, err := findLast(r.TorrentName, `\b(?i)[0-9]{3,4}(?:p|i)\b`)
	if err != nil {
		return err
	}
	r.Resolution = v

	return nil
}

func (r *Release) extractResolutionFromTags(tag string) error {
	if r.Resolution != "" {
		return nil
	}
	v, err := findLast(tag, `\b(?i)(([0-9]{3,4}p|i))\b`)
	if err != nil {
		return err
	}
	r.Resolution = v

	return nil
}

func (r *Release) extractSource() error {
	v, err := findLast(r.TorrentName, `(?i)\b(((?:PPV\.)?[HP]DTV|(?:HD)?CAM|B[DR]Rip|(?:HD-?)?TS|(?:PPV )?WEB-?DL(?: DVDRip)?|HDRip|DVDRip|DVDRIP|CamRip|WEB|W[EB]BRip|Blu-?Ray|DvDScr|telesync|CD|DVD|Vinyl|DAT|Cassette))\b`)
	if err != nil {
		return err
	}
	r.Source = v

	return nil
}

func (r *Release) extractSourceFromTags(tag string) error {
	if r.Source != "" {
		return nil
	}
	v, err := findLast(tag, `(?i)\b(((?:PPV\.)?[HP]DTV|(?:HD)?CAM|B[DR]Rip|(?:HD-?)?TS|(?:PPV )?WEB-?DL(?: DVDRip)?|HDRip|DVDRip|DVDRIP|CamRip|WEB|W[EB]BRip|Blu-?Ray|DvDScr|telesync|CD|DVD|Vinyl|DAT|Cassette))\b`)
	if err != nil {
		return err
	}
	r.Source = v

	return nil
}

func (r *Release) extractCodec() error {
	v, err := findLast(r.TorrentName, `(?i)\b(HEVC|[hx]\.?26[45] 10-bit|[hx]\.?26[45]|xvid|divx|AVC|MPEG-?2|AV1|VC-?1|VP9|WebP)\b`)
	if err != nil {
		return err
	}
	r.Codec = v

	return nil
}

func (r *Release) extractCodecFromTags(tag string) error {
	if r.Codec != "" {
		return nil
	}

	v, err := findLast(tag, `(?i)\b(HEVC|[hx]\.?26[45] 10-bit|[hx]\.?26[45]|xvid|divx|AVC|MPEG-?2|AV1|VC-?1|VP9|WebP)\b`)
	if err != nil {
		return err
	}
	r.Codec = v

	return nil
}

func (r *Release) extractContainer() error {
	v, err := findLast(r.TorrentName, `(?i)\b(AVI|MPG|MKV|MP4|VOB|m2ts|ISO|IMG)\b`)
	if err != nil {
		return err
	}
	r.Container = v

	return nil
}

func (r *Release) extractContainerFromTags(tag string) error {
	if r.Container != "" {
		return nil
	}

	v, err := findLast(tag, `(?i)\b(AVI|MPG|MKV|MP4|VOB|m2ts|ISO|IMG)\b`)
	if err != nil {
		return err
	}
	r.Container = v

	return nil
}

func (r *Release) extractHDR() error {
	v, err := findLast(r.TorrentName, `(?i)[\. ](HDR10\+|HDR10|DoVi[\. ]HDR|DV[\. ]HDR10\+|DV[\. ]HDR10|DV[\. ]HDR|HDR|DV|DoVi|Dolby[\. ]Vision[\. ]\+[\. ]HDR10|Dolby[\. ]Vision)[\. ]`)
	if err != nil {
		return err
	}
	r.HDR = v

	return nil
}

func (r *Release) extractAudio() error {
	v, err := findLast(r.TorrentName, `(?i)(FLAC[\. ][1-7][\. ][0-2]|FLAC|Opus|DD-EX|DDP[\. ]?[124567][\. ][012] Atmos|DDP[\. ]?[124567][\. ][012]|DDP|DD[1-7][\. ][0-2]|Dual[\- ]Audio|LiNE|PCM|Dolby TrueHD [0-9][\. ][0-4]|TrueHD [0-9][\. ][0-4] Atmos|TrueHD [0-9][\. ][0-4]|DTS X|DTS-HD MA [0-9][\. ][0-4]|DTS-HD MA|DTS-ES|DTS [1-7][\. ][0-2]|DTS|DD|DD[12][\. ]0|Dolby Atmos|TrueHD ATMOS|TrueHD|Atmos|Dolby Digital Plus|Dolby Digital Audio|Dolby Digital|AAC[.-]LC|AAC (?:\.?[1-7]\.[0-2])?|AAC|eac3|AC3(?:\.5\.1)?)`)
	if err != nil {
		return err
	}
	r.Audio = v

	return nil
}

func (r *Release) extractAudioFromTags(tag string) error {
	if r.Audio != "" {
		return nil
	}

	// "(?i)(FLAC|Opus|DD-EX|DDP Atmos|DDP|DD|Dual[\- ]Audio|LiNE|PCM|Dolby TrueHD|TrueHD Atmos|TrueHD|DTS X|DTS-HD MA|DTS-ES|DTS|DD|DD|Dolby Atmos|TrueHD ATMOS|TrueHD|Atmos|Dolby Digital Plus|Dolby Digital Audio|Dolby Digital|AAC[.-]LC|AAC|eac3|AC3)"
	v, err := findLast(tag, `(?i)(FLAC[\. ][1-7][\. ][0-2]|FLAC|Opus|DD-EX|DDP[\. ]?[124567][\. ][012] Atmos|DDP[\. ]?[124567][\. ][012]|DDP|DD[1-7][\. ][0-2]|Dual[\- ]Audio|LiNE|PCM|Dolby TrueHD [0-9][\. ][0-4]|TrueHD [0-9][\. ][0-4] Atmos|TrueHD [0-9][\. ][0-4]|DTS X|DTS-HD MA [0-9][\. ][0-4]|DTS-HD MA|DTS-ES|DTS [1-7][\. ][0-2]|DTS|DD|DD[12][\. ]0|Dolby Atmos|TrueHD ATMOS|TrueHD|Atmos|Dolby Digital Plus|Dolby Digital Audio|Dolby Digital|AAC[.-]LC|AAC (?:\.?[1-7]\.[0-2])?|AAC|eac3|AC3(?:\.5\.1)?)`)
	if err != nil {
		return err
	}
	r.Audio = v

	return nil
}

func (r *Release) extractAudioChannels(tag string) error {
	if r.AudioChannels != "" {
		return nil
	}

	v, err := findLast(tag, `(?i)([1-9][\. ][0-4])?)`)
	if err != nil {
		return err
	}
	r.AudioChannels = v

	return nil
}

func (r *Release) extractFormatsFromTags(tag string) error {
	if r.Format != "" {
		return nil
	}

	v, err := findLast(tag, `(?i)(?:MP3|FLAC|Ogg Vorbis|AAC|AC3|DTS)`)
	if err != nil {
		return err
	}
	r.Format = v

	return nil
}

//func (r *Release) extractCueFromTags(tag string) error {
//	v, err := findLast(tag, `Cue`)
//	if err != nil {
//		return err
//	}
//	r.HasCue = v
//
//	return nil
//}

func (r *Release) extractGroup() error {
	// try first for wierd anime group names [group] show name, or in brackets at the end

	//g, err := findLast(r.Clean, `\[(.*?)\]`)
	group, err := findLast(r.TorrentName, `\-([a-zA-Z0-9_\.]+)$`)
	if err != nil {
		return err
	}

	r.Group = group

	return nil
}

func (r *Release) extractAnimeGroupFromTags(tag string) error {
	if r.Group != "" {
		return nil
	}
	v, err := findLast(tag, `(?:RAW|Softsubs|Hardsubs)\s\((.+)\)`)
	if err != nil {
		return err
	}
	r.Group = v

	return nil
}

func (r *Release) extractRegion() error {
	v, err := findLast(r.TorrentName, `(?i)\b(R([0-9]))\b`)
	if err != nil {
		return err
	}
	r.Region = v

	return nil
}

func (r *Release) extractLanguage() error {
	v, err := findLast(r.TorrentName, `(?i)\b((DK|DKSUBS|DANiSH|DUTCH|NL|NLSUBBED|ENG|FI|FLEMiSH|FiNNiSH|DE|FRENCH|GERMAN|HE|HEBREW|HebSub|HiNDi|iCELANDiC|KOR|MULTi|MULTiSUBS|NORWEGiAN|NO|NORDiC|PL|PO|POLiSH|PLDUB|RO|ROMANiAN|RUS|SPANiSH|SE|SWEDiSH|SWESUB||))\b`)
	if err != nil {
		return err
	}
	r.Language = v

	return nil
}

func (r *Release) extractEdition() error {
	v, err := findLast(r.TorrentName, `(?i)\b((?:DIRECTOR'?S|EXTENDED|INTERNATIONAL|THEATRICAL|ORIGINAL|FINAL|BOOTLEG)(?:.CUT)?)\b`)
	if err != nil {
		return err
	}
	r.Edition = v

	return nil
}

func (r *Release) extractUnrated() error {
	v, err := findLastBool(r.TorrentName, `(?i)\b((UNRATED))\b`)
	if err != nil {
		return err
	}
	r.Unrated = v

	return nil
}

func (r *Release) extractHybrid() error {
	v, err := findLastBool(r.TorrentName, `(?i)\b((HYBRID))\b`)
	if err != nil {
		return err
	}
	r.Hybrid = v

	return nil
}

func (r *Release) extractProper() error {
	v, err := findLastBool(r.TorrentName, `(?i)\b((PROPER))\b`)
	if err != nil {
		return err
	}
	r.Proper = v

	return nil
}

func (r *Release) extractRepack() error {
	v, err := findLastBool(r.TorrentName, `(?i)\b((REPACK))\b`)
	if err != nil {
		return err
	}
	r.Repack = v

	return nil
}

func (r *Release) extractWebsite() error {
	// Start with the basic most common ones
	v, err := findLast(r.TorrentName, `(?i)\b((AMBC|AS|AMZN|AMC|ANPL|ATVP|iP|CORE|BCORE|CMOR|CN|CBC|CBS|CMAX|CNBC|CC|CRIT|CR|CSPN|CW|DAZN|DCU|DISC|DSCP|DSNY|DSNP|DPLY|ESPN|FOX|FUNI|PLAY|HBO|HMAX|HIST|HS|HOTSTAR|HULU|iT|MNBC|MTV|NATG|NBC|NF|NICK|NRK|PMNT|PMNP|PCOK|PBS|PBSK|PSN|QIBI|SBS|SHO|STAN|STZ|SVT|SYFY|TLC|TRVL|TUBI|TV3|TV4|TVL|VH1|VICE|VMEO|UFC|USAN|VIAP|VIAPLAY|VL|WWEN|XBOX|YHOO|YT|RED))\b`)
	if err != nil {
		return err
	}
	r.Website = v

	return nil
}

func (r *Release) extractFreeleechFromTags(tag string) error {
	if r.Freeleech == true {
		return nil
	}

	// Start with the basic most common ones
	v, err := findLast(tag, `(?i)(Freeleech!|Freeleech)`)
	if err != nil {
		return err
	}
	if v != "" {
		r.Freeleech = true
		return nil
	}

	r.Freeleech = false

	return nil
}

func (r *Release) extractLogScoreFromTags(tag string) error {
	if r.LogScore > 0 {
		return nil
	}

	// Start with the basic most common ones

	rxp, err := regexp.Compile(`([\d\.]+)%`)
	if err != nil {
		return err
		//return errors.Wrapf(err, "invalid regex: %s", value)
	}

	matches := rxp.FindStringSubmatch(tag)
	if matches != nil {
		// first value is the match, second value is the text
		if len(matches) >= 1 {
			last := matches[len(matches)-1]
			score, err := strconv.ParseInt(last, 10, 32)
			if err != nil {
				return err
			}

			r.LogScore = int(score)
			return nil
		}
	}

	return nil
}

func (r *Release) extractQualityFromTags(tag string) error {
	if r.Quality != "" {
		return nil
	}

	// Start with the basic most common ones

	rxp, err := regexp.Compile(`(?i)(Lossless|24bit Lossless|V0 \(VBR\)|V1 \(VBR\)|V2 \(VBR\)|APS \(VBR\)|APX \(VBR\)|320|256|192)`)
	if err != nil {
		return err
		//return errors.Wrapf(err, "invalid regex: %s", value)
	}

	matches := rxp.FindStringSubmatch(tag)
	if matches != nil {
		// first value is the match, second value is the text
		if len(matches) >= 1 {
			last := matches[len(matches)-1]

			r.Quality = last
			return nil
		}
	}

	return nil
}

func (r *Release) extractReleaseTags() error {
	if r.ReleaseTags == "" {
		return nil
	}

	tags := SplitAny(r.ReleaseTags, ",|/")

	for _, t := range tags {
		t = strings.Trim(t, " ")

		var err error
		err = r.extractAudioFromTags(t)
		err = r.extractFormatsFromTags(t)
		err = r.extractResolutionFromTags(t)
		err = r.extractCodecFromTags(t)
		err = r.extractContainerFromTags(t)
		err = r.extractSourceFromTags(t)
		err = r.extractFreeleechFromTags(t)
		err = r.extractLogScoreFromTags(t)
		err = r.extractQualityFromTags(t)
		//err = r.extractAnimeGroupFromTags(t)

		if err != nil {
			continue
		}

		switch t {
		case "Cue":
			r.HasCue = true
		case "Log":
			r.HasLog = true
			// check percent
		}
	}

	return nil
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
		return err
	}

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{
		Transport: customTransport,
		Jar:       jar,
	}

	req, err := http.NewRequest("GET", r.TorrentURL, nil)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error downloading file")
		return err
	}

	if r.RawCookie != "" {
		// set the cookie on the header instead of req.AddCookie
		// since we have a raw cookie like "uid=10; pass=000"
		req.Header.Set("Cookie", r.RawCookie)
	}

	// Get the data
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error downloading file")
		return err
	}
	defer resp.Body.Close()

	// retry logic

	if resp.StatusCode != http.StatusOK {
		log.Error().Stack().Err(err).Msgf("error downloading file from: %v - bad status: %d", r.TorrentURL, resp.StatusCode)
		return fmt.Errorf("error downloading torrent (%v) file (%v) from '%v' - status code: %d", r.TorrentName, r.TorrentURL, r.Indexer, resp.StatusCode)
	}

	// Create tmp file
	tmpFile, err := os.CreateTemp("", "autobrr-")
	if err != nil {
		log.Error().Stack().Err(err).Msg("error creating temp file")
		return err
	}
	defer tmpFile.Close()

	// Write the body to file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error writing downloaded file: %v", tmpFile.Name())
		return err
	}

	meta, err := metainfo.LoadFromFile(tmpFile.Name())
	if err != nil {
		log.Error().Stack().Err(err).Msgf("metainfo could not load file contents: %v", tmpFile.Name())
		return err
	}

	torrentMetaInfo, err := meta.UnmarshalInfo()
	if err != nil {
		log.Error().Stack().Err(err).Msgf("metainfo could not unmarshal info from torrent: %v", tmpFile.Name())
		return err
	}

	r.TorrentTmpFile = tmpFile.Name()
	r.TorrentHash = meta.HashInfoBytes().String()
	r.Size = uint64(torrentMetaInfo.TotalLength())

	// remove file if fail

	log.Debug().Msgf("successfully downloaded file: %v", tmpFile.Name())

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
func (r *Release) MapVars(def IndexerDefinition, varMap map[string]string) error {

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
		r.Freeleech = strings.EqualFold(freeleech, "freeleech") || strings.EqualFold(freeleech, "yes") || strings.EqualFold(freeleech, "1")
	}

	if freeleechPercent, err := getStringMapValue(varMap, "freeleechPercent"); err == nil {
		// remove % and trim spaces
		freeleechPercent = strings.Replace(freeleechPercent, "%", "", -1)
		freeleechPercent = strings.Trim(freeleechPercent, " ")

		freeleechPercentInt, err := strconv.Atoi(freeleechPercent)
		if err != nil {
			//log.Debug().Msgf("bad freeleechPercent var: %v", year)
		}

		r.FreeleechPercent = freeleechPercentInt
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
		r.ReleaseTagsArr = SplitAny(r.ReleaseTags, ",|/") // TODO keep this?
	}

	if resolution, err := getStringMapValue(varMap, "resolution"); err == nil {
		r.Resolution = resolution
	}

	return nil
}

func getStringMapValue(stringMap map[string]string, key string) (string, error) {
	lowerKey := strings.ToLower(key)

	// case-sensitive match
	//if caseSensitive {
	//	v, ok := stringMap[key]
	//	if !ok {
	//		return "", fmt.Errorf("key was not found in map: %q", key)
	//	}
	//
	//	return v, nil
	//}

	// case-insensitive match
	for k, v := range stringMap {
		if strings.ToLower(k) == lowerKey {
			return v, nil
		}
	}

	return "", fmt.Errorf("key was not found in map: %q", lowerKey)
}

//func getFirstStringMapValue(stringMap map[string]string, keys []string) (string, error) {
//	for _, k := range keys {
//		if val, err := getStringMapValue(stringMap, k); err == nil {
//			return val, nil
//		}
//	}
//
//	return "", fmt.Errorf("key were not found in map: %q", strings.Join(keys, ", "))
//}

func findLast(input string, pattern string) (string, error) {
	matched := make([]string, 0)
	//for _, s := range arr {

	rxp, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
		//return errors.Wrapf(err, "invalid regex: %s", value)
	}

	matches := rxp.FindStringSubmatch(input)
	if matches != nil {
		// first value is the match, second value is the text
		if len(matches) >= 1 {
			last := matches[len(matches)-1]

			// add to temp slice
			matched = append(matched, last)
		}
	}

	//}

	// check if multiple values in temp slice, if so get the last one
	if len(matched) >= 1 {
		last := matched[len(matched)-1]

		return last, nil
	}

	return "", nil
}

func findLastBool(input string, pattern string) (bool, error) {
	matched := make([]string, 0)

	rxp, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}

	matches := rxp.FindStringSubmatch(input)
	if matches != nil {
		// first value is the match, second value is the text
		if len(matches) >= 1 {
			last := matches[len(matches)-1]

			// add to temp slice
			matched = append(matched, last)
		}
	}

	//}

	// check if multiple values in temp slice, if so get the last one
	if len(matched) >= 1 {
		//last := matched[len(matched)-1]

		return true, nil
	}

	return false, nil
}

func findLastInt(input string, pattern string) (int, error) {
	matched := make([]string, 0)
	//for _, s := range arr {

	rxp, err := regexp.Compile(pattern)
	if err != nil {
		return 0, err
		//return errors.Wrapf(err, "invalid regex: %s", value)
	}

	matches := rxp.FindStringSubmatch(input)
	if matches != nil {
		// first value is the match, second value is the text
		if len(matches) >= 1 {
			last := matches[len(matches)-1]

			// add to temp slice
			matched = append(matched, last)
		}
	}

	//}

	// check if multiple values in temp slice, if so get the last one
	if len(matched) >= 1 {
		last := matched[len(matched)-1]

		i, err := strconv.Atoi(last)
		if err != nil {
			return 0, err
		}

		return i, nil
	}

	return 0, nil
}

func SplitAny(s string, seps string) []string {
	splitter := func(r rune) bool {
		return strings.ContainsRune(seps, r)
	}
	return strings.FieldsFunc(s, splitter)
}

//func Splitter(s string, splits string) []string {
//	m := make(map[rune]int)
//	for _, r := range splits {
//		m[r] = 1
//	}
//
//	splitter := func(r rune) bool {
//		return m[r] == 1
//	}
//
//	return strings.FieldsFunc(s, splitter)
//}
//
//func canonicalizeString(s string) []string {
//	//a := strings.FieldsFunc(s, split)
//	a := Splitter(s, " .")
//
//	return a
//}

func cleanReleaseName(input string) string {
	// Make a Regex to say we only want letters and numbers
	reg, err := regexp.Compile(`[\x00-\x1F\x2D\x2E\x5F\x7F]`)
	if err != nil {
		return ""
	}
	processedString := reg.ReplaceAllString(input, " ")

	return processedString
}
