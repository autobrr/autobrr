package domain

import (
	"context"
	"time"
)

/*
Works the same way as for autodl-irssi
https://autodl-community.github.io/autodl-irssi/configuration/filter/
*/

type FilterRepo interface {
	FindByID(filterID int) (*Filter, error)
	FindFiltersForSite(site string) ([]Filter, error)
	FindByIndexerIdentifier(indexer string) ([]Filter, error)
	ListFilters() ([]Filter, error)
	Store(filter Filter) (*Filter, error)
	Update(ctx context.Context, filter Filter) (*Filter, error)
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
	Years               string    `json:"years"`
	Artists             string    `json:"artists"`
	Albums              string    `json:"albums"`
	MatchReleaseTypes   string    `json:"match_release_types"` // Album,Single,EP
	ExceptReleaseTypes  string    `json:"except_release_types"`
	Formats             []string  `json:"formats"`  // MP3, FLAC, Ogg, AAC, AC3, DTS
	Bitrates            []string  `json:"bitrates"` // 192, 320, APS (VBR), V2 (VBR), V1 (VBR), APX (VBR), V0 (VBR), q8.x (VBR), Lossless, 24bit Lossless, Other
	Media               []string  `json:"media"`    // CD, DVD, Vinyl, Soundboard, SACD, DAT, Cassette, WEB, Other
	Cue                 bool      `json:"cue"`
	Log                 bool      `json:"log"`
	LogScores           string    `json:"log_scores"`
	MatchCategories     string    `json:"match_categories"`
	ExceptCategories    string    `json:"except_categories"`
	MatchUploaders      string    `json:"match_uploaders"`
	ExceptUploaders     string    `json:"except_uploaders"`
	Tags                string    `json:"tags"`
	ExceptTags          string    `json:"except_tags"`
	TagsAny             string    `json:"tags_any"`
	ExceptTagsAny       string    `json:"except_tags_any"`
	Actions             []Action  `json:"actions"`
	Indexers            []Indexer `json:"indexers"`
}
