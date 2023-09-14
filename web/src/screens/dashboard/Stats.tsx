/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useQuery } from "@tanstack/react-query";
import { APIClient } from "@api/APIClient";
import { classNames } from "@utils";

interface StatsItemProps {
    name: string;
    value?: number;
    placeholder?: string;
}

const StatsItem = ({ name, placeholder, value }: StatsItemProps) => (
  <div
    className="relative px-4 py-3 overflow-hidden bg-white rounded-lg shadow-lg dark:bg-gray-800"
    title="All time"
  >
    <dt>
      <p className="pb-0.5 text-sm font-medium text-gray-500 truncate">{name}</p>
    </dt>

    <dd className="flex items-baseline">
      <p className="text-3xl font-extrabold text-gray-900 dark:text-gray-200">{placeholder}</p>
    </dd>

    <dd className="flex items-baseline">
      <p className="text-3xl font-extrabold text-gray-900 dark:text-gray-200">{value}</p>
    </dd>
  </div>
);

export const Stats = () => {
  const { isLoading, data } = useQuery({
    queryKey: ["dash_release_stats"],
    queryFn: APIClient.release.stats,
    refetchOnWindowFocus: false
  });

  return (
    <div>
      <h1 className="text-3xl font-bold text-black dark:text-white">
        Stats
      </h1>

      <dl className={classNames("grid grid-cols-1 gap-5 mt-5 sm:grid-cols-2 lg:grid-cols-3", isLoading ? "animate-pulse" : "")}>
        <StatsItem name="Filtered Releases" value={data?.filtered_count ?? 0} />
        {/* <StatsItem name="Filter Rejected Releases" stat={data?.filter_rejected_count} /> */}
        <StatsItem name="Rejected Pushes" value={data?.push_rejected_count ?? 0 } />
        <StatsItem name="Approved Pushes" value={data?.push_approved_count ?? 0} />
      </dl>
    </div>
  );
};
