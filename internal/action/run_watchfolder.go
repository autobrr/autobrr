package action

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

func (s *Service) runWatchFolder(ctx context.Context, action *domain.Action, release *domain.Release) error {
	l := zerolog.Ctx(ctx)

	if release.HasMagnetUri() {
		return fmt.Errorf("action watch folder does not support magnet links: %s", release.TorrentName)
	}

	l.Debug().Str("watch_folder", action.WatchFolder).Str("release", release.TorrentName).Msg("running Watch Folder action")

	if len(release.TorrentDataRawBytes) < 1 {
		return fmt.Errorf("watch_folder: missing torrent %s", release.TorrentName)
	}

	// default dir to watch folder
	//  /mnt/watch/{{.Indexer}}
	//  /mnt/watch/mock
	//  /mnt/watch/{{.Indexer}}-{{.TorrentName}}.torrent
	//  /mnt/watch/mock-Torrent.Name-GROUP.torrent
	dir := action.WatchFolder
	newFileName := action.WatchFolder

	// if watchFolderArgs does not contain .torrent, create
	if !strings.HasSuffix(action.WatchFolder, ".torrent") {
		// The torrent is no longer backed by a tmp file, so name it after the
		// infohash. It is unique, safe to use as a file name as it stands, and
		// set alongside the raw bytes. Anything richer belongs in a client or
		// one of our other tools rather than in the file name.
		newFileName = filepath.Join(action.WatchFolder, "autobrr-"+release.TorrentHash+".torrent")
	} else {
		dir, _ = filepath.Split(action.WatchFolder)
	}

	// Create folder
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return errors.Wrap(err, "could not create new folders %v", dir)
	}

	// Create new file
	newFile, err := os.Create(newFileName)
	if err != nil {
		return errors.Wrap(err, "could not create new file %v", newFileName)
	}
	defer newFile.Close()

	// Copy file
	if _, err := io.Copy(newFile, release.TorrentReader()); err != nil {
		return errors.Wrap(err, "could not copy file %v to watch folder", newFileName)
	}

	l.Info().Str("file", newFileName).Msg("saved file to watch folder")

	return nil
}
