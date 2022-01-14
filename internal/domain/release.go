package domain

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/autobrr/autobrr/pkg/wildcard"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type ReleaseRepo interface {
	Store(ctx context.Context, release *Release) (*Release, error)
	Find(ctx context.Context, params QueryParams) (res []Release, nextCursor int64, count int64, err error)
	GetActionStatusByReleaseID(ctx context.Context, releaseID int64) ([]ReleaseActionStatus, error)
	Stats(ctx context.Context) (*ReleaseStats, error)
	StoreReleaseActionStatus(ctx context.Context, actionStatus *ReleaseActionStatus) error
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
	Container                   string                `json:"container"`
	HDR                         string                `json:"hdr"`
	Audio                       string                `json:"audio"`
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
	Bitrate                     string                `json:"bitrate"` // bitrate
	LogScore                    int                   `json:"log_score"`
	HasLog                      bool                  `json:"has_log"`
	HasCue                      bool                  `json:"has_cue"`
	IsScene                     bool                  `json:"is_scene"`
	Origin                      string                `json:"origin"` // P2P, Internal
	Tags                        []string              `json:"tags"`
	ReleaseTags                 string                `json:"-"`
	Freeleech                   bool                  `json:"freeleech"`
	FreeleechPercent            int                   `json:"freeleech_percent"`
	Uploader                    string                `json:"uploader"`
	PreTime                     string                `json:"pre_time"`
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

func (r *Release) extractYear() error {
	y, err := findLastInt(r.TorrentName, `\b(((?:19[0-9]|20[0-9])[0-9]))\b`)
	if err != nil {
		return err
	}
	r.Year = y

	return nil
}

func (r *Release) extractSeason() error {
	s, err := findLastInt(r.TorrentName, `(?:S|Season\s*)(\d{1,3})`)
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
	v, err := findLast(r.TorrentName, `\b(([0-9]{3,4}p|i))\b`)
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
	v, err := findLast(r.TorrentName, `(?i)\b(HEVC|[hx]\.?26[45]|xvid|divx|AVC|MPEG-?2|AV1|VC-?1|VP9|WebP)\b`)
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
	v, err := findLast(r.TorrentName, `(?i)[\. ](HDR10\+|HDR10|DoVi[\. ]HDR|DV[\. ]HDR|HDR|DV|DoVi|Dolby[\. ]Vision[\. ]\+[\. ]HDR10|Dolby[\. ]Vision)[\. ]`)
	if err != nil {
		return err
	}
	r.HDR = v

	return nil
}

