package action

import (
	"context"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/avast/retry-go"

	"github.com/hekmon/transmissionrpc/v2"
)

func (s *service) transmission(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error) {
	s.log.Debug().Msgf("action Transmission: %v", action.Name)

	var err error

	// get client for action
	client, err := s.clientSvc.FindByID(ctx, action.ClientID)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("error finding client: %v", action.ClientID)
		return nil, err
	}

	if client == nil {
		return nil, errors.New("could not find client by id: %v", action.ClientID)
	}

	var rejections []string

	if release.TorrentTmpFile == "" {
		if err := release.DownloadTorrentFileCtx(ctx); err != nil {
			s.log.Error().Err(err).Msgf("could not download torrent file for release: %v", release.TorrentName)
			return nil, err
		}
	}

	tbt, err := transmissionrpc.New(client.Host, client.Username, client.Password, &transmissionrpc.AdvancedConfig{
		HTTPS: client.TLS,
		Port:  uint16(client.Port),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error logging into client: %v", client.Host)
	}

	b64, err := transmissionrpc.File2Base64(release.TorrentTmpFile)
	if err != nil {
		return nil, errors.Wrap(err, "cant encode file %v into base64", release.TorrentTmpFile)
	}

	payload := transmissionrpc.TorrentAddPayload{
		MetaInfo: &b64,
	}
	if action.SavePath != "" {
		payload.DownloadDir = &action.SavePath
	}
	if action.Paused {
		payload.Paused = &action.Paused
	}

	// Prepare and send payload
	torrent, err := tbt.TorrentAdd(ctx, payload)
	if err != nil {
		return nil, errors.Wrap(err, "could not add torrent %v to client: %v", release.TorrentTmpFile, client.Host)
	}

	s.log.Info().Msgf("torrent with hash %v successfully added to client: '%v'", torrent.HashString, client.Name)

	if !action.ReAnnounceSkip {
		err = retry.Do(func() error {
			if err := tbt.TorrentReannounceIDs(ctx, []int64{*torrent.ID}); err != nil {
				return errors.Wrap(err, "failed to reannounce")
			}

			t, err := tbt.TorrentGet(ctx, []string{"trackerStats"}, []int64{*torrent.ID})
			if err != nil {
				return errors.Wrap(err, "reannounced, failed to find torrentid")
			}

			if len(t) < 1 {
				return errors.Wrap(err, "reannounced, failed to get torrent from id")
			}

			seeds := int64(0)
			for _, trackers := range t[0].TrackerStats {
				seeds += trackers.SeederCount
			}

			if seeds != 0 {
				return nil
			}

			retry.Delay(time.Second * time.Duration(action.ReAnnounceInterval))

			return errors.New("no seeds yet")
		},
			retry.OnRetry(func(n uint, err error) {
				s.log.Error().Err(err).Msgf("%q: attempt %d - %v\n", err, n, int(action.ReAnnounceMaxAttempts))
			}),
			//retry.Delay(time.Second*3),
			retry.Attempts(uint(action.ReAnnounceMaxAttempts)),
			retry.MaxJitter(time.Second*1),
		)

		if err != nil && action.ReAnnounceDelete {
			tbt.TorrentRemove(ctx, transmissionrpc.TorrentRemovePayload{IDs: []int64{*torrent.ID}})
		}
	}

	return rejections, nil
}
