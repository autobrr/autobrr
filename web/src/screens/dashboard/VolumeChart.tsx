/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Chart } from "@tanstack/react-charts";
import { defineChart, barY } from "@tanstack/charts";
import { scaleBand, scaleLinear } from "d3-scale";
import { format } from "date-fns";

import { ReleasesVolumeQueryOptions } from "@api/queries";
import { humanFileSize } from "@utils";
import { ChartCard, ChartError, ChartSkeleton, chartTheme, seriesColors, useIsDark } from "./charts";

const asDate = (value: string) => new Date(`${value}T00:00:00Z`);

export const VolumeChart = () => {
  const { t } = useTranslation("common");
  const isDark = useIsDark();
  const { isLoading, isError, data } = useQuery(ReleasesVolumeQueryOptions());

  const definition = useMemo(() => {
    const days = data?.daily ?? [];
    const downloadedLabel = t("dashboardCharts.downloaded");

    return defineChart({
      marks: [
        barY(days, {
          x: "date",
          y: "downloaded_bytes",
          fill: seriesColors(isDark).matches,
          radius: 4,
          inset: 1
        })
      ],
      x: {
        scale: scaleBand,
        grid: false,
        format: (value: string) => (asDate(value).getUTCDay() === 1 ? format(asDate(value), "MMM d") : "")
      },
      y: {
        scale: scaleLinear,
        nice: true,
        ticks: 4,
        grid: true,
        format: (value: number) => humanFileSize(value)
      },
      theme: chartTheme(isDark),
      tooltip: {
        content: (points) => {
          const point = points[0];
          if (!point) {
            return { rows: [] };
          }
          const datum = point.datum as ReleaseVolumeDaily;
          return {
            title: format(asDate(datum.date), "EEE MMM d"),
            rows: [{ label: downloadedLabel, value: humanFileSize(datum.downloaded_bytes) }]
          };
        }
      }
    });
  }, [data, isDark, t]);

  return (
    <ChartCard title={t("dashboardCharts.volumeTitle")}>
      {isLoading ? (
        <ChartSkeleton heightClass="h-56" />
      ) : isError ? (
        <ChartError heightClass="h-56" />
      ) : (
        <Chart
          definition={definition}
          ariaLabel={t("dashboardCharts.volumeTitle")}
          height={224}
          className="mt-2 w-full"
        />
      )}
    </ChartCard>
  );
};