func (r *Release) extractAudio() error {
	v, err := findLast(r.TorrentName, `(?i)(MP3|FLAC[\. ][1-7][\. ][0-2]|FLAC|Opus|DD-EX|DDP[\. ]?[124567][\. ][012] Atmos|DDP[\. ]?[124567][\. ][012]|DDP|DD[1-7][\. ][0-2]|Dual[\- ]Audio|LiNE|PCM|Dolby TrueHD [0-9][\. ][0-4]|TrueHD [0-9][\. ][0-4] Atmos|TrueHD [0-9][\. ][0-4]|DTS X|DTS-HD MA [0-9][\. ][0-4]|DTS-HD MA|DTS-ES|DTS [1-7][\. ][0-2]|DTS|DD|DD[12][\. ]0|Dolby Atmos|TrueHD ATMOS|TrueHD|Atmos|Dolby Digital Plus|Dolby Digital Audio|Dolby Digital|AAC[.-]LC|AAC (?:\.?[1-7]\.[0-2])?|AAC|eac3|AC3(?:\.5\.1)?)`)
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

	v, err := findLast(tag, `(?i)(MP3|Ogg Vorbis|FLAC[\. ][1-7][\. ][0-2]|FLAC|Opus|DD-EX|DDP[\. ]?[124567][\. ][012] Atmos|DDP[\. ]?[124567][\. ][012]|DDP|DD[1-7][\. ][0-2]|Dual[\- ]Audio|LiNE|PCM|Dolby TrueHD [0-9][\. ][0-4]|TrueHD [0-9][\. ][0-4] Atmos|TrueHD [0-9][\. ][0-4]|DTS X|DTS-HD MA [0-9][\. ][0-4]|DTS-HD MA|DTS-ES|DTS [1-7][\. ][0-2]|DTS|DD|DD[12][\. ]0|Dolby Atmos|TrueHD ATMOS|TrueHD|Atmos|Dolby Digital Plus|Dolby Digital Audio|Dolby Digital|AAC[.-]LC|AAC (?:\.?[1-7]\.[0-2])?|AAC|eac3|AC3(?:\.5\.1)?)`)
	if err != nil {
		return err
	}
	r.Audio = v

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
	v, err := findLast(tag, `Freeleech!`)
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

func (r *Release) extractBitrateFromTags(tag string) error {
	if r.Bitrate != "" {
		return nil
	}

	// Start with the basic most common ones

	rxp, err := regexp.Compile(`^(?:vbr|aps|apx|v\d|\d{2,4}|\d+\.\d+|q\d+\.[\dx]+|Other)?(?:\s*kbps|\s*kbits?|\s*k)?(?:\s*\(?(?:vbr|cbr)\)?)?$`)
	if err != nil {
		return err
		//return errors.Wrapf(err, "invalid regex: %s", value)
	}

	matches := rxp.FindStringSubmatch(tag)
	if matches != nil {
		// first value is the match, second value is the text
		if len(matches) >= 1 {
			last := matches[len(matches)-1]

			r.Bitrate = last
			return nil
		}
	}

	return nil
}

func (r *Release) extractReleaseTags() error {
	if r.ReleaseTags == "" {
		return nil
	}

	tags := SplitAny(r.ReleaseTags, ",|/ ")

	for _, t := range tags {
		var err error
		err = r.extractAudioFromTags(t)
		err = r.extractContainerFromTags(t)
		err = r.extractSourceFromTags(t)
		err = r.extractFreeleechFromTags(t)
		err = r.extractLogScoreFromTags(t)
		err = r.extractBitrateFromTags(t)

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

func (r *Release) ParseTorrentUrl(match string, vars map[string]string, extraVars map[string]string, encode []string) error {
	tmpVars := map[string]string{}

	// copy vars to new tmp map
	for k, v := range vars {
		tmpVars[k] = v
	}

	// merge extra vars with vars
	if extraVars != nil {
		for k, v := range extraVars {
			tmpVars[k] = v
		}
	}

	// handle url encode of values
	if encode != nil {
		for _, e := range encode {
			if v, ok := tmpVars[e]; ok {
				// url encode  value
				t := url.QueryEscape(v)
				tmpVars[e] = t
			}
		}
	}

	// setup text template to inject variables into
	tmpl, err := template.New("torrenturl").Parse(match)
	if err != nil {
		log.Error().Err(err).Msg("could not create torrent url template")
		return err
	}

	var urlBytes bytes.Buffer
	err = tmpl.Execute(&urlBytes, &tmpVars)
	if err != nil {
		log.Error().Err(err).Msg("could not write torrent url template output")
		return err
	}

	r.TorrentURL = urlBytes.String()

	// TODO handle cookies

	return nil
}

func (r *Release) DownloadTorrentFile(opts map[string]string) (*DownloadTorrentFileResponse, error) {
	if r.TorrentURL == "" {
		return nil, errors.New("download_file: url can't be empty")
	} else if r.TorrentTmpFile != "" {
		// already downloaded
		return nil, nil
	}

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{Transport: customTransport}

	// Get the data
	resp, err := client.Get(r.TorrentURL)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error downloading file")
		return nil, err
	}
	defer resp.Body.Close()

	// retry logic

	if resp.StatusCode != http.StatusOK {
		log.Error().Stack().Err(err).Msgf("error downloading file from: %v - bad status: %d", r.TorrentURL, resp.StatusCode)
		return nil, err
	}

	// Create tmp file
	tmpFile, err := os.CreateTemp("", "autobrr-")
	if err != nil {
		log.Error().Stack().Err(err).Msg("error creating temp file")
		return nil, err
	}
	defer tmpFile.Close()

	r.TorrentTmpFile = tmpFile.Name()

	// Write the body to file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error writing downloaded file: %v", tmpFile.Name())
		return nil, err
	}

	meta, err := metainfo.LoadFromFile(tmpFile.Name())
	if err != nil {
		log.Error().Stack().Err(err).Msgf("metainfo could not load file contents: %v", tmpFile.Name())
		return nil, err
	}

	// remove file if fail

	res := DownloadTorrentFileResponse{
		MetaInfo:    meta,
		TmpFileName: tmpFile.Name(),
	}

	if res.TmpFileName == "" || res.MetaInfo == nil {
		log.Error().Stack().Err(err).Msgf("tmp file error - empty body: %v", r.TorrentURL)
		return nil, errors.New("error downloading file, no tmp file")
	}

	log.Debug().Msgf("successfully downloaded file: %v", tmpFile.Name())

	return &res, nil
}

func (r *Release) addRejection(reason string) {
	r.Rejections = append(r.Rejections, reason)
}

// ResetRejections reset rejections between filter checks
func (r *Release) resetRejections() {
	r.Rejections = []string{}
}

func (r *Release) CheckFilter(filter Filter) bool {
	// reset rejections first to clean previous checks
	r.resetRejections()

	if !filter.Enabled {
		return false
	}

	// FIXME what if someone explicitly doesnt want scene, or toggles in filter. Make enum? 0,1,2? Yes, No, Dont care
	if filter.Scene && r.IsScene != filter.Scene {
		r.addRejection("wanted: scene")
		return false
	}

	if filter.Freeleech && r.Freeleech != filter.Freeleech {
		r.addRejection("wanted: freeleech")
		return false
	}

	if filter.FreeleechPercent != "" && !checkFreeleechPercent(r.FreeleechPercent, filter.FreeleechPercent) {
		r.addRejection("freeleech percent not matching")
		return false
	}

	// check against TorrentName and Clean which is a cleaned name without (. _ -)
	if filter.Shows != "" && !checkMultipleFilterStrings(filter.Shows, r.TorrentName, r.Clean) {
		r.addRejection("shows not matching")
		return false
	}

	if filter.Seasons != "" && !checkFilterIntStrings(r.Season, filter.Seasons) {
		r.addRejection("season not matching")
		return false
	}

	if filter.Episodes != "" && !checkFilterIntStrings(r.Episode, filter.Episodes) {
		r.addRejection("episode not matching")
		return false
	}

	// matchRelease
	// TODO allow to match against regex
	if filter.MatchReleases != "" && !checkMultipleFilterStrings(filter.MatchReleases, r.TorrentName, r.Clean) {
		r.addRejection("match release not matching")
		return false
	}

	if filter.ExceptReleases != "" && checkMultipleFilterStrings(filter.ExceptReleases, r.TorrentName, r.Clean) {
		r.addRejection("except_releases: unwanted release")
		return false
	}

	if filter.MatchReleaseGroups != "" && !checkMultipleFilterGroups(filter.MatchReleaseGroups, r.Group, r.Clean) {
		r.addRejection("release groups not matching")
		return false
	}

	if filter.ExceptReleaseGroups != "" && checkMultipleFilterGroups(filter.ExceptReleaseGroups, r.Group, r.Clean) {
		r.addRejection("unwanted release group")
		return false
	}

	if filter.MatchUploaders != "" && !checkFilterStrings(r.Uploader, filter.MatchUploaders) {
		r.addRejection("uploaders not matching")
		return false
	}

	if filter.ExceptUploaders != "" && checkFilterStrings(r.Uploader, filter.ExceptUploaders) {
		r.addRejection("unwanted uploaders")
		return false
	}

	if len(filter.Resolutions) > 0 && !checkFilterSlice(r.Resolution, filter.Resolutions) {
		r.addRejection("resolution not matching")
		return false
	}

	if len(filter.Codecs) > 0 && !checkFilterSlice(r.Codec, filter.Codecs) {
		r.addRejection("codec not matching")
		return false
	}

	if len(filter.Sources) > 0 && !checkFilterSource(r.Source, filter.Sources) {
		r.addRejection("source not matching")
		return false
	}

	if len(filter.Containers) > 0 && !checkFilterSlice(r.Container, filter.Containers) {
		r.addRejection("container not matching")
		return false
	}

	if len(filter.MatchHDR) > 0 && !checkMultipleFilterHDR(filter.MatchHDR, r.HDR, r.TorrentName) {
		r.addRejection("hdr not matching")
		return false
	}

	if len(filter.ExceptHDR) > 0 && checkMultipleFilterHDR(filter.ExceptHDR, r.HDR, r.TorrentName) {
		r.addRejection("unwanted hdr")
		return false
	}

	if filter.Years != "" && !checkFilterIntStrings(r.Year, filter.Years) {
		r.addRejection("year not matching")
		return false
	}

	if filter.MatchCategories != "" && !checkFilterStrings(r.Category, filter.MatchCategories) {
		r.addRejection("category not matching")
		return false
	}

	if filter.ExceptCategories != "" && checkFilterStrings(r.Category, filter.ExceptCategories) {
		r.addRejection("unwanted category")
		return false
	}

	if (filter.MinSize != "" || filter.MaxSize != "") && !r.CheckSizeFilter(filter.MinSize, filter.MaxSize) {
		return false
	}

	if filter.Tags != "" && !checkFilterTags(r.Tags, filter.Tags) {
		r.addRejection("tags not matching")
		return false
	}

	if filter.ExceptTags != "" && checkFilterTags(r.Tags, filter.ExceptTags) {
		r.addRejection("unwanted tags")
		return false
	}

	return true
}

// CheckSizeFilter additional size check
// for indexers that doesn't announce size, like some gazelle based
// set flag r.AdditionalSizeCheckRequired if there's a size in the filter, otherwise go a head
// implement API for ptp,btn,ggn to check for size if needed
// for others pull down torrent and do check
func (r *Release) CheckSizeFilter(minSize string, maxSize string) bool {

	if r.Size == 0 {
		r.AdditionalSizeCheckRequired = true

		return true
	} else {
		r.AdditionalSizeCheckRequired = false
	}

	// if r.Size parse filter to bytes and compare
	// handle both min and max
	if minSize != "" {
		// string to bytes
		minSizeBytes, err := humanize.ParseBytes(minSize)
		if err != nil {
			// log could not parse into bytes
		}

		if r.Size <= minSizeBytes {
			r.addRejection("size: smaller than min size")
			return false
		}

	}

	if maxSize != "" {
		// string to bytes
		maxSizeBytes, err := humanize.ParseBytes(maxSize)
		if err != nil {
			// log could not parse into bytes
		}

		if r.Size >= maxSizeBytes {
			r.addRejection("size: larger than max size")
			return false
		}
	}

	return true
}

// MapVars better name
func (r *Release) MapVars(varMap map[string]string) error {

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
		r.Freeleech = strings.EqualFold(freeleech, "freeleech") || strings.EqualFold(freeleech, "yes")
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
		size, err := humanize.ParseBytes(torrentSize)
		if err != nil {
			// log could not parse into bytes
		}
		r.Size = size
		// TODO implement other size checks in filter
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
	}

	return nil
}

