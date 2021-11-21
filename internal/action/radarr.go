package action

import (
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/radarr"

	"github.com/rs/zerolog/log"
)

func (s *service) radarr(release domain.Release, action domain.Action) error {
	log.Trace().Msg("action RADARR")

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
	cfg := radarr.Config{
		Hostname: client.Host,
		APIKey:   client.Settings.APIKey,
	}

	// only set basic auth if enabled
	if client.Settings.Basic.Auth {
		cfg.BasicAuth = client.Settings.Basic.Auth
		cfg.Username = client.Settings.Basic.Username
		cfg.Password = client.Settings.Basic.Password
	}

	arr := radarr.New(cfg)

	r := radarr.Release{
		Title:            release.Name,
		DownloadUrl:      release.TorrentURL,
		Size:             int64(release.Size),
		Indexer:          release.Indexer,
		DownloadProtocol: "torrent",
		Protocol:         "torrent",
		PublishDate:      time.Now().Format(time.RFC3339),
	}

	success, err := arr.Push(r)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("radarr: failed to push release: %v", r)
		return err
	}

	if success {
		// TODO save pushed release
		log.Debug().Msgf("radarr: successfully pushed release: %v, indexer %v to %v", r.Title, r.Indexer, client.Host)
	}

	return nil
}
