package action

import (
	"context"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/sonarr"

	"github.com/rs/zerolog/log"
)

func (s *service) sonarr(release domain.Release, action domain.Action) error {
	log.Trace().Msg("action SONARR")

	// TODO validate data

	// get client for action
	client, err := s.clientSvc.FindByID(context.TODO(), action.ClientID)
	if err != nil {
		log.Error().Err(err).Msgf("error finding client: %v", action.ClientID)
		return err
	}

	// return early if no client found
	if client == nil {
		return err
	}

	// initial config
	cfg := sonarr.Config{
		Hostname: client.Host,
		APIKey:   client.Settings.APIKey,
	}

	// only set basic auth if enabled
	if client.Settings.Basic.Auth {
		cfg.BasicAuth = client.Settings.Basic.Auth
		cfg.Username = client.Settings.Basic.Username
		cfg.Password = client.Settings.Basic.Password
	}

	arr := sonarr.New(cfg)

	r := sonarr.Release{
		Title:            release.TorrentName,
		DownloadUrl:      release.TorrentURL,
		Size:             int64(release.Size),
		Indexer:          release.Indexer,
		DownloadProtocol: "torrent",
		Protocol:         "torrent",
		PublishDate:      time.Now().Format(time.RFC3339),
	}

	success, rejections, err := arr.Push(r)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("sonarr: failed to push release: %v", r)
		return err
	}

	if !success {
		log.Debug().Msgf("sonarr: release push rejected: %v, indexer %v to %v reasons: '%v'", r.Title, r.Indexer, client.Host, rejections)

		// save pushed release
		s.bus.Publish("release:update-push-status-rejected", release.ID, rejections)
		return nil
	}

	log.Debug().Msgf("sonarr: successfully pushed release: %v, indexer %v to %v", r.Title, r.Indexer, client.Host)

	s.bus.Publish("release:update-push-status", release.ID, domain.ReleasePushStatusApproved)

	return nil
}
