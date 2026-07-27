// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build e2e

package e2e_test

import (
	"log"
	"os"
	"testing"

	"github.com/autobrr/autobrr/test/e2e/harness"
)

// TestMain compiles the binaries under test and launches the browser once for
// the whole package. See test/e2e/README.md for how to run these.
func TestMain(m *testing.M) {
	code, err := harness.Run(m)
	if err != nil {
		log.Fatalf("e2e: %v", err)
	}

	os.Exit(code)
}
