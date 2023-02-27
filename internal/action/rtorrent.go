package action

import (
	"context"
	"os"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/mrobinsn/go-rtorrent/rtorrent"
)

func (s *service) rtorrent(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error) {
	s.log.Debug().Msgf("action rTorrent: %s", action.Name)

	var err error

	// get client for action
	client, err := s.clientSvc.FindByID(ctx, action.ClientID)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("error finding client: %d", action.ClientID)
		return nil, err
	}

	if client == nil {
		return nil, errors.New("could not find client by id: %d", action.ClientID)
	}

	var rejections []string

	// create client
	rt := rtorrent.New(client.Host, true)

	if release.HasMagnetUri() {
		if err := release.ResolveMagnetUri(ctx); err != nil {
			return nil, err
		}

		var args []*rtorrent.FieldValue

		if action.Label != "" {
			args = append(args, &rtorrent.FieldValue{
				Field: rtorrent.DLabel,
				Value: action.Label,
			})
		}
		if action.SavePath != "" {
			args = append(args, &rtorrent.FieldValue{
				Field: rtorrent.DDirectory,
				Value: action.SavePath,
			})
		}

		if err := rt.Add(release.MagnetURI, args...); err != nil {
			return nil, errors.Wrap(err, "could not add torrent from magnet: %s", release.MagnetURI)
		}

		s.log.Info().Msgf("torrent from magnet successfully added to client: '%s'", client.Name)

		return nil, nil

	} else {
		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFileCtx(ctx); err != nil {
				s.log.Error().Err(err).Msgf("could not download torrent file for release: %s", release.TorrentName)
				return nil, err
			}
		}

		tmpFile, err := os.ReadFile(release.TorrentTmpFile)
		if err != nil {
			return nil, errors.Wrap(err, "could not read torrent file: %s", release.TorrentTmpFile)
		}

		var args []*rtorrent.FieldValue

		if action.Label != "" {
			args = append(args, &rtorrent.FieldValue{
				Field: rtorrent.DLabel,
				Value: action.Label,
			})
		}
		if action.SavePath != "" {
			args = append(args, &rtorrent.FieldValue{
				Field: rtorrent.DDirectory,
				Value: action.SavePath,
			})
		}

		if err := rt.AddTorrent(tmpFile, args...); err != nil {
			return nil, errors.Wrap(err, "could not add torrent file: %s", release.TorrentTmpFile)
		}

		s.log.Info().Msgf("torrent successfully added to client: '%s'", client.Name)
	}

	return rejections, nil
}
