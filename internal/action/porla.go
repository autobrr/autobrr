package action

import (
	"bufio"
	"context"
	"encoding/base64"
	"io/ioutil"
	"os"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/porla"
	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/rs/zerolog"
)

func (s *service) porla(action domain.Action, release domain.Release) ([]string, error) {
	s.log.Debug().Msgf("action Porla: %v", action.Name)

	client, err := s.clientSvc.FindByID(context.TODO(), action.ClientID)
	if err != nil {
		return nil, errors.Wrap(err, "error finding client: %v", action.ClientID)
	}

	if client == nil {
		return nil, errors.New("could not find client by id: %v", action.ClientID)
	}

	porlaSettings := porla.Settings{
		Hostname:  client.Host,
		AuthToken: client.Settings.APIKey,
	}

	porlaSettings.Log = zstdlog.NewStdLoggerWithLevel(s.log.With().Str("type", "Porla").Str("client", client.Name).Logger(), zerolog.TraceLevel)

	prl := porla.NewClient(porlaSettings)

	if release.TorrentTmpFile == "" {
		if err := release.DownloadTorrentFile(); err != nil {
			return nil, errors.Wrap(err, "error downloading torrent file for release: %v", release.TorrentName)
		}
	}

	file, err := os.Open(release.TorrentTmpFile)
	if err != nil {
		return nil, errors.Wrap(err, "error opening file %v", release.TorrentTmpFile)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	content, err := ioutil.ReadAll(reader)

	if err != nil {
		return nil, errors.Wrap(err, "failed to read file: %v", release.TorrentTmpFile)
	}

	opts := &porla.TorrentsAddReq{
		SavePath: action.SavePath,
		Ti:       base64.StdEncoding.EncodeToString(content),
	}

	if err = prl.TorrentsAdd(opts); err != nil {
		return nil, errors.Wrap(err, "could not add torrent %v to client: %v", release.TorrentTmpFile, client.Name)
	}

	s.log.Info().Msgf("torrent with hash %v successfully added to client: '%v'", release.TorrentHash, client.Name)

	return nil, nil
}
