// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build simulation

package simulation

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var (
	flagReleases   = flag.Int("releases", 80, "number of releases to deliver")
	flagSeed       = flag.Uint64("seed", 0, "release generator seed, 0 picks one from the clock")
	flagInterval   = flag.Duration("interval", 25*time.Millisecond, "pause between delivered releases")
	flagFeedBatch  = flag.Int("feed-batch", 5, "feed items published per forced feed run")
	flagTimeout    = flag.Duration("wait", 2*time.Minute, "how long to wait for expected outcomes")
	flagSettle     = flag.Duration("settle", 3*time.Second, "extra wait after outcomes converge, to catch duplicates")
	flagOut        = flag.String("out", "", "directory for autobrr.log and report files, defaults to a temp dir kept only when the run fails")
	flagAutobrrBin = flag.String("autobrr-bin", "", "prebuilt autobrr binary, defaults to building ./cmd/autobrr")
	flagKeep       = flag.Bool("keep", false, "keep autobrr and the mocks running after the run until interrupted")
	flagPGOProfile = flag.String("pgo-profile", "", "write autobrr's CPU profile from delivery to verification here, usable with go build -pgo")
	flagLogLevel   = flag.String("log-level", "DEBUG", "autobrr log level, use INFO when recording a PGO profile")
)

func TestSimulation(t *testing.T) {
	start := time.Now()

	seed := *flagSeed
	if seed == 0 {
		seed = uint64(start.UnixNano())
	}

	out := *flagOut
	if out == "" {
		// not t.TempDir, which would delete the log and report of a failed run
		dir, err := os.MkdirTemp("", "autobrr-simulation-")
		if err != nil {
			t.Fatalf("could not create out dir: %v", err)
		}

		t.Cleanup(func() {
			if !t.Failed() {
				_ = os.RemoveAll(dir)
			}
		})

		out = dir
	} else if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatalf("could not create out dir: %v", err)
	}

	out, err := filepath.Abs(out)
	if err != nil {
		t.Fatalf("could not resolve out dir: %v", err)
	}

	bin := *flagAutobrrBin
	if bin == "" {
		bin = BuildAutobrr(t, t.TempDir())
	}

	scenario := NewScenario(t)

	app := StartApp(t, bin, scenario.DefinitionsDir(), out, AppOptions{
		LogLevel:  *flagLogLevel,
		Profiling: *flagPGOProfile != "",
	})
	t.Logf("autobrr running at %s, log at %s", app.BaseURL, app.LogPath())

	scenario.Configure(app)
	scenario.WaitReady(t, app, 30*time.Second)

	releases := GenerateReleases(seed, *flagReleases, scenario.Indexers)
	t.Logf("delivering %d releases with seed %d", len(releases), seed)

	var profiler *CPUProfiler
	if *flagPGOProfile != "" {
		profiler = StartCPUProfile(app, t.TempDir())
	}

	injectStart := time.Now()
	injected := scenario.Inject(app, releases, *flagInterval, *flagFeedBatch)
	injection := time.Since(injectStart)

	report := scenario.Verify(releases, injected, *flagTimeout, *flagSettle)

	if profiler != nil {
		if err := profiler.Stop(*flagPGOProfile); err != nil {
			t.Errorf("could not write cpu profile: %v", err)
		} else {
			t.Logf("cpu profile written to %s", *flagPGOProfile)
		}
	}

	report.AppExit = app.UnexpectedExit()
	report.Seed = seed
	report.Injection = injection
	report.Duration = time.Since(start)

	if err := report.Write(out); err != nil {
		t.Errorf("could not write report: %v", err)
	}

	t.Logf("\n%s", report.Markdown())
	t.Logf("report written to %s", out)

	if report.Failed() {
		t.Errorf("simulation failed, see the report above and %s", app.LogPath())
	}

	if *flagKeep {
		t.Logf("-keep: autobrr stays up at %s (user %q, password %q), interrupt to stop", app.BaseURL, username, password)
		app.Wait()
	}
}
