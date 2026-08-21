/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { FilteredTile, ApprovedTile, RejectedTile, ErroredTile } from "./StatTiles";
import { ActivityChart } from "./ActivityChart";
import { VolumeChart } from "./VolumeChart";
import { HourHeatmap } from "./HourHeatmap";
import { TopIndexers, TopFilters } from "./TopLists";
import { ActivityTable } from "./ActivityTable";

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
