package action

import (
	"os/exec"
	"time"

	"github.com/mattn/go-shellwords"

	"github.com/autobrr/autobrr/internal/domain"
)

func (s *service) execCmd(release domain.Release, action domain.Action) {
	s.log.Debug().Msgf("action exec: %v release: %v", action.Name, release.TorrentName)

	// check if program exists
	cmd, err := exec.LookPath(action.ExecCmd)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("exec failed, could not find program: %v", action.ExecCmd)
		return
	}

	args, err := s.parseExecArgs(release, action.ExecArgs)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("parsing args failed: command: %v args: %v torrent: %v", cmd, action.ExecArgs, release.TorrentTmpFile)
		return
	}

	// we need to split on space into a string slice, so we can spread the args into exec

	start := time.Now()

	// setup command and args
	command := exec.Command(cmd, args...)

	// execute command
	output, err := command.CombinedOutput()
	if err != nil {
		// everything other than exit 0 is considered an error
		s.log.Error().Stack().Err(err).Msgf("command: %v args: %v failed, torrent: %v", cmd, args, release.TorrentTmpFile)
		return
	}

	s.log.Trace().Msgf("executed command: '%v'", string(output))

	duration := time.Since(start)

	s.log.Info().Msgf("executed command: '%v', args: '%v' %v,%v, total time %v", cmd, args, release.TorrentName, release.Indexer, duration)
}

func (s *service) parseExecArgs(release domain.Release, execArgs string) ([]string, error) {
	// handle args and replace vars
	m := NewMacro(release)

	// parse and replace values in argument string before continuing
	parsedArgs, err := m.Parse(execArgs)
	if err != nil {
		return nil, err
	}

	p := shellwords.NewParser()
	p.ParseBacktick = true
	args, err := p.Parse(parsedArgs)
	if err != nil {
		return nil, err
	}

	return args, nil
}
