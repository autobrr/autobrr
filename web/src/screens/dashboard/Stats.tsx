/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useQuery } from "@tanstack/react-query";
import { APIClient } from "@api/APIClient";
import { classNames } from "@utils";
import { useNavigate } from "react-router-dom";
import { LinkIcon } from "@heroicons/react/24/solid";

interface StatsItemProps {
    name: string;
    value?: number;
    placeholder?: string;
    onClick?: () => void;
}

const StatsItem = ({ name, placeholder, value, onClick }: StatsItemProps) => (
  <div
    className="group relative px-4 py-3 cursor-pointer overflow-hidden rounded-lg shadow-lg bg-white dark:bg-gray-800 border border-gray-150 dark:border-gray-775 hover:border-gray-300 dark:hover:border-gray-725"
    title="All time"
    onClick={onClick}
  >
    <dt>
      <div className="flex items-center">
        <p className="pb-0.5 text-sm font-medium text-gray-500 group-hover:dark:text-gray-475 group-hover:text-gray-600 truncate">{name}</p>
        <LinkIcon className="h-3 w-3 ml-2 text-gray-500 group-hover:dark:text-gray-475 group-hover:text-gray-600" aria-hidden="true" />
      </div>
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
  const navigate = useNavigate();
  const handleStatClick = (filterType: string) => {
    if (filterType) {
      navigate(`/releases?filter=${filterType}`);
    } else {
      navigate("/releases");
    }
  };

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

      <dl className={classNames("grid grid-cols-2 gap-5 mt-5 sm:grid-cols-2 lg:grid-cols-4", isLoading ? "animate-pulse" : "")}>
        <StatsItem name="Filtered Releases" onClick={() => handleStatClick("")} value={data?.filtered_count ?? 0} />
        {/* <StatsItem name="Filter Rejected Releases" stat={data?.filter_rejected_count} /> */}
        <StatsItem name="Approved Pushes" onClick={() => handleStatClick("PUSH_APPROVED")}  value={data?.push_approved_count ?? 0} />
        <StatsItem name="Rejected Pushes" onClick={() => handleStatClick("PUSH_REJECTED")}  value={data?.push_rejected_count ?? 0 } />
        <StatsItem name="Errored Pushes" onClick={() => handleStatClick("PUSH_ERROR")}  value={data?.push_error_count ?? 0} />
      </dl>
    </div>
  );
};
