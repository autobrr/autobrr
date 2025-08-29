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
}

type NotificationSender interface {
	Send(event NotificationEvent, payload NotificationPayload) error
	CanSend(event NotificationEvent) bool
	Name() string
}

type Notification struct {
	ID        int              `json:"id"`
	Name      string           `json:"name"`
	Type      NotificationType `json:"type"`
	Enabled   bool             `json:"enabled"`
	Events    []string         `json:"events"`
	Token     string           `json:"token"`
	APIKey    string           `json:"api_key"`
	Webhook   string           `json:"webhook"`
	Title     string           `json:"title"`
	Icon      string           `json:"icon"`
	Username  string           `json:"username"`
	Host      string           `json:"host"`
	Password  string           `json:"password"`
	Channel   string           `json:"channel"`
	Rooms     string           `json:"rooms"`
	Targets   string           `json:"targets"`
	Devices   string           `json:"devices"`
	Priority  int32            `json:"priority"`
	Topic     string           `json:"topic"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
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

func (n *Notification) UnmarshalJSON(data []byte) error {
	type Alias Notification
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(n),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// If any of the secret fields appear to be redacted, don't overwrite the existing values
	if isRedactedValue(n.Token) {
		// Keep the original token by not updating it
		// This assumes the original struct already had the real value
	}
	if isRedactedValue(n.APIKey) {
		// Keep the original api key by not updating it
	}
	if isRedactedValue(n.Password) {
		// Keep the original password by not updating it
	}

	return nil
}

type NotificationPayload struct {
	Subject        string
	Message        string
	Event          NotificationEvent
	ReleaseName    string
	Filter         string
	Indexer        string
	InfoHash       string
	Size           uint64
	Status         ReleasePushStatus
	Action         string
	ActionType     ActionType
	ActionClient   string
	Rejections     []string
	Protocol       ReleaseProtocol       // torrent, usenet
	Implementation ReleaseImplementation // irc, rss, api
	Timestamp      time.Time
	Sender         string
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
