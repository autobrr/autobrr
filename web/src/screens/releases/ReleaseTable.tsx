/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { Column, useFilters, usePagination, useSortBy, useTable } from "react-table";
import {
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  EyeIcon,
  EyeSlashIcon
} from "@heroicons/react/24/solid";

import { ReleasesRoute } from "@app/routes";
import { ReleasesListQueryOptions } from "@api/queries";
import { RandomLinuxIsos } from "@utils";
import { RingResizeSpinner, SortDownIcon, SortIcon, SortUpIcon } from "@components/Icons";
import { IndexerSelectColumnFilter, PushStatusSelectColumnFilter, SearchColumnFilter } from "./ReleaseFilters";
import { EmptyListState } from "@components/emptystates";
import { TableButton, TablePageButton } from "@components/data-table/Buttons.tsx";
import { AgeCell, IndexerCell, LinksCell, NameCell, ReleaseStatusCell } from "@components/data-table";

type TableState = {
  queryPageIndex: number;
  queryPageSize: number;
  totalCount: number;
  queryFilters: ReleaseFilter[];
};

const initialState: TableState = {
  queryPageIndex: 0,
  queryPageSize: 10,
  totalCount: 0,
  queryFilters: []
};

enum ActionType {
  PAGE_CHANGED = "PAGE_CHANGED",
  PAGE_SIZE_CHANGED = "PAGE_SIZE_CHANGED",
  TOTAL_COUNT_CHANGED = "TOTAL_COUNT_CHANGED",
  FILTER_CHANGED = "FILTER_CHANGED"
}

type Actions =
  | { type: ActionType.FILTER_CHANGED; payload: ReleaseFilter[]; }
  | { type: ActionType.PAGE_CHANGED; payload: number; }
  | { type: ActionType.PAGE_SIZE_CHANGED; payload: number; }
  | { type: ActionType.TOTAL_COUNT_CHANGED; payload: number; };

const TableReducer = (state: TableState, action: Actions): TableState => {
  switch (action.type) {
  case ActionType.PAGE_CHANGED: {
    return { ...state, queryPageIndex: action.payload };
  }
  case ActionType.PAGE_SIZE_CHANGED: {
    return { ...state, queryPageSize: action.payload };
  }
  case ActionType.FILTER_CHANGED: {
    return { ...state, queryFilters: action.payload };
  }
  case ActionType.TOTAL_COUNT_CHANGED: {
    return { ...state, totalCount: action.payload };
  }
  default: {
    throw new Error(`Unhandled action type: ${action}`);
  }
  }
};

