package domain

import (
	"context"
	"time"
)

type ListRepo interface {
	List(ctx context.Context) ([]List, error)
	FindByID(ctx context.Context, listID int64) (*List, error)
	Store(ctx context.Context, listID *List) error
	Update(ctx context.Context, listID *List) error
	UpdateLastRefresh(ctx context.Context, list List) error
	ToggleEnabled(ctx context.Context, listID int64, enabled bool) error
	Delete(ctx context.Context, listID int64) error
}

type ListType string

const (
	ListTypeRadarr     ListType = "RADARR"
	ListTypeSonarr     ListType = "SONARR"
	ListTypeLidarr     ListType = "LIDARR"
	ListTypeReadarr    ListType = "READARR"
	ListTypeWhisparr   ListType = "WHISPARR"
	ListTypeMDBList    ListType = "MDBLIST"
	ListTypeMetacritic ListType = "METACRITIC"
	ListTypeTrakt      ListType = "TRAKT"
	ListTypeSteam      ListType = "STEAM"
)

type List struct {
	ID                     int       `json:"id"`
	Name                   string    `json:"name"`
	Type                   ListType  `json:"type"`
	Enabled                bool      `json:"enabled"`
	ClientID               int       `json:"client_id"`
	URL                    string    `json:"url"`
	Headers                []string  `json:"headers"`
	APIKey                 string    `json:"api_key"`
	Filters                []int     `json:"filters"`
	MatchRelease           bool      `json:"match_release"`
	TagsInclude            []string  `json:"tags_include"`
	TagsExclude            []string  `json:"tags_exclude"`
	IncludeUnmonitored     bool      `json:"include_unmonitored"`
	ExcludeAlternateTitles bool      `json:"exclude_alternate_titles"`
	LastRefreshTime        time.Time `json:"last_refresh_time"`
	LastRefreshError       string    `json:"last_refresh_error"`
	LastRefreshStatus      string    `json:"last_refresh_status"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}
