// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package events

import "github.com/autobrr/autobrr/internal/domain"

type EventType string

const (
	ApplicationUpdate EventType = "application.update"

	IndexerDeleted       EventType = "indexer.deleted"
	IndexerToggleEnabled EventType = "indexer.toggle-enabled"

	ProxyUpdated EventType = "proxy.updated"
	ProxyDeleted EventType = "proxy.deleted"

	FilterApproved EventType = "filter.approved"
	FilterRejected EventType = "filter.rejected"
	FilterError    EventType = "filter.error"

	FilterExternalApproved EventType = "filter.external.approved"
	FilterExternalRejected EventType = "filter.external.rejected"
	FilterExternalError    EventType = "filter.external.error"

	ReleaseNew          EventType = "release.new"
	ReleasePushApproved EventType = "release.push_approved"
	ReleasePushRejected EventType = "release.push_rejected"
	ReleasePushError    EventType = "release.push_error"

	IRCDisconnected EventType = "irc.disconnected"
	IRCReconnected  EventType = "irc.reconnected"
	IRCFlapping     EventType = "irc.flapping"

	ListRefreshSuccess EventType = "list.refresh_success"
	ListRefreshError   EventType = "list.refresh_error"

	FeedRefreshSuccess EventType = "feed.refresh_success"
	FeedRefreshError   EventType = "feed.refresh_error"
)

type Event struct {
	Type EventType
}

// GetType returns the event type. It is promoted to every event embedding Event.
func (e Event) GetType() EventType {
	return e.Type
}

type AppUpdateEvent struct {
	Event
	CurrentVersion string
	NewVersion     string
	URL            string
}

type ReleaseEvent struct {
	Event
	Release *domain.Release
}

type ReleasePushEvent struct {
	Event
	Action       *domain.Action
	ActionStatus *domain.ReleaseActionStatus
	Release      *domain.Release
}

type IndexerChangeEvent struct {
	Event
	Indexer *domain.Indexer
}

// ProxyChangeEvent carries the entities that pointed at the proxy before the change, so
// subscribers can reconcile their running state against the persisted rows.
type ProxyChangeEvent struct {
	Event
	ProxyID int64
	Usage   *domain.ProxyUsage
}

// ListRefreshEvent carries the list after its refresh outcome is recorded, so LastRefreshData
// holds the error message of a failed refresh.
type ListRefreshEvent struct {
	Event
	List *domain.List
}

// FeedRefreshEvent carries the outcome of a feed refresh; Error is empty on success.
type FeedRefreshEvent struct {
	Event
	Feed  *domain.Feed
	Error string
}

type IRCEvent struct {
	Event
	State   string
	Network string
	Message string
}
