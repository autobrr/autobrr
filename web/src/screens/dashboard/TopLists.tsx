/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";

import { ReleasesTopFiltersQueryOptions, ReleasesTopIndexersQueryOptions } from "@api/queries";
import { SettingsContext } from "@utils/Context";
import { ChartCard, ChartError, ChartSkeleton, seriesColors, useIsDark } from "./charts";

interface BreakdownRow {
  name: string;
  matched: number;
  approved: number;
}

interface BreakdownTableProps {
  title: string;
  nameHeader: string;
  rows: BreakdownRow[];
  isLoading: boolean;
  isError: boolean;
}

const BreakdownTable = ({ title, nameHeader, rows, isLoading, isError }: BreakdownTableProps) => {
  const { t } = useTranslation("common");
  const isDark = useIsDark();

  const maxApproved = Math.max(...rows.map((row) => row.approved), 1);
  const barColor = seriesColors(isDark).matches;
  const trackColor = isDark ? "#1e3a8a66" : "#dbeafe";

  return (
    <ChartCard title={title}>
      {isLoading ? (
        <ChartSkeleton heightClass="h-56" />
      ) : isError ? (
        <ChartError heightClass="h-56" />
      ) : (
        <div className="mt-2 min-h-56 min-w-0 flex-1">
          <table className="w-full table-fixed">
            <thead>
              <tr>
                <th className="py-2 pr-3 text-xs font-medium tracking-wider uppercase text-left text-gray-600 dark:text-gray-400">{nameHeader}</th>
                <th className="w-20 px-2 py-2 text-right text-xs font-medium uppercase tracking-wider text-gray-600 sm:px-3 dark:text-gray-400">{t("dashboardCharts.matches")}</th>
                <th className="w-20 px-2 py-2 text-right text-xs font-medium uppercase tracking-wider text-gray-600 sm:px-3 dark:text-gray-400">{t("dashboardCharts.approved")}</th>
                <th className="hidden w-28 py-2 pl-3 sm:table-cell"><span className="sr-only">{t("dashboardCharts.approved")}</span></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-150 dark:divide-gray-750">
              {rows.map((row) => (
                <tr key={row.name}>
                  <td className="max-w-0 truncate whitespace-nowrap py-2 pr-3 text-sm font-medium text-gray-900 dark:text-gray-300">{row.name}</td>
                  <td className="px-2 py-2 text-right text-sm tabular-nums text-gray-600 sm:px-3 dark:text-gray-400">{row.matched.toLocaleString()}</td>
                  <td className="px-2 py-2 text-right text-sm tabular-nums text-gray-900 sm:px-3 dark:text-gray-300">{row.approved.toLocaleString()}</td>
                  <td className="hidden py-2 pl-3 sm:table-cell">
                    <div className="h-1.5 w-full rounded-full" style={{ backgroundColor: trackColor }}>
                      <div
                        className="h-1.5 rounded-full"
                        style={{ backgroundColor: barColor, width: `${Math.max((row.approved / maxApproved) * 100, 2)}%` }}
                      />
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </ChartCard>
  );
};

export const TopIndexers = () => {
  const { t } = useTranslation("common");
  const settings = SettingsContext.useValue();
  const { isLoading, isError, data } = useQuery(ReleasesTopIndexersQueryOptions());

  const rows = (data?.top ?? []).map((indexer, i) => ({
    name: settings.incognitoMode ? `tracker-${i + 1}` : indexer.indexer,
    matched: indexer.matched_count,
    approved: indexer.push_approved_count
  }));

  return (
    <BreakdownTable
      title={t("dashboardCharts.topIndexersTitle")}
      nameHeader={t("dashboardCharts.indexer")}
      rows={rows}
      isLoading={isLoading}
      isError={isError}
    />
  );
};

export const TopFilters = () => {
  const { t } = useTranslation("common");
  const settings = SettingsContext.useValue();
  const { isLoading, isError, data } = useQuery(ReleasesTopFiltersQueryOptions());

  const rows = (data?.top ?? []).map((filter, i) => ({
    name: settings.incognitoMode ? `filter-${i + 1}` : filter.filter,
    matched: filter.matched_count,
    approved: filter.push_approved_count
  }));

  return (
    <BreakdownTable
      title={t("dashboardCharts.topFiltersTitle")}
      nameHeader={t("dashboardCharts.filter")}
      rows={rows}
      isLoading={isLoading}
      isError={isError}
    />
  );
};
