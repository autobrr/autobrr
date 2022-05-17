package domain

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/wildcard"

	"github.com/dustin/go-humanize"
)

/*
Works the same way as for autodl-irssi
https://autodl-community.github.io/autodl-irssi/configuration/filter/
*/

type FilterRepo interface {
	FindByID(ctx context.Context, filterID int) (*Filter, error)
	FindByIndexerIdentifier(indexer string) ([]Filter, error)
	ListFilters(ctx context.Context) ([]Filter, error)
	Store(ctx context.Context, filter Filter) (*Filter, error)
	Update(ctx context.Context, filter Filter) (*Filter, error)
	ToggleEnabled(ctx context.Context, filterID int, enabled bool) error
	Delete(ctx context.Context, filterID int) error
	StoreIndexerConnection(ctx context.Context, filterID int, indexerID int) error
	StoreIndexerConnections(ctx context.Context, filterID int, indexers []Indexer) error
	DeleteIndexerConnections(ctx context.Context, filterID int) error
}

type Filter struct {
	ID                  int       `json:"id"`
	Name                string    `json:"name"`
	Enabled             bool      `json:"enabled"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	MinSize             string    `json:"min_size"`
	MaxSize             string    `json:"max_size"`
	Delay               int       `json:"delay"`
	Priority            int32     `json:"priority"`
	MatchReleases       string    `json:"match_releases"`
	ExceptReleases      string    `json:"except_releases"`
	UseRegex            bool      `json:"use_regex"`
	MatchReleaseGroups  string    `json:"match_release_groups"`
	ExceptReleaseGroups string    `json:"except_release_groups"`
	Scene               bool      `json:"scene"`
	Origins             []string  `json:"origins"`
	Freeleech           bool      `json:"freeleech"`
	FreeleechPercent    string    `json:"freeleech_percent"`
	Shows               string    `json:"shows"`
	Seasons             string    `json:"seasons"`
	Episodes            string    `json:"episodes"`
	Resolutions         []string  `json:"resolutions"` // SD, 480i, 480p, 576p, 720p, 810p, 1080i, 1080p.
	Codecs              []string  `json:"codecs"`      // XviD, DivX, x264, h.264 (or h264), mpeg2 (or mpeg-2), VC-1 (or VC1), WMV, Remux, h.264 Remux (or h264 Remux), VC-1 Remux (or VC1 Remux).
	Sources             []string  `json:"sources"`     // DSR, PDTV, HDTV, HR.PDTV, HR.HDTV, DVDRip, DVDScr, BDr, BD5, BD9, BDRip, BRRip, DVDR, MDVDR, HDDVD, HDDVDRip, BluRay, WEB-DL, TVRip, CAM, R5, TELESYNC, TS, TELECINE, TC. TELESYNC and TS are synonyms (you don't need both). Same for TELECINE and TC
	Containers          []string  `json:"containers"`
	MatchHDR            []string  `json:"match_hdr"`
	ExceptHDR           []string  `json:"except_hdr"`
	MatchOther          []string  `json:"match_other"`
	ExceptOther         []string  `json:"except_other"`
	Years               string    `json:"years"`
	Artists             string    `json:"artists"`
	Albums              string    `json:"albums"`
	MatchReleaseTypes   []string  `json:"match_release_types"` // Album,Single,EP
	ExceptReleaseTypes  string    `json:"except_release_types"`
	Formats             []string  `json:"formats"` // MP3, FLAC, Ogg, AAC, AC3, DTS
	Quality             []string  `json:"quality"` // 192, 320, APS (VBR), V2 (VBR), V1 (VBR), APX (VBR), V0 (VBR), q8.x (VBR), Lossless, 24bit Lossless, Other
	Media               []string  `json:"media"`   // CD, DVD, Vinyl, Soundboard, SACD, DAT, Cassette, WEB, Other
	PerfectFlac         bool      `json:"perfect_flac"`
	Cue                 bool      `json:"cue"`
	Log                 bool      `json:"log"`
	LogScore            int       `json:"log_score"`
	MatchCategories     string    `json:"match_categories"`
	ExceptCategories    string    `json:"except_categories"`
	MatchUploaders      string    `json:"match_uploaders"`
	ExceptUploaders     string    `json:"except_uploaders"`
	Tags                string    `json:"tags"`
	ExceptTags          string    `json:"except_tags"`
	TagsAny             string    `json:"tags_any"`
	ExceptTagsAny       string    `json:"except_tags_any"`
	Actions             []*Action `json:"actions"`
	Indexers            []Indexer `json:"indexers"`
}

func (f Filter) CheckFilter(r *Release) ([]string, bool) {
	// reset rejections first to clean previous checks
	r.resetRejections()

	if f.Freeleech && r.Freeleech != f.Freeleech {
		r.addRejection("wanted: freeleech")
	}

	if f.FreeleechPercent != "" && !checkFreeleechPercent(r.FreeleechPercent, f.FreeleechPercent) {
		r.addRejectionF("freeleech percent not matching. got: %v want: %v", r.FreeleechPercent, f.FreeleechPercent)
	}

	if len(f.Origins) > 0 && !containsSlice(r.Origin, f.Origins) {
		r.addRejectionF("origin not matching. got: %v want: %v", r.Origin, f.Origins)
	}

	// title is the parsed title
	if f.Shows != "" && !contains(r.Title, f.Shows) {
		r.addRejectionF("shows not matching. got: %v want: %v", r.Title, f.Shows)
	}

	if f.Seasons != "" && !containsIntStrings(r.Season, f.Seasons) {
		r.addRejectionF("season not matching. got: %d want: %v", r.Season, f.Seasons)
	}

	if f.Episodes != "" && !containsIntStrings(r.Episode, f.Episodes) {
		r.addRejectionF("episodes not matching. got: %d want: %v", r.Episode, f.Episodes)
	}

	// matchRelease
	// match against regex
	if f.UseRegex {
		if f.MatchReleases != "" && !matchRegex(r.TorrentName, f.MatchReleases) {
			r.addRejectionF("match release regex not matching. got: %v want: %v", r.TorrentName, f.MatchReleases)
		}

		if f.ExceptReleases != "" && matchRegex(r.TorrentName, f.ExceptReleases) {
			r.addRejectionF("except releases regex: unwanted release. got: %v want: %v", r.TorrentName, f.ExceptReleases)
		}

	} else {
		if f.MatchReleases != "" && !containsFuzzy(r.TorrentName, f.MatchReleases) {
			r.addRejectionF("match release not matching. got: %v want: %v", r.TorrentName, f.MatchReleases)
		}

		if f.ExceptReleases != "" && containsFuzzy(r.TorrentName, f.ExceptReleases) {
			r.addRejectionF("except releases: unwanted release. got: %v want: %v", r.TorrentName, f.ExceptReleases)
		}
	}

	if f.MatchReleaseGroups != "" && !contains(r.Group, f.MatchReleaseGroups) {
		r.addRejectionF("release groups not matching. got: %v want: %v", r.Group, f.MatchReleaseGroups)
	}

	if f.ExceptReleaseGroups != "" && contains(r.Group, f.ExceptReleaseGroups) {
		r.addRejectionF("unwanted release group. got: %v unwanted: %v", r.Group, f.ExceptReleaseGroups)
	}

	if f.MatchUploaders != "" && !contains(r.Uploader, f.MatchUploaders) {
		r.addRejectionF("uploaders not matching. got: %v want: %v", r.Uploader, f.MatchUploaders)
	}

	if f.ExceptUploaders != "" && contains(r.Uploader, f.ExceptUploaders) {
		r.addRejectionF("unwanted uploaders. got: %v unwanted: %v", r.Uploader, f.ExceptUploaders)
	}

	if len(f.Resolutions) > 0 && !containsSlice(r.Resolution, f.Resolutions) {
		r.addRejectionF("resolution not matching. got: %v want: %v", r.Resolution, f.Resolutions)
	}

	if len(f.Codecs) > 0 && !sliceContainsSlice(r.Codec, f.Codecs) {
		r.addRejectionF("codec not matching. got: %v want: %v", r.Codec, f.Codecs)
	}

	if len(f.Sources) > 0 && !containsSlice(r.Source, f.Sources) {
		r.addRejectionF("source not matching. got: %v want: %v", r.Source, f.Sources)
	}

	if len(f.Containers) > 0 && !containsSlice(r.Container, f.Containers) {
		r.addRejectionF("container not matching. got: %v want: %v", r.Container, f.Containers)
	}

	// HDR is parsed into the Codec slice from rls
	if len(f.MatchHDR) > 0 && !sliceContainsSlice(r.HDR, f.MatchHDR) {
		r.addRejectionF("hdr not matching. got: %v want: %v", r.HDR, f.MatchHDR)
	}

	// HDR is parsed into the Codec slice from rls
	if len(f.ExceptHDR) > 0 && sliceContainsSlice(r.HDR, f.ExceptHDR) {
		r.addRejectionF("hdr unwanted. got: %v want: %v", r.HDR, f.ExceptHDR)
	}

	if f.Years != "" && !containsIntStrings(r.Year, f.Years) {
		r.addRejectionF("year not matching. got: %d want: %v", r.Year, f.Years)
	}

	if f.MatchCategories != "" && !contains(r.Category, f.MatchCategories) {
		r.addRejectionF("category not matching. got: %v want: %v", r.Category, f.MatchCategories)
	}

	if f.ExceptCategories != "" && contains(r.Category, f.ExceptCategories) {
		r.addRejectionF("category unwanted. got: %v want: %v", r.Category, f.ExceptCategories)
	}

	if len(f.MatchReleaseTypes) > 0 && !containsSlice(r.Category, f.MatchReleaseTypes) {
		r.addRejectionF("release type not matching. got: %v want: %v", r.Category, f.MatchReleaseTypes)
	}

	if (f.MinSize != "" || f.MaxSize != "") && !f.checkSizeFilter(r, f.MinSize, f.MaxSize) {
		r.addRejectionF("size not matching. got: %v want min: %v max: %v", r.Size, f.MinSize, f.MaxSize)
	}

	if f.Tags != "" && !containsAny(r.Tags, f.Tags) {
		r.addRejectionF("tags not matching. got: %v want: %v", r.Tags, f.Tags)
	}

	if f.ExceptTags != "" && containsAny(r.Tags, f.ExceptTags) {
		r.addRejectionF("tags unwanted. got: %v want: %v", r.Tags, f.ExceptTags)
	}

	if len(f.Artists) > 0 && !containsFuzzy(r.TorrentName, f.Artists) {
		r.addRejectionF("artists not matching. got: %v want: %v", r.TorrentName, f.Artists)
	}

	if len(f.Albums) > 0 && !containsFuzzy(r.TorrentName, f.Albums) {
		r.addRejectionF("albums not matching. got: %v want: %v", r.TorrentName, f.Albums)
	}

	// Perfect flac requires Cue, Log, Log Score 100, FLAC and 24bit Lossless
	if f.PerfectFlac && !f.isPerfectFLAC(r) {
		r.addRejectionF("wanted: perfect flac. got: %v", r.Audio)
	}

	if len(f.Formats) > 0 && !sliceContainsSlice(r.Audio, f.Formats) {
		r.addRejectionF("formats not matching. got: %v want: %v", r.Audio, f.Formats)
	}

	if len(f.Quality) > 0 && !sliceContainsSlice(r.Audio, f.Quality) {
		r.addRejectionF("quality not matching. got: %v want: %v", r.Audio, f.Quality)
	}

	if len(f.Media) > 0 && !containsSlice(r.Source, f.Media) {
		r.addRejectionF("media not matching. got: %v want: %v", r.Source, f.Media)
	}

	if f.Cue && !containsAny(r.Audio, "Cue") {
		r.addRejection("wanted: cue")
	}

	if f.Log && !containsAny(r.Audio, "Log") {
		r.addRejection("wanted: log")
	}

	if f.Log && f.LogScore != 0 && r.LogScore != f.LogScore {
		r.addRejectionF("log score. got: %v want: %v", r.LogScore, f.LogScore)
	}

	if len(r.Rejections) > 0 {
		return r.Rejections, false
	}

	return nil, true
}

// isPerfectFLAC Perfect is "CD FLAC Cue Log 100% Lossless or 24bit Lossless"
func (f Filter) isPerfectFLAC(r *Release) bool {
	if !contains(r.Source, "CD") {
		return false
	}
	if !containsAny(r.Audio, "Cue") {
		return false
	}
	if !containsAny(r.Audio, "Log") {
		return false
	}
	if !containsAny(r.Audio, "Log100") {
		return false
	}
	if !containsAny(r.Audio, "FLAC") {
		return false
	}
	if !containsAnySlice(r.Audio, []string{"Lossless", "24bit Lossless"}) {
		return false
	}

	return true
}

// checkSizeFilter additional size check
// for indexers that doesn't announce size, like some gazelle based
// set flag r.AdditionalSizeCheckRequired if there's a size in the filter, otherwise go a head
// implement API for ptp,btn,ggn to check for size if needed
// for others pull down torrent and do check
func (f Filter) checkSizeFilter(r *Release, minSize string, maxSize string) bool {

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

func matchRegex(tag string, filterList string) bool {
	if tag == "" {
		return false
	}
	filters := strings.Split(filterList, ",")

	for _, filter := range filters {
		if filter == "" {
			continue
		}
		re, err := regexp.Compile(`(?i)(?:` + filter + `)`)
		if err != nil {
			return false
		}
		match := re.MatchString(tag)
		if match {
			return true
		}
	}

	return false
}

// checkFilterIntStrings "1,2,3-20"
func containsIntStrings(value int, filterList string) bool {
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

func contains(tag string, filter string) bool {
	return containsMatch([]string{tag}, strings.Split(filter, ","))
}

func containsFuzzy(tag string, filter string) bool {
	return containsMatchFuzzy([]string{tag}, strings.Split(filter, ","))
}

func containsSlice(tag string, filters []string) bool {
	return containsMatch([]string{tag}, filters)
}

func containsAny(tags []string, filter string) bool {
	return containsMatch(tags, strings.Split(filter, ","))
}

func sliceContainsSlice(tags []string, filters []string) bool {
	return containsMatchBasic(tags, filters)
}

func containsMatchFuzzy(tags []string, filters []string) bool {
	for _, tag := range tags {
		tag = strings.ToLower(tag)

		for _, filter := range filters {
			filter = strings.ToLower(filter)
			filter = strings.Trim(filter, " ")
			// check if line contains * or ?, if so try wildcard match, otherwise try substring match
			a := strings.ContainsAny(filter, "?|*")
			if a {
				match := wildcard.Match(filter, tag)
				if match {
					return true
				}
			} else if strings.Contains(tag, filter) {
				return true
			}
		}
	}

	return false
}

func containsMatch(tags []string, filters []string) bool {
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		tag = strings.ToLower(tag)

		for _, filter := range filters {
			if filter == "" {
				continue
			}
			filter = strings.ToLower(filter)
			filter = strings.Trim(filter, " ")
			// check if line contains * or ?, if so try wildcard match, otherwise try substring match
			a := strings.ContainsAny(filter, "?|*")
			if a {
				match := wildcard.Match(filter, tag)
				if match {
					return true
				}
			} else if tag == filter {
				return true
			}
		}
	}

	return false
}

func containsMatchBasic(tags []string, filters []string) bool {
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		tag = strings.ToLower(tag)

		for _, filter := range filters {
			if filter == "" {
				continue
			}
			filter = strings.ToLower(filter)
			filter = strings.Trim(filter, " ")

			if tag == filter {
				return true
			}
		}
	}

	return false
}

func containsAnySlice(tags []string, filters []string) bool {
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		tag = strings.ToLower(tag)

		for _, filter := range filters {
			if filter == "" {
				continue
			}
			filter = strings.ToLower(filter)
			filter = strings.Trim(filter, " ")
			// check if line contains * or ?, if so try wildcard match, otherwise try substring match
			wild := strings.ContainsAny(filter, "?|*")
			if wild {
				match := wildcard.Match(filter, tag)
				if match {
					return true
				}
			} else if tag == filter {
				return true
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
