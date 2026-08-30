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
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			// not an exit error (e.g. the command could not be started) - hard error
			return errors.Wrap(err, "error executing command: %s args: %s", cmd, args)
		}

		// the command ran but exited with a non-zero code, only treat it as an
		// error when it does not match the configured expected exit status.
		if exitCode := exitErr.ExitCode(); exitCode != action.ExecExpectStatus {
			return errors.Wrap(err, "command exited with unexpected exit code: %d (expected %d) command: %s args: %s", exitCode, action.ExecExpectStatus, cmd, args)
		}
	} else if action.ExecExpectStatus != 0 {
		// the command exited 0 but a non-zero exit status was expected
		return errors.New("command exited with code 0 but expected %d - command: %s args: %s", action.ExecExpectStatus, cmd, args)
	}

	l.Trace().Str("output", string(output)).Msg("executed command")

	duration := time.Since(start)

	l.Info().Str("cmd", cmd).Strs("args", args).Str("indexer", release.Indexer.Identifier).Dur("duration", duration).Msg("executed command")

	return nil
}
