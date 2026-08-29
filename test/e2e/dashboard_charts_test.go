// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build e2e

package e2e_test

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/autobrr/autobrr/test/e2e/harness"
	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
)

func TestDashboardChartInteractions(t *testing.T) {
	app := harness.Start(t)
	page := app.NewAuthedPage()

	heatmap := make([]int, 7*24)
	for day := 0; day < 7; day++ {
		heatmap[day*24+2] = day + 1
		heatmap[day*24+13] = day + 8
	}
	volume := make([]map[string]any, 30)
	start := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	for day := range volume {
		bytes := 1024
		if day == 10 {
			bytes = 8 * 1024 * 1024
		} else if day == 11 {
			bytes = 1024 * 1024
		}
		volume[day] = map[string]any{
			"date":             start.AddDate(0, 0, day).Format(time.DateOnly),
			"downloaded_bytes": bytes,
		}
	}

	mockJSON(t, page.Page, "**/api/release/stats/heatmap?days=30", map[string]any{
		"days":    30,
		"heatmap": heatmap,
	})
	mockJSON(t, page.Page, "**/api/release/stats/volume?days=30", map[string]any{
		"days":  30,
		"daily": volume,
	})

	page.Open("/")
	page.ExpectText("Activity by hour")
	page.ExpectText("Download volume")

	t.Run("orders heatmap hours", func(t *testing.T) {
		heatmapChart := page.GetByLabel("Activity by hour")
		previousX := -1.0
		for hour := 0; hour < 24; hour++ {
			tick := heatmapChart.Locator(fmt.Sprintf(`[data-ts-key="x-tick-rule:number:%d"]`, hour))
			box, err := tick.BoundingBox()
			require.NoError(t, err)
			require.NotNil(t, box)
			require.Greater(t, box.X, previousX, "hour %d must appear after hour %d", hour, hour-1)
			previousX = box.X
		}

		offsetValue, err := page.Evaluate("() => Math.round(-new Date().getTimezoneOffset() / 60) || 0")
		require.NoError(t, err)
		offset, ok := offsetValue.(int)
		require.True(t, ok)
		shiftedHour := 2 + offset
		localHour := ((shiftedHour % 24) + 24) % 24
		localDay := (1 + int(math.Floor(float64(shiftedHour)/24)) + 7) % 7
		dayLabels := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

		cell := heatmapChart.Locator(fmt.Sprintf(
			`g[data-ts-key="rect-1"] rect[data-ts-key*='"number:%d"'][data-ts-key*='"string:%s"']`,
			localHour,
			dayLabels[localDay],
		))
		require.NoError(t, cell.Hover())

		expectedTitle := fmt.Sprintf("%s %02d:00", dayLabels[localDay], localHour)
		tooltipTitle := page.Locator(fmt.Sprintf(".ts-chart-tooltip__title:has-text(%q)", expectedTitle))
		require.NoError(t, tooltipTitle.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(2000)}))
		tooltip := tooltipTitle.Locator("xpath=..")
		tooltipRow, err := tooltip.Locator(".ts-chart-tooltip__row").TextContent()
		require.NoError(t, err)
		require.Equal(t, "Releases2", tooltipRow)
	})

	t.Run("selects the volume bar under the pointer", func(t *testing.T) {
		volumeChart := page.GetByLabel("Download volume")
		bar := volumeChart.Locator("g.ts-chart__bar rect").Nth(10)
		require.NoError(t, bar.ScrollIntoViewIfNeeded())
		box, err := bar.BoundingBox()
		require.NoError(t, err)
		require.NotNil(t, box)
		require.NoError(t, page.Mouse().Move(box.X+box.Width-2, box.Y+box.Height-2))

		tooltipTitle := page.Locator(`.ts-chart-tooltip__title:has-text("Tue Aug 11")`)
		require.NoError(t, tooltipTitle.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(2000)}))
		tooltip := tooltipTitle.Locator("xpath=..")
		title, err := tooltipTitle.TextContent()
		require.NoError(t, err)
		require.Equal(t, "Tue Aug 11", title)
		tooltipRow, err := tooltip.Locator(".ts-chart-tooltip__row").TextContent()
		require.NoError(t, err)
		require.Equal(t, "Downloaded8.4 MB", tooltipRow)
	})
}

func mockJSON(t *testing.T, page playwright.Page, pattern string, value any) {
	t.Helper()

	body, err := json.Marshal(value)
	require.NoError(t, err)
	require.NoError(t, page.Route(pattern, func(route playwright.Route) {
		require.NoError(t, route.Fulfill(playwright.RouteFulfillOptions{
			Body:        string(body),
			ContentType: playwright.String("application/json"),
		}))
	}))
}
