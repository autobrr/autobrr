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
	Sound         string               `json:"sound"`
	EventSounds   map[string]string    `json:"event_sounds,omitempty"` // event -> sound mapping
	UsedByFilters []FilterNotification `json:"used_by_filters,omitempty"`
	Method        string               `json:"method,omitempty"`
	Headers       string               `json:"headers,omitempty"`
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
	case NotificationTypeWebhook:
		if n.Webhook != "" {
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
	Release             *Release             // full release data for webhook
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
	NotificationTypeWebhook    NotificationType = "WEBHOOK"
)

type NotificationEvent string

const (
	NotificationEventAppUpdateAvailable NotificationEvent = "APP_UPDATE_AVAILABLE"
	NotificationEventPushApproved       NotificationEvent = "PUSH_APPROVED"
	NotificationEventPushRejected       NotificationEvent = "PUSH_REJECTED"
	NotificationEventPushError          NotificationEvent = "PUSH_ERROR"
	NotificationEventIRCDisconnected    NotificationEvent = "IRC_DISCONNECTED"
	NotificationEventIRCReconnected     NotificationEvent = "IRC_RECONNECTED"
	NotificationEventReleaseNew         NotificationEvent = "RELEASE_NEW"
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

// WebhookEventType represents namespaced event types
type WebhookEventType string

const (
	WebhookEventReleaseNew      WebhookEventType = "release.new"
	WebhookEventActionApproved  WebhookEventType = "action.approved"
	WebhookEventActionRejected  WebhookEventType = "action.rejected"
	WebhookEventActionError     WebhookEventType = "action.error"
	WebhookEventIRCDisconnected WebhookEventType = "irc.disconnected"
	WebhookEventIRCReconnected  WebhookEventType = "irc.reconnected"
	WebhookEventAppUpdate       WebhookEventType = "app.update_available"
	WebhookEventTest            WebhookEventType = "test"
)

// WebhookEvent is the top-level webhook payload structure
type WebhookEvent struct {
	Event     WebhookEventType `json:"event"`
	ID        string           `json:"id"`
	Timestamp time.Time        `json:"timestamp"`
	Version   string           `json:"version"`
	Data      *WebhookData     `json:"data"`
}

// WebhookData contains all nested event data
type WebhookData struct {
	Release *WebhookRelease `json:"release,omitempty"`
	Indexer *WebhookIndexer `json:"indexer,omitempty"`
	Filter  *WebhookFilter  `json:"filter,omitempty"`
	Action  *WebhookAction  `json:"action,omitempty"`
	Result  *WebhookResult  `json:"result,omitempty"`
}

// WebhookRelease contains release-specific data
type WebhookRelease struct {
	Protocol         string       `json:"protocol,omitempty"`
	Implementation   string       `json:"implementation,omitempty"`
	Timestamp        time.Time    `json:"timestamp,omitempty"`
	Type             string       `json:"type"`
	AnnounceType     AnnounceType `json:"announce_type"`
	Link             string       `json:"link,omitempty"`
	DownloadURL      string       `json:"download_url,omitempty"`
	InfoURL          string       `json:"info_url,omitempty"`
	MagnetURI        string       `json:"magnet_uri,omitempty"`
	Category         string       `json:"category,omitempty"`
	Categories       []string     `json:"categories,omitempty"`
	ExternalID       string       `json:"external_id,omitempty"`
	ExternalGroupID  string       `json:"external_group_id,omitempty"`
	Size             uint64       `json:"size,omitempty"`
	Name             string       `json:"name"`
	Title            string       `json:"title,omitempty"`
	SubTitle         string       `json:"sub_title,omitempty"`
	Season           int          `json:"season,omitempty"`
	Episode          int          `json:"episode,omitempty"`
	Year             int          `json:"year,omitempty"`
	Month            int          `json:"month,omitempty"`
	Day              int          `json:"day,omitempty"`
	Resolution       string       `json:"resolution,omitempty"`
	Source           string       `json:"source,omitempty"`
	Codec            []string     `json:"codec,omitempty"`
	Container        string       `json:"container,omitempty"`
	DynamicRange     []string     `json:"dynamic_range,omitempty"`
	Audio            []string     `json:"audio,omitempty"`
	AudioChannels    string       `json:"audio_channels,omitempty"`
	AudioFormat      string       `json:"audio_format,omitempty"`
	Bitrate          string       `json:"bitrate,omitempty"`
	MediaProcessing  string       `json:"media_processing,omitempty"`
	Group            string       `json:"group,omitempty"`
	Website          string       `json:"website,omitempty"`
	Origin           string       `json:"origin,omitempty"`
	Uploader         string       `json:"uploader,omitempty"`
	PreTime          string       `json:"pre_time,omitempty"`
	Edition          []string     `json:"edition,omitempty"`
	Cut              []string     `json:"cut,omitempty"`
	Language         []string     `json:"language,omitempty"`
	Region           string       `json:"region,omitempty"`
	Tags             []string     `json:"tags,omitempty"`
	Proper           bool         `json:"proper,omitempty"`
	Repack           bool         `json:"repack,omitempty"`
	Hybrid           bool         `json:"hybrid,omitempty"`
	Artists          string       `json:"artists,omitempty"`
	RecordLabel      string       `json:"record_label,omitempty"`
	HasCue           bool         `json:"has_cue,omitempty"`
	HasLog           bool         `json:"has_log,omitempty"`
	LogScore         int          `json:"log_score,omitempty"`
	Freeleech        bool         `json:"freeleech,omitempty"`
	FreeleechPercent int          `json:"freeleech_percent,omitempty"`
	Seeders          int          `json:"seeders,omitempty"`
	Leechers         int          `json:"leechers,omitempty"`
	MetaIMDB         string       `json:"meta_imdb,omitempty"`
}

// WebhookIndexer contains indexer information
type WebhookIndexer struct {
	Name           string `json:"name"`
	Identifier     string `json:"identifier,omitempty"`
	Protocol       string `json:"protocol,omitempty"`
	Implementation string `json:"implementation,omitempty"`
}

// WebhookFilter contains filter information
type WebhookFilter struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// WebhookAction contains action information
type WebhookAction struct {
	Name   string `json:"name,omitempty"`
	Type   string `json:"type,omitempty"`
	Client string `json:"client,omitempty"`
}

// WebhookResult contains push result information
type WebhookResult struct {
	Status     string   `json:"status,omitempty"`
	Rejections []string `json:"rejections,omitempty"`
}

func mapNotificationEventToWebhookEvent(event NotificationEvent) WebhookEventType {
	switch event {
	case NotificationEventReleaseNew:
		return WebhookEventReleaseNew
	case NotificationEventPushApproved:
		return WebhookEventActionApproved
	case NotificationEventPushRejected:
		return WebhookEventActionRejected
	case NotificationEventPushError:
		return WebhookEventActionError
	case NotificationEventIRCDisconnected:
		return WebhookEventIRCDisconnected
	case NotificationEventIRCReconnected:
		return WebhookEventIRCReconnected
	case NotificationEventAppUpdateAvailable:
		return WebhookEventAppUpdate
	case NotificationEventTest:
		return WebhookEventTest
	default:
		return WebhookEventTest
	}
}

// NewWebhookEvent creates a structured webhook payload
func NewWebhookEvent(event NotificationEvent, payload NotificationPayload, id string) *WebhookEvent {
	eventPayload := &WebhookEvent{
		Event:     mapNotificationEventToWebhookEvent(event),
		ID:        id,
		Timestamp: payload.Timestamp,
		Version:   "1.0",
	}
	data := &WebhookData{}

	// Populate Release data if available
	if payload.Release != nil {
		release := payload.Release
		data.Release = &WebhookRelease{
			Protocol:         string(release.Protocol),
			Implementation:   string(release.Implementation),
			Timestamp:        release.Timestamp,
			Type:             release.Type.String(),
			AnnounceType:     release.AnnounceType,
			DownloadURL:      release.DownloadURL,
			InfoURL:          release.InfoURL,
			MagnetURI:        release.MagnetURI,
			Category:         release.Category,
			Categories:       release.Categories,
			ExternalID:       release.TorrentID,
			ExternalGroupID:  release.GroupID,
			Size:             release.Size,
			Name:             release.TorrentName,
			Title:            release.Title,
			SubTitle:         release.SubTitle,
			Season:           release.Season,
			Episode:          release.Episode,
			Year:             release.Year,
			Month:            release.Month,
			Day:              release.Day,
			Resolution:       release.Resolution,
			Source:           release.Source,
			Codec:            release.Codec,
			Container:        release.Container,
			DynamicRange:     release.HDR,
			Audio:            release.Audio,
			AudioChannels:    release.AudioChannels,
			AudioFormat:      release.AudioFormat,
			Bitrate:          release.Bitrate,
			MediaProcessing:  release.MediaProcessing,
			Group:            release.Group,
			Website:          release.Website,
			Origin:           release.Origin,
			Uploader:         release.Uploader,
			PreTime:          release.PreTime,
			Edition:          release.Edition,
			Cut:              release.Cut,
			Language:         release.Language,
			Region:           release.Region,
			Tags:             release.Tags,
			Proper:           release.Proper,
			Repack:           release.Repack,
			Hybrid:           release.Hybrid,
			Artists:          release.Artists,
			RecordLabel:      release.RecordLabel,
			HasCue:           release.HasCue,
			HasLog:           release.HasLog,
			LogScore:         release.LogScore,
			Freeleech:        release.Freeleech,
			FreeleechPercent: release.FreeleechPercent,
			Seeders:          release.Seeders,
			Leechers:         release.Leechers,
			MetaIMDB:         release.MetaIMDB,
		}
	} else if payload.ReleaseName != "" {
		// Fallback if full release object is missing but we have basic info in payload
		data.Release = &WebhookRelease{
			Name: payload.ReleaseName,
			Size: payload.Size,
		}
	}

	// Populate Indexer data
	if payload.Indexer != "" {
		data.Indexer = &WebhookIndexer{
			Name:     payload.Indexer,
			Protocol: string(payload.Protocol),
		}
		// If we have full release object, we might have more indexer info
		if payload.Release != nil {
			data.Indexer.Identifier = payload.Release.Indexer.Identifier
		}
	}

	// Populate Filter data
	if payload.Filter != "" || payload.FilterID > 0 {
		data.Filter = &WebhookFilter{
			ID:   payload.FilterID,
			Name: payload.Filter,
		}
	}

	// Populate Action data
	if payload.Action != "" || payload.ActionType != "" {
		data.Action = &WebhookAction{
			Name:   payload.Action,
			Type:   string(payload.ActionType),
			Client: payload.ActionClient,
		}
	}

	// Populate Result data for action events
	if event == NotificationEventPushApproved || event == NotificationEventPushRejected || event == NotificationEventPushError {
		data.Result = &WebhookResult{
			Status:     string(payload.Status),
			Rejections: payload.Rejections,
		}
	}

	eventPayload.Data = data

	return eventPayload
}
