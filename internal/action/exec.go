// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"os/exec"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/mattn/go-shellwords"
)

func (s *service) execCmd(ctx context.Context, action *domain.Action, release domain.Release) error {
	s.log.Debug().Msgf("action exec: %s release: %s", action.Name, release.TorrentName)

	// check if program exists
	cmd, err := exec.LookPath(action.ExecCmd)
	if err != nil {
		return errors.Wrap(err, "exec failed, could not find program: %s", action.ExecCmd)
	}

	p := shellwords.NewParser()
	p.ParseBacktick = true
	args, err := p.Parse(action.ExecArgs)
	if err != nil {
		return errors.Wrap(err, "could not parse exec args: %s", action.ExecArgs)
	}

	// we need to split on space into a string slice, so we can spread the args into exec

	start := time.Now()

	// setup command and args
	command := exec.CommandContext(ctx, cmd, args...)

	// execute command
	output, err := command.CombinedOutput()
	if err != nil {
		// everything other than exit 0 is considered an error
		return errors.Wrap(err, "error executing command: %s args: %s", cmd, args)
	}

	s.log.Trace().Msgf("executed command: '%s'", string(output))

	duration := time.Since(start)

	s.log.Info().Msgf("executed command: '%s', args: '%s' %s,%s, total time %v", cmd, args, release.TorrentName, release.Indexer.Name, duration)

	return nil
}
