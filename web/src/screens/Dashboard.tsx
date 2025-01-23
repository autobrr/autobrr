/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Stats } from "./dashboard/Stats";
import { ActivityTable } from "./dashboard/ActivityTable";

export const Dashboard = () => (
  <main>
    <div className="my-6 max-w-(--breakpoint-xl) mx-auto pb-6 px-2 sm:px-6 lg:pb-16 lg:px-8">
      <Stats />
      <ActivityTable />
    </div>
  </main>
);
