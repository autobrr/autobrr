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
  spanClass: string;
  Component: React.ComponentType;
}

// Registry order is the default layout. Adding a widget here is enough:
// stored configs are reconciled against the registry, unknown ids dropped
// and new ones appended.
export const DASHBOARD_WIDGETS: DashboardWidgetDef[] = [
  { id: "stat-filtered", spanClass: "sm:col-span-1", Component: FilteredTile },
  { id: "stat-approved", spanClass: "sm:col-span-1", Component: ApprovedTile },
  { id: "stat-rejected", spanClass: "sm:col-span-1", Component: RejectedTile },
  { id: "stat-errored", spanClass: "sm:col-span-1", Component: ErroredTile },
  { id: "activity", spanClass: "sm:col-span-2 lg:col-span-4", Component: ActivityChart },
  { id: "volume", spanClass: "sm:col-span-2 lg:col-span-2", Component: VolumeChart },
  { id: "heatmap", spanClass: "sm:col-span-2 lg:col-span-2", Component: HourHeatmap },
  { id: "top-indexers", spanClass: "sm:col-span-2 lg:col-span-2", Component: TopIndexers },
  { id: "top-filters", spanClass: "sm:col-span-2 lg:col-span-2", Component: TopFilters },
  { id: "recent-activity", spanClass: "sm:col-span-2 lg:col-span-4", Component: ActivityTable }
];
