// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build e2e

package e2e_test

import (
	"testing"

	"github.com/autobrr/autobrr/test/e2e/harness"
)

// Adding an indexer is the setup step every other autobrr workflow depends on,
// and it exercises the slide-over, the react-select implementation picker and
// the fields that picker reveals.
func TestAddIndexer(t *testing.T) {
	app := harness.Start(t)
	page := app.NewAuthedPage()

	page.Open("/settings/indexers")
	page.AddNew()

	page.Select("Generic RSS")
	page.Fill("feed.url", "https://distrowatch.com/news/torrents.xml")
	page.Submit()

	page.ExpectRow("Generic RSS")
}

// Notifications have their own event toggles, which are the only place the
// Checkbox component appears inside a slide-over form.
func TestAddNotification(t *testing.T) {
	app := harness.Start(t)
	page := app.NewAuthedPage()

	page.Open("/settings/notifications")
	page.AddNew()

	page.Select("Discord")
	page.Fill("name", "Discord E2E")
	page.Fill("webhook", "https://discord.com/api/webhooks/e2e")
	page.Toggle("events-PUSH_APPROVED")
	page.Submit()

	page.ExpectRow("Discord E2E")
}

func TestAddAPIKey(t *testing.T) {
	app := harness.Start(t)
	page := app.NewAuthedPage()

	page.Open("/settings/api")
	page.AddNew()

	page.Fill("name", "e2e-key")
	page.Submit()

	page.ExpectRow("e2e-key")
}

// The download client form validates the host before it will save, so this also
// covers the form actually reaching the backend.
func TestAddDownloadClient(t *testing.T) {
	app := harness.Start(t)
	page := app.NewAuthedPage()

	page.Open("/settings/clients")
	page.AddNew()

	page.Fill("name", "qbit-e2e")
	page.Fill("host", "http://localhost:8080")
	page.Submit()

	page.ExpectRow("qbit-e2e")
}

// The log level select saves on change rather than on submit, so the toast is
// the only signal that the write reached the config file.
func TestChangeLogLevel(t *testing.T) {
	app := harness.Start(t)
	page := app.NewAuthedPage()

	page.Open("/settings/logs")
	page.Select("DEBUG")

	page.ExpectText("Config successfully updated!")
}

// A filter with no criteria still has to save and show up in the list.
func TestCreateFilter(t *testing.T) {
	app := harness.Start(t)
	page := app.NewAuthedPage()

	page.Open("/filters")
	page.ClickButton("Add new")

	page.Fill("name", "E2E Filter")
	page.Submit()

	page.Open("/filters")
	page.ExpectRow("E2E Filter")
}
