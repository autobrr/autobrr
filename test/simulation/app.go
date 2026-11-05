// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build simulation

package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	username = "simulation"
	password = "simulation-password"
)

// App is a running autobrr binary with its own config directory and SQLite database.
type App struct {
	BaseURL string

	// ProfilingURL is the pprof server, empty unless StartApp was asked to enable it.
	ProfilingURL string

	t       testing.TB
	cmd     *exec.Cmd
	logFile *os.File
	logPath string
	client  *http.Client

	exited   chan struct{}
	exitErr  error
	stopping atomic.Bool
}

// BuildAutobrr compiles ./cmd/autobrr into dir. The binary is built rather than
// started with `go run` so the harness owns the process it signals on shutdown.
func BuildAutobrr(t testing.TB, dir string) string {
	t.Helper()

	root := repoRoot(t)

	out := filepath.Join(dir, "autobrr")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", out, "./cmd/autobrr")
	cmd.Dir = root

	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("simulation: could not build autobrr: %v\n%s", err, combined)
	}

	return out
}

// AppOptions tunes the autobrr instance StartApp boots.
type AppOptions struct {
	// LogLevel defaults to DEBUG. Profile at INFO, the default for real installs,
	// or PGO learns to optimise log formatting nobody runs.
	LogLevel string

	// Profiling serves pprof on a second free port.
	Profiling bool
}

// StartApp boots bin on a free port with a fresh config directory, onboards the
// simulation user and logs in. Process output goes to autobrr.log in outDir.
func StartApp(t testing.TB, bin, definitionsDir, outDir string, opts AppOptions) *App {
	t.Helper()

	if opts.LogLevel == "" {
		opts.LogLevel = "DEBUG"
	}

	port := freePort(t)

	confDir := t.TempDir()

	config := fmt.Sprintf(`host = "127.0.0.1"
port = %d
logLevel = %q
checkForUpdates = false
customDefinitions = %q
sessionSecret = "simulation-session-secret"
`, port, opts.LogLevel, definitionsDir)

	var profilingURL string
	if opts.Profiling {
		profilingPort := freePort(t)
		profilingURL = fmt.Sprintf("http://127.0.0.1:%d", profilingPort)

		config += fmt.Sprintf("profilingEnabled = true\nprofilingHost = \"127.0.0.1\"\nprofilingPort = %d\n", profilingPort)
	}

	if err := os.WriteFile(filepath.Join(confDir, "config.toml"), []byte(config), 0o644); err != nil {
		t.Fatalf("simulation: could not write config.toml: %v", err)
	}

	logPath := filepath.Join(outDir, "autobrr.log")

	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("simulation: could not create autobrr log: %v", err)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("simulation: could not create cookie jar: %v", err)
	}

	app := &App{
		BaseURL:      fmt.Sprintf("http://127.0.0.1:%d", port),
		ProfilingURL: profilingURL,
		t:            t,
		logFile:      logFile,
		logPath:      logPath,
		client:       &http.Client{Jar: jar, Timeout: 30 * time.Second},
	}

	app.cmd = exec.Command(bin, "--config", confDir)
	app.cmd.Stdout = logFile
	app.cmd.Stderr = logFile

	if err := app.cmd.Start(); err != nil {
		t.Fatalf("simulation: could not start autobrr: %v", err)
	}

	app.exited = make(chan struct{})
	go func() {
		app.exitErr = app.cmd.Wait()
		close(app.exited)
	}()

	t.Cleanup(app.stop)

	if err := waitForHTTP(app.BaseURL+"/api/healthz/liveness", 30*time.Second); err != nil {
		t.Fatalf("simulation: autobrr did not become ready: %v\n%s", err, app.LogTail(50))
	}

	app.Post("/api/auth/onboard", map[string]string{"username": username, "password": password}, nil)
	app.Post("/api/auth/login", map[string]string{"username": username, "password": password}, nil)

	return app
}

// LogPath is where the instance's stdout and stderr are written.
func (a *App) LogPath() string {
	return a.logPath
}

// LogTail returns the last n lines of the instance log.
func (a *App) LogTail(n int) string {
	b, err := os.ReadFile(a.logPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}

	return strings.Join(lines, "\n")
}

// Get decodes the JSON response of GET path into out.
func (a *App) Get(path string, out any) {
	a.t.Helper()

	a.do(http.MethodGet, path, nil, out)
}

// Post sends in as JSON and decodes the response into out when out is non-nil.
func (a *App) Post(path string, in, out any) {
	a.t.Helper()

	a.do(http.MethodPost, path, in, out)
}

// Put sends in as JSON and decodes the response into out when out is non-nil.
func (a *App) Put(path string, in, out any) {
	a.t.Helper()

	a.do(http.MethodPut, path, in, out)
}

// Patch sends in as JSON and decodes the response into out when out is non-nil.
func (a *App) Patch(path string, in, out any) {
	a.t.Helper()

	a.do(http.MethodPatch, path, in, out)
}

func (a *App) do(method, path string, in, out any) {
	a.t.Helper()

	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			a.t.Fatalf("simulation: could not marshal %s %s body: %v", method, path, err)
		}

		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, a.BaseURL+path, body)
	if err != nil {
		a.t.Fatalf("simulation: could not build %s %s: %v", method, path, err)
	}

	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := a.client.Do(req)
	if err != nil {
		a.t.Fatalf("simulation: %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		a.t.Fatalf("simulation: %s %s returned %d: %s", method, path, resp.StatusCode, respBody)
	}

	if out == nil || len(respBody) == 0 {
		return
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		a.t.Fatalf("simulation: could not decode %s %s response: %v\n%s", method, path, err, respBody)
	}
}

// Wait blocks until autobrr exits on its own, which with -keep means until the
// developer interrupts the test.
func (a *App) Wait() {
	<-a.exited
}

// UnexpectedExit describes how autobrr exited if it did so before the harness
// stopped it, and is empty while it is still running.
func (a *App) UnexpectedExit() string {
	select {
	case <-a.exited:
	default:
		return ""
	}

	if a.stopping.Load() {
		return ""
	}

	if a.exitErr != nil {
		return a.exitErr.Error()
	}

	return "exit status 0"
}

func (a *App) stop() {
	defer a.logFile.Close()

	a.stopping.Store(true)

	select {
	case <-a.exited:
	default:
		// a plain Kill would leave the SQLite WAL behind, autobrr shuts down cleanly on interrupt.
		// Windows can not deliver one, and the config dir is thrown away anyway
		timeout := 10 * time.Second
		if err := a.cmd.Process.Signal(os.Interrupt); err != nil {
			timeout = 0
		}

		select {
		case <-a.exited:
		case <-time.After(timeout):
			_ = a.cmd.Process.Kill()
			<-a.exited
		}
	}

	if state := a.cmd.ProcessState; state != nil {
		a.t.Logf("autobrr used %s user + %s system cpu", state.UserTime().Round(time.Millisecond), state.SystemTime().Round(time.Millisecond))
	}
}

func repoRoot(t testing.TB) string {
	t.Helper()

	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		t.Fatalf("simulation: could not locate repo root: %v", err)
	}

	return strings.TrimSpace(string(out))
}

func freePort(t testing.TB) int {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("simulation: could not find a free port: %v", err)
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port
}

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