func checkFilterSlice(name string, filterList []string) bool {
	name = strings.ToLower(name)

	for _, filter := range filterList {
		filter = strings.ToLower(filter)
		filter = strings.Trim(filter, " ")
		// check if line contains * or ?, if so try wildcard match, otherwise try substring match
		a := strings.ContainsAny(filter, "?|*")
		if a {
			match := wildcard.Match(filter, name)
			if match {
				return true
			}
		} else {
			b := strings.Contains(name, filter)
			if b {
				return true
			}
		}
	}

	return false
}

func checkFilterStrings(name string, filterList string) bool {
	filterSplit := strings.Split(filterList, ",")
	name = strings.ToLower(name)

	for _, s := range filterSplit {
		s = strings.ToLower(s)
		s = strings.Trim(s, " ")
		// check if line contains * or ?, if so try wildcard match, otherwise try substring match
		a := strings.ContainsAny(s, "?|*")
		if a {
			match := wildcard.Match(s, name)
			if match {
				return true
			}
		} else {
			b := strings.Contains(name, s)
			if b {
				return true
			}
		}

	}

	return false
}

// checkMultipleFilterStrings check against multiple vars of unknown length
func checkMultipleFilterStrings(filterList string, vars ...string) bool {
	filterSplit := strings.Split(filterList, ",")

	for _, name := range vars {
		name = strings.ToLower(name)

		for _, s := range filterSplit {
			s = strings.ToLower(s)
			s = strings.Trim(s, " ")
			// check if line contains * or ?, if so try wildcard match, otherwise try substring match
			a := strings.ContainsAny(s, "?|*")
			if a {
				match := wildcard.Match(s, name)
				if match {
					return true
				}
			} else {
				b := strings.Contains(name, s)
				if b {
					return true
				}
			}
		}
	}

	return false
}

