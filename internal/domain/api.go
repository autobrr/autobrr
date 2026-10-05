// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"slices"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

type APIKey struct {
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	Scopes    []string  `json:"scopes"`
	CreatedAt time.Time `json:"created_at"`
}

// APIScopeAll grants full access, including the endpoints that manage API keys and users.
const APIScopeAll = "*"

type APIResource string

const (
	APIResourceConfig        APIResource = "config"
	APIResourceDownloaders   APIResource = "downloaders"
	APIResourceFeeds         APIResource = "feeds"
	APIResourceFilters       APIResource = "filters"
	APIResourceIndexers      APIResource = "indexers"
	APIResourceIRC           APIResource = "irc"
	APIResourceLists         APIResource = "lists"
	APIResourceLogs          APIResource = "logs"
	APIResourceNotifications APIResource = "notifications"
	APIResourceProxies       APIResource = "proxies"
	APIResourceReleases      APIResource = "releases"
	APIResourceUpdates       APIResource = "updates"
	APIResourceWebhooks      APIResource = "webhooks"
)

func (r APIResource) String() string {
	return string(r)
}

type APIAccess string

const (
	APIAccessRead  APIAccess = "read"
	APIAccessWrite APIAccess = "write"
)

func (a APIAccess) String() string {
	return string(a)
}

var apiResourceAccess = map[APIResource][]APIAccess{
	APIResourceConfig:        {APIAccessRead, APIAccessWrite},
	APIResourceDownloaders:   {APIAccessRead, APIAccessWrite},
	APIResourceFeeds:         {APIAccessRead, APIAccessWrite},
	APIResourceFilters:       {APIAccessRead, APIAccessWrite},
	APIResourceIndexers:      {APIAccessRead, APIAccessWrite},
	APIResourceIRC:           {APIAccessRead, APIAccessWrite},
	APIResourceLists:         {APIAccessRead, APIAccessWrite},
	APIResourceLogs:          {APIAccessRead},
	APIResourceNotifications: {APIAccessRead, APIAccessWrite},
	APIResourceProxies:       {APIAccessRead, APIAccessWrite},
	APIResourceReleases:      {APIAccessRead, APIAccessWrite},
	APIResourceUpdates:       {APIAccessRead},
	APIResourceWebhooks:      {APIAccessWrite},
}

// Validate requires either the lone full access scope or one resource:access scope per resource.
func (k *APIKey) Validate() error {
	if len(k.Scopes) == 0 {
		return errors.Wrap(ErrInvalidAPIKeyScopes, "at least one scope is required")
	}

	if slices.Contains(k.Scopes, APIScopeAll) {
		if len(k.Scopes) > 1 {
			return errors.Wrap(ErrInvalidAPIKeyScopes, "scope '%s' can not be combined with other scopes", APIScopeAll)
		}

		return nil
	}

	seen := make(map[APIResource]struct{}, len(k.Scopes))

	for _, scope := range k.Scopes {
		resource, access, ok := parseAPIScope(scope)
		if !ok {
			return errors.Wrap(ErrInvalidAPIKeyScopes, "invalid scope: '%s'", scope)
		}

		allowed, ok := apiResourceAccess[resource]
		if !ok || !slices.Contains(allowed, access) {
			return errors.Wrap(ErrInvalidAPIKeyScopes, "unsupported scope: '%s'", scope)
		}

		if _, ok := seen[resource]; ok {
			return errors.Wrap(ErrInvalidAPIKeyScopes, "duplicate scope for resource: '%s'", resource)
		}

		seen[resource] = struct{}{}
	}

	return nil
}

// HasFullAccess reports whether the key carries the full access scope.
func (k *APIKey) HasFullAccess() bool {
	return slices.Contains(k.Scopes, APIScopeAll)
}

// HasAccess reports whether the key grants access to resource. Write access implies read access.
func (k *APIKey) HasAccess(resource APIResource, access APIAccess) bool {
	for _, scope := range k.Scopes {
		if scope == APIScopeAll {
			return true
		}

		r, a, ok := parseAPIScope(scope)
		if !ok || r != resource {
			continue
		}

		if a == access || a == APIAccessWrite {
			return true
		}
	}

	return false
}

func parseAPIScope(scope string) (APIResource, APIAccess, bool) {
	resource, access, ok := strings.Cut(scope, ":")
	if !ok {
		return "", "", false
	}

	return APIResource(resource), APIAccess(access), true
}

const RedactedStr = "<redacted>"

func RedactString(s string) string {
	if len(s) == 0 {
		return ""
	}

	return RedactedStr
}

func IsRedactedString(s string) bool {
	if s == "" {
		return false
	}
	return s == RedactedStr
}
