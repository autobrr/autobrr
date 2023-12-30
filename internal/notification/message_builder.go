package notification

import (
	"fmt"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/dustin/go-humanize"
)

type NotificationBuilderPlainText struct{}

// BuildBody constructs the body of the notification message.
func (b *NotificationBuilderPlainText) BuildBody(payload domain.NotificationPayload) string {
	var parts []string

	buildPart := func(condition bool, format string, a ...interface{}) {
		if condition {
			parts = append(parts, fmt.Sprintf(format, a...))
		}
	}

	buildPart(payload.Subject != "" && payload.Message != "", "%v\n%v", payload.Subject, payload.Message)
	buildPart(payload.ReleaseName != "", "\nNew release: %v", payload.ReleaseName)
	buildPart(payload.Size > 0, "\nSize: %v", humanize.Bytes(payload.Size))
	buildPart(payload.Status != "", "\nStatus: %v", payload.Status.String())
	buildPart(payload.Indexer != "", "\nIndexer: %v", payload.Indexer)
	buildPart(payload.Filter != "", "\nFilter: %v", payload.Filter)
	buildPart(payload.Action != "", "\nAction: %v: %v", payload.ActionType, payload.Action)
	if payload.Action != "" && payload.ActionClient != "" {
		parts = append(parts, fmt.Sprintf(" Client: %v", payload.ActionClient))
	}
	buildPart(len(payload.Rejections) > 0, "\nRejections: %v", strings.Join(payload.Rejections, ", "))

	return strings.Join(parts, "\n")
}

// BuildTitle constructs the title of the notification message.
func (b *NotificationBuilderPlainText) BuildTitle(event domain.NotificationEvent) string {
	titles := map[domain.NotificationEvent]string{
		domain.NotificationEventAppUpdateAvailable: "Autobrr update available",
		domain.NotificationEventPushApproved:       "Push Approved",
		domain.NotificationEventPushRejected:       "Push Rejected",
		domain.NotificationEventPushError:          "Push Error",
		domain.NotificationEventIRCDisconnected:    "IRC Disconnected",
		domain.NotificationEventIRCReconnected:     "IRC Reconnected",
		domain.NotificationEventTest:               "Test",
	}

	if title, ok := titles[event]; ok {
		return title
	}

	return "New Event"
}
