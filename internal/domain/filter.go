// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/regexcache"
	"github.com/autobrr/autobrr/pkg/sanitize"
	"github.com/autobrr/autobrr/pkg/wildcard"

	"github.com/dustin/go-humanize"
	"github.com/go-andiamo/splitter"
)

/*
Works the same way as for autodl-irssi
https://autodl-community.github.io/autodl-irssi/configuration/filter/
*/

type FilterRepo interface {
	ListFilters(ctx context.Context) ([]Filter, error)
	Find(ctx context.Context, params FilterQueryParams) ([]Filter, error)
	FindByID(ctx context.Context, filterID int) (*Filter, error)
	FindByIndexerIdentifier(ctx context.Context, indexer string) ([]*Filter, error)
	FindExternalFiltersByID(ctx context.Context, filterId int) ([]FilterExternal, error)
	Store(ctx context.Context, filter *Filter) error
	Update(ctx context.Context, filter *Filter) error
	UpdatePartial(ctx context.Context, filter FilterUpdate) error
	ToggleEnabled(ctx context.Context, filterID int, enabled bool) error
	Delete(ctx context.Context, filterID int) error
	StoreIndexerConnection(ctx context.Context, filterID int, indexerID int) error
	StoreIndexerConnections(ctx context.Context, filterID int, indexers []Indexer) error
	StoreFilterExternal(ctx context.Context, filterID int, externalFilters []FilterExternal) error
	DeleteIndexerConnections(ctx context.Context, filterID int) error
	DeleteFilterExternal(ctx context.Context, filterID int) error
	GetDownloadsByFilterId(ctx context.Context, filterID int) (*FilterDownloads, error)
}

type FilterDownloads struct {
	HourCount  int
	DayCount   int
	WeekCount  int
	MonthCount int
	TotalCount int
}

type FilterMaxDownloadsUnit string

const (
	FilterMaxDownloadsHour  FilterMaxDownloadsUnit = "HOUR"
	FilterMaxDownloadsDay   FilterMaxDownloadsUnit = "DAY"
	FilterMaxDownloadsWeek  FilterMaxDownloadsUnit = "WEEK"
	FilterMaxDownloadsMonth FilterMaxDownloadsUnit = "MONTH"
	FilterMaxDownloadsEver  FilterMaxDownloadsUnit = "EVER"
)

type SmartEpisodeParams struct {
	Title   string
	Season  int
	Episode int
	Year    int
	Month   int
	Day     int
	Repack  bool
	Proper  bool
	Group   string
}

func (p *SmartEpisodeParams) IsDailyEpisode() bool {
	return p.Year != 0 && p.Month != 0 && p.Day != 0
}

type FilterQueryParams struct {
	Sort    map[string]string
	Filters struct {
		Indexers []string
	}
	Search string
}

