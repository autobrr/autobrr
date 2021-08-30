package action

import (
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/sonarr"

	"github.com/rs/zerolog/log"
)

func (s *service) sonarr(announce domain.Announce, action domain.Action) error {
	log.Trace().Msg("action SONARR")

	// TODO validate data

	// get client for action
	client, err := s.clientSvc.FindByID(action.ClientID)
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

	r := sonarr.New(cfg)

	release := sonarr.Release{
		Title:            announce.TorrentName,
		DownloadUrl:      announce.TorrentUrl,
		Size:             0,
		Indexer:          announce.Site,
		DownloadProtocol: "torrent",
		Protocol:         "torrent",
		PublishDate:      time.Now().Format(time.RFC3339),
	}

	success, err := r.Push(release)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("sonarr: failed to push release: %v", release)
		return err
	}

	if success {
		// TODO save pushed release
		log.Debug().Msgf("sonarr: successfully pushed release: %v, indexer %v to %v", release.Title, release.Indexer, client.Host)
	}

	return nil
}
