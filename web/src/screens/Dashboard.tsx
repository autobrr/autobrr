/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Stats } from "./dashboard/Stats";
import { ActivityChart } from "./dashboard/ActivityChart";
import { VolumeChart } from "./dashboard/VolumeChart";
import { HourHeatmap } from "./dashboard/HourHeatmap";
import { TopIndexers, TopFilters } from "./dashboard/TopLists";
import { ActivityTable } from "./dashboard/ActivityTable";

export const Dashboard = () => (
  <main>
    <div className="my-6 max-w-(--breakpoint-xl) mx-auto pb-6 px-2 sm:px-6 lg:pb-16 lg:px-8">
      <Stats />
      <ActivityChart />
      <div className="grid grid-cols-1 gap-2 sm:gap-5 mt-5 lg:grid-cols-2">
        <VolumeChart />
        <HourHeatmap />
      </div>
      <div className="grid grid-cols-1 gap-2 sm:gap-5 mt-5 lg:grid-cols-2">
        <TopIndexers />
        <TopFilters />
      </div>
      <ActivityTable />
    </div>
  </main>
);
