/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useSearch } from "@tanstack/react-router";
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  ColumnDef,
  Column,
  RowData,
  PaginationState,
} from "@tanstack/react-table";
import {
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  EyeIcon,
  EyeSlashIcon
} from "@heroicons/react/24/solid";

import { ReleasesListQueryOptions } from "@api/queries";
import { RandomLinuxIsos } from "@utils";
import { RingResizeSpinner } from "@components/Icons";
import { IndexerSelectColumnFilter, PushStatusSelectColumnFilter, SearchColumnFilter } from "./ReleaseFilters";
import { EmptyListState } from "@components/emptystates";
import { TableButton, TablePageButton, AgeCell, IndexerCell, LinksCell, NameCell, ReleaseStatusCell } from "@components/data-table";

declare module '@tanstack/react-table' {
  //allows us to define custom properties for our columns
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  interface ColumnMeta<TData extends RowData, TValue> {
    filterVariant?: 'text' | 'range' | 'select' | 'search' | 'actionPushStatus' | 'indexerSelect';
  }
}

const EmptyReleaseList = () => (
  <div
    className="bg-white dark:bg-gray-800 border border-gray-250 dark:border-gray-775 shadow-table rounded-md overflow-auto">
    <table className="min-w-full rounded-md divide-y divide-gray-200 dark:divide-gray-750">
      <thead className="bg-gray-100 dark:bg-gray-850 border-b border-gray-200 dark:border-gray-750">
      <tr>
        <th>
          <div className="flex items-center justify-between">
            <span className="h-10"/>
          </div>
        </th>
      </tr>
      </thead>
    </table>
    <div className="flex items-center justify-center py-52">
      <EmptyListState text="No results"/>
    </div>
  </div>
);

function Filter({ column }: { column: Column<Release, unknown> }) {
  const { filterVariant } = column.columnDef.meta ?? {}

  switch (filterVariant) {
    case "search":
      return <SearchColumnFilter column={column}/>

    case "indexerSelect":
      return <IndexerSelectColumnFilter column={column}/>

    case "actionPushStatus":
      return <PushStatusSelectColumnFilter column={column}/>

    default:
      return null;
  }
}

interface ColumnFilter {
  id: string
  value: unknown
}

type ColumnFiltersState = ColumnFilter[];

