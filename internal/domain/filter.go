package domain

import (
	"context"
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
	Origins             string    `json:"origins"`
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

	//if !f.Enabled {
	//	return nil, false
	//}

	// FIXME what if someone explicitly doesnt want scene, or toggles in filter. Make enum? 0,1,2? Yes, No, Dont care
	if f.Scene && r.IsScene != f.Scene {
		r.addRejection("wanted: scene")
	}

	if f.Freeleech && r.Freeleech != f.Freeleech {
		r.addRejection("wanted: freeleech")
	}

	if f.FreeleechPercent != "" && !checkFreeleechPercent(r.FreeleechPercent, f.FreeleechPercent) {
		r.addRejectionF("freeleech percent not matching. wanted: %v got: %v", f.FreeleechPercent, r.FreeleechPercent)
	}

	// check against TorrentName and Clean which is a cleaned name without (. _ -)
	if f.Shows != "" && !f.contains(r.Title, f.Shows) {
		r.addRejection("shows not matching")
	}

	if f.Seasons != "" && !checkFilterIntStrings(r.Season, f.Seasons) {
		r.addRejectionF("season not matching. wanted: %v got: %d", f.Seasons, r.Season)
	}

	if f.Episodes != "" && !checkFilterIntStrings(r.Episode, f.Episodes) {
		r.addRejectionF("episodes not matching. wanted: %v got: %d", f.Episodes, r.Episode)
	}

	// matchRelease
	// TODO allow to match against regex
	if f.MatchReleases != "" && !checkMultipleFilterStrings(f.MatchReleases, r.TorrentName, r.Clean) {
		r.addRejection("match release not matching")
	}

	if f.ExceptReleases != "" && checkMultipleFilterStrings(f.ExceptReleases, r.TorrentName, r.Clean) {
		r.addRejection("except_releases: unwanted release")
	}

	if f.MatchReleaseGroups != "" && !f.contains(r.Group, f.MatchReleaseGroups) {
		r.addRejectionF("release groups not matching. wanted: %v got: %v", f.MatchReleaseGroups, r.Group)
	}

	if f.ExceptReleaseGroups != "" && f.contains(r.Group, f.ExceptReleaseGroups) {
		r.addRejectionF("unwanted release group. unwanted: %v got: %v", f.ExceptReleaseGroups, r.Group)
	}

	if f.MatchUploaders != "" && !f.contains(r.Uploader, f.MatchUploaders) {
		r.addRejectionF("uploaders not matching. wanted: %v got: %v", f.MatchUploaders, r.Uploader)
	}

	if f.ExceptUploaders != "" && f.contains(r.Uploader, f.ExceptUploaders) {
		r.addRejectionF("unwanted uploaders. unwanted: %v got: %v", f.MatchUploaders, r.Uploader)
	}

	if len(f.Resolutions) > 0 && !checkFilterSlice(r.Resolution, f.Resolutions) {
		r.addRejectionF("resolution not matching. wanted: %v got: %v", f.Resolutions, r.Resolution)
	}

	if len(f.Codecs) > 0 && !f.sliceContainsSlice(r.CodecArr, f.Codecs) {
		r.addRejectionF("codec not matching. wanted: %v got: %v", f.Codecs, r.CodecArr)
	}

	if len(f.Sources) > 0 && !f.containsSlice(r.Source, f.Sources) {
		r.addRejectionF("source not matching. wanted: %v got: %v", f.Sources, r.Source)
	}

	if len(f.Containers) > 0 && !checkFilterSlice(r.Container, f.Containers) {
		r.addRejectionF("container not matching. wanted: %v got: %v", f.Containers, r.Container)
	}

	// HDR is parsed into the Codec slice from rls
	if len(f.MatchHDR) > 0 && !f.sliceContainsSlice(r.HDRArr, f.MatchHDR) {
		r.addRejectionF("hdr not matching. wanted: %v got: %v", f.MatchHDR, r.HDRArr)
	}

	// HDR is parsed into the Codec slice from rls
	if len(f.ExceptHDR) > 0 && f.sliceContainsSlice(r.HDRArr, f.ExceptHDR) {
		r.addRejectionF("hdr unwanted. %v got: %v", f.ExceptHDR, r.HDRArr)
	}

	if f.Years != "" && !checkFilterIntStrings(r.Year, f.Years) {
		r.addRejectionF("year not matching. wanted: %v got: %d", f.Years, r.Year)
	}

	if f.MatchCategories != "" && !f.contains(r.Category, f.MatchCategories) {
		r.addRejectionF("category not matching. wanted: %v got: %v", f.MatchCategories, r.Category)
	}

	if f.ExceptCategories != "" && f.contains(r.Category, f.ExceptCategories) {
		r.addRejectionF("category unwanted. %v got: %v", f.ExceptCategories, r.Category)
	}

	if len(f.MatchReleaseTypes) > 0 && !checkFilterSlice(r.Category, f.MatchReleaseTypes) {
		r.addRejectionF("release type not matching. wanted: %v got: %v", f.MatchReleaseTypes, r.Category)
	}

	if (f.MinSize != "" || f.MaxSize != "") && !f.CheckSizeFilter(r, f.MinSize, f.MaxSize) {
		r.addRejectionF("size not matching. wanted min: %v max: %v got: %v", f.MinSize, f.MaxSize, r.Size)
	}

	if f.Tags != "" && !f.containsAny(r.Tags, f.Tags) {
		r.addRejectionF("tags not matching. wanted: %v got: %v", f.Tags, r.Tags)
	}

	if f.ExceptTags != "" && f.containsAny(r.Tags, f.ExceptTags) {
		r.addRejectionF("tags unwanted. wanted: %v got: %v", f.ExceptTags, r.Tags)
	}

	if len(f.Artists) > 0 && !f.contains(r.TorrentName, f.Artists) {
		r.addRejection("artists not matching")
	}

	if len(f.Albums) > 0 && !f.contains(r.TorrentName, f.Albums) {
		r.addRejection("albums not matching")
	}

	// Perfect flac requires Cue, Log, Log Score 100, FLAC and 24bit Lossless
	if f.PerfectFlac {
		if !r.HasLog || !r.HasCue || r.LogScore != 100 || r.Format != "FLAC" && !checkFilterSlice(r.Quality, []string{"Lossless", "24bit Lossless"}) {
			r.addRejectionF("wanted: perfect flac. got: cue %v log %v log score %v format %v quality %v", r.HasCue, r.HasLog, r.LogScore, r.Format, r.Quality)
		}
	}

	if len(f.Formats) > 0 && !checkFilterSlice(r.Format, f.Formats) {
		r.addRejectionF("formats not matching. wanted: %v got: %v", f.Formats, r.Format)
	}

	if len(f.Quality) > 0 && !checkFilterSlice(r.Quality, f.Quality) {
		r.addRejectionF("quality not matching. wanted: %v got: %v", f.Quality, r.Quality)
	}

	if len(f.Media) > 0 && !checkFilterSource(r.Source, f.Media) {
		r.addRejectionF("media not matching. wanted: %v got: %v", f.Media, r.Source)
	}

	if f.Log && r.HasLog != f.Log {
		r.addRejection("wanted: log")
	}

	if f.Log && f.LogScore != 0 && r.LogScore != f.LogScore {
		r.addRejectionF("wanted: log score %v got: %v", f.LogScore, r.LogScore)
	}

	if f.Cue && r.HasCue != f.Cue {
		r.addRejection("wanted: cue")
	}

	if len(r.Rejections) > 0 {
		return r.Rejections, false
	}

	return nil, true
}

