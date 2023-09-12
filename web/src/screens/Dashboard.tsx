/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Stats } from "./dashboard/Stats";
import { ActivityTable } from "./dashboard/ActivityTable";

export const Dashboard = () => (
  <div className="my-6 max-w-screen-xl mx-auto pb-6 px-4 sm:px-6 lg:pb-16 lg:px-8">
    <Stats />
    <ActivityTable />
  </div>
);
