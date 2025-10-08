// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
	"encoding/json"
	"time"
)

type NotificationRepo interface {
	List(ctx context.Context) ([]Notification, error)
	Find(ctx context.Context, params NotificationQueryParams) ([]Notification, int, error)
	FindByID(ctx context.Context, id int) (*Notification, error)
	Store(ctx context.Context, notification *Notification) error
	Update(ctx context.Context, notification *Notification) error
	Delete(ctx context.Context, notificationID int) error

	GetNotificationFilters(ctx context.Context, notificationID int) ([]FilterNotification, error)
	GetFilterNotifications(ctx context.Context, filterID int) ([]FilterNotification, error)
	StoreFilterNotifications(ctx context.Context, filterID int, notifications []FilterNotification) error
	DeleteFilterNotifications(ctx context.Context, filterID int) error
}

type NotificationSender interface {
	Send(event NotificationEvent, payload NotificationPayload) error
	CanSend(event NotificationEvent) bool
	CanSendPayload(event NotificationEvent, payload NotificationPayload) bool
	IsEnabled() bool
	Name() string
	HasFilterEvents(filterID int) bool
}

type Notification struct {
	ID            int                  `json:"id"`
	Name          string               `json:"name"`
	Type          NotificationType     `json:"type"`
	Enabled       bool                 `json:"enabled"`
	Events        []string             `json:"events"`
	Token         string               `json:"token"`
	APIKey        string               `json:"api_key"`
	Webhook       string               `json:"webhook"`
	Title         string               `json:"title"`
	Icon          string               `json:"icon"`
	Username      string               `json:"username"`
	Host          string               `json:"host"`
	Password      string               `json:"password"`
	Channel       string               `json:"channel"`
	Rooms         string               `json:"rooms"`
	Targets       string               `json:"targets"`
	Devices       string               `json:"devices"`
	Priority      int32                `json:"priority"`
	Topic         string               `json:"topic"`
	UsedByFilters []FilterNotification `json:"used_by_filters,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`

	filters map[int]NotificationEvents
}

func NewNotification() *Notification {
	return &Notification{
		filters: make(map[int]NotificationEvents),
	}
}

func (n *Notification) IsEnabled() bool {
	if !n.Enabled {
		return false
	}

	switch n.Type {
	case NotificationTypeDiscord:
		if n.Webhook != "" {
			return true
		}
	case NotificationTypeGotify:
		if n.Host != "" && n.Token != "" {
			return true
		}
	case NotificationTypeLunaSea:
		if n.Webhook != "" {
			return true
		}
	case NotificationTypeNotifiarr:
		if n.APIKey != "" {
			return true
		}
	case NotificationTypeNtfy:
		if n.Host != "" {
			return true
		}
	case NotificationTypePushover:
		if n.APIKey != "" && n.Token != "" {
			return true
		}
	case NotificationTypeShoutrrr:
		if n.Host != "" {
			return true
		}
	case NotificationTypeTelegram:
		if n.Token != "" && n.Channel != "" {
			return true
		}
	}
	return false
}

func (n *Notification) FilterMuted(filterID int) bool {
	if n.filters != nil && filterID > 0 {
		if events, ok := n.filters[filterID]; ok {
			return events.IsMuted()
		}
	}

	return false
}

func (n *Notification) HasFilterNotifications(filterID int) bool {
	if n.filters != nil && filterID > 0 {
		_, ok := n.filters[filterID]
		return ok
	}
	return false
}

func (n *Notification) FilterEventEnabled(filterID int, event NotificationEvent) bool {
	if filterID > 0 {
		if n.filters == nil {
			return false
		}

		if events, ok := n.filters[filterID]; ok {
			return events.EventEnabled(string(event))
		}
	}

	return false
}

func (n *Notification) EventEnabled(event string) bool {
	for _, e := range n.Events {
		if e == event {
			return true
		}
	}
	return false
}