type Filter struct {
	ID                   int                    `json:"id"`
	Name                 string                 `json:"name"`
	Enabled              bool                   `json:"enabled"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
	MinSize              string                 `json:"min_size,omitempty"`
	MaxSize              string                 `json:"max_size,omitempty"`
	Delay                int                    `json:"delay,omitempty"`
	Priority             int32                  `json:"priority"`
	MaxDownloads         int                    `json:"max_downloads,omitempty"`
	MaxDownloadsUnit     FilterMaxDownloadsUnit `json:"max_downloads_unit,omitempty"`
	MatchReleases        string                 `json:"match_releases,omitempty"`
	ExceptReleases       string                 `json:"except_releases,omitempty"`
	UseRegex             bool                   `json:"use_regex,omitempty"`
	MatchReleaseGroups   string                 `json:"match_release_groups,omitempty"`
	ExceptReleaseGroups  string                 `json:"except_release_groups,omitempty"`
	Scene                bool                   `json:"scene,omitempty"`
	Origins              []string               `json:"origins,omitempty"`
	ExceptOrigins        []string               `json:"except_origins,omitempty"`
	Bonus                []string               `json:"bonus,omitempty"`
	Freeleech            bool                   `json:"freeleech,omitempty"`
	FreeleechPercent     string                 `json:"freeleech_percent,omitempty"`
	SmartEpisode         bool                   `json:"smart_episode"`
	Shows                string                 `json:"shows,omitempty"`
	Seasons              string                 `json:"seasons,omitempty"`
	Episodes             string                 `json:"episodes,omitempty"`
	Resolutions          []string               `json:"resolutions,omitempty"` // SD, 480i, 480p, 576p, 720p, 810p, 1080i, 1080p.
	Codecs               []string               `json:"codecs,omitempty"`      // XviD, DivX, x264, h.264 (or h264), mpeg2 (or mpeg-2), VC-1 (or VC1), WMV, Remux, h.264 Remux (or h264 Remux), VC-1 Remux (or VC1 Remux).
	Sources              []string               `json:"sources,omitempty"`     // DSR, PDTV, HDTV, HR.PDTV, HR.HDTV, DVDRip, DVDScr, BDr, BD5, BD9, BDRip, BRRip, DVDR, MDVDR, HDDVD, HDDVDRip, BluRay, WEB-DL, TVRip, CAM, R5, TELESYNC, TS, TELECINE, TC. TELESYNC and TS are synonyms (you don't need both). Same for TELECINE and TC
	Containers           []string               `json:"containers,omitempty"`
	MatchHDR             []string               `json:"match_hdr,omitempty"`
	ExceptHDR            []string               `json:"except_hdr,omitempty"`
	MatchOther           []string               `json:"match_other,omitempty"`
	ExceptOther          []string               `json:"except_other,omitempty"`
	Years                string                 `json:"years,omitempty"`
	Months               string                 `json:"months,omitempty"`
	Days                 string                 `json:"days,omitempty"`
	Artists              string                 `json:"artists,omitempty"`
	Albums               string                 `json:"albums,omitempty"`
	MatchReleaseTypes    []string               `json:"match_release_types,omitempty"` // Album,Single,EP
	ExceptReleaseTypes   string                 `json:"except_release_types,omitempty"`
	Formats              []string               `json:"formats,omitempty"` // MP3, FLAC, Ogg, AAC, AC3, DTS
	Quality              []string               `json:"quality,omitempty"` // 192, 320, APS (VBR), V2 (VBR), V1 (VBR), APX (VBR), V0 (VBR), q8.x (VBR), Lossless, 24bit Lossless, Other
	Media                []string               `json:"media,omitempty"`   // CD, DVD, Vinyl, Soundboard, SACD, DAT, Cassette, WEB, Other
	PerfectFlac          bool                   `json:"perfect_flac,omitempty"`
	Cue                  bool                   `json:"cue,omitempty"`
	Log                  bool                   `json:"log,omitempty"`
	LogScore             int                    `json:"log_score,omitempty"`
	MatchCategories      string                 `json:"match_categories,omitempty"`
	ExceptCategories     string                 `json:"except_categories,omitempty"`
	MatchUploaders       string                 `json:"match_uploaders,omitempty"`
	ExceptUploaders      string                 `json:"except_uploaders,omitempty"`
	MatchLanguage        []string               `json:"match_language,omitempty"`
	ExceptLanguage       []string               `json:"except_language,omitempty"`
	Tags                 string                 `json:"tags,omitempty"`
	ExceptTags           string                 `json:"except_tags,omitempty"`
	TagsAny              string                 `json:"tags_any,omitempty"`
	ExceptTagsAny        string                 `json:"except_tags_any,omitempty"`
	TagsMatchLogic       string                 `json:"tags_match_logic,omitempty"`
	ExceptTagsMatchLogic string                 `json:"except_tags_match_logic,omitempty"`
	MatchReleaseTags     string                 `json:"match_release_tags,omitempty"`
	ExceptReleaseTags    string                 `json:"except_release_tags,omitempty"`
	UseRegexReleaseTags  bool                   `json:"use_regex_release_tags,omitempty"`
	MatchDescription     string                 `json:"match_description,omitempty"`
	ExceptDescription    string                 `json:"except_description,omitempty"`
	UseRegexDescription  bool                   `json:"use_regex_description,omitempty"`
	MinSeeders           int                    `json:"min_seeders,omitempty"`
	MaxSeeders           int                    `json:"max_seeders,omitempty"`
	MinLeechers          int                    `json:"min_leechers,omitempty"`
	MaxLeechers          int                    `json:"max_leechers,omitempty"`
	ActionsCount         int                    `json:"actions_count"`
	ActionsEnabledCount  int                    `json:"actions_enabled_count"`
	Actions              []*Action              `json:"actions,omitempty"`
	External             []FilterExternal       `json:"external,omitempty"`
	Indexers             []Indexer              `json:"indexers"`
	Downloads            *FilterDownloads       `json:"-"`
	Rejections           []string               `json:"-"`
}

type FilterExternal struct {
	ID                       int                `json:"id"`
	Name                     string             `json:"name"`
	Index                    int                `json:"index"`
	Type                     FilterExternalType `json:"type"`
	Enabled                  bool               `json:"enabled"`
	ExecCmd                  string             `json:"exec_cmd,omitempty"`
	ExecArgs                 string             `json:"exec_args,omitempty"`
	ExecExpectStatus         int                `json:"exec_expect_status,omitempty"`
	WebhookHost              string             `json:"webhook_host,omitempty"`
	WebhookMethod            string             `json:"webhook_method,omitempty"`
	WebhookData              string             `json:"webhook_data,omitempty"`
	WebhookHeaders           string             `json:"webhook_headers,omitempty"`
	WebhookExpectStatus      int                `json:"webhook_expect_status,omitempty"`
	WebhookRetryStatus       string             `json:"webhook_retry_status,omitempty"`
	WebhookRetryAttempts     int                `json:"webhook_retry_attempts,omitempty"`
	WebhookRetryDelaySeconds int                `json:"webhook_retry_delay_seconds,omitempty"`
	FilterId                 int                `json:"-"`
}

func (f FilterExternal) NeedTorrentDownloaded() bool {
	if strings.Contains(f.ExecArgs, "TorrentHash") || strings.Contains(f.WebhookData, "TorrentHash") {
		return true
	}

	if strings.Contains(f.ExecArgs, "TorrentPathName") || strings.Contains(f.WebhookData, "TorrentPathName") {
		return true
	}

	if strings.Contains(f.WebhookData, "TorrentDataRawBytes") {
		return true
	}

	return false
}

type FilterExternalType string

const (
	ExternalFilterTypeExec    FilterExternalType = "EXEC"
	ExternalFilterTypeWebhook FilterExternalType = "WEBHOOK"
)

type FilterUpdate struct {
	ID                   int                     `json:"id"`
	Name                 *string                 `json:"name,omitempty"`
	Enabled              *bool                   `json:"enabled,omitempty"`
	MinSize              *string                 `json:"min_size,omitempty"`
	MaxSize              *string                 `json:"max_size,omitempty"`
	Delay                *int                    `json:"delay,omitempty"`
	Priority             *int32                  `json:"priority,omitempty"`
	MaxDownloads         *int                    `json:"max_downloads,omitempty"`
	MaxDownloadsUnit     *FilterMaxDownloadsUnit `json:"max_downloads_unit,omitempty"`
	MatchReleases        *string                 `json:"match_releases,omitempty"`
	ExceptReleases       *string                 `json:"except_releases,omitempty"`
	UseRegex             *bool                   `json:"use_regex,omitempty"`
	MatchReleaseGroups   *string                 `json:"match_release_groups,omitempty"`
	ExceptReleaseGroups  *string                 `json:"except_release_groups,omitempty"`
	MatchReleaseTags     *string                 `json:"match_release_tags,omitempty"`
	ExceptReleaseTags    *string                 `json:"except_release_tags,omitempty"`
	UseRegexReleaseTags  *bool                   `json:"use_regex_release_tags,omitempty"`
	MatchDescription     *string                 `json:"match_description,omitempty"`
	ExceptDescription    *string                 `json:"except_description,omitempty"`
	UseRegexDescription  *bool                   `json:"use_regex_description,omitempty"`
	Scene                *bool                   `json:"scene,omitempty"`
	Origins              *[]string               `json:"origins,omitempty"`
	ExceptOrigins        *[]string               `json:"except_origins,omitempty"`
	Bonus                *[]string               `json:"bonus,omitempty"`
	Freeleech            *bool                   `json:"freeleech,omitempty"`
	FreeleechPercent     *string                 `json:"freeleech_percent,omitempty"`
	SmartEpisode         *bool                   `json:"smart_episode,omitempty"`
	Shows                *string                 `json:"shows,omitempty"`
	Seasons              *string                 `json:"seasons,omitempty"`
	Episodes             *string                 `json:"episodes,omitempty"`
	Resolutions          *[]string               `json:"resolutions,omitempty"` // SD, 480i, 480p, 576p, 720p, 810p, 1080i, 1080p.
	Codecs               *[]string               `json:"codecs,omitempty"`      // XviD, DivX, x264, h.264 (or h264), mpeg2 (or mpeg-2), VC-1 (or VC1), WMV, Remux, h.264 Remux (or h264 Remux), VC-1 Remux (or VC1 Remux).
	Sources              *[]string               `json:"sources,omitempty"`     // DSR, PDTV, HDTV, HR.PDTV, HR.HDTV, DVDRip, DVDScr, BDr, BD5, BD9, BDRip, BRRip, DVDR, MDVDR, HDDVD, HDDVDRip, BluRay, WEB-DL, TVRip, CAM, R5, TELESYNC, TS, TELECINE, TC. TELESYNC and TS are synonyms (you don't need both). Same for TELECINE and TC
	Containers           *[]string               `json:"containers,omitempty"`
	MatchHDR             *[]string               `json:"match_hdr,omitempty"`
	ExceptHDR            *[]string               `json:"except_hdr,omitempty"`
	MatchOther           *[]string               `json:"match_other,omitempty"`
	ExceptOther          *[]string               `json:"except_other,omitempty"`
	Years                *string                 `json:"years,omitempty"`
	Months               *string                 `json:"months,omitempty"`
	Days                 *string                 `json:"days,omitempty"`
	Artists              *string                 `json:"artists,omitempty"`
	Albums               *string                 `json:"albums,omitempty"`
	MatchReleaseTypes    *[]string               `json:"match_release_types,omitempty"` // Album,Single,EP
	ExceptReleaseTypes   *string                 `json:"except_release_types,omitempty"`
	Formats              *[]string               `json:"formats,omitempty"` // MP3, FLAC, Ogg, AAC, AC3, DTS
	Quality              *[]string               `json:"quality,omitempty"` // 192, 320, APS (VBR), V2 (VBR), V1 (VBR), APX (VBR), V0 (VBR), q8.x (VBR), Lossless, 24bit Lossless, Other
	Media                *[]string               `json:"media,omitempty"`   // CD, DVD, Vinyl, Soundboard, SACD, DAT, Cassette, WEB, Other
	PerfectFlac          *bool                   `json:"perfect_flac,omitempty"`
	Cue                  *bool                   `json:"cue,omitempty"`
	Log                  *bool                   `json:"log,omitempty"`
	LogScore             *int                    `json:"log_score,omitempty"`
	MatchCategories      *string                 `json:"match_categories,omitempty"`
	ExceptCategories     *string                 `json:"except_categories,omitempty"`
	MatchUploaders       *string                 `json:"match_uploaders,omitempty"`
	ExceptUploaders      *string                 `json:"except_uploaders,omitempty"`
	MatchLanguage        *[]string               `json:"match_language,omitempty"`
	ExceptLanguage       *[]string               `json:"except_language,omitempty"`
	Tags                 *string                 `json:"tags,omitempty"`
	ExceptTags           *string                 `json:"except_tags,omitempty"`
	TagsAny              *string                 `json:"tags_any,omitempty"`
	ExceptTagsAny        *string                 `json:"except_tags_any,omitempty"`
	TagsMatchLogic       *string                 `json:"tags_match_logic,omitempty"`
	ExceptTagsMatchLogic *string                 `json:"except_tags_match_logic,omitempty"`
	MinSeeders           *int                    `json:"min_seeders,omitempty"`
	MaxSeeders           *int                    `json:"max_seeders,omitempty"`
	MinLeechers          *int                    `json:"min_leechers,omitempty"`
	MaxLeechers          *int                    `json:"max_leechers,omitempty"`
	Actions              []*Action               `json:"actions,omitempty"`
	External             []FilterExternal        `json:"external,omitempty"`
	Indexers             []Indexer               `json:"indexers,omitempty"`
}

func (f *Filter) Validate() error {
	if f.Name == "" {
		return errors.New("validation: name can't be empty")
	}

	if _, _, err := f.parsedSizeLimits(); err != nil {
		return fmt.Errorf("error validating filter size limits: %w", err)
	}

	for _, external := range f.External {
		if external.Type == ExternalFilterTypeExec {
			if external.ExecCmd != "" && external.Enabled {
				// check if program exists
				_, err := exec.LookPath(external.ExecCmd)
				if err != nil {
					return errors.Wrap(err, "could not find external exec command: %s", external.ExecCmd)
				}
			}
		}
	}

	for _, action := range f.Actions {
		if action.Type == ActionTypeExec {
			if action.ExecCmd != "" && action.Enabled {
				// check if program exists
				_, err := exec.LookPath(action.ExecCmd)
				if err != nil {
					return errors.Wrap(err, "could not find action exec command: %s", action.ExecCmd)
				}
			}
		}
	}

	return nil
}

func (f *Filter) Sanitize() error {
	f.Shows = sanitize.FilterString(f.Shows)

	if !f.UseRegex {
		f.MatchReleases = sanitize.FilterString(f.MatchReleases)
		f.ExceptReleases = sanitize.FilterString(f.ExceptReleases)
	}

	f.MatchReleaseGroups = sanitize.FilterString(f.MatchReleaseGroups)
	f.ExceptReleaseGroups = sanitize.FilterString(f.ExceptReleaseGroups)

	f.MatchCategories = sanitize.FilterString(f.MatchCategories)
	f.ExceptCategories = sanitize.FilterString(f.ExceptCategories)

	f.MatchUploaders = sanitize.FilterString(f.MatchUploaders)
	f.ExceptUploaders = sanitize.FilterString(f.ExceptUploaders)

	f.TagsAny = sanitize.FilterString(f.TagsAny)
	f.ExceptTags = sanitize.FilterString(f.ExceptTags)

	if !f.UseRegexReleaseTags {
		f.MatchReleaseTags = sanitize.FilterString(f.MatchReleaseTags)
		f.ExceptReleaseTags = sanitize.FilterString(f.ExceptReleaseTags)
	}

	f.Years = sanitize.FilterString(f.Years)
	f.Months = sanitize.FilterString(f.Months)
	f.Days = sanitize.FilterString(f.Days)

	f.Artists = sanitize.FilterString(f.Artists)
	f.Albums = sanitize.FilterString(f.Albums)

	return nil
}

func (f *Filter) CheckFilter(r *Release) ([]string, bool) {
	// max downloads check. If reached return early so other filters can be checked as quick as possible.
	if f.MaxDownloads > 0 && !f.checkMaxDownloads() {
		f.addRejectionF("max downloads (%d) this (%v) reached", f.MaxDownloads, f.MaxDownloadsUnit)
		return f.Rejections, false
	}

	if len(f.Bonus) > 0 && !sliceContainsSlice(r.Bonus, f.Bonus) {
		r.addRejectionF("bonus not matching. got: %v want: %v", r.Bonus, f.Bonus)
	}

	if f.Freeleech && r.Freeleech != f.Freeleech {
		f.addRejection("wanted: freeleech")
	}

	if f.FreeleechPercent != "" && !checkFreeleechPercent(r.FreeleechPercent, f.FreeleechPercent) {
		f.addRejectionF("freeleech percent not matching. got: %v want: %v", r.FreeleechPercent, f.FreeleechPercent)
	}

	if len(f.Origins) > 0 && !containsSlice(r.Origin, f.Origins) {
		f.addRejectionF("origin not matching. got: %v want: %v", r.Origin, f.Origins)
	}
	if len(f.ExceptOrigins) > 0 && containsSlice(r.Origin, f.ExceptOrigins) {
		f.addRejectionF("except origin not matching. got: %v unwanted: %v", r.Origin, f.ExceptOrigins)
	}

	// title is the parsed title
	if f.Shows != "" && !contains(r.Title, f.Shows) {
		f.addRejectionF("shows not matching. got: %v want: %v", r.Title, f.Shows)
	}

	if f.Seasons != "" && !containsIntStrings(r.Season, f.Seasons) {
		f.addRejectionF("season not matching. got: %d want: %v", r.Season, f.Seasons)
	}

	if f.Episodes != "" && !containsIntStrings(r.Episode, f.Episodes) {
		f.addRejectionF("episodes not matching. got: %d want: %v", r.Episode, f.Episodes)
	}

	// matchRelease
	// match against regex
	if f.UseRegex {
		if f.MatchReleases != "" && !matchRegex(r.TorrentName, f.MatchReleases) {
			f.addRejectionF("match release regex not matching. got: %v want: %v", r.TorrentName, f.MatchReleases)
		}

		if f.ExceptReleases != "" && matchRegex(r.TorrentName, f.ExceptReleases) {
			f.addRejectionF("except releases regex: unwanted release. got: %v want: %v", r.TorrentName, f.ExceptReleases)
		}

	} else {
		if f.MatchReleases != "" && !containsFuzzy(r.TorrentName, f.MatchReleases) {
			f.addRejectionF("match release not matching. got: %v want: %v", r.TorrentName, f.MatchReleases)
		}

		if f.ExceptReleases != "" && containsFuzzy(r.TorrentName, f.ExceptReleases) {
			f.addRejectionF("except releases: unwanted release. got: %v want: %v", r.TorrentName, f.ExceptReleases)
		}
	}

	if f.MatchReleaseGroups != "" && !contains(r.Group, f.MatchReleaseGroups) {
		f.addRejectionF("release groups not matching. got: %v want: %v", r.Group, f.MatchReleaseGroups)
	}

	if f.ExceptReleaseGroups != "" && contains(r.Group, f.ExceptReleaseGroups) {
		f.addRejectionF("unwanted release group. got: %v unwanted: %v", r.Group, f.ExceptReleaseGroups)
	}

	// check raw releaseTags string
	if f.UseRegexReleaseTags {
		if f.MatchReleaseTags != "" && !matchRegex(r.ReleaseTags, f.MatchReleaseTags) {
			f.addRejectionF("match release tags regex not matching. got: %v want: %v", r.ReleaseTags, f.MatchReleaseTags)
		}

		if f.ExceptReleaseTags != "" && matchRegex(r.ReleaseTags, f.ExceptReleaseTags) {
			f.addRejectionF("except release tags regex: unwanted release. got: %v want: %v", r.ReleaseTags, f.ExceptReleaseTags)
		}

	} else {
		if f.MatchReleaseTags != "" && !containsFuzzy(r.ReleaseTags, f.MatchReleaseTags) {
			f.addRejectionF("match release tags not matching. got: %v want: %v", r.ReleaseTags, f.MatchReleaseTags)
		}

		if f.ExceptReleaseTags != "" && containsFuzzy(r.ReleaseTags, f.ExceptReleaseTags) {
			f.addRejectionF("except release tags: unwanted release. got: %v want: %v", r.ReleaseTags, f.ExceptReleaseTags)
		}
	}

	if f.MatchUploaders != "" && !contains(r.Uploader, f.MatchUploaders) {
		f.addRejectionF("uploaders not matching. got: %v want: %v", r.Uploader, f.MatchUploaders)
	}

	if f.ExceptUploaders != "" && contains(r.Uploader, f.ExceptUploaders) {
		f.addRejectionF("unwanted uploaders. got: %v unwanted: %v", r.Uploader, f.ExceptUploaders)
	}

	if len(f.MatchLanguage) > 0 && !sliceContainsSlice(r.Language, f.MatchLanguage) {
		f.addRejectionF("language not matching. got: %v want: %v", r.Language, f.MatchLanguage)
	}

	if len(f.ExceptLanguage) > 0 && sliceContainsSlice(r.Language, f.ExceptLanguage) {
		f.addRejectionF("language unwanted. got: %v want: %v", r.Language, f.ExceptLanguage)
	}

	if len(f.Resolutions) > 0 && !containsSlice(r.Resolution, f.Resolutions) {
		f.addRejectionF("resolution not matching. got: %v want: %v", r.Resolution, f.Resolutions)
	}

	if len(f.Codecs) > 0 && !sliceContainsSlice(r.Codec, f.Codecs) {
		f.addRejectionF("codec not matching. got: %v want: %v", r.Codec, f.Codecs)
	}

	if len(f.Sources) > 0 && !containsSlice(r.Source, f.Sources) {
		f.addRejectionF("source not matching. got: %v want: %v", r.Source, f.Sources)
	}

	if len(f.Containers) > 0 && !containsSlice(r.Container, f.Containers) {
		f.addRejectionF("container not matching. got: %v want: %v", r.Container, f.Containers)
	}

	// HDR is parsed into the Codec slice from rls
	if len(f.MatchHDR) > 0 && !matchHDR(r.HDR, f.MatchHDR) {
		f.addRejectionF("hdr not matching. got: %v want: %v", r.HDR, f.MatchHDR)
	}

	// HDR is parsed into the Codec slice from rls
	if len(f.ExceptHDR) > 0 && matchHDR(r.HDR, f.ExceptHDR) {
		f.addRejectionF("hdr unwanted. got: %v want: %v", r.HDR, f.ExceptHDR)
	}

	// Other is parsed into the Other slice from rls
	if len(f.MatchOther) > 0 && !sliceContainsSlice(r.Other, f.MatchOther) {
		f.addRejectionF("match other not matching. got: %v want: %v", r.Other, f.MatchOther)
	}

	// Other is parsed into the Other slice from rls
	if len(f.ExceptOther) > 0 && sliceContainsSlice(r.Other, f.ExceptOther) {
		f.addRejectionF("except other unwanted. got: %v unwanted: %v", r.Other, f.ExceptOther)
	}

	if f.Years != "" && !containsIntStrings(r.Year, f.Years) {
		f.addRejectionF("year not matching. got: %d want: %v", r.Year, f.Years)
	}

	if f.Months != "" && !containsIntStrings(r.Month, f.Months) {
		f.addRejectionF("month not matching. got: %d want: %v", r.Month, f.Months)
	}

	if f.Days != "" && !containsIntStrings(r.Day, f.Days) {
		f.addRejectionF("day not matching. got: %d want: %v", r.Day, f.Days)
	}

	if f.MatchCategories != "" {
		var categories []string
		categories = append(categories, r.Categories...)
		if r.Category != "" {
			categories = append(categories, r.Category)
		}
		if !contains(r.Category, f.MatchCategories) && !containsAny(categories, f.MatchCategories) {
			f.addRejectionF("category not matching. got: %v want: %v", strings.Join(categories, ","), f.MatchCategories)
		}
	}

	if f.ExceptCategories != "" {
		var categories []string
		categories = append(categories, r.Categories...)
		if r.Category != "" {
			categories = append(categories, r.Category)
		}
		if contains(r.Category, f.ExceptCategories) && containsAny(categories, f.ExceptCategories) {
			f.addRejectionF("category unwanted. got: %v unwanted: %v", strings.Join(categories, ","), f.ExceptCategories)
		}
	}

	if len(f.MatchReleaseTypes) > 0 && !containsSlice(r.Category, f.MatchReleaseTypes) {
		f.addRejectionF("release type not matching. got: %v want: %v", r.Category, f.MatchReleaseTypes)
	}

	if (f.MinSize != "" || f.MaxSize != "") && !f.checkSizeFilter(r) {
		f.addRejectionF("size not matching. got: %v want min: %v max: %v", r.Size, f.MinSize, f.MaxSize)
	}

	if f.Tags != "" {
		if f.TagsMatchLogic == "ALL" && !containsAll(r.Tags, f.Tags) {
			f.addRejectionF("tags not matching. got: %v want(all): %v", r.Tags, f.Tags)
		} else if !containsAny(r.Tags, f.Tags) { // TagsMatchLogic is set to "" by default, this makes sure that "" and "ANY" are treated the same way.
			f.addRejectionF("tags not matching. got: %v want: %v", r.Tags, f.Tags)
		}
	}

	if f.ExceptTags != "" {
		if f.ExceptTagsMatchLogic == "ALL" && containsAll(r.Tags, f.ExceptTags) {
			f.addRejectionF("tags unwanted. got: %v don't want: %v", r.Tags, f.ExceptTags)
		} else if containsAny(r.Tags, f.ExceptTags) { // ExceptTagsMatchLogic is set to "" by default, this makes sure that "" and "ANY" are treated the same way.
			f.addRejectionF("tags unwanted. got: %v don't want: %v", r.Tags, f.ExceptTags)
		}
	}

	if len(f.Artists) > 0 && !contains(r.Artists, f.Artists) {
		f.addRejectionF("artists not matching. got: %v want: %v", r.Artists, f.Artists)
	}

	if len(f.Albums) > 0 && !contains(r.Title, f.Albums) {
		f.addRejectionF("albums not matching. got: %v want: %v", r.Title, f.Albums)
	}

	// Perfect flac requires Cue, Log, Log Score 100, FLAC and 24bit Lossless
	if f.PerfectFlac && !f.isPerfectFLAC(r) {
		f.addRejectionF("wanted: perfect flac. got: %v", r.Audio)
	}

	if len(f.Formats) > 0 && !sliceContainsSlice(r.Audio, f.Formats) {
		f.addRejectionF("formats not matching. got: %v want: %v", r.Audio, f.Formats)
	}

	if len(f.Quality) > 0 && !containsMatchBasic(r.Audio, f.Quality) {
		f.addRejectionF("quality not matching. got: %v want: %v", r.Audio, f.Quality)
	}

	if len(f.Media) > 0 && !containsSlice(r.Source, f.Media) {
		f.addRejectionF("media not matching. got: %v want: %v", r.Source, f.Media)
	}

	if f.Cue && !containsAny(r.Audio, "Cue") {
		f.addRejection("wanted: cue")
	}

	if f.Log && !containsAny(r.Audio, "Log") {
		f.addRejection("wanted: log")
	}

	if f.Log && f.LogScore != 0 && r.LogScore != f.LogScore {
		f.addRejectionF("log score. got: %v want: %v", r.LogScore, f.LogScore)
	}

	// check description string
	if f.UseRegexDescription {
		if f.MatchDescription != "" && !matchRegex(r.Description, f.MatchDescription) {
			f.addRejectionF("match description regex not matching. got: %v want: %v", r.Description, f.MatchDescription)
		}

		if f.ExceptDescription != "" && matchRegex(r.Description, f.ExceptDescription) {
			f.addRejectionF("except description regex: unwanted release. got: %v want: %v", r.Description, f.ExceptDescription)
		}

	} else {
		if f.MatchDescription != "" && !containsFuzzy(r.Description, f.MatchDescription) {
			f.addRejectionF("match description not matching. got: %v want: %v", r.Description, f.MatchDescription)
		}

		if f.ExceptDescription != "" && containsFuzzy(r.Description, f.ExceptDescription) {
			f.addRejectionF("except description: unwanted release. got: %v want: %v", r.Description, f.ExceptDescription)
		}
	}

	// Min and Max Seeders/Leechers is only for Torznab feeds
	if f.MinSeeders > 0 {
		if f.MinSeeders > r.Seeders {
			f.addRejectionF("min seeders not matcing. got: %d want %d", r.Seeders, f.MinSeeders)
		}
	}

	if f.MaxSeeders > 0 {
		if f.MaxSeeders < r.Seeders {
			f.addRejectionF("max seeders not matcing. got: %d want %d", r.Seeders, f.MaxSeeders)
		}
	}

	if f.MinLeechers > 0 {
		if f.MinLeechers > r.Leechers {
			f.addRejectionF("min leechers not matcing. got: %d want %d", r.Leechers, f.MinLeechers)
		}
	}

	if f.MaxLeechers > 0 {
		if f.MaxLeechers < r.Leechers {
			f.addRejectionF("max leechers not matcing. got: %d want %d", r.Leechers, f.MaxLeechers)
		}
	}

	if len(f.Rejections) > 0 {
		return f.Rejections, false
	}

	return nil, true
}

func (f *Filter) checkMaxDownloads() bool {
	if f.Downloads == nil {
		return false
	}

	var count int
	switch f.MaxDownloadsUnit {
	case FilterMaxDownloadsHour:
		count = f.Downloads.HourCount
	case FilterMaxDownloadsDay:
		count = f.Downloads.DayCount
	case FilterMaxDownloadsWeek:
		count = f.Downloads.WeekCount
	case FilterMaxDownloadsMonth:
		count = f.Downloads.MonthCount
	case FilterMaxDownloadsEver:
		count = f.Downloads.TotalCount
	}

	return count < f.MaxDownloads
}

// isPerfectFLAC Perfect is "CD FLAC Cue Log 100% Lossless or 24bit Lossless"
func (f *Filter) isPerfectFLAC(r *Release) bool {
	if !contains(r.Source, "CD") {
		return false
	}
	if !containsAny(r.Audio, "Cue") {
		return false
	}
	if !containsAny(r.Audio, "Log") {
		return false
	}
	if !containsAny(r.Audio, "Log100") || r.LogScore != 100 {
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

// checkSizeFilter compares the filter size limits to a release's size if it is
// known from the announce line.
func (f *Filter) checkSizeFilter(r *Release) bool {
	if r.Size == 0 {
		r.AdditionalSizeCheckRequired = true
		return true
	} else {
		r.AdditionalSizeCheckRequired = false
	}

	sizeOK, err := f.CheckReleaseSize(r.Size)
	if err != nil {
		f.addRejectionF("size: error checking release size against filter: %+v", err)
		return false
	}

	if !sizeOK {
		return false
	}

	return true
}

// IsPerfectFLAC Perfect is "CD FLAC Cue Log 100% Lossless or 24bit Lossless"
func (f *Filter) IsPerfectFLAC(r *Release) ([]string, bool) {
	rejections := []string{}

	if r.Source != "CD" {
		rejections = append(rejections, fmt.Sprintf("wanted Source CD, got %s", r.Source))
	}
	if r.AudioFormat != "FLAC" {
		rejections = append(rejections, fmt.Sprintf("wanted Format FLAC, got %s", r.AudioFormat))
	}
	if !r.HasCue {
		rejections = append(rejections, fmt.Sprintf("wanted Cue, got %t", r.HasCue))
	}
	if !r.HasLog {
		rejections = append(rejections, fmt.Sprintf("wanted Log, got %t", r.HasLog))
	}
	if r.LogScore != 100 {
		rejections = append(rejections, fmt.Sprintf("wanted Log Score 100, got %d", r.LogScore))
	}
	if !containsSlice(r.Bitrate, []string{"Lossless", "24bit Lossless"}) {
		rejections = append(rejections, fmt.Sprintf("wanted Bitrate Lossless / 24bit Lossless, got %s", r.Bitrate))
	}

	return rejections, len(rejections) == 0
}

func (f *Filter) addRejection(reason string) {
	f.Rejections = append(f.Rejections, reason)
}

func (f *Filter) AddRejectionF(format string, v ...interface{}) {
	f.addRejectionF(format, v...)
}

func (f *Filter) addRejectionF(format string, v ...interface{}) {
	f.Rejections = append(f.Rejections, fmt.Sprintf(format, v...))
}

// ResetRejections reset rejections
func (f *Filter) resetRejections() {
	f.Rejections = []string{}
}

func (f *Filter) RejectionsString(trim bool) string {
	if len(f.Rejections) > 0 {
		out := strings.Join(f.Rejections, ", ")
		if trim && len(out) > 1024 {
			out = out[:1024]
		}

		return out
	}
	return ""
}

func matchRegex(tag string, filterList string) bool {
	if tag == "" {
		return false
	}

	sp, err := splitter.NewSplitter(',',
		splitter.DoubleQuotes,
		splitter.Parenthesis,
		splitter.CurlyBrackets,
		splitter.SquareBrackets,
	)

	if err != nil {
		return false
	}

	filters, err := sp.Split(filterList)
	if err != nil {
		return false
	}

	for _, filter := range filters {
		if filter == "" {
			continue
		}
		re, err := regexcache.Compile(`(?i)(?:` + filter + `)`)
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

func containsAll(tags []string, filter string) bool {
	return containsAllMatch(tags, strings.Split(filter, ","))
}

func containsAnyOther(filter string, tags ...string) bool {
	return containsMatch(tags, strings.Split(filter, ","))
}

func sliceContainsSlice(tags []string, filters []string) bool {
	return containsMatchBasic(tags, filters)
}

func containsMatchFuzzy(tags []string, filters []string) bool {
	advanced := make([]string, 0, len(filters))
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		tag = strings.ToLower(tag)

		clear(advanced)
		for _, filter := range filters {
			if filter == "" {
				continue
			}

			filter = strings.ToLower(filter)
			// check if line contains * or ?, if so try wildcard match, otherwise try substring match
			a := strings.ContainsAny(filter, "?|*")
			if a {
				advanced = append(advanced, filter)
			} else if strings.Contains(tag, filter) {
				return true
			}
		}

		if wildcard.MatchSlice(advanced, tag) {
			return true
		}
	}

	return false
}

func containsMatch(tags []string, filters []string) bool {
	advanced := make([]string, 0, len(filters))
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		tag = strings.ToLower(tag)

		clear(advanced)
		for _, filter := range filters {
			if filter == "" {
				continue
			}

			filter = strings.ToLower(filter)
			// check if line contains * or ?, if so try wildcard match, otherwise try substring match
			a := strings.ContainsAny(filter, "?|*")
			if a {
				advanced = append(advanced, filter)
			} else if tag == filter {
				return true
			}
		}

		if wildcard.MatchSlice(advanced, tag) {
			return true
		}
	}

	return false
}

func containsAllMatch(tags []string, filters []string) bool {
	for _, filter := range filters {
		if filter == "" {
			continue
		}
		filter = strings.ToLower(filter)
		found := false

		wildFilter := strings.ContainsAny(filter, "?|*")

		for _, tag := range tags {
			if tag == "" {
				continue
			}
			tag = strings.ToLower(tag)

			if tag == filter {
				found = true
				break
			} else if wildFilter {
				if wildcard.Match(filter, tag) {
					found = true
					break
				}
			}
		}
		if !found {
			return false
		}
	}

	return true
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

			if tag == filter {
				return true
			}
		}
	}

	return false
}

func containsAnySlice(tags []string, filters []string) bool {
	advanced := make([]string, 0, len(filters))
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		tag = strings.ToLower(tag)

		clear(advanced)
		for _, filter := range filters {
			if filter == "" {
				continue
			}

			filter = strings.ToLower(filter)
			// check if line contains * or ?, if so try wildcard match, otherwise try substring match
			a := strings.ContainsAny(filter, "?|*")
			if a {
				advanced = append(advanced, filter)
			} else if tag == filter {
				return true
			}
		}

		if wildcard.MatchSlice(advanced, tag) {
			return true
		}
	}

	return false
}

func checkFreeleechPercent(announcePercent int, filterPercent string) bool {
	filters := strings.Split(filterPercent, ",")

	for _, s := range filters {
		s = strings.Replace(s, "%", "", -1)

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

func matchHDR(releaseValues []string, filterValues []string) bool {

	for _, filter := range filterValues {
		if filter == "" {
			continue
		}
		filter = strings.ToLower(filter)

		parts := strings.Split(filter, " ")
		if len(parts) == 2 {
			partsMatched := 0
			for _, part := range parts {
				for _, tag := range releaseValues {
					if tag == "" {
						continue
					}
					tag = strings.ToLower(tag)
					if tag == part {
						partsMatched++
					}
					if len(parts) == partsMatched {
						return true
					}
				}
			}
		} else {
			for _, tag := range releaseValues {
				if tag == "" {
					continue
				}
				tag = strings.ToLower(tag)
				if tag == filter {
					return true
				}
			}
		}
	}

	return false
}

func (f *Filter) CheckReleaseSize(releaseSize uint64) (bool, error) {
	minBytes, maxBytes, err := f.parsedSizeLimits()
	if err != nil {
		return false, err
	}

	if minBytes != nil && releaseSize <= *minBytes {
		f.addRejectionF("release size %d bytes smaller than filter min size %d bytes", releaseSize, *minBytes)
		return false, nil
	}

	if maxBytes != nil && releaseSize >= *maxBytes {
		f.addRejectionF("release size %d bytes is larger than filter max size %d bytes", releaseSize, *maxBytes)
		return false, nil
	}

	return true, nil
}

// parsedSizeLimits parses filter bytes limits (expressed as a string) into a
// uint64 number of bytes. The bounds are returned as *uint64 number of bytes,
// with "nil" representing "no limit". We break out filter size limit parsing
// into a discrete step so that we can more easily check parsability at filter
// creation time.
func (f *Filter) parsedSizeLimits() (*uint64, *uint64, error) {
	minBytes, err := parseBytes(f.MinSize)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not parse filter min size")
	}

	maxBytes, err := parseBytes(f.MaxSize)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not parse filter max size")
	}

	return minBytes, maxBytes, nil
}

// parseBytes parses a string representation of a file size into a number of
// bytes. It returns a *uint64 where "nil" represents "none" (corresponding to
// the empty string)
func parseBytes(s string) (*uint64, error) {
	if s == "" {
		return nil, nil
	}
	b, err := humanize.ParseBytes(s)
	return &b, err
}
