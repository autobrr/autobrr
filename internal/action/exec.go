// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"slices"
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

	return parseExecRejections(output), nil
}

// parseExecRejections reads the trailing block of "REJECT:"-prefixed lines from a script's
// output. Only lines at the very end of the output are considered, so scripts can log freely
// before signalling rejection; the scan stops at the first line (from the bottom) that isn't
// a REJECT: line, and any output before that is ignored for backwards compatibility.
func parseExecRejections(output []byte) []string {
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	var rejections []string
	for i := len(lines) - 1; i >= 0; i-- {
		reason, ok := strings.CutPrefix(lines[i], "REJECT:")
		if !ok {
			break
		}
		if reason = strings.TrimSpace(reason); reason != "" {
			rejections = append(rejections, reason)
		}
	}

	slices.Reverse(rejections)

	return rejections
}
