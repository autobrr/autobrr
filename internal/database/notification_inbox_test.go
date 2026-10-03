// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package database

import (
	"fmt"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationInboxRepo_Lifecycle(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		repo := NewNotificationInboxRepo(setupLoggerForTest(), testDb.db)

		t.Run(fmt.Sprintf("Lifecycle [%s]", dbType), func(t *testing.T) {
			require.NoError(t, repo.DeleteAll(ctx))
			t.Cleanup(func() { _ = repo.DeleteAll(ctx) })

			now := time.Now().UTC()
			messages := []*domain.InboxMessage{
				{Event: domain.NotificationEventPushError, Title: "Push Error", Message: "old", CreatedAt: now.Add(-48 * time.Hour)},
				{Event: domain.NotificationEventIRCDisconnected, Title: "IRC Disconnected", Message: "middle", CreatedAt: now.Add(-time.Hour)},
				{Event: domain.NotificationEventPushError, Title: "Push Error", Message: "new", ReleaseName: "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP", FilterName: "TV", FilterID: 4, Rejections: []string{"error pushing to client"}, CreatedAt: now},
			}
			for _, msg := range messages {
				require.NoError(t, repo.Store(ctx, msg))
				assert.NotZero(t, msg.ID)
			}

			resp, err := repo.Find(ctx, domain.InboxQueryParams{Limit: 2})
			require.NoError(t, err)
			assert.Equal(t, 3, resp.TotalCount)
			assert.Equal(t, 3, resp.AllCount)
			assert.Equal(t, 3, resp.UnreadCount)
			require.Len(t, resp.Data, 2)
			assert.Equal(t, "new", resp.Data[0].Message)
			assert.Equal(t, "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP", resp.Data[0].ReleaseName)
			assert.Equal(t, "TV", resp.Data[0].FilterName)
			assert.Equal(t, 4, resp.Data[0].FilterID)
			assert.Equal(t, []string{"error pushing to client"}, resp.Data[0].Rejections)
			assert.Equal(t, []string{}, resp.Data[1].Rejections)
			assert.Nil(t, resp.Data[0].ReadAt)

			resp, err = repo.Find(ctx, domain.InboxQueryParams{Limit: 10, Events: []string{string(domain.NotificationEventPushError)}})
			require.NoError(t, err)
			assert.Equal(t, 2, resp.AllCount)
			require.Len(t, resp.Data, 2)
			assert.Equal(t, domain.NotificationEventPushError, resp.Data[1].Event)

			require.NoError(t, repo.MarkRead(ctx, []int64{messages[2].ID}, now))
			require.NoError(t, repo.MarkRead(ctx, []int64{messages[2].ID}, now.Add(time.Hour)))

			resp, err = repo.Find(ctx, domain.InboxQueryParams{Limit: 10, Unread: true})
			require.NoError(t, err)
			assert.Equal(t, 2, resp.TotalCount)
			assert.Equal(t, 3, resp.AllCount)
			assert.Equal(t, 2, resp.UnreadCount)
			require.Len(t, resp.Data, 2)
			assert.Equal(t, "middle", resp.Data[0].Message)

			resp, err = repo.Find(ctx, domain.InboxQueryParams{Limit: 1})
			require.NoError(t, err)
			require.NotNil(t, resp.Data[0].ReadAt)
			assert.WithinDuration(t, now, *resp.Data[0].ReadAt, time.Second)

			require.NoError(t, repo.MarkRead(ctx, nil, now))
			resp, err = repo.Find(ctx, domain.InboxQueryParams{Limit: 10})
			require.NoError(t, err)
			assert.Equal(t, 0, resp.UnreadCount)

			deleted, err := repo.Cleanup(ctx, domain.InboxCleanupParams{MaxMessages: 2, OlderThan: now.Add(-24 * time.Hour)})
			require.NoError(t, err)
			assert.Equal(t, int64(1), deleted)

			require.NoError(t, repo.Delete(ctx, []int64{messages[1].ID, messages[2].ID}))
			resp, err = repo.Find(ctx, domain.InboxQueryParams{Limit: 10})
			require.NoError(t, err)
			assert.Empty(t, resp.Data)
		})
	}
}