const EmptyReleaseList = () => (
  <div className="bg-white dark:bg-gray-800 border border-gray-250 dark:border-gray-775 shadow-table rounded-md overflow-auto">
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

export const ReleaseTable = () => {
  const search = ReleasesRoute.useSearch()

  const columns = React.useMemo(() => [
    {
      Header: "Age",
      accessor: "timestamp",
      Cell: AgeCell
    },
    {
      Header: "Release",
      accessor: "name",
      Cell: NameCell,
      Filter: SearchColumnFilter
    },
    {
      Header: "Links",
      accessor: (row) => ({ download_url: row.download_url, info_url: row.info_url }),
      id: "links",
      Cell: LinksCell
    },
    {
      Header: "Actions",
      accessor: "action_status",
      Cell: ReleaseStatusCell,
      Filter: PushStatusSelectColumnFilter
    },
    {
      Header: "Indexer",
      accessor: "indexer.identifier",
      Cell: IndexerCell,
      Filter: IndexerSelectColumnFilter,
      filter: "equal"
    }
  ] as Column<Release>[], []);

  if (search.action_status != "") {
    initialState.queryFilters = [{id: "action_status", value: search.action_status! }]
  }

  const [{ queryPageIndex, queryPageSize, totalCount, queryFilters }, dispatch] =
        React.useReducer(TableReducer, initialState);

  const { isLoading, error, data, isSuccess } = useQuery(ReleasesListQueryOptions(queryPageIndex * queryPageSize, queryPageSize, queryFilters));

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

  const displayData = showLinuxIsos ? modifiedData : (data?.data ?? []);

  const {
    getTableProps,
    getTableBodyProps,
    headerGroups,
    prepareRow,
    page, // Instead of using 'rows', we'll use page,
    // which has only the rows for the active page

    // The rest of these things are super handy, too ;)
    canPreviousPage,
    canNextPage,
    pageOptions,
    pageCount,
    gotoPage,
    nextPage,
    previousPage,
    setPageSize,
    state: { pageIndex, pageSize, filters }
  } = useTable(
    {
      columns,
      data: displayData, // Use displayData here
      initialState: {
        pageIndex: queryPageIndex,
        pageSize: queryPageSize,
        filters: queryFilters,
      },
      manualPagination: true,
      manualFilters: true,
      manualSortBy: true,
      pageCount: isSuccess ? Math.ceil(totalCount / queryPageSize) : 0,
      autoResetSortBy: false,
      autoResetExpanded: false,
      autoResetPage: false
    },
    useFilters,
    useSortBy,
    usePagination
  );

  React.useEffect(() => {
    dispatch({ type: ActionType.PAGE_CHANGED, payload: pageIndex });
  }, [pageIndex]);

  React.useEffect(() => {
    dispatch({ type: ActionType.PAGE_SIZE_CHANGED, payload: pageSize });
    gotoPage(0);
  }, [pageSize, gotoPage]);

  React.useEffect(() => {
    if (data?.count) {
      dispatch({
        type: ActionType.TOTAL_COUNT_CHANGED,
        payload: data.count
      });
    }
  }, [data?.count]);

  React.useEffect(() => {
    dispatch({ type: ActionType.FILTER_CHANGED, payload: filters });
    gotoPage(0);
  }, [filters]);

  React.useEffect(() => {
    if (search.action_status != null) {
      dispatch({ type: ActionType.FILTER_CHANGED, payload: [{ id: "action_status", value: search.action_status! }] });
    }
  }, [search.action_status]);

  if (error) {
    return <p>Error</p>;
  }

  if (isLoading) {
    return (
      <div>
        <div className="flex mb-6 flex-col sm:flex-row">
          { headerGroups.map((headerGroup) => headerGroup.headers.map((column) => (
              column.Filter ? (
                <React.Fragment key={ column.id }>{ column.render("Filter") }</React.Fragment>
              ) : null
            ))
          ) }
        </div>
        <div className="bg-white dark:bg-gray-800 border border-gray-250 dark:border-gray-775 shadow-lg rounded-md mt-4">
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

  // Render the UI for your table
  return (
    <div className="flex flex-col">
      <div className="flex mb-6 flex-col sm:flex-row">
        {headerGroups.map((headerGroup) =>
          headerGroup.headers.map((column) => (
            column.Filter ? (
              <React.Fragment key={column.id}>{column.render("Filter")}</React.Fragment>
            ) : null
          ))
        )}
      </div>
      <div className="relative">
        {displayData.length === 0
          ? <EmptyReleaseList/>
          : (
          <div className="bg-white dark:bg-gray-800 border border-gray-250 dark:border-gray-775 shadow-table rounded-md overflow-auto">
            <table {...getTableProps()} className="min-w-full rounded-md divide-y divide-gray-200 dark:divide-gray-750">
              <thead className="bg-gray-100 dark:bg-gray-850">
              {headerGroups.map((headerGroup) => {
                const {key: rowKey, ...rowRest} = headerGroup.getHeaderGroupProps();
                return (
                  <tr key={rowKey} {...rowRest}>
                    {headerGroup.headers.map((column) => {
                      const {key: columnKey, ...columnRest} = column.getHeaderProps(column.getSortByToggleProps());
                      return (
                        // Add the sorting props to control sorting. For this example
                        // we can add them into the header props
                        <th
                          key={`${rowKey}-${columnKey}`}
                          scope="col"
                          className="first:pl-5 first:rounded-tl-md last:rounded-tr-md pl-3 pr-3 py-3 text-xs font-medium tracking-wider text-left uppercase group text-gray-600 dark:text-gray-400 transition hover:bg-gray-200 dark:hover:bg-gray-775"
                          {...columnRest}
                        >
                          <div className="flex items-center justify-between">
                            <>{column.render("Header")}</>
                            {/* Add a sort direction indicator */}
                            <span>
                              {column.isSorted ? (
                                column.isSortedDesc ? (
                                  <SortDownIcon className="w-4 h-4 text-gray-400"/>
                                ) : (
                                  <SortUpIcon className="w-4 h-4 text-gray-400"/>
                                )
                              ) : (
                                <SortIcon className="w-4 h-4 text-gray-400 opacity-0 group-hover:opacity-100"/>
                              )}
                            </span>
                          </div>
                        </th>
                      );
                    })}
                  </tr>
                );
              })}
              </thead>
              <tbody
                {...getTableBodyProps()}
                className="divide-y divide-gray-150 dark:divide-gray-750"
              >
              {page.map((row) => {
                prepareRow(row);

                const {key: bodyRowKey, ...bodyRowRest} = row.getRowProps();
                return (
                  <tr key={bodyRowKey} {...bodyRowRest}>
                    {row.cells.map((cell) => {
                      const {key: cellRowKey, ...cellRowRest} = cell.getCellProps();
                      return (
                        <td
                          key={cellRowKey}
                          className="first:pl-5 pl-3 pr-3 whitespace-nowrap"
                          role="cell"
                          {...cellRowRest}
                        >
                          <>{cell.render("Cell")}</>
                        </td>
                      );
                    })}
                  </tr>
                );
              })}
              </tbody>
            </table>
            {/* Pagination */}
            <div className="flex items-center justify-between px-6 py-3 border-t border-gray-200 dark:border-gray-700">
              <div className="flex justify-between flex-1 sm:hidden">
                <TableButton onClick={() => previousPage()} disabled={!canPreviousPage}>Previous</TableButton>
                <TableButton onClick={() => nextPage()} disabled={!canNextPage}>Next</TableButton>
              </div>
              <div className="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
                <div className="flex items-baseline gap-x-2">
                <span className="text-sm text-gray-700 dark:text-gray-500">
                Page <span className="font-medium">{pageIndex + 1}</span> of <span
                  className="font-medium">{pageOptions.length}</span>
                </span>
                  <label>
                    <span className="sr-only bg-gray-700">Items Per Page</span>
                    <select
                      className="py-1 pl-2 pr-8 text-sm block w-full border-gray-300 rounded-md shadow-sm cursor-pointer transition-colors dark:bg-gray-800 dark:border-gray-600 dark:text-gray-400 dark:hover:text-gray-200 focus:border-blue-300 focus:ring focus:ring-blue-200 focus:ring-opacity-50"
                      value={pageSize}
                      onChange={e => {
                        setPageSize(Number(e.target.value));
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
                      onClick={() => gotoPage(0)}
                      disabled={!canPreviousPage}
                    >
                      <span className="sr-only">First</span>
                      <ChevronDoubleLeftIcon className="w-4 h-4" aria-hidden="true"/>
                    </TablePageButton>
                    <TablePageButton
                      className="pl-1 pr-2"
                      onClick={() => previousPage()}
                      disabled={!canPreviousPage}
                    >
                      <ChevronLeftIcon className="w-4 h-4 mr-1" aria-hidden="true"/>
                      <span>Prev</span>
                    </TablePageButton>
                    <TablePageButton
                      className="pl-2 pr-1"
                      onClick={() => nextPage()}
                      disabled={!canNextPage}>
                      <span>Next</span>
                      <ChevronRightIcon className="w-4 h-4 ml-1" aria-hidden="true"/>
                    </TablePageButton>
                    <TablePageButton
                      className="rounded-r-md"
                      onClick={() => gotoPage(pageCount - 1)}
                      disabled={!canNextPage}
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