export const ReleaseTable = () => {
  const search = useSearch({
    from: "/auth/authenticated-routes/releases" as const,
  });
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);

  // Set initial filter based on URL params
  React.useEffect(() => {
    if (search.action_status) {
      setColumnFilters([{ id: "action_status", value: search.action_status }]);
    }
  }, [search.action_status]);

  const columns = React.useMemo<ColumnDef<Release, unknown>[]>(() => [
    {
      header: "Age",
      accessorKey: "timestamp",
      cell: AgeCell
    },
    {
      header: "Release",
      accessorKey: "name",
      cell: NameCell,
      meta: {
        filterVariant: 'search',
      },
    },
    {
      header: "Links",
      accessorFn: (row) => ({ download_url: row.download_url, info_url: row.info_url }),
      id: "links",
      cell: LinksCell
    },
    {
      header: "Actions",
      accessorKey: "action_status",
      cell: ReleaseStatusCell,
      meta: {
        filterVariant: 'actionPushStatus',
      },
    },
    {
      header: "Indexer",
      accessorKey: "indexer.identifier",
      cell: IndexerCell,
      meta: {
        filterVariant: 'indexerSelect',
      },
    }
  ], []);

  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const {
    isLoading,
    error,
    data,
  } = useQuery(ReleasesListQueryOptions(pagination.pageIndex * pagination.pageSize, pagination.pageSize, columnFilters));

  const [modifiedData, setModifiedData] = useState<Release[]>([]);
  const [showLinuxIsos, setShowLinuxIsos] = useState(false);

  const toggleReleaseNames = () => {
    setShowLinuxIsos(!showLinuxIsos);
    if (!showLinuxIsos && data && data.data) {
      const randomNames = RandomLinuxIsos(data.data.length);
      const newData: Release[] = data.data.map((item, index) => ({
        ...item,
        name: `${randomNames[index]}.iso`,
        indexer: {
          id: 0,
          name: index % 2 === 0 ? "distrowatch" : "linuxtracker",
          identifier: index % 2 === 0 ? "distrowatch" : "linuxtracker",
          identifier_external: index % 2 === 0 ? "distrowatch" : "linuxtracker",
        },
        category: "Linux ISOs",
        size: index % 2 === 0 ? 4566784529 : (index % 3 === 0 ? 7427019812 : 2312122455),
        source: "",
        container: "",
        codec: "",
        resolution: "",
      }));
      setModifiedData(newData);
    }
  };

  const defaultData = React.useMemo(() => [], [])
  const displayData = showLinuxIsos ? modifiedData : (data?.data ?? defaultData);

  const tableInstance = useReactTable({
    columns,
    data: displayData,
    getCoreRowModel: getCoreRowModel(),
    manualFiltering: true,
    manualPagination: true,
    manualSorting: true,
    rowCount: data?.count,
    state: {
      columnFilters,
      pagination,
    },
    initialState: {
      pagination
    },
    onPaginationChange: setPagination,
    onColumnFiltersChange: setColumnFilters,
  });

  // Manage your own state
  // const [state, setState] = React.useState(tableInstance.initialState)

  // Override the state managers for the table to your own
  // tableInstance.setOptions(prev => ({
  //   ...prev,
  //   state,
  //   onStateChange: setState,
  //   // These are just table options, so if things
  //   // need to change based on your state, you can
  //   // derive them here
  //
  //   // Just for fun, let's debug everything if the pageIndex
  //   // is greater than 2
  //   // debugTable: state.pagination.pageIndex > 2,
  // }))

  if (error) {
    return <p>Error</p>;
  }

  if (isLoading) {
    return (
      <div>
        <div className="flex mb-6 flex-col sm:flex-row">
          {tableInstance.getHeaderGroups().map((headerGroup) =>
            headerGroup.headers.map((header) => (
              header.column.getCanFilter() ? (
                <Filter key={header.column.id} column={header.column}/>
              ) : null
            ))
          )}
        </div>
        <div
          className="bg-white dark:bg-gray-800 border border-gray-250 dark:border-gray-775 shadow-lg rounded-md mt-4">
          <div className="bg-gray-100 dark:bg-gray-850 border-b border-gray-200 dark:border-gray-750">
            <div className="flex h-10"/>
          </div>
          <div className="flex items-center justify-center py-64">
            <RingResizeSpinner className="text-blue-500 size-24"/>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col">
      <div className="flex mb-6 flex-col sm:flex-row">
        {tableInstance.getHeaderGroups().map((headerGroup) =>
          headerGroup.headers.map((header) => (
            header.column.getCanFilter() ? (
              <Filter key={header.column.id} column={header.column}/>
            ) : null
          ))
        )}
      </div>
      <div className="relative">
        {displayData.length === 0
          ? <EmptyReleaseList/>
          : (
            <div
              className="bg-white dark:bg-gray-800 border border-gray-250 dark:border-gray-775 shadow-table rounded-md overflow-auto">
              <table className="min-w-full rounded-md divide-y divide-gray-200 dark:divide-gray-750">
                <thead className="bg-gray-100 dark:bg-gray-850">
                {tableInstance.getHeaderGroups().map((headerGroup) => (
                  <tr key={headerGroup.id}>
                    {headerGroup.headers.map((header) => (
                      <th
                        key={header.id}
                        scope="col"
                        colSpan={header.colSpan}
                        className="first:pl-5 first:rounded-tl-md last:rounded-tr-md pl-3 pr-3 py-3 text-xs font-medium tracking-wider text-left uppercase group text-gray-600 dark:text-gray-400 transition hover:bg-gray-200 dark:hover:bg-gray-775"
                      >
                        <div className="flex items-center justify-between">
                          {header.isPlaceholder
                            ? null
                            : flexRender(
                              header.column.columnDef.header,
                              header.getContext()
                            )}
                          {/*<>{header.render("Header")}</>*/}
                          {/*/!* Add a sort direction indicator *!/*/}
                          {/*<span>*/}
                          {/*  {header.isSorted ? (*/}
                          {/*    header.isSortedDesc ? (*/}
                          {/*      <SortDownIcon className="w-4 h-4 text-gray-400"/>*/}
                          {/*    ) : (*/}
                          {/*      <SortUpIcon className="w-4 h-4 text-gray-400"/>*/}
                          {/*    )*/}
                          {/*  ) : (*/}
                          {/*    <SortIcon className="w-4 h-4 text-gray-400 opacity-0 group-hover:opacity-100"/>*/}
                          {/*  )}*/}
                          {/*</span>*/}
                        </div>
                      </th>
                    ))}
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
                        <>
                          {flexRender(
                            cell.column.columnDef.cell,
                            cell.getContext()
                          )}
                        </>
                      </td>
                    ))}
                  </tr>
                ))}
                </tbody>
              </table>

              {/* Pagination */}
              <div className="flex items-center justify-between px-6 py-3 border-t border-gray-200 dark:border-gray-700">
                <div className="flex justify-between flex-1 sm:hidden">
                  <TableButton onClick={() => tableInstance.previousPage()} disabled={!tableInstance.getCanPreviousPage()}>Previous</TableButton>
                  <TableButton onClick={() => tableInstance.nextPage()} disabled={!tableInstance.getCanNextPage()}>Next</TableButton>
                </div>
                <div className="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
                  <div className="flex items-baseline gap-x-2">
                  <span className="text-sm text-gray-700 dark:text-gray-500">
                  Page <span className="font-medium">{tableInstance.getState().pagination.pageIndex + 1}</span> of <span
                    className="font-medium">{tableInstance.getPageCount()}</span>
                  </span>
                    <label>
                      <span className="sr-only bg-gray-700">Items Per Page</span>
                      <select
                        className="py-1 pl-2 pr-8 text-sm block w-full border-gray-300 rounded-md shadow-sm cursor-pointer transition-colors dark:bg-gray-800 dark:border-gray-600 dark:text-gray-400 dark:hover:text-gray-200 focus:border-blue-300 focus:ring focus:ring-blue-200 focus:ring-opacity-50"
                        value={tableInstance.getState().pagination.pageSize}
                        onChange={e => {
                          tableInstance.setPageSize(Number(e.target.value));
                        }}
                      >
                        {[5, 10, 20, 50].map(pageSize => (
                          <option key={pageSize} value={pageSize}>
                            {pageSize} entries
                          </option>
                        ))}
                      </select>
                    </label>
                  </div>
                  <div>
                    <nav className="inline-flex -space-x-px rounded-md shadow-sm" aria-label="Pagination">
                      <TablePageButton
                        className="rounded-l-md"
                        onClick={() => tableInstance.firstPage()}
                        disabled={!tableInstance.getCanPreviousPage()}
                      >
                        <span className="sr-only">First</span>
                        <ChevronDoubleLeftIcon className="w-4 h-4" aria-hidden="true"/>
                      </TablePageButton>
                      <TablePageButton
                        className="pl-1 pr-2"
                        onClick={() => tableInstance.previousPage()}
                        disabled={!tableInstance.getCanPreviousPage()}
                      >
                        <ChevronLeftIcon className="w-4 h-4 mr-1" aria-hidden="true"/>
                        <span>Prev</span>
                      </TablePageButton>
                      <TablePageButton
                        className="pl-2 pr-1"
                        onClick={() => tableInstance.nextPage()}
                        disabled={!tableInstance.getCanNextPage()}>
                        <span>Next</span>
                        <ChevronRightIcon className="w-4 h-4 ml-1" aria-hidden="true"/>
                      </TablePageButton>
                      <TablePageButton
                        className="rounded-r-md"
                        onClick={() => tableInstance.lastPage()}
                        disabled={!tableInstance.getCanNextPage()}
                      >
                        <ChevronDoubleRightIcon className="w-4 h-4" aria-hidden="true"/>
                        <span className="sr-only">Last</span>
                      </TablePageButton>
                    </nav>
                  </div>
                </div>
              </div>

              <div className="absolute -bottom-11 right-0 p-2">
                <button
                  onClick={toggleReleaseNames}
                  className="p-2 absolute bottom-0 right-0 bg-gray-750 text-white rounded-full opacity-10 hover:opacity-100 transition-opacity duration-300"
                  aria-label="Toggle view"
                  title="Go incognito"
                >
                  {showLinuxIsos ? (
                    <EyeIcon className="h-4 w-4"/>
                  ) : (
                    <EyeSlashIcon className="h-4 w-4"/>
                  )}
                </button>
              </div>
            </div>
          )}
      </div>
    </div>
  );
};
