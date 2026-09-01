/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { ReleaseKeys } from "@api/query_keys";
import { HourHeatmap } from "@screens/dashboard/HourHeatmap";
import { VolumeChart } from "@screens/dashboard/VolumeChart";
import { humanFileSize } from "@utils";
import "@app/i18n";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const heatmapStats = () => {
  const heatmap = new Array<number>(7 * 24).fill(0);
  for (let day = 0; day < 7; day++) {
    heatmap[day * 24 + 2] = day + 1;
    heatmap[day * 24 + 13] = day + 8;
  }

  return { days: 30, heatmap };
};

const volumeStats = () => ({
  days: 30,
  daily: Array.from({ length: 30 }, (_, day) => ({
    date: new Date(Date.UTC(2026, 7, 1 + day)).toISOString().slice(0, 10),
    downloaded_bytes: day === 10 ? 8 * 1024 * 1024 : day === 11 ? 1024 * 1024 : 1024
  }))
});

const renderChart = (ui: ReactNode, queryKey: readonly unknown[], data: unknown) => {
  const queryClient = new QueryClient();
  queryClient.setQueryData(queryKey, data);

  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>).container;
};

const chartSvg = (label: string) => screen.getByLabelText(label) as unknown as SVGSVGElement;

// jsdom does no layout, so give the svg a rect matching its viewBox; client
// coordinates then map 1:1 onto scene coordinates and the chart's own
// hit-testing decides what the pointer is over.
const hoverAt = (svg: SVGSVGElement, x: number, y: number) => {
  const [, , width, height] = (svg.getAttribute("viewBox") ?? "").split(" ").map(Number);
  svg.getBoundingClientRect = () =>
    ({ x: 0, y: 0, left: 0, top: 0, right: width, bottom: height, width, height, toJSON: () => ({}) }) as DOMRect;
  svg.dispatchEvent(new MouseEvent("pointermove", { clientX: x, clientY: y, bubbles: true }));
};

const hoverRect = (svg: SVGSVGElement, rect: Element, insetX = 0, insetY = 0) => {
  const x = Number(rect.getAttribute("x"));
  const y = Number(rect.getAttribute("y"));
  const width = Number(rect.getAttribute("width"));
  const height = Number(rect.getAttribute("height"));
  hoverAt(svg, insetX === 0 ? x + width / 2 : x + width - insetX, insetY === 0 ? y + height / 2 : y + height - insetY);
};

const heatmapCell = (svg: SVGSVGElement, hour: number, day: string) =>
  svg.querySelector(`g[data-ts-key="rect-1"] rect[data-ts-key*='"number:${hour}"'][data-ts-key*='"string:${day.length}:${day}"']`);

test("orders heatmap hours on the x axis", () => {
  renderChart(<HourHeatmap />, ReleaseKeys.statsHeatmap(30), heatmapStats());

  const svg = chartSvg("Activity by hour");
  let previousX = -1;
  for (let hour = 0; hour < 24; hour++) {
    const tick = svg.querySelector(`[data-ts-key="x-tick-rule:number:${hour}"]`);
    expect(tick, `tick for hour ${hour}`).not.toBeNull();
    const x = Number(tick?.getAttribute("x1"));
    expect(x, `hour ${hour} must appear after hour ${hour - 1}`).toBeGreaterThan(previousX);
    previousX = x;
  }
});

// The fixture puts releases at UTC Mon 02:00 (count 2) and Mon 13:00 (count 9).
test.each([
  { zone: "UTC", offsetMinutes: 0, hour: 2, day: "Mon", count: 2 },
  { zone: "UTC+2", offsetMinutes: -120, hour: 4, day: "Mon", count: 2 },
  { zone: "UTC-5", offsetMinutes: 300, hour: 21, day: "Sun", count: 2 },
  { zone: "UTC+12", offsetMinutes: -720, hour: 1, day: "Tue", count: 9 }
])("shifts heatmap cells from UTC to $zone", ({ offsetMinutes, hour, day, count }) => {
  vi.spyOn(Date.prototype, "getTimezoneOffset").mockReturnValue(offsetMinutes);

  const container = renderChart(<HourHeatmap />, ReleaseKeys.statsHeatmap(30), heatmapStats());

  const svg = chartSvg("Activity by hour");
  const cell = heatmapCell(svg, hour, day);
  expect(cell).not.toBeNull();
  if (offsetMinutes !== 0) {
    expect(heatmapCell(svg, 2, "Mon")).toBeNull();
  }

  hoverRect(svg, cell as Element);

  const title = container.querySelector(".ts-chart-tooltip__title");
  expect(title?.textContent).toBe(`${day} ${String(hour).padStart(2, "0")}:00`);
  const row = container.querySelector(".ts-chart-tooltip__row");
  expect(row?.textContent).toBe(`Releases${count}`);
});

test("selects the volume bar under the pointer", () => {
  const container = renderChart(<VolumeChart />, ReleaseKeys.statsVolume(30), volumeStats());

  const svg = chartSvg("Download volume");
  const bars = svg.querySelectorAll("g.ts-chart__bar rect");
  expect(bars).toHaveLength(30);

  hoverRect(svg, bars[10], 2, 2);

  const title = container.querySelector(".ts-chart-tooltip__title");
  expect(title?.textContent).toBe("Tue Aug 11");
  const row = container.querySelector(".ts-chart-tooltip__row");
  expect(row?.textContent).toBe(`Downloaded${humanFileSize(8 * 1024 * 1024)}`);
});
