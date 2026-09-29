// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build e2e

package harness

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/mxschmitt/playwright-go"
)

// artifactsDir collects traces and screenshots from failed tests. CI uploads it.
const artifactsDir = "artifacts"

// NewPage opens a logged-out browser context. Use it for the onboarding and
// login flows; everything else wants NewAuthedPage.
func (a *App) NewPage() *UI {
	a.t.Helper()

	return a.newPage(nil)
}

// NewAuthedPage onboards the instance, authenticates over the API and hands the
// session cookie to a fresh browser context. Tests start already signed in, so
// a broken login form fails exactly one test instead of all of them.
func (a *App) NewAuthedPage() *UI {
	a.t.Helper()

	a.Onboard()

	return a.newPage(a.login())
}

func (a *App) newPage(session *http.Cookie) *UI {
	a.t.Helper()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		BaseURL: new(a.BaseURL),
		// The UI picks its language from navigator.language and ships eight of
		// them. Pin the locale so text-based assertions stay meaningful.
		Locale: new("en-US"),
	})
	if err != nil {
		a.t.Fatalf("harness: could not create browser context: %v", err)
	}

	if session != nil {
		u, _ := url.Parse(a.BaseURL)

		err = context.AddCookies([]playwright.OptionalCookie{{
			Name:     session.Name,
			Value:    session.Value,
			Domain:   new(u.Hostname()),
			Path:     new("/"),
			HttpOnly: new(true),
		}})
		if err != nil {
			a.t.Fatalf("harness: could not set session cookie: %v", err)
		}
	}

	// Trace unconditionally but only keep it when the test fails: a trace is
	// the difference between "a locator timed out" and being able to see what
	// the page actually looked like at that moment.
	err = context.Tracing().Start(playwright.TracingStartOptions{
		Screenshots: new(true),
		Snapshots:   new(true),
		Sources:     new(true),
	})
	if err != nil {
		a.t.Fatalf("harness: could not start tracing: %v", err)
	}

	page, err := context.NewPage()
	if err != nil {
		a.t.Fatalf("harness: could not create page: %v", err)
	}

	a.t.Cleanup(func() {
		if a.t.Failed() {
			a.saveArtifacts(page, context)
		}

		_ = context.Tracing().Stop()
		_ = context.Close()
	})

	return &UI{Page: page, t: a.t}
}

// saveArtifacts writes a screenshot and a Playwright trace for a failed test.
func (a *App) saveArtifacts(page playwright.Page, context playwright.BrowserContext) {
	base := filepath.Join(artifactsDir, sanitize(a.t.Name()))

	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		a.t.Logf("harness: could not create artifacts dir: %v", err)
		return
	}

	shot := base + ".png"
	if _, err := page.Screenshot(playwright.PageScreenshotOptions{
		Path:     new(shot),
		FullPage: new(true),
	}); err != nil {
		a.t.Logf("harness: could not save screenshot: %v", err)
	} else {
		a.t.Logf("screenshot: %s", shot)
	}

	trace := base + ".trace.zip"
	if err := context.Tracing().Stop(trace); err != nil {
		a.t.Logf("harness: could not save trace: %v", err)
	} else {
		a.t.Logf("trace: %s (view with `go run github.com/mxschmitt/playwright-go/cmd/playwright@%s show-trace %s`)", trace, playwrightVersion, trace)
	}
}

// sanitize turns a test name into something safe for a file name.
func sanitize(name string) string {
	return strings.NewReplacer("/", "-", " ", "_", `\`, "-").Replace(name)
}
