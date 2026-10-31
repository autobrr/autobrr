// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package cli

import (
	"github.com/autobrr/autobrr/internal/application"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/spf13/cobra"
)

func CommandVersion() *cobra.Command {
	var command = &cobra.Command{
		Use:     "version",
		Short:   "Print version info",
		Example: "  autobrr version\n  autobrr version --output=json",
	}

	var output string

	command.Flags().StringVar(&output, "output", "text", "output as text or json. Default: text")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		switch output {
		case "json":
			application.PrintVersionJSON()

		case "text", "":
			application.PrintVersion()

		default:
			return errors.New("unsupported output %q: must be text or json", output)
		}

		return nil
	}

	return command
}
