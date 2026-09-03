// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

type ListType string

const (
	ListTypeRadarr     ListType = "RADARR"
	ListTypeSonarr     ListType = "SONARR"
	ListTypeLidarr     ListType = "LIDARR"
	ListTypeReadarr    ListType = "READARR"
	ListTypeWhisparr   ListType = "WHISPARR"
	ListTypeWhisparrV3 ListType = "WHISPARR_V3"
	ListTypeSportarr   ListType = "SPORTARR"
	ListTypeMDBList    ListType = "MDBLIST"
	ListTypeMetacritic ListType = "METACRITIC"
	ListTypePlaintext  ListType = "PLAINTEXT"
	ListTypeTrakt      ListType = "TRAKT"
	ListTypeSteam      ListType = "STEAM"
	ListTypeAniList    ListType = "ANILIST"
)

func (l ListType) String() string {
	return string(l)
}

func (l ListType) Valid() bool {
	return l.ArrClient() || l.RegularList()
}

func (l ListType) ArrClient() bool {
	switch l {
	case ListTypeRadarr, ListTypeSonarr, ListTypeLidarr, ListTypeReadarr, ListTypeWhisparr, ListTypeWhisparrV3, ListTypeSportarr:
		return true
	default:
		return false
	}
}

func (l ListType) RegularList() bool {
	switch l {
	case ListTypeMDBList, ListTypeMetacritic, ListTypePlaintext, ListTypeTrakt, ListTypeSteam, ListTypeAniList:
		return true
	default:
		return false
	}
}

type ListRefreshStatus string

const (
	ListRefreshStatusSuccess ListRefreshStatus = "SUCCESS"
	ListRefreshStatusError   ListRefreshStatus = "ERROR"
)

type List struct {
	ID                     int64             `json:"id"`
	Name                   string            `json:"name"`
	Type                   ListType          `json:"type"`
	Enabled                bool              `json:"enabled"`
	ClientID               int               `json:"client_id"`
	URL                    string            `json:"url"`
	Headers                []string          `json:"headers"`
	APIKey                 string            `json:"api_key"`
	Filters                []ListFilter      `json:"filters"`
	MatchRelease           bool              `json:"match_release"`
	TagsInclude            []string          `json:"tags_included"`
	TagsExclude            []string          `json:"tags_excluded"`
	IncludeUnmonitored     bool              `json:"include_unmonitored"`
	IncludeAlternateTitles bool              `json:"include_alternate_titles"`
	IncludeYear            bool              `json:"include_year"`
	SkipCleanSanitize      bool              `json:"skip_clean_sanitize"`
	LastRefreshTime        time.Time         `json:"last_refresh_time"`
	LastRefreshData        string            `json:"last_refresh_error"`
	LastRefreshStatus      ListRefreshStatus `json:"last_refresh_status"`
	CreatedAt              time.Time         `json:"created_at"`
	UpdatedAt              time.Time         `json:"updated_at"`
}

func (l List) MarshalJSON() ([]byte, error) {
	type Alias List
	return json.Marshal(&struct {
		*Alias
		APIKey string `json:"api_key"`
	}{
		APIKey: RedactString(l.APIKey),
		Alias:  (*Alias)(&l),
	})
}

func (l *List) Validate() error {
	if l.Name == "" {
		return errors.New("name is required")
	}

	if l.Type == "" {
		return errors.New("type is required")
	}

	if !l.Type.Valid() {
		return errors.New("invalid list type: %s", l.Type)
	}

	if l.Type.ArrClient() && l.ClientID == 0 {
		return errors.New("arr client id is required")
	}

	if l.Type.RegularList() {
		if l.URL == "" {
			return errors.New("list url is required")
		}

		_, err := url.Parse(l.URL)
		if err != nil {
			return errors.Wrap(err, "could not parse list url: %s", l.URL)
		}
	}

	if len(l.Filters) == 0 {
		return errors.New("at least one filter is required")
	}

	return nil
}

func (l *List) ShouldProcessItem(monitored bool) bool {
	if l.IncludeUnmonitored {
		return true
	}

	return monitored
}

// SetRequestHeaders set headers from list on the request
func (l *List) SetRequestHeaders(req *http.Request) {
	for _, header := range l.Headers {
		parts := strings.Split(header, "=")
		if len(parts) != 2 {
			continue
		}
		req.Header.Set(parts[0], parts[1])
	}
}

type ListFilter struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
