package action

import (
	"context"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/radarr"
)

func (s *service) radarr(action domain.Action, release domain.Release) ([]string, error) {
	s.log.Trace().Msg("action RADARR")

	// TODO validate data

	// get client for action
	client, err := s.clientSvc.FindByID(context.TODO(), action.ClientID)
	if err != nil {
		return nil, errors.Wrap(err, "error finding client: %v", action.ClientID)
	}

	// return early if no client found
	if client == nil {
		return nil, errors.New("could not find client by id: %v", action.ClientID)
	}

	// initial config
	cfg := radarr.Config{
		Hostname: client.Host,
		APIKey:   client.Settings.APIKey,
		Log:      s.subLogger,
	}

	// only set basic auth if enabled
	if client.Settings.Basic.Auth {
		cfg.BasicAuth = client.Settings.Basic.Auth
		cfg.Username = client.Settings.Basic.Username
		cfg.Password = client.Settings.Basic.Password
	}

	arr := radarr.New(cfg)

	r := radarr.Release{
		Title:            release.TorrentName,
		DownloadUrl:      release.TorrentURL,
		Size:             int64(release.Size),
		Indexer:          release.Indexer,
		DownloadProtocol: string(release.Protocol),
		Protocol:         string(release.Protocol),
		PublishDate:      time.Now().Format(time.RFC3339),
	}

	rejections, err := arr.Push(r)
	if err != nil {
		return nil, errors.Wrap(err, "radarr failed to push release: %v", r)
	}

	if rejections != nil {
		s.log.Debug().Msgf("radarr: release push rejected: %v, indexer %v to %v reasons: '%v'", r.Title, r.Indexer, client.Host, rejections)

		return rejections, nil
	}

	s.log.Debug().Msgf("radarr: successfully pushed release: %v, indexer %v to %v", r.Title, r.Indexer, client.Host)

	return nil, nil
}
