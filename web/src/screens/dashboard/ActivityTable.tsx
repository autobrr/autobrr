/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMemo } from "react";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
  useTable,
  flexRender,
  type ColumnDef
} from "@tanstack/react-table";
import { useTranslation } from "react-i18next";

import { EmptyListState } from "@components/emptystates";
import * as DataTable from "@components/data-table";
import { RandomLinuxIsos, RandomIsoTracker } from "@utils";
import { ReleasesLatestQueryOptions } from "@api/queries";
import { IndexerCell } from "@components/data-table";
import { dataTableFeatures, type DataTableFeatures } from "@components/data-table/features";
import { SettingsContext } from "@utils/Context";

interface TableProps {
  columns: ColumnDef<DataTableFeatures, Release>[];
  data: Release[];
}

function Table({ columns, data }: TableProps) {
  const { t } = useTranslation("common");
  const tableInstance = useTable({
    features: dataTableFeatures,
    columns,
    data,
  })

  if (data.length === 0) {
    return (
      <div className="mb-2 mt-4 overflow-auto rounded-md border border-gray-250 bg-white shadow-table dark:border-gray-775 dark:bg-gray-800">
        <div className="flex items-center justify-center py-16">
          <EmptyListState text={t("activityTable.noRecentActivity")}/>
        </div>
      </div>
    )
  }

  return (
    <div className="mb-2 mt-4 min-w-0 flex-1 overflow-auto rounded-md border border-gray-250 bg-white align-middle shadow-table dark:border-gray-775 dark:bg-gray-800">
      <table className="min-w-full divide-y divide-gray-200 rounded-md dark:divide-gray-750">
        <thead className="bg-gray-100 dark:bg-gray-850">
          {tableInstance.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <th
                  key={header.id}
                  scope="col"
                  className="first:pl-5 first:rounded-tl-md last:rounded-tr-md pl-3 pr-3 py-3 text-xs font-medium tracking-wider text-left uppercase group text-gray-600 dark:text-gray-400"
                  colSpan={header.colSpan}
                >
                  <div className="flex items-center justify-between">
                    {header.isPlaceholder
                      ? null
                      : flexRender(
                        header.column.columnDef.header,
                        header.getContext()
                      )}
                  </div>
                </th>
                )
              )}
            </tr>
          ))}
        </thead>

        <tbody className="divide-y divide-gray-150 dark:divide-gray-750">
          {tableInstance.getRowModel().rows.map((row) => (
            <tr key={row.id}>
              {row.getVisibleCells().map((cell) => (
                <td
                  key={cell.id}
                  className="first:pl-5 pl-3 pr-3 whitespace-nowrap"
                  role="cell"
                >
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export const ActivityTable = () => {
  const { t } = useTranslation("common");
  const columns = useMemo<ColumnDef<DataTableFeatures, Release>[]>(() => [
    {
      header: t("activityTable.columns.age"),
      accessorKey: "timestamp",
      cell: DataTable.AgeCell
    },
    {
      header: t("activityTable.columns.release"),
      accessorKey: "name",
      cell: DataTable.TitleCell
    },
    {
      header: t("activityTable.columns.actions"),
      accessorKey: "action_status",
      cell: DataTable.ReleaseStatusCell
    },
    {
      header: t("activityTable.columns.indexer"),
      accessorKey: "indexer.identifier",
      cell: IndexerCell,
    }
  ], [t]);

  const { isLoading, data } = useSuspenseQuery(ReleasesLatestQueryOptions());

  const settings = SettingsContext.useValue();
  const displayData = useMemo(() => {
    const releases = data?.data ?? [];
    if (!settings.incognitoMode) {
      return releases;
    }

    const randomIsoNames = RandomLinuxIsos(releases.length);
    const randomTorrentSiteNames = RandomIsoTracker(releases.length);
    return releases.map((item, index) => {
      const siteName = randomTorrentSiteNames[index % randomTorrentSiteNames.length];
      return {
        ...item,
        name: randomIsoNames[index],
        indexer: {
          id: 0,
          name: siteName,
          identifier: siteName,
          identifier_external: siteName,
        },
      };
    });
  }, [settings.incognitoMode, data?.data]);

  if (isLoading) {
    return (
      <div className="flex h-full min-w-0 flex-col">
        <h3 className="text-2xl font-medium leading-6 text-gray-900 dark:text-gray-200">
          {t("activityTable.title")}
        </h3>
        <div className="animate-pulse text-black dark:text-white">
          <EmptyListState text={t("activityTable.loading")}/>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full min-w-0 flex-col">
      <h3 className="text-2xl font-medium leading-6 text-black dark:text-white">
        {t("activityTable.title")}
      </h3>
      <Table columns={columns} data={displayData}/>
    </div>
  );
};
