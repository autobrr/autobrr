// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import "time"

type EventsReleasePushed struct {
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
}