// checkFilterIntStrings "1,2,3-20"
func checkFilterIntStrings(value int, filterList string) bool {
	filters := strings.Split(filterList, ",")

	for _, s := range filters {
		s = strings.Replace(s, "%", "", -1)
		s = strings.Trim(s, " ")

		if strings.Contains(s, "-") {
			minMax := strings.Split(s, "-")

			// to int
			min, err := strconv.ParseInt(minMax[0], 10, 32)
			if err != nil {
				return false
			}

			max, err := strconv.ParseInt(minMax[1], 10, 32)
			if err != nil {
				return false
			}

			if min > max {
				// handle error
				return false
			} else {
				// if announcePercent is greater than min and less than max return true
				if value >= int(min) && value <= int(max) {
					return true
				}
			}
		}

		filterInt, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return false
		}

		if int(filterInt) == value {
			return true
		}
	}

	return false
}

func checkMultipleFilterGroups(filterList string, vars ...string) bool {
	filterSplit := strings.Split(filterList, ",")

	for _, name := range vars {
		name = strings.ToLower(name)

		for _, s := range filterSplit {
			s = strings.ToLower(strings.Trim(s, " "))
			// check if line contains * or ?, if so try wildcard match, otherwise try substring match
			a := strings.ContainsAny(s, "?|*")
			if a {
				match := wildcard.Match(s, name)
				if match {
					return true
				}
			} else {
				split := SplitAny(name, " .-")
				for _, c := range split {
					if c == s {
						return true
					}
				}
				continue
			}
		}
	}

	return false
}

