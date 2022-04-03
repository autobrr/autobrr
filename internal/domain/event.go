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
	Rejections     []string
	Protocol       ReleaseProtocol       // torrent
	Implementation ReleaseImplementation // irc, rss, api
	Timestamp      time.Time
}
