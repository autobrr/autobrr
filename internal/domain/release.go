package domain

import (
	"context"
	"time"
)

type ReleaseRepo interface {
	Store(release Release) (*Release, error)
	Find(ctx context.Context, params QueryParams) (res []Release, nextCursor int64, err error)
}

type Release struct {
	ID         int64         `json:"id"`
	Status     ReleaseStatus `json:"status"`
	Rejections []string      `json:"rejections"`
	Indexer    string        `json:"indexer"`
	Client     string        `json:"client"`
	Protocol   string        `json:"protocol"`
	Title      string        `json:"title"`
	Size       string        `json:"size"`
	Raw        string        `json:"raw"`
	CreatedAt  time.Time     `json:"created_at"`
}

type ReleaseStatus string

const (
	ReleaseStatusFiltered       ReleaseStatus = "FILTERED"
	ReleaseStatusFilterRejected ReleaseStatus = "FILTER_REJECTED"
	ReleaseStatusPushApproved   ReleaseStatus = "PUSH_APPROVED"
	ReleaseStatusPushRejected   ReleaseStatus = "PUSH_REJECTED"
)

type QueryParams struct {
	Limit  uint64
	Cursor uint64
	Sort   map[string]string
	Filter map[string]string
	Search string
}
