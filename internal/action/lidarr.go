package action

import (
	"context"
	"fmt"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/lidarr"
)

func (s *service) lidarr(release domain.Release, action domain.Action) ([]string, error) {
	s.log.Trace().Msg("action LIDARR")

	// TODO validate data

	// get client for action
	client, err := s.clientSvc.FindByID(context.TODO(), action.ClientID)
	if err != nil {
		s.log.Error().Err(err).Msgf("lidarr: error finding client: %v", action.ClientID)
		return nil, err
	}

	// return early if no client found
	if client == nil {
		return nil, err
	}

	// initial config
	cfg := lidarr.Config{
		Hostname: client.Host,
		APIKey:   client.Settings.APIKey,
	}

	// only set basic auth if enabled
	if client.Settings.Basic.Auth {
		cfg.BasicAuth = client.Settings.Basic.Auth
		cfg.Username = client.Settings.Basic.Username
		cfg.Password = client.Settings.Basic.Password
	}

	arr := lidarr.New(cfg)

	r := lidarr.Release{
		Title:            release.TorrentName,
		DownloadUrl:      release.TorrentURL,
		Size:             int64(release.Size),
		Indexer:          release.Indexer,
		DownloadProtocol: "torrent",
		Protocol:         "torrent",
		PublishDate:      time.Now().Format(time.RFC3339),
	}

	// special handling for RED and OPS because their torrent names contain to little info
	// "Artist - Album" is not enough for Lidarr to make a decision. It needs year like "Artist - Album 2022"
	if release.Indexer == "redacted" || release.Indexer == "ops" {
		r.Title = fmt.Sprintf("%v (%d)", release.TorrentName, release.Year)
	}

	rejections, err := arr.Push(r)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("lidarr: failed to push release: %v", r)
		return nil, err
	}

	if rejections != nil {
		s.log.Debug().Msgf("lidarr: release push rejected: %v, indexer %v to %v reasons: '%v'", r.Title, r.Indexer, client.Host, rejections)

		return rejections, nil
	}

	s.log.Debug().Msgf("lidarr: successfully pushed release: %v, indexer %v to %v", r.Title, r.Indexer, client.Host)

	return nil, nil
}
