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
	case NotificationTypeGenericWebhook:
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
	Release             *Release             // full release data for generic webhook
}

// GenericWebhookPayload contains all available release and event data for generic webhook notifications
type GenericWebhookPayload struct {
	// Event metadata
	Event     NotificationEvent `json:"event"`
	Timestamp time.Time         `json:"timestamp"`

	// Release identification
	ReleaseName string `json:"release_name"`
	TorrentName string `json:"torrent_name"`
	InfoHash    string `json:"info_hash,omitempty"`
	Size        uint64 `json:"size"`
	Title       string `json:"title"`
	SubTitle    string `json:"sub_title,omitempty"`
	Type        string `json:"type"`

	// Source information
	Indexer        string                `json:"indexer"`
	Protocol       ReleaseProtocol       `json:"protocol"`
	Implementation ReleaseImplementation `json:"implementation"`
	InfoURL        string                `json:"info_url,omitempty"`
	DownloadURL    string                `json:"download_url,omitempty"`

	// Filter/Action information
	Filter       string            `json:"filter,omitempty"`
	FilterID     int               `json:"filter_id,omitempty"`
	Action       string            `json:"action,omitempty"`
	ActionType   ActionType        `json:"action_type,omitempty"`
	ActionClient string            `json:"action_client,omitempty"`
	Status       ReleasePushStatus `json:"status,omitempty"`
	Rejections   []string          `json:"rejections,omitempty"`

	// Media identification
	Category   string   `json:"category,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Season     int      `json:"season,omitempty"`
	Episode    int      `json:"episode,omitempty"`
	Year       int      `json:"year,omitempty"`
	Month      int      `json:"month,omitempty"`
	Day        int      `json:"day,omitempty"`

	// Media quality
	Resolution      string   `json:"resolution,omitempty"`
	Source          string   `json:"source,omitempty"`
	Codec           []string `json:"codec,omitempty"`
	Container       string   `json:"container,omitempty"`
	HDR             []string `json:"hdr,omitempty"`
	Audio           []string `json:"audio,omitempty"`
	AudioChannels   string   `json:"audio_channels,omitempty"`
	AudioFormat     string   `json:"audio_format,omitempty"`
	MediaProcessing string   `json:"media_processing,omitempty"`

	// Release metadata
	Group    string   `json:"group,omitempty"`
	Website  string   `json:"website,omitempty"`
	Origin   string   `json:"origin,omitempty"`
	Uploader string   `json:"uploader,omitempty"`
	PreTime  string   `json:"pre_time,omitempty"`
	Edition  []string `json:"edition,omitempty"`
	Cut      []string `json:"cut,omitempty"`
	Language []string `json:"language,omitempty"`
	Region   string   `json:"region,omitempty"`
	Tags     []string `json:"tags,omitempty"`

	// Flags
	Proper           bool `json:"proper"`
	Repack           bool `json:"repack"`
	Hybrid           bool `json:"hybrid"`
	Freeleech        bool `json:"freeleech"`
	FreeleechPercent int  `json:"freeleech_percent,omitempty"`

	// Music specific
	Artists     string   `json:"artists,omitempty"`
	RecordLabel string   `json:"record_label,omitempty"`
	LogScore    int      `json:"log_score,omitempty"`
	HasCue      bool     `json:"has_cue,omitempty"`
	HasLog      bool     `json:"has_log,omitempty"`
	Bonus       []string `json:"bonus,omitempty"`

	// Torrent stats
	Seeders  int `json:"seeders,omitempty"`
	Leechers int `json:"leechers,omitempty"`
}

// NewGenericWebhookPayload creates a GenericWebhookPayload from a NotificationPayload
func NewGenericWebhookPayload(payload NotificationPayload) *GenericWebhookPayload {
	p := &GenericWebhookPayload{
		Event:          payload.Event,
		Timestamp:      payload.Timestamp,
		ReleaseName:    payload.ReleaseName,
		InfoHash:       payload.InfoHash,
		Size:           payload.Size,
		Indexer:        payload.Indexer,
		Protocol:       payload.Protocol,
		Implementation: payload.Implementation,
		Filter:         payload.Filter,
		FilterID:       payload.FilterID,
		Action:         payload.Action,
		ActionType:     payload.ActionType,
		ActionClient:   payload.ActionClient,
		Status:         payload.Status,
		Rejections:     payload.Rejections,
	}

	if payload.Release != nil {
		p.TorrentName = payload.Release.TorrentName
		p.Size = payload.Release.Size
		p.Title = payload.Release.Title
		p.SubTitle = payload.Release.SubTitle
		p.Type = payload.Release.Type.String()
		p.InfoURL = payload.Release.InfoURL
		p.DownloadURL = payload.Release.DownloadURL
		p.Category = payload.Release.Category
		p.Categories = payload.Release.Categories
		p.Season = payload.Release.Season
		p.Episode = payload.Release.Episode
		p.Year = payload.Release.Year
		p.Month = payload.Release.Month
		p.Day = payload.Release.Day
		p.Resolution = payload.Release.Resolution
		p.Source = payload.Release.Source
		p.Codec = payload.Release.Codec
		p.Container = payload.Release.Container
		p.HDR = payload.Release.HDR
		p.Audio = payload.Release.Audio
		p.AudioChannels = payload.Release.AudioChannels
		p.AudioFormat = payload.Release.AudioFormat
		p.MediaProcessing = payload.Release.MediaProcessing
		p.Group = payload.Release.Group
		p.Website = payload.Release.Website
		p.Origin = payload.Release.Origin
		p.Uploader = payload.Release.Uploader
		p.PreTime = payload.Release.PreTime
		p.Edition = payload.Release.Edition
		p.Cut = payload.Release.Cut
		p.Language = payload.Release.Language
		p.Region = payload.Release.Region
		p.Tags = payload.Release.Tags
		p.Proper = payload.Release.Proper
		p.Repack = payload.Release.Repack
		p.Hybrid = payload.Release.Hybrid
		p.Freeleech = payload.Release.Freeleech
		p.FreeleechPercent = payload.Release.FreeleechPercent
		p.Artists = payload.Release.Artists
		p.RecordLabel = payload.Release.RecordLabel
		p.LogScore = payload.Release.LogScore
		p.HasCue = payload.Release.HasCue
		p.HasLog = payload.Release.HasLog
		p.Bonus = payload.Release.Bonus
		p.Seeders = payload.Release.Seeders
		p.Leechers = payload.Release.Leechers
	}

	return p
}

type NotificationType string

const (
	NotificationTypeDiscord        NotificationType = "DISCORD"
	NotificationTypeNotifiarr      NotificationType = "NOTIFIARR"
	NotificationTypeIFTTT          NotificationType = "IFTTT"
	NotificationTypeJoin           NotificationType = "JOIN"
	NotificationTypeMattermost     NotificationType = "MATTERMOST"
	NotificationTypeMatrix         NotificationType = "MATRIX"
	NotificationTypePushBullet     NotificationType = "PUSH_BULLET"
	NotificationTypePushover       NotificationType = "PUSHOVER"
	NotificationTypeRocketChat     NotificationType = "ROCKETCHAT"
	NotificationTypeSlack          NotificationType = "SLACK"
	NotificationTypeTelegram       NotificationType = "TELEGRAM"
	NotificationTypeGotify         NotificationType = "GOTIFY"
	NotificationTypeNtfy           NotificationType = "NTFY"
	NotificationTypeLunaSea        NotificationType = "LUNASEA"
	NotificationTypeShoutrrr       NotificationType = "SHOUTRRR"
	NotificationTypeGenericWebhook NotificationType = "GENERIC_WEBHOOK"
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
	Name            string    `json:"name"`
	Title           string    `json:"title,omitempty"`
	SubTitle        string    `json:"sub_title,omitempty"`
	Multiplier      string    `json:"multiplier,omitempty"` // Added to handle release metadata
	Category        string    `json:"category,omitempty"`
	Categories      []string  `json:"categories,omitempty"`
	Season          int       `json:"season,omitempty"`
	Episode         int       `json:"episode,omitempty"`
	Year            int       `json:"year,omitempty"`
	Month           int       `json:"month,omitempty"`
	Day             int       `json:"day,omitempty"`
	Resolution      string    `json:"resolution,omitempty"`
	Source          string    `json:"source,omitempty"`
	Codec           []string  `json:"codec,omitempty"`
	Container       string    `json:"container,omitempty"`
	HDR             []string  `json:"hdr,omitempty"`
	Audio           []string  `json:"audio,omitempty"`
	AudioChannels   string    `json:"audio_channels,omitempty"`
	AudioFormat     string    `json:"audio_format,omitempty"`
	MediaProcessing string    `json:"media_processing,omitempty"`
	Group           string    `json:"group,omitempty"`
	Website         string    `json:"website,omitempty"`
	Origin          string    `json:"origin,omitempty"`
	Uploader        string    `json:"uploader,omitempty"`
	PreTime         string    `json:"pre_time,omitempty"`
	Edition         []string  `json:"edition,omitempty"`
	Cut             []string  `json:"cut,omitempty"`
	Language        []string  `json:"language,omitempty"`
	Region          string    `json:"region,omitempty"`
	Tags            []string  `json:"tags,omitempty"`
	Proper          bool      `json:"proper,omitempty"`
	Repack          bool      `json:"repack,omitempty"`
	Hybrid          bool      `json:"hybrid,omitempty"`
	Freeleech       bool      `json:"freeleech,omitempty"`
	Link            string    `json:"link,omitempty"`
	DownloadURL     string    `json:"download_url,omitempty"`
	InfoURL         string    `json:"info_url,omitempty"`
	Size            uint64    `json:"size,omitempty"`
	Seeders         int       `json:"seeders,omitempty"`
	Leechers        int       `json:"leechers,omitempty"`
	Protocol        string    `json:"protocol,omitempty"`
	Implementation  string    `json:"implementation,omitempty"`
	Timestamp       time.Time `json:"timestamp,omitempty"`
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

func mapToWebhookEvent(event NotificationEvent) WebhookEventType {
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
	// If release is nil but available in payload, use that
	var release *Release
	if payload.Release != nil {
		release = payload.Release
	}

	data := &WebhookData{}

	// Populate Release data if available
	if release != nil {
		data.Release = &WebhookRelease{
			Name:            release.TorrentName,
			Title:           release.Title,
			SubTitle:        release.SubTitle,
			Category:        release.Category,
			Categories:      release.Categories,
			Season:          release.Season,
			Episode:         release.Episode,
			Year:            release.Year,
			Month:           release.Month,
			Day:             release.Day,
			Resolution:      release.Resolution,
			Source:          release.Source,
			Codec:           release.Codec,
			Container:       release.Container,
			HDR:             release.HDR,
			Audio:           release.Audio,
			AudioChannels:   release.AudioChannels,
			AudioFormat:     release.AudioFormat,
			MediaProcessing: release.MediaProcessing,
			Group:           release.Group,
			Website:         release.Website,
			Origin:          release.Origin,
			Uploader:        release.Uploader,
			PreTime:         release.PreTime,
			Edition:         release.Edition,
			Cut:             release.Cut,
			Language:        release.Language,
			Region:          release.Region,
			Tags:            release.Tags,
			Proper:          release.Proper,
			Repack:          release.Repack,
			Hybrid:          release.Hybrid,
			Freeleech:       release.Freeleech,
			DownloadURL:     release.DownloadURL,
			InfoURL:         release.InfoURL,
			Size:            release.Size,
			Seeders:         release.Seeders,
			Leechers:        release.Leechers,
			Protocol:        string(release.Protocol),
			Implementation:  string(release.Implementation),
			Timestamp:       release.Timestamp,
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
		if release != nil {
			data.Indexer.Identifier = release.Indexer.Identifier
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

	return &WebhookEvent{
		Event:     mapToWebhookEvent(event),
		ID:        id,
		Timestamp: payload.Timestamp,
		Version:   "1.0",
		Data:      data,
	}
}
