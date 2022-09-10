package action

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/mattn/go-shellwords"
)

func (s *service) execCmd(action domain.Action, release domain.Release) error {
	s.log.Debug().Msgf("action exec: %v release: %v", action.Name, release.TorrentName)

	if release.TorrentTmpFile == "" && strings.Contains(action.ExecArgs, "TorrentPathName") {
		if err := release.DownloadTorrentFile(); err != nil {
			return errors.Wrap(err, "error downloading torrent file for release: %v", release.TorrentName)
		}
	}

	// read the file into bytes we can then use in the macro
	if len(release.TorrentDataRawBytes) == 0 && release.TorrentTmpFile != "" {
		t, err := os.ReadFile(release.TorrentTmpFile)
		if err != nil {
			return errors.Wrap(err, "could not read torrent file: %v", release.TorrentTmpFile)
		}

		release.TorrentDataRawBytes = t
	}

	// check if program exists
	cmd, err := exec.LookPath(action.ExecCmd)
	if err != nil {
		return errors.Wrap(err, "exec failed, could not find program: %v", action.ExecCmd)
	}

	args, err := s.parseExecArgs(release, action.ExecArgs)
	if err != nil {
		return errors.Wrap(err, "could not parse exec args: %v", action.ExecArgs)
	}

	// we need to split on space into a string slice, so we can spread the args into exec

	start := time.Now()

	// setup command and args
	command := exec.Command(cmd, args...)

	// execute command
	output, err := command.CombinedOutput()
	if err != nil {
		// everything other than exit 0 is considered an error
		return errors.Wrap(err, "error executing command: %v args: %v", cmd, args)
	}

	s.log.Trace().Msgf("executed command: '%v'", string(output))

	duration := time.Since(start)

	s.log.Info().Msgf("executed command: '%v', args: '%v' %v,%v, total time %v", cmd, args, release.TorrentName, release.Indexer, duration)

	return nil
}

func (s *service) parseExecArgs(release domain.Release, execArgs string) ([]string, error) {
	// handle args and replace vars
	m := domain.NewMacro(release)

	// parse and replace values in argument string before continuing
	parsedArgs, err := m.Parse(execArgs)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse macro")
	}

	p := shellwords.NewParser()
	p.ParseBacktick = true
	args, err := p.Parse(parsedArgs)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse into shell-words")
	}

	return args, nil
}
