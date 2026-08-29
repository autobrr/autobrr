/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Chart } from "@tanstack/react-charts";
import { defineChart, cell } from "@tanstack/charts";
import { scaleBand, scaleQuantize } from "d3-scale";

import { ReleasesHeatmapQueryOptions } from "@api/queries";
import { ChartCard, ChartError, ChartSkeleton, chartTheme, sequentialRamp, useIsDark } from "./charts";

interface HeatmapDatum {
  hour: number;
  day: string;
  count: number;
}

const squareCellChartAspectRatio = 3.15;

const DAY_LABELS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const DISPLAY_DOWS = [1, 2, 3, 4, 5, 6, 0];
const DISPLAY_DAYS = DISPLAY_DOWS.map((dow) => DAY_LABELS[dow]);
const HOURS = Array.from({ length: 24 }, (_, hour) => hour);

// The API grid is UTC-indexed (dow*24+hour); shift whole cells into the
// browser's timezone so the evening peak lands where the user expects it.
const toLocalCells = (heatmap: number[]): HeatmapDatum[] => {
  const offset = Math.round(-new Date().getTimezoneOffset() / 60);
  const local = new Array<number>(7 * 24).fill(0);

  for (let dow = 0; dow < 7; dow++) {
    for (let hour = 0; hour < 24; hour++) {
      const shifted = hour + offset;
      const localHour = ((shifted % 24) + 24) % 24;
      const localDow = (dow + Math.floor(shifted / 24) + 7) % 7;
      local[localDow * 24 + localHour] = heatmap[dow * 24 + hour] ?? 0;
    }
  }

  return DISPLAY_DOWS.flatMap((dow) =>
    Array.from({ length: 24 }, (_, hour) => ({
      hour,
      day: DAY_LABELS[dow],
      count: local[dow * 24 + hour]
    }))
  );
};

export const HourHeatmap = () => {
  const { t } = useTranslation("common");
  const isDark = useIsDark();
  const { isLoading, isError, data } = useQuery(ReleasesHeatmapQueryOptions());

  const ramp = sequentialRamp(isDark);
  const zeroFill = isDark ? "#303034" : "#f4f4f5";

  const definition = useMemo(() => {
    const cells = toLocalCells(data?.heatmap ?? []);
    const max = Math.max(...cells.map((c) => c.count), 1);
    const releasesLabel = t("dashboardCharts.releases");

    return defineChart({
      marks: [
        cell(cells.filter((c) => c.count === 0), {
          x: "hour",
          y: "day",
          fill: zeroFill,
          inset: 1,
          radius: 2
        }),
        cell(cells.filter((c) => c.count > 0), {
          x: "hour",
          y: "day",
          color: "count",
          inset: 1,
          radius: 2
        })
      ],
      x: {
        scale: scaleBand<number>().domain(HOURS),
        grid: false,
        format: (value: number) => (value % 3 === 0 ? String(value).padStart(2, "0") : "")
      },
      y: {
        scale: scaleBand<string>().domain(DISPLAY_DAYS),
        grid: false
      },
      color: {
        scale: scaleQuantize<string>().domain([1, max]).range(ramp)
      },
      theme: chartTheme(isDark),
      tooltip: {
        content: (points) => {
          const point = points[0];
          if (!point) {
            return { rows: [] };
          }
          const datum = point.datum as HeatmapDatum;
          return {
            title: `${datum.day} ${String(datum.hour).padStart(2, "0")}:00`,
            rows: [{ label: releasesLabel, value: String(datum.count) }]
          };
        }
      }
    });
  }, [data, isDark, ramp, zeroFill, t]);

  return (
    <ChartCard title={t("dashboardCharts.heatmapTitle")}>
      {isLoading ? (
        <ChartSkeleton heightClass="h-56" />
      ) : isError ? (
        <ChartError heightClass="h-56" />
      ) : (
        <>
          <Chart
            definition={definition}
            ariaLabel={t("dashboardCharts.heatmapTitle")}
            aspectRatio={squareCellChartAspectRatio}
            className="mt-2 w-full"
          />
          <div className="mt-auto flex items-center justify-end gap-1 pt-1 text-xs text-gray-500 dark:text-gray-400">
            {t("dashboardCharts.heatmapLess")}
            <span className="inline-block w-3 h-3 rounded-xs" style={{ backgroundColor: zeroFill }} aria-hidden="true" />
            {ramp.map((color) => (
              <span key={color} className="inline-block w-3 h-3 rounded-xs" style={{ backgroundColor: color }} aria-hidden="true" />
            ))}
            {t("dashboardCharts.heatmapMore")}
          </div>
        </>
      )}
    </ChartCard>
  );
};
