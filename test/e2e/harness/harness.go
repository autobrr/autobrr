// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build e2e

// Package harness boots everything an end-to-end run needs and tears it down
// again: the autobrr binary under test, the mock indexer that stands in for a
// tracker, and a Playwright browser.
//
// The expensive work happens once per package in Run: compiling the binaries
// and launching the browser. Everything a single test touches is cheap and
// per-test, so tests do not share state. Each one gets its own autobrr process,
// config directory and SQLite database via Start, and its own browser context
// via App.NewPage.
package harness

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mxschmitt/playwright-go"
)

// playwrightVersion must track the playwright-go version in go.mod. It only
// appears in the "how do I install the driver" hint on startup failure.
const playwrightVersion = "v0.6201.1"

// Binaries compiled once by Run and reused by every test in the package.
var (
	autobrrBin      string
	mockIndexerBin  string
	repoRoot        string
	browser         playwright.Browser
	playwrightSetup *playwright.Playwright
)

// Run performs the package-level setup, runs the tests and cleans up. Call it
// from TestMain. It returns the exit code m.Run produced so TestMain can hand
// it straight to os.Exit.
func Run(m *testing.M) (int, error) {
	root, err := findRepoRoot()
	if err != nil {
		return 1, err
	}
	repoRoot = root

	buildDir, err := os.MkdirTemp("", "autobrr-e2e-build")
	if err != nil {
		return 1, fmt.Errorf("could not create build dir: %w", err)
	}
	defer os.RemoveAll(buildDir)

	autobrrBin, err = goBuild(buildDir, "autobrr", "./cmd/autobrr")
	if err != nil {
		return 1, err
	}

	mockIndexerBin, err = goBuild(buildDir, "mockindexer", "./test/mockindexer")
	if err != nil {
		return 1, err
	}

	playwrightSetup, err = playwright.Run(&playwright.RunOptions{Browsers: []string{"chromium"}})
	if err != nil {
		return 1, fmt.Errorf("could not start playwright, run `go run github.com/mxschmitt/playwright-go/cmd/playwright@%s install --with-deps chromium` first: %w", playwrightVersion, err)
	}
	defer playwrightSetup.Stop()

	browser, err = playwrightSetup.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: new(!headed()),
	})
	if err != nil {
		return 1, fmt.Errorf("could not launch chromium: %w", err)
	}
	defer browser.Close()

	return m.Run(), nil
}

// headed reports whether the browser should be visible, which is useful when
// debugging a failing test locally: HEADED=1 go test -tags=e2e ./test/e2e/...
func headed() bool {
	v := os.Getenv("HEADED")
	return v != "" && v != "0" && v != "false"
}

// goBuild compiles pkg into dir and returns the path to the resulting binary.
// The binaries are built rather than `go run`, so the test owns the process it
// starts and can reliably signal it. `go run` would leave the real process as a
// grandchild that outlives the one we hold.
func goBuild(dir, name, pkg string) (string, error) {
	out := filepath.Join(dir, name)
	if runtime.GOOS == "windows" {
		out += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", out, pkg)
	cmd.Dir = repoRoot

	if combined, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("could not build %s: %w\n%s", pkg, err, combined)
	}

	return out, nil
}

// findRepoRoot locates the module root so the harness can build packages and
// read fixtures by repo-relative path regardless of the test's working
// directory.
func findRepoRoot() (string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not locate repo root: %w", err)
	}

	root := strings.TrimSpace(string(out))
	if root == "" {
		return "", fmt.Errorf("could not locate repo root: go list returned nothing")
	}

	return root, nil
}

// definitionsDir is the custom indexer definition directory holding mock.yaml,
// which teaches autobrr how to parse the mock indexer's announces.
func definitionsDir() string {
	return filepath.Join(repoRoot, "test", "definitions")
}
