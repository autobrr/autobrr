/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useQuery} from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { classNames } from "@utils";
import { LinkIcon } from "@heroicons/react/24/solid";
import { ReleasesStatsQueryOptions } from "@api/queries";

interface StatsItemProps {
  name: string;
  value?: number;
  placeholder?: string;
  to?: string;
  eventType?: "" | "PUSH_APPROVED" | "PUSH_REJECTED" | "PUSH_ERROR" | undefined;
}

const StatsItem = ({ name, placeholder, value, to, eventType }: StatsItemProps) => (
  <Link
    className="group relative px-4 py-3 cursor-pointer overflow-hidden rounded-lg shadow-lg bg-white dark:bg-gray-800 hover:scale-110 hover:shadow-xl transition-all duration-200 ease-in-out"
    to={to}
    search={{
      action_status: eventType
    }}
    params={{}}
  >
    <dt>
      <div className="flex items-center text-sm font-medium text-gray-500 group-hover:dark:text-gray-475 group-hover:text-gray-600 transition-colors duration-200 ease-in-out">
        <p className="pb-0.5 truncate">{name}</p>
        <LinkIcon className="h-3 w-3 ml-2" aria-hidden="true" />
      </div>
    </dt>

    <div className="flex items-baseline text-3xl font-extrabold text-gray-900 dark:text-gray-200">
      <dd>
        <p>{placeholder}</p>
      </dd>
      <dd>
        <p>{value}</p>
      </dd>
    </div>
  </Link>
);

export const Stats = () => {
  const { isLoading, data } = useQuery(ReleasesStatsQueryOptions());

  return (
    <div>
      <h1 className="text-3xl font-bold text-black dark:text-white">
        Stats
      </h1>

      <dl className={classNames("grid grid-cols-2 gap-2 sm:gap-5 mt-5 sm:grid-cols-2 lg:grid-cols-4", isLoading ? "animate-pulse" : "")}>
        <StatsItem name="Filtered Releases" to="/releases" value={data?.filtered_count ?? 0} />
        {/* <StatsItem name="Filter Rejected Releases" stat={data?.filter_rejected_count} /> */}
        <StatsItem name="Approved Pushes" to="/releases" eventType="PUSH_APPROVED" value={data?.push_approved_count ?? 0} />
        <StatsItem name="Rejected Pushes" to="/releases" eventType="PUSH_REJECTED" value={data?.push_rejected_count ?? 0 } />
        <StatsItem name="Errored Pushes" to="/releases" eventType="PUSH_ERROR"  value={data?.push_error_count ?? 0} />
      </dl>
    </div>
  );
};