func (n *Notification) SetFilterEvents(filterID int, events NotificationEvents) {
	if n.filters == nil {
		n.filters = make(map[int]NotificationEvents)
	}
	n.filters[filterID] = events
}

func (n *Notification) RemoveFilterEvents(filterID int) {
	delete(n.filters, filterID)
}

func (n *Notification) ClearFilterEvents() {
	n.filters = nil
}

func (n Notification) MarshalJSON() ([]byte, error) {
	type Alias Notification
	return json.Marshal(&struct {
		*Alias
		Token    string `json:"token"`
		APIKey   string `json:"api_key"`
		Password string `json:"password"`
	}{
		Alias:    (*Alias)(&n),
		Token:    RedactString(n.Token),
		APIKey:   RedactString(n.APIKey),
		Password: RedactString(n.Password),
	})
}

type NotificationPayload struct {
	Subject             string
	Message             string
	Event               NotificationEvent
	ReleaseName         string
	Filter              string
	FilterID            int
	Indexer             string
	InfoHash            string
	Size                uint64
	Status              ReleasePushStatus
	Action              string
	ActionType          ActionType
	ActionClient        string
	Rejections          []string
	Protocol            ReleaseProtocol       // torrent, usenet
	Implementation      ReleaseImplementation // irc, rss, api
	Timestamp           time.Time
	Sender              string
	FilterNotifications []FilterNotification // per-filter notifications
}

type NotificationType string

const (
	NotificationTypeDiscord    NotificationType = "DISCORD"
	NotificationTypeNotifiarr  NotificationType = "NOTIFIARR"
	NotificationTypeIFTTT      NotificationType = "IFTTT"
	NotificationTypeJoin       NotificationType = "JOIN"
	NotificationTypeMattermost NotificationType = "MATTERMOST"
	NotificationTypeMatrix     NotificationType = "MATRIX"
	NotificationTypePushBullet NotificationType = "PUSH_BULLET"
	NotificationTypePushover   NotificationType = "PUSHOVER"
	NotificationTypeRocketChat NotificationType = "ROCKETCHAT"
	NotificationTypeSlack      NotificationType = "SLACK"
	NotificationTypeTelegram   NotificationType = "TELEGRAM"
	NotificationTypeGotify     NotificationType = "GOTIFY"
	NotificationTypeNtfy       NotificationType = "NTFY"
	NotificationTypeLunaSea    NotificationType = "LUNASEA"
	NotificationTypeShoutrrr   NotificationType = "SHOUTRRR"
)

type NotificationEvent string

const (
	NotificationEventAppUpdateAvailable NotificationEvent = "APP_UPDATE_AVAILABLE"
	NotificationEventPushApproved       NotificationEvent = "PUSH_APPROVED"
	NotificationEventPushRejected       NotificationEvent = "PUSH_REJECTED"
	NotificationEventPushError          NotificationEvent = "PUSH_ERROR"
	NotificationEventIRCDisconnected    NotificationEvent = "IRC_DISCONNECTED"
	NotificationEventIRCReconnected     NotificationEvent = "IRC_RECONNECTED"
	NotificationEventTest               NotificationEvent = "TEST"
)

func (e NotificationEvent) String() string {
	return string(e)
}

type NotificationEvents []NotificationEvent

func NewNotificationEventsFromStrings(events []string) NotificationEvents {
	result := make(NotificationEvents, 0)
	for _, e := range events {
		result = append(result, NotificationEvent(e))
	}
	return result
}

func (events NotificationEvents) IsMuted() bool {
	return len(events) == 0
}

func (events NotificationEvents) EventEnabled(event string) bool {
	for _, e := range events {
		if string(e) == event {
			return true
		}
	}
	return false
}

type NotificationEventArr []NotificationEvent

type NotificationQueryParams struct {
	Limit   uint64
	Offset  uint64
	Sort    map[string]string
	Filters struct {
		Indexers   []string
		PushStatus string
	}
	Search string
}
