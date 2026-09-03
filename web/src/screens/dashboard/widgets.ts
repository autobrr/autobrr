/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { lazy } from "react";

import type { DashboardConfigType } from "@utils/Context";

import { FilteredTile, ApprovedTile, RejectedTile, ErroredTile } from "./StatTiles";
import { TopIndexers, TopFilters } from "./TopLists";
import { ActivityTable } from "./ActivityTable";

// The chart widgets carry @tanstack/react-charts and d3, so they load on demand.
const ActivityChart = lazy(() => import("./ActivityChart").then((m) => ({ default: m.ActivityChart })));
const VolumeChart = lazy(() => import("./VolumeChart").then((m) => ({ default: m.VolumeChart })));
const HourHeatmap = lazy(() => import("./HourHeatmap").then((m) => ({ default: m.HourHeatmap })));

export interface DashboardWidgetDef {
  id: string;
  size: "compact" | "half" | "full";
  spanClass: string;
  titleKey: string;
  Component: React.ComponentType;
}

export const DASHBOARD_WIDGETS: DashboardWidgetDef[] = [
  {
    id: "stat-filtered",
    size: "compact",
    spanClass: "sm:col-span-1",
    titleKey: "dashboardStats.filteredReleases",
    Component: FilteredTile
  },
  {
    id: "stat-approved",
    size: "compact",
    spanClass: "sm:col-span-1",
    titleKey: "dashboardStats.approvedPushes",
    Component: ApprovedTile
  },
  {
    id: "stat-rejected",
    size: "compact",
    spanClass: "sm:col-span-1",
    titleKey: "dashboardStats.rejectedPushes",
    Component: RejectedTile
  },
  {
    id: "stat-errored",
    size: "compact",
    spanClass: "sm:col-span-1",
    titleKey: "dashboardStats.erroredPushes",
    Component: ErroredTile
  },
  {
    id: "activity",
    size: "full",
    spanClass: "sm:col-span-2 lg:col-span-4",
    titleKey: "dashboardCharts.activityTitle",
    Component: ActivityChart
  },
  {
    id: "volume",
    size: "half",
    spanClass: "sm:col-span-2 lg:col-span-2",
    titleKey: "dashboardCharts.volumeTitle",
    Component: VolumeChart
  },
  {
    id: "heatmap",
    size: "half",
    spanClass: "sm:col-span-2 lg:col-span-2",
    titleKey: "dashboardCharts.heatmapTitle",
    Component: HourHeatmap
  },
  {
    id: "top-indexers",
    size: "half",
    spanClass: "sm:col-span-2 lg:col-span-2",
    titleKey: "dashboardCharts.topIndexersTitle",
    Component: TopIndexers
  },
  {
    id: "top-filters",
    size: "half",
    spanClass: "sm:col-span-2 lg:col-span-2",
    titleKey: "dashboardCharts.topFiltersTitle",
    Component: TopFilters
  },
  {
    id: "recent-activity",
    size: "full",
    spanClass: "sm:col-span-2 lg:col-span-4",
    titleKey: "activityTable.title",
    Component: ActivityTable
  }
];

export interface WidgetLayout {
  def: DashboardWidgetDef;
  hidden: boolean;
}

export const resolveLayout = (config: DashboardConfigType): WidgetLayout[] => {
  const byId = new Map(DASHBOARD_WIDGETS.map((def) => [def.id, def]));
  const seen = new Set<string>();
  const layout: WidgetLayout[] = [];

  // Preserve the saved order while dropping removed widgets and appending new ones.
  for (const entry of config.widgets) {
    const def = byId.get(entry.id);
    if (def && !seen.has(entry.id)) {
      seen.add(entry.id);
      layout.push({ def, hidden: entry.hidden });
    }
  }
  for (const def of DASHBOARD_WIDGETS) {
    if (!seen.has(def.id)) {
      layout.push({ def, hidden: false });
    }
  }

  return layout;
};
