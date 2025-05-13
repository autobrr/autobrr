// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

type ListRepo interface {
	List(ctx context.Context) ([]*List, error)
	FindByID(ctx context.Context, listID int64) (*List, error)
	Store(ctx context.Context, listID *List) error
	Update(ctx context.Context, listID *List) error
	UpdateLastRefresh(ctx context.Context, list *List) error
	ToggleEnabled(ctx context.Context, listID int64, enabled bool) error
	Delete(ctx context.Context, listID int64) error
	GetListFilters(ctx context.Context, listID int64) ([]ListFilter, error)
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
	ListTypePlaintext  ListType = "PLAINTEXT"
	ListTypeTrakt      ListType = "TRAKT"
	ListTypeSteam      ListType = "STEAM"
	ListTypeAniList    ListType = "ANILIST"
)

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
	LastRefreshTime        time.Time         `json:"last_refresh_time"`
	LastRefreshData        string            `json:"last_refresh_error"`
	LastRefreshStatus      ListRefreshStatus `json:"last_refresh_status"`
	CreatedAt              time.Time         `json:"created_at"`
	UpdatedAt              time.Time         `json:"updated_at"`
	SkipCleanSanitize      bool              `json:"skip_clean_sanitize"`
}

func (l *List) Validate() error {
	if l.Name == "" {
		return errors.New("name is required")
	}

	if l.Type == "" {
		return errors.New("type is required")
	}

	if !l.ListTypeArr() && !l.ListTypeList() {
		return errors.New("invalid list type: %s", l.Type)
	}

	if l.ListTypeArr() && l.ClientID == 0 {
		return errors.New("arr client id is required")
	}

	if l.ListTypeList() {
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

func (l *List) ListTypeArr() bool {
	return l.Type == ListTypeRadarr || l.Type == ListTypeSonarr || l.Type == ListTypeLidarr || l.Type == ListTypeReadarr || l.Type == ListTypeWhisparr
}

func (l *List) ListTypeList() bool {
	return l.Type == ListTypeMDBList || l.Type == ListTypeMetacritic || l.Type == ListTypePlaintext || l.Type == ListTypeTrakt || l.Type == ListTypeSteam || l.Type == ListTypeAniList
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
