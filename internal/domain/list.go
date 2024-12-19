package domain

import "time"

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
}

//	interface List {
//	id: number;
//	name: string;
//	enabled: boolean;
//	type: ListType;
//	client_id: number;
//	url: string;
//	headers: string[];
//	api_key: string;
//	// cookie: string;
//	filters: number[];
//	match_release: boolean;
//	tags_include: string[];
//	tags_exclude: string[];
//	include_unmonitored: boolean;
//	exclude_alternate_titles: boolean;
//}
