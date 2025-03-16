// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
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
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	Name      string           `json:"name"`
	Type      NotificationType `json:"type"`
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
	Topic     string           `json:"topic"`
	Events    []string         `json:"events"`
	ID        int              `json:"id"`
	Priority  int32            `json:"priority"`
	Enabled   bool             `json:"enabled"`
}

type NotificationPayload struct {
	Timestamp      time.Time
	Subject        string
	Message        string
	Event          NotificationEvent
	ReleaseName    string
	Filter         string
	Indexer        string
	InfoHash       string
	Status         ReleasePushStatus
	Action         string
	ActionType     ActionType
	ActionClient   string
	Protocol       ReleaseProtocol       // torrent, usenet
	Implementation ReleaseImplementation // irc, rss, api
	Sender         string
	Rejections     []string
	Size           uint64
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
	Sort    map[string]string
	Filters struct {
		PushStatus string
		Indexers   []string
	}
	Search string
	Limit  uint64
	Offset uint64
}
