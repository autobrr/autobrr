// Copyright (c) 2021-2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestNotificationBuilderPlainText_BuildBody(t *testing.T) {
	t.Parallel()
	type args struct {
		payload domain.NotificationPayload
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "build body",
			args: args{payload: domain.NotificationPayload{
				Subject:        "",
				Message:        "",
				Event:          domain.NotificationEventPushApproved,
				ReleaseName:    "Movie 2024 UHD BluRay 2160p DTS-HD MA 5.1 DV HEVC HYBRID REMUX-GROUP",
				Filter:         "test",
				Indexer:        "mock",
				InfoHash:       "",
				Size:           0,
				Status:         domain.ReleasePushStatusApproved,
				Action:         "mock",
				ActionType:     domain.ActionTypeQbittorrent,
				ActionClient:   "mock",
				Rejections:     nil,
				Protocol:       domain.ReleaseProtocolTorrent,
				Implementation: domain.ReleaseImplementationIRC,
				Timestamp:      time.Time{},
			}},
			want: "New release: Movie 2024 UHD BluRay 2160p DTS-HD MA 5.1 DV HEVC HYBRID REMUX-GROUP\nStatus: Approved\nIndexer: mock\nFilter: test\nAction: QBITTORRENT: mock\nClient: mock\n",
		},
		{
			name: "build body with rejections",
			args: args{payload: domain.NotificationPayload{
				Subject:        "",
				Message:        "",
				Event:          domain.NotificationEventPushRejected,
				ReleaseName:    "Movie 2024 UHD BluRay 2160p DTS-HD MA 5.1 DV HEVC HYBRID REMUX-GROUP",
				Filter:         "test",
				Indexer:        "mock",
				InfoHash:       "",
				Size:           0,
				Status:         domain.ReleasePushStatusRejected,
				Action:         "mock",
				ActionType:     domain.ActionTypeRadarr,
				ActionClient:   "mock",
				Rejections:     []string{"Item already exists"},
				Protocol:       domain.ReleaseProtocolTorrent,
				Implementation: domain.ReleaseImplementationIRC,
				Timestamp:      time.Time{},
			}},
			want: "New release: Movie 2024 UHD BluRay 2160p DTS-HD MA 5.1 DV HEVC HYBRID REMUX-GROUP\nStatus: Rejected\nIndexer: mock\nFilter: test\nAction: RADARR: mock\nClient: mock\nRejections: Item already exists\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &MessageBuilderPlainText{}
			assert.Equal(t, tt.want, b.BuildBody(tt.args.payload))
		})
	}
}
