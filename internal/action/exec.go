// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"os/exec"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/Hellseher/go-shellquote"
	"github.com/rs/zerolog"
)

func (s *Service) execCmd(ctx context.Context, action *domain.Action, release *domain.Release) error {
	l := zerolog.Ctx(ctx)

	l.Debug().Msg("running Exec action")

	// check if program exists
	cmd, err := exec.LookPath(action.ExecCmd)
	if err != nil {
		return errors.Wrap(err, "exec failed, could not find program: %s", action.ExecCmd)
	}

	args, err := shellquote.Split(action.ExecArgs)
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

	l.Trace().Str("output", string(output)).Msg("executed command")

	duration := time.Since(start)

	l.Info().Str("cmd", cmd).Strs("args", args).Str("indexer", release.Indexer.Identifier).Dur("duration", duration).Msg("executed command")

	return nil
}
