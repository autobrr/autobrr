// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/Hellseher/go-shellquote"
	"github.com/rs/zerolog"
)

func (s *Service) execCmd(ctx context.Context, action *domain.Action, release *domain.Release) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Debug().Msg("running Exec action")

	cmd, err := exec.LookPath(action.ExecCmd)
	if err != nil {
		return nil, errors.Wrap(err, "exec failed, could not find program: %s", action.ExecCmd)
	}

	args, err := shellquote.Split(action.ExecArgs)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse exec args: %s", action.ExecArgs)
	}

	start := time.Now()

	command := exec.CommandContext(ctx, cmd, args...)

	output, err := command.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "error executing command: %s args: %s", cmd, args)
	}

	l.Trace().Str("output", string(output)).Msg("executed command")

	duration := time.Since(start)

	l.Info().Str("cmd", cmd).Strs("args", args).Str("indexer", release.Indexer.Identifier).Dur("duration", duration).Msg("executed command")

	// Scripts signal rejection by printing one or more lines prefixed with "REJECT:".
	// Any other output is ignored, preserving backwards compatibility with existing scripts.
	var rejections []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		if line := scanner.Text(); strings.HasPrefix(line, "REJECT:") {
			if reason := strings.TrimSpace(strings.TrimPrefix(line, "REJECT:")); reason != "" {
				rejections = append(rejections, reason)
			}
		}
	}

	return rejections, nil
}
