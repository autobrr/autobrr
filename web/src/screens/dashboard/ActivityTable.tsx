/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect } from "react";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  ColumnDef
} from "@tanstack/react-table";
import { EyeIcon, EyeSlashIcon } from "@heroicons/react/24/solid";

import { EmptyListState } from "@components/emptystates";
import * as DataTable from "@components/data-table";
import { RandomLinuxIsos, RandomIsoTracker } from "@utils";
import { ReleasesLatestQueryOptions } from "@api/queries";
import { IndexerCell } from "@components/data-table";
import { SettingsContext } from "@utils/Context";

interface TableProps {
  columns: ColumnDef<Release>[];
  data: Release[];
}

function Table({ columns, data }: TableProps) {
  const tableInstance = useReactTable({
    columns,
    data,
    getCoreRowModel: getCoreRowModel(),
  })

  if (data.length === 0) {
    return (
      <div
        className="mt-4 mb-2 bg-white dark:bg-gray-800 border border-gray-250 dark:border-gray-775 shadow-table rounded-md overflow-auto">
        <div className="flex items-center justify-center py-16">
          <EmptyListState text="No recent activity"/>
        </div>
      </div>
    )
  }

  return (
    <div className="inline-block min-w-full mt-4 mb-2 align-middle">
      <div className="bg-white dark:bg-gray-800 border border-gray-250 dark:border-gray-775 shadow-table rounded-md overflow-auto">
        <table className="min-w-full rounded-md divide-y divide-gray-200 dark:divide-gray-750">
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
    </div>
  );
}

export const ActivityTable = () => {
  const columns = React.useMemo<ColumnDef<Release>[]>(() => [
    {
      header: "Age",
      accessorKey: "timestamp",
      cell: DataTable.AgeCell
    },
    {
      header: "Release",
      accessorKey: "name",
      cell: DataTable.TitleCell
    },
    {
      header: "Actions",
      accessorKey: "action_status",
      cell: DataTable.ReleaseStatusCell
    },
    {
      header: "Indexer",
      accessorKey: "indexer.identifier",
      cell: IndexerCell,
    }
  ], []);

  const { isLoading, data, dataUpdatedAt } = useSuspenseQuery(ReleasesLatestQueryOptions());

  const [modifiedData, setModifiedData] = useState<Release[]>([]);
  const [settings, setSettings] = SettingsContext.use();

  useEffect(() => {
    if (settings.incognitoMode && data?.data) {
      const randomIsoNames = RandomLinuxIsos(data.data.length);
      const randomTorrentSiteNames = RandomIsoTracker(data.data.length);
      const newData: Release[] = data.data.map((item, index) => {
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
      setModifiedData(newData);
    } else {
      setModifiedData([]);
    }
  }, [settings.incognitoMode, data?.data, dataUpdatedAt]);

  if (isLoading) {
    return (
      <div className="flex flex-col mt-12">
        <h3 className="text-2xl font-medium leading-6 text-gray-900 dark:text-gray-200">
          Recent activity
        </h3>
        <div className="animate-pulse text-black dark:text-white">
          <EmptyListState text="Loading..."/>
        </div>
      </div>
    );
  }

  const toggleReleaseNames = () => {
    setSettings(prev => ({ ...prev, incognitoMode: !prev.incognitoMode }));
  };

  const displayData = settings.incognitoMode ? modifiedData : [...(data?.data ?? [])];

  return (
    <div className="flex flex-col mt-12 relative">
      <h3 className="text-2xl font-medium leading-6 text-black dark:text-white">
        Recent activity
      </h3>

      <Table columns={columns} data={displayData}/>

      <button
        onClick={toggleReleaseNames}
        className="p-2 absolute -bottom-8 right-0 bg-gray-750 text-white rounded-full opacity-10 hover:opacity-100 transition-opacity duration-300"
        aria-label="Toggle view"
        title="Go incognito"
      >
        {settings.incognitoMode ? (
          <EyeIcon className="h-4 w-4"/>
        ) : (
          <EyeSlashIcon className="h-4 w-4"/>
        )}
      </button>
    </div>
  );
};
