/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Chart } from "@tanstack/react-charts";
import { defineChart, lineY, d3Curve } from "@tanstack/charts";
import { scaleUtc, scaleLinear } from "d3-scale";
import { curveMonotoneX } from "d3-shape";
import { format } from "date-fns";

import { ReleasesActivityQueryOptions } from "@api/queries";
import { ChartCard, ChartLegend, ChartSkeleton, RangeSelect, chartTheme, seriesColors, useIsDark } from "./charts";

interface ActivityDatum {
  date: Date;
  series: string;
  count: number;
}

const RANGES = [
  { value: "7d", days: 7 },
  { value: "30d", days: 30 },
  { value: "90d", days: 90 },
  { value: "1y", days: 365 },
  { value: "all", days: 0 }
] as const;

type RangeValue = (typeof RANGES)[number]["value"];

export const ActivityChart = () => {
  const { t } = useTranslation("common");
  const isDark = useIsDark();
  const [range, setRange] = useState<RangeValue>("30d");
  const rangeDays = RANGES.find((r) => r.value === range)?.days ?? 30;
  const { isLoading, data } = useQuery(ReleasesActivityQueryOptions(rangeDays));

  const colors = seriesColors(isDark);
  const series = useMemo(() => [
    { key: "push_approved_count" as const, label: t("dashboardCharts.seriesApproved"), color: colors.approved },
    { key: "matched_count" as const, label: t("dashboardCharts.seriesMatches"), color: colors.matches },
    { key: "push_rejected_count" as const, label: t("dashboardCharts.seriesRejected"), color: colors.rejected },
    { key: "push_error_count" as const, label: t("dashboardCharts.seriesErrored"), color: colors.errored }
  ], [t, colors.approved, colors.matches, colors.rejected, colors.errored]);

  const definition = useMemo(() => {
    const points: ActivityDatum[] = (data?.daily ?? []).flatMap((day) =>
      series.map((s) => ({
        date: new Date(`${day.date}T00:00:00Z`),
        series: s.label,
        count: day[s.key]
      }))
    );

    return defineChart({
      marks: [
        lineY(points, {
          x: "date",
          y: "count",
          z: "series",
          color: "series",
          strokeWidth: 2,
          curve: d3Curve(curveMonotoneX)
        })
      ],
      x: {
        scale: scaleUtc,
        ticks: 6,
        grid: false,
        format: (value: Date) => format(value, rangeDays === 0 || rangeDays > 180 ? "MMM yyyy" : "MMM d")
      },
      y: {
        scale: scaleLinear,
        nice: true,
        ticks: 4,
        grid: true
      },
      color: {
        domain: series.map((s) => s.label),
        range: series.map((s) => s.color)
      },
      theme: chartTheme(isDark),
      focus: "group-x",
      tooltip: {
        content: (points) => {
          const first = points[0];
          if (!first) {
            return { rows: [] };
          }
          const order = new Map(series.map((s, i) => [s.label, i]));
          const colorByLabel = new Map(series.map((s) => [s.label, s.color]));
          return {
            title: format(first.datum.date, "EEE MMM d"),
            rows: [...points]
              .sort((a, b) => (order.get(a.datum.series) ?? 0) - (order.get(b.datum.series) ?? 0))
              .map((point) => ({
                label: point.datum.series,
                value: point.datum.count.toLocaleString(),
                color: colorByLabel.get(point.datum.series)
              }))
          };
        }
      }
    });
  }, [data, series, isDark, rangeDays]);

  return (
    <ChartCard
      title={t("dashboardCharts.activityTitle")}
      className="mt-5"
      action={
        <RangeSelect
          options={RANGES.map((r) => ({ value: r.value, label: t(`dashboardCharts.range_${r.value}`) }))}
          value={range}
          onChange={setRange}
        />
      }
    >
      {isLoading ? (
        <ChartSkeleton heightClass="h-64" />
      ) : (
        <>
          <ChartLegend items={series.map((s) => ({ label: s.label, color: s.color }))} />
          <Chart
            definition={definition}
            ariaLabel={t("dashboardCharts.activityTitle")}
            height={240}
            className="mt-2 w-full"
          />
        </>
      )}
    </ChartCard>
  );
};
