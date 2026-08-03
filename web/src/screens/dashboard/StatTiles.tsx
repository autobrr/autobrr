/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { classNames } from "@utils";
import { LinkIcon } from "@heroicons/react/24/solid";
import { ReleasesStatsQueryOptions } from "@api/queries";

interface StatTileProps {
  name: string;
  value: number;
  isLoading: boolean;
  isError: boolean;
  eventType?: "" | "PUSH_APPROVED" | "PUSH_REJECTED" | "PUSH_ERROR";
}

const StatTile = ({ name, value, isLoading, isError, eventType }: StatTileProps) => (
  <Link
    className={classNames(
      "group relative block h-full px-4 py-3 cursor-pointer overflow-hidden rounded-lg shadow-lg bg-white dark:bg-gray-800 hover:shadow-xl transition-all duration-200 ease-in-out",
      isLoading ? "animate-pulse" : ""
    )}
    to="/releases"
    search={{
      action_status: eventType
    }}
    params={{}}
  >
    <div className="flex items-center text-sm font-medium text-gray-500 dark:group-hover:text-gray-475 group-hover:text-gray-600 transition-colors duration-200 ease-in-out">
      <p className="pb-0.5 truncate">{name}</p>
      <LinkIcon className="h-3 w-3 ml-2" aria-hidden="true" />
    </div>
    <p className="text-3xl font-extrabold text-gray-900 dark:text-gray-200">{isError ? "-" : value}</p>
  </Link>
);

export const FilteredTile = () => {
  const { t } = useTranslation("common");
  const { isLoading, isError, data } = useQuery(ReleasesStatsQueryOptions());
  return <StatTile name={t("dashboardStats.filteredReleases")} value={data?.filtered_count ?? 0} isLoading={isLoading} isError={isError} />;
};

export const ApprovedTile = () => {
  const { t } = useTranslation("common");
  const { isLoading, isError, data } = useQuery(ReleasesStatsQueryOptions());
  return <StatTile name={t("dashboardStats.approvedPushes")} value={data?.push_approved_count ?? 0} isLoading={isLoading} isError={isError} eventType="PUSH_APPROVED" />;
};

export const RejectedTile = () => {
  const { t } = useTranslation("common");
  const { isLoading, isError, data } = useQuery(ReleasesStatsQueryOptions());
  return <StatTile name={t("dashboardStats.rejectedPushes")} value={data?.push_rejected_count ?? 0} isLoading={isLoading} isError={isError} eventType="PUSH_REJECTED" />;
};

export const ErroredTile = () => {
  const { t } = useTranslation("common");
  const { isLoading, isError, data } = useQuery(ReleasesStatsQueryOptions());
  return <StatTile name={t("dashboardStats.erroredPushes")} value={data?.push_error_count ?? 0} isLoading={isLoading} isError={isError} eventType="PUSH_ERROR" />;
};
