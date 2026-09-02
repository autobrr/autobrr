// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build e2e

package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// Credentials the harness onboards every instance with. Tests that drive the
// login form need to know them, so they are exported rather than hidden.
const (
	Username = "e2e"
	Password = "e2e-password"
)

// App is a running autobrr process with its own config directory and database.
type App struct {
	BaseURL string

	t       testing.TB
	cmd     *exec.Cmd
	logs    *syncBuffer
	confDir string
	client  *http.Client
}

// Start compiles nothing and boots a fresh autobrr: a free port, an empty
// config directory and therefore an empty database. It blocks until the
// instance answers its liveness probe and stops it when the test ends.
func Start(t testing.TB) *App {
	t.Helper()

	port, err := freePort()
	if err != nil {
		t.Fatalf("harness: could not find a free port: %v", err)
	}

	confDir := t.TempDir()
	writeConfig(t, confDir, port)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("harness: could not create cookie jar: %v", err)
	}

	app := &App{
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		t:       t,
		logs:    &syncBuffer{},
		confDir: confDir,
		client:  &http.Client{Jar: jar, Timeout: 10 * time.Second},
	}

	app.cmd = exec.Command(autobrrBin, "--config", confDir)
	app.cmd.Stdout = app.logs
	app.cmd.Stderr = app.logs

	if err := app.cmd.Start(); err != nil {
		t.Fatalf("harness: could not start autobrr: %v", err)
	}

	t.Cleanup(app.stop)

	if err := waitForHTTP(app.BaseURL+"/api/healthz/liveness", 30*time.Second); err != nil {
		t.Fatalf("harness: autobrr did not become ready: %v\n%s", err, app.logs.String())
	}

	return app
}

// Logs returns everything the instance has written to stdout and stderr so far.
// Tests dump it when they fail, which is usually the fastest way to tell a
// broken assertion apart from a broken backend.
func (a *App) Logs() string {
	return a.logs.String()
}

func (a *App) stop() {
	if a.cmd.Process == nil {
		return
	}

	// A plain Kill would leave the SQLite WAL behind. autobrr shuts down
	// cleanly on SIGTERM, so give it that and only escalate if it hangs.
	_ = a.cmd.Process.Signal(os.Interrupt)

	done := make(chan struct{})
	go func() {
		_ = a.cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		_ = a.cmd.Process.Kill()
		<-done
	}

	if a.t.Failed() {
		a.t.Logf("autobrr logs:\n%s", a.logs.String())
	}
}

// Onboard creates the initial user over the API. Driving the onboarding form is
// the subject of its own test; every other test just needs an account to exist.
func (a *App) Onboard() {
	a.t.Helper()

	body, _ := json.Marshal(map[string]string{"username": Username, "password": Password})

	resp, err := a.client.Post(a.BaseURL+"/api/auth/onboard", "application/json", bytes.NewReader(body))
	if err != nil {
		a.t.Fatalf("harness: could not onboard: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		a.t.Fatalf("harness: onboard returned %d", resp.StatusCode)
	}
}

// login authenticates over the API and returns the session cookie, which
// NewAuthedPage hands to the browser so tests can start on the page they
// actually care about.
func (a *App) login() *http.Cookie {
	a.t.Helper()

	body, _ := json.Marshal(map[string]any{"username": Username, "password": Password})

	resp, err := a.client.Post(a.BaseURL+"/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		a.t.Fatalf("harness: could not log in: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		a.t.Fatalf("harness: login returned %d", resp.StatusCode)
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Value != "" {
			return cookie
		}
	}

	a.t.Fatal("harness: login did not set a session cookie")

	return nil
}

// writeConfig writes a complete config.toml rather than letting autobrr
// generate one, because the generated file hardcodes port 7474 and would
// collide with a developer's own instance.
func writeConfig(t testing.TB, dir string, port int) {
	t.Helper()

	config := fmt.Sprintf(`host = "127.0.0.1"
port = %d
logLevel = "DEBUG"
checkForUpdates = false
customDefinitions = %q
sessionSecret = "e2e-session-secret"
`, port, definitionsDir())

	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(config), 0o644); err != nil {
		t.Fatalf("harness: could not write config.toml: %v", err)
	}
}

// freePort asks the kernel for an unused port. There is an unavoidable gap
// between closing the listener and the process binding it, but nothing else on
// the machine is competing for ephemeral ports during a test run.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

// waitForHTTP polls url until it answers 200 or timeout elapses.
func waitForHTTP(url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Second}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error

	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				return nil
			}

			lastErr = fmt.Errorf("got status %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for %s: %w", url, lastErr)
		case <-ticker.C:
		}
	}
}

// syncBuffer collects process output. The process writes from its own goroutine
// while the test reads on failure, so the writes need guarding.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}
