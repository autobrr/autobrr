package action

import (
	"context"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"os"

	"github.com/mrobinsn/go-rtorrent/rtorrent"
)

func (s *service) rtorrent(action domain.Action, release domain.Release) ([]string, error) {
	s.log.Debug().Msgf("action rTorrent: %v", action.Name)

	var err error

	// get client for action
	client, err := s.clientSvc.FindByID(context.TODO(), action.ClientID)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("error finding client: %v", action.ClientID)
		return nil, err
	}

	if client == nil {
		return nil, errors.New("could not find client by id: %v", action.ClientID)
	}

	var rejections []string

	if release.TorrentTmpFile == "" {
		if err := release.DownloadTorrentFile(); err != nil {
			s.log.Error().Err(err).Msgf("could not download torrent file for release: %v", release.TorrentName)
			return nil, err
		}
	}

	// create client
	rt := rtorrent.New(client.Host, true)

	tmpFile, err := os.ReadFile(release.TorrentTmpFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not read torrent file: %v", release.TorrentTmpFile)
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
		return nil, errors.Wrap(err, "could not add torrent file: %v", release.TorrentTmpFile)
	}

	s.log.Info().Msgf("torrent with hash %v successfully added to client: '%v'", "", client.Name)

	return rejections, nil
}
