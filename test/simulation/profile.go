// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build simulation

package simulation

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// profileChunk is the length of each CPU profile request. The pprof endpoint
// needs its duration up front and drops the profile if the request is
// cancelled, so the run is recorded as back-to-back chunks and merged.
const profileChunk = 2 * time.Second

// CPUProfiler records autobrr's CPU profile over its pprof endpoint while the
// simulation runs, so the profile covers the announce, filter and action path
// rather than startup.
type CPUProfiler struct {
	url string
	dir string

	stop chan struct{}
	wg   sync.WaitGroup

	m      sync.Mutex
	chunks []string
	err    error
}

// StartCPUProfile begins recording app's CPU profile into chunk files in a temp dir.
func StartCPUProfile(app *App, dir string) *CPUProfiler {
	p := &CPUProfiler{
		url:  app.ProfilingURL + fmt.Sprintf("/debug/pprof/profile?seconds=%d", int(profileChunk.Seconds())),
		dir:  dir,
		stop: make(chan struct{}),
	}

	p.wg.Go(p.run)

	return p
}

func (p *CPUProfiler) run() {
	client := &http.Client{Timeout: profileChunk + 30*time.Second}

	for i := 0; ; i++ {
		select {
		case <-p.stop:
			return
		default:
		}

		path := filepath.Join(p.dir, fmt.Sprintf("cpu-%03d.pprof", i))

		if err := fetchProfile(client, p.url, path); err != nil {
			p.m.Lock()
			p.err = err
			p.m.Unlock()

			return
		}

		p.m.Lock()
		p.chunks = append(p.chunks, path)
		p.m.Unlock()
	}
}

func fetchProfile(client *http.Client, url, path string) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("could not fetch cpu profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cpu profile returned %s: %s", resp.Status, body)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("could not write cpu profile: %w", err)
	}

	return nil
}

// Stop waits for the chunk in flight to finish and merges every chunk into out,
// in the proto format `go build -pgo` takes.
func (p *CPUProfiler) Stop(out string) error {
	close(p.stop)
	p.wg.Wait()

	p.m.Lock()
	defer p.m.Unlock()

	if p.err != nil {
		return p.err
	}

	if len(p.chunks) == 0 {
		return fmt.Errorf("no cpu profile was recorded")
	}

	args := append([]string{"tool", "pprof", "-proto"}, p.chunks...)

	cmd := exec.Command("go", args...)

	merged, err := cmd.Output()
	if err != nil {
		stderr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}

		return fmt.Errorf("could not merge cpu profiles: %w\n%s", err, stderr)
	}

	return os.WriteFile(out, merged, 0o644)
}
