package action

import (
	"os/exec"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog/log"
)

func (s *service) execCmd(announce domain.Announce, action domain.Action, torrentFile string) {
	log.Trace().Msgf("action EXEC: release: %v", announce.TorrentName)

	// check if program exists
	cmd, err := exec.LookPath(action.ExecCmd)
	if err != nil {
		log.Error().Err(err).Msgf("exec failed, could not find program: %v", action.ExecCmd)
		return
	}

	// handle args and replace vars
	m := Macro{
		TorrentName:     announce.TorrentName,
		TorrentPathName: torrentFile,
		TorrentUrl:      announce.TorrentUrl,
	}

	// parse and replace values in argument string before continuing
	parsedArgs, err := m.Parse(action.ExecArgs)
	if err != nil {
		log.Error().Err(err).Msgf("exec failed, could not parse arguments: %v", action.ExecCmd)
		return
	}

	// we need to split on space into a string slice, so we can spread the args into exec
	args := strings.Split(parsedArgs, " ")

	start := time.Now()

	// setup command and args
	command := exec.Command(cmd, args...)

	// execute command
	output, err := command.CombinedOutput()
	if err != nil {
		// everything other than exit 0 is considered an error
		log.Error().Err(err).Msgf("command: %v args: %v failed, torrent: %v", cmd, parsedArgs, torrentFile)
	}

	log.Trace().Msgf("executed command: '%v'", string(output))

	duration := time.Since(start)

	log.Info().Msgf("executed command: '%v', args: '%v' %v,%v, total time %v", cmd, parsedArgs, announce.TorrentName, announce.Site, duration)
}