// CheckSizeFilter additional size check
// for indexers that doesn't announce size, like some gazelle based
// set flag r.AdditionalSizeCheckRequired if there's a size in the filter, otherwise go a head
// implement API for ptp,btn,ggn to check for size if needed
// for others pull down torrent and do check
func (f Filter) CheckSizeFilter(r *Release, minSize string, maxSize string) bool {

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

func (f Filter) contains(tag string, filter string) bool {
	return containsMatch([]string{tag}, strings.Split(filter, ","))
}

func (f Filter) containsSlice(tag string, filters []string) bool {
	return containsMatch([]string{tag}, filters)
}

func (f Filter) containsFilterList(tag string, filter string) bool {
	return containsMatch([]string{tag}, strings.Split(filter, ","))
}

func (f Filter) containsAny(tags []string, filter string) bool {
	return containsMatch(tags, strings.Split(filter, ","))
}

func (f Filter) sliceContainsSlice(tags []string, filters []string) bool {
	return containsMatchBasic(tags, filters)
}

func containsMatch(tags []string, filters []string) bool {
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
			} else if tag == filter {
				return true
			}
		}
	}

	return false
}

func containsMatchBasic(tags []string, filters []string) bool {
	for _, tag := range tags {
		tag = strings.ToLower(tag)

		for _, filter := range filters {
			filter = strings.ToLower(filter)
			filter = strings.Trim(filter, " ")

			if tag == filter {
				return true
			}

			//match := strings.Contains(tag, filter)
			//if match {
			//	return true
			//}
		}
	}

	return false
}

func (f Filter) containsAnySlice(tags []string, filters []string) bool {

	for _, tag := range tags {
		tag = strings.ToLower(tag)

		for _, filter := range filters {
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