func checkMultipleFilterHDR(filterList []string, vars ...string) bool {
	for _, name := range vars {
		name = strings.ToLower(name)

		for _, s := range filterList {
			s = strings.ToLower(strings.Trim(s, " "))
			// check if line contains * or ?, if so try wildcard match, otherwise try substring match
			a := strings.ContainsAny(s, "?|*")
			if a {
				match := wildcard.Match(s, name)
				if match {
					return true
				}
			} else {
				split := SplitAny(name, " .-")
				for _, c := range split {
					if c == s {
						return true
					}
				}
				continue
			}
		}
	}

	return false
}

func checkFilterSource(name string, filterList []string) bool {
	// remove dash (-) in blu-ray web-dl and make lowercase
	name = strings.ToLower(strings.ReplaceAll(name, "-", ""))

	for _, filter := range filterList {
		// remove dash (-) in blu-ray web-dl, trim spaces and make lowercase
		filter = strings.ToLower(strings.Trim(strings.ReplaceAll(filter, "-", ""), " "))

		b := strings.Contains(name, filter)
		if b {
			return true
		}
	}

	return false
}

func checkFilterTags(tags []string, filter string) bool {
	filterTags := strings.Split(filter, ",")

	for _, tag := range tags {
		tag = strings.ToLower(tag)

		for _, filter := range filterTags {
			filter = strings.ToLower(filter)
			filter = strings.Trim(filter, " ")
			// check if line contains * or ?, if so try wildcard match, otherwise try substring match
			a := strings.ContainsAny(filter, "?|*")
			if a {
				match := wildcard.Match(filter, tag)
				if match {
					return true
				}
			} else {
				b := strings.Contains(tag, filter)
				if b {
					return true
				}
			}
		}
	}

	return false
}

func checkFreeleechPercent(announcePercent int, filterPercent string) bool {
	filters := strings.Split(filterPercent, ",")

	// remove % and trim spaces
	//announcePercent = strings.Replace(announcePercent, "%", "", -1)
	//announcePercent = strings.Trim(announcePercent, " ")

	//announcePercentInt, err := strconv.ParseInt(announcePercent, 10, 32)
	//if err != nil {
	//	return false
	//}

	for _, s := range filters {
		s = strings.Replace(s, "%", "", -1)
		s = strings.Trim(s, " ")

		if strings.Contains(s, "-") {
			minMax := strings.Split(s, "-")

			// to int
			min, err := strconv.ParseInt(minMax[0], 10, 32)
			if err != nil {
				return false
			}

			max, err := strconv.ParseInt(minMax[1], 10, 32)
			if err != nil {
				return false
			}

			if min > max {
				// handle error
				return false
			} else {
				// if announcePercent is greater than min and less than max return true
				if announcePercent >= int(min) && announcePercent <= int(max) {
					return true
				}
			}
		}

		filterPercentInt, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return false
		}

		if int(filterPercentInt) == announcePercent {
			return true
		}
	}

	return false
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

func getFirstStringMapValue(stringMap map[string]string, keys []string) (string, error) {
	for _, k := range keys {
		if val, err := getStringMapValue(stringMap, k); err == nil {
			return val, nil
		}
	}

	return "", fmt.Errorf("key were not found in map: %q", strings.Join(keys, ", "))
}

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
	ReleasePushStatusMixed    ReleasePushStatus = "MIXED"   // For multiple actions, one might go and the other not
	ReleasePushStatusPending  ReleasePushStatus = "PENDING" // Initial status
)

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
	ReleaseImplementationIRC ReleaseImplementation = "IRC"
)

type QueryParams struct {
	Limit  uint64
	Offset uint64
	Cursor uint64
	Sort   map[string]string
	Filter map[string]string
	Search string
}
