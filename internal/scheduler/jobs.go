package scheduler

import (
	"context"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/pkg/version"

	"github.com/rs/zerolog"
)

type CheckUpdatesJob struct {
	Name     string
	Log      zerolog.Logger
	Version  string
	NotifSvc notification.Service

	lastCheckVersion string
}

func (j *CheckUpdatesJob) Run() {
	v := version.Checker{
		Owner:          "autobrr",
		Repo:           "autobrr",
		CurrentVersion: j.Version,
	}

	newAvailable, newVersion, err := v.CheckNewVersion(context.TODO(), j.Version)
	if err != nil {
		j.Log.Error().Err(err).Msg("could not check for new release")
		return
	}

	if newAvailable {
		j.Log.Info().Msgf("a new release has been found: %v Consider updating.", newVersion)

		// this is not persisted so this can trigger more than once
		// lets check if we have different versions between runs
		if newVersion.TagName != j.lastCheckVersion {
			j.NotifSvc.Send(domain.NotificationEventAppUpdateAvailable, domain.NotificationPayload{
				Subject:   "New update available!",
				Message:   newVersion.TagName,
				Event:     domain.NotificationEventAppUpdateAvailable,
				Timestamp: time.Now(),
			})
		}

		j.lastCheckVersion = newVersion.TagName
	}
}
