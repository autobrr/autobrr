import * as React from "react";
import { useQuery } from "react-query";
import { CellProps, Column, useFilters, usePagination, useSortBy, useTable } from "react-table";
import {
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon
} from "@heroicons/react/24/solid";

import { APIClient } from "../../api/APIClient";
import { EmptyListState } from "../../components/emptystates";

import * as Icons from "../../components/Icons";
import * as DataTable from "../../components/data-table";

import { IndexerSelectColumnFilter, PushStatusSelectColumnFilter, SearchColumnFilter } from "./Filters";
import { classNames } from "../../utils";
import { ArrowTopRightOnSquareIcon } from "@heroicons/react/24/outline";
import { Tooltip } from "../../components/tooltips/Tooltip";

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
  case ActionType.PAGE_CHANGED:
    return { ...state, queryPageIndex: action.payload };
  case ActionType.PAGE_SIZE_CHANGED:
    return { ...state, queryPageSize: action.payload };
  case ActionType.FILTER_CHANGED:
    return { ...state, queryFilters: action.payload };
  case ActionType.TOTAL_COUNT_CHANGED:
    return { ...state, totalCount: action.payload };
  default:
    throw new Error(`Unhandled action type: ${action}`);
  }
};

export const ReleaseTable = () => {
  const columns = React.useMemo(() => [
    {
      Header: "Age",
      accessor: "timestamp",
      Cell: DataTable.AgeCell
    },
    {
      Header: "Release",
      accessor: "torrent_name",
      Cell: (props: CellProps<Release>) => {
        return (
          <div
            className={classNames(
              "flex justify-between py-3 text-sm font-medium box-content text-gray-900 dark:text-gray-300",
              "max-w-[96px] sm:max-w-[216px] md:max-w-[360px] lg:max-w-[640px] xl:max-w-[840px]"
            )}
          >
            <Tooltip
              label={props.cell.value}
              maxWidth="max-w-[90vw]"
            >
              <span className="whitespace-pre-wrap break-words">
                {String(props.cell.value)}
              </span>
            </Tooltip>
            {props.row.original.info_url && (
              <a
                rel="noopener noreferrer"
                target="_blank"
                href={props.row.original.info_url}
                className="max-w-[90vw] mr-2"
              >
                <ArrowTopRightOnSquareIcon className="h-5 w-5 text-blue-400 hover:text-blue-500 dark:text-blue-500 dark:hover:text-blue-600" aria-hidden="true" />
              </a>
            )}
          </div>
        );
      },
      Filter: SearchColumnFilter
    },
    {
      Header: "Actions",
      accessor: "action_status",
      Cell: DataTable.ReleaseStatusCell,
      Filter: PushStatusSelectColumnFilter
    },
    {
      Header: "Indexer",
      accessor: "indexer",
      Cell: DataTable.IndexerCell,
      Filter: IndexerSelectColumnFilter,
      filter: "equal"
    }
  ] as Column<Release>[], []);

  const [{ queryPageIndex, queryPageSize, totalCount, queryFilters }, dispatch] =
        React.useReducer(TableReducer, initialState);

  const { isLoading, error, data, isSuccess } = useQuery(
    ["releases", queryPageIndex, queryPageSize, queryFilters],
    () => APIClient.release.findQuery(queryPageIndex * queryPageSize, queryPageSize, queryFilters),
    {
      keepPreviousData: true,
      staleTime: 5000
    }
  );

  // Use the state and functions returned from useTable to build your UI
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
      data: data && isSuccess ? data.data : [],
      initialState: {
        pageIndex: queryPageIndex,
        pageSize: queryPageSize,
        filters: []
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
  }, [filters]);

  if (error)
    return <p>Error</p>;

  if (isLoading)
    return (
      <div className="animate-pulse flex flex-col">
        <div className="flex mb-6 flex-col sm:flex-row">
          {headerGroups.map((headerGroup) =>
            headerGroup.headers.map((column) => (
              column.Filter ? (
                <React.Fragment key={column.id}>{column.render("Filter")}</React.Fragment>
              ) : null
            ))
          )}
        </div>
        <div className="bg-white shadow-lg dark:bg-gray-800 rounded-md overflow-auto">
          <table {...getTableProps()} className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-800">
              {headerGroups.map((headerGroup) => {
                const { key: rowKey, ...rowRest } = headerGroup.getHeaderGroupProps();
                return (
                  <tr key={rowKey} {...rowRest}>
                    {headerGroup.headers.map((column) => {
                      const { key: columnKey, ...columnRest } = column.getHeaderProps(column.getSortByToggleProps());
                      return (
                      // Add the sorting props to control sorting. For this example
                      // we can add them into the header props
                        <th
                          key={`${rowKey}-${columnKey}`}
                          scope="col"
                          className="first:pl-5 pl-3 pr-3 py-3 first:rounded-tl-md last:rounded-tr-md text-xs font-medium tracking-wider text-left text-gray-500 uppercase group"
                          {...columnRest}
                        >
                          <div className="flex items-center justify-between">
                            <>{column.render("Header")}</>
                            {/* Add a sort direction indicator */}
                            <span>
                              {column.isSorted ? (
                                column.isSortedDesc ? (
                                  <Icons.SortDownIcon className="w-4 h-4 text-gray-400" />
                                ) : (
                                  <Icons.SortUpIcon className="w-4 h-4 text-gray-400" />
                                )
                              ) : (
                                <Icons.SortIcon className="w-4 h-4 text-gray-400 opacity-0 group-hover:opacity-100" />
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
            <tbody className=" divide-gray-200 dark:divide-gray-700">
              <tr className="flex justify-between py-4 text-sm font-medium box-content text-gray-900 dark:text-gray-300 max-w-[96px] sm:max-w-[216px] md:max-w-[360px] lg:max-w-[640px] xl:max-w-[840px]">
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
              </tr>
              <tr className="flex justify-between py-4 text-sm font-medium box-content text-gray-900 dark:text-gray-300 max-w-[96px] sm:max-w-[216px] md:max-w-[360px] lg:max-w-[640px] xl:max-w-[840px]">
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
              </tr>
              <tr className="flex justify-between py-4 text-sm font-medium box-content text-gray-900 dark:text-gray-300 max-w-[96px] sm:max-w-[216px] md:max-w-[360px] lg:max-w-[640px] xl:max-w-[840px]">
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap">&nbsp;</td>
              </tr>
              <tr className="flex justify-between py-4 text-sm font-medium box-content text-gray-900 dark:text-gray-300 max-w-[96px] sm:max-w-[216px] md:max-w-[360px] lg:max-w-[640px] xl:max-w-[840px]">
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
              </tr>
              <tr className="justify-between py-3 text-sm font-medium box-content text-gray-900 dark:text-gray-300">
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap text-center">
                  <p className="text-black dark:text-white">Loading release table...</p>
                </td>
              </tr>
              <tr className="flex justify-between py-3 text-sm font-medium box-content text-gray-900 dark:text-gray-300 max-w-[96px] sm:max-w-[216px] md:max-w-[360px] lg:max-w-[640px] xl:max-w-[840px]">
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
              </tr>
              <tr className="flex justify-between py-3 text-sm font-medium box-content text-gray-900 dark:text-gray-300 max-w-[96px] sm:max-w-[216px] md:max-w-[360px] lg:max-w-[640px] xl:max-w-[840px]">
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
              </tr>
              <tr className="flex justify-between py-3 text-sm font-medium box-content text-gray-900 dark:text-gray-300 max-w-[96px] sm:max-w-[216px] md:max-w-[360px] lg:max-w-[640px] xl:max-w-[840px]">
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
              </tr>
              <tr className="flex justify-between py-3 text-sm font-medium box-content text-gray-900 dark:text-gray-300 max-w-[96px] sm:max-w-[216px] md:max-w-[360px] lg:max-w-[640px] xl:max-w-[840px]">
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
              </tr>
              <tr className="flex justify-between py-3 text-sm font-medium box-content text-gray-900 dark:text-gray-300 max-w-[96px] sm:max-w-[216px] md:max-w-[360px] lg:max-w-[640px] xl:max-w-[840px]">
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
                <td className="first:pl-5 pl-3 pr-3 whitespace-nowrap ">&nbsp;</td>
              </tr>
            </tbody>
          </table>

          {/* Pagination */}
          <div className="flex items-center justify-between px-6 py-3 border-t border-gray-200 dark:border-gray-700">
            <div className="flex justify-between flex-1 sm:hidden">
              <DataTable.Button onClick={() => previousPage()} disabled={!canPreviousPage}>Previous</DataTable.Button>
              <DataTable.Button onClick={() => nextPage()} disabled={!canNextPage}>Next</DataTable.Button>
            </div>
            <div className="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
              <div className="flex items-baseline gap-x-2">
                <span className="text-sm text-gray-700 dark:text-gray-500">
                Page <span className="font-medium">{pageIndex + 1}</span> of <span className="font-medium">{pageOptions.length}</span>
                </span>
                <label>
                  <span className="sr-only bg-gray-700">Items Per Page</span>
                  <select
                    className="py-1 pl-2 pr-8 text-sm block w-full border-gray-300 rounded-md shadow-sm cursor-pointer dark:bg-gray-800 dark:border-gray-600 dark:text-gray-400 dark:hover:text-gray-500 focus:border-blue-300 focus:ring focus:ring-blue-200 focus:ring-opacity-50"
                    value={pageSize}
                    onChange={e => {
                      setPageSize(Number(e.target.value));
                    }}
                  >
                    {[5, 10, 20, 50].map(pageSize => (
                      <option key={pageSize} value={pageSize}>
                      Show {pageSize}
                      </option>
                    ))}
                  </select>
                </label>
              </div>
              <div>
                <nav className="inline-flex -space-x-px rounded-md shadow-sm" aria-label="Pagination">
                  <DataTable.PageButton
                    className="rounded-l-md"
                    onClick={() => gotoPage(0)}
                    disabled={!canPreviousPage}
                  >
                    <span className="sr-only text-gray-400 dark:text-gray-500 dark:bg-gray-700">First</span>
                    <ChevronDoubleLeftIcon className="w-4 h-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
                  </DataTable.PageButton>
                  <DataTable.PageButton
                    onClick={() => previousPage()}
                    disabled={!canPreviousPage}
                  >
                    <span className="sr-only text-gray-400 dark:text-gray-500 dark:bg-gray-700">Previous</span>
                    <ChevronLeftIcon className="w-4 h-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
                  </DataTable.PageButton>
                  <DataTable.PageButton
                    onClick={() => nextPage()}
                    disabled={!canNextPage}>
                    <span className="sr-only text-gray-400 dark:text-gray-500 dark:bg-gray-700">Next</span>
                    <ChevronRightIcon className="w-4 h-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
                  </DataTable.PageButton>
                  <DataTable.PageButton
                    className="rounded-r-md"
                    onClick={() => gotoPage(pageCount - 1)}
                    disabled={!canNextPage}
                  >
                    <span className="sr-only text-gray-400 dark:text-gray-500 dark:bg-gray-700">Last</span>
                    <ChevronDoubleRightIcon className="w-4 h-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
                  </DataTable.PageButton>
                </nav>
              </div>
            </div>
          </div>
        </div>
        <EmptyListState text="Loading release table..." />
      </div>
    );

  if (!data)
    return <EmptyListState text="No recent activity" />;

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
      <div className="bg-white shadow-lg dark:bg-gray-800 rounded-md overflow-auto">
        <table {...getTableProps()} className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead className="bg-gray-50 dark:bg-gray-800">
            {headerGroups.map((headerGroup) => {
              const { key: rowKey, ...rowRest } = headerGroup.getHeaderGroupProps();
              return (
                <tr key={rowKey} {...rowRest}>
                  {headerGroup.headers.map((column) => {
                    const { key: columnKey, ...columnRest } = column.getHeaderProps(column.getSortByToggleProps());
                    return (
                    // Add the sorting props to control sorting. For this example
                    // we can add them into the header props
                      <th
                        key={`${rowKey}-${columnKey}`}
                        scope="col"
                        className="first:pl-5 pl-3 pr-3 py-3 first:rounded-tl-md last:rounded-tr-md text-xs font-medium tracking-wider text-left text-gray-500 uppercase group"
                        {...columnRest}
                      >
                        <div className="flex items-center justify-between">
                          <>{column.render("Header")}</>
                          {/* Add a sort direction indicator */}
                          <span>
                            {column.isSorted ? (
                              column.isSortedDesc ? (
                                <Icons.SortDownIcon className="w-4 h-4 text-gray-400" />
                              ) : (
                                <Icons.SortUpIcon className="w-4 h-4 text-gray-400" />
                              )
                            ) : (
                              <Icons.SortIcon className="w-4 h-4 text-gray-400 opacity-0 group-hover:opacity-100" />
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
            className="divide-y divide-gray-200 dark:divide-gray-700"
          >
            {page.map((row) => {
              prepareRow(row);

              const { key: bodyRowKey, ...bodyRowRest } = row.getRowProps();
              return (
                <tr key={bodyRowKey} {...bodyRowRest}>
                  {row.cells.map((cell) => {
                    const { key: cellRowKey, ...cellRowRest } = cell.getCellProps();
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
            <DataTable.Button onClick={() => previousPage()} disabled={!canPreviousPage}>Previous</DataTable.Button>
            <DataTable.Button onClick={() => nextPage()} disabled={!canNextPage}>Next</DataTable.Button>
          </div>
          <div className="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
            <div className="flex items-baseline gap-x-2">
              <span className="text-sm text-gray-700 dark:text-gray-500">
                Page <span className="font-medium">{pageIndex + 1}</span> of <span className="font-medium">{pageOptions.length}</span>
              </span>
              <label>
                <span className="sr-only bg-gray-700">Items Per Page</span>
                <select
                  className="py-1 pl-2 pr-8 text-sm block w-full border-gray-300 rounded-md shadow-sm cursor-pointer dark:bg-gray-800 dark:border-gray-600 dark:text-gray-400 dark:hover:text-gray-500 focus:border-blue-300 focus:ring focus:ring-blue-200 focus:ring-opacity-50"
                  value={pageSize}
                  onChange={e => {
                    setPageSize(Number(e.target.value));
                  }}
                >
                  {[5, 10, 20, 50].map(pageSize => (
                    <option key={pageSize} value={pageSize}>
                      Show {pageSize}
                    </option>
                  ))}
                </select>
              </label>
            </div>
            <div>
              <nav className="inline-flex -space-x-px rounded-md shadow-sm" aria-label="Pagination">
                <DataTable.PageButton
                  className="rounded-l-md"
                  onClick={() => gotoPage(0)}
                  disabled={!canPreviousPage}
                >
                  <span className="sr-only text-gray-400 dark:text-gray-500 dark:bg-gray-700">First</span>
                  <ChevronDoubleLeftIcon className="w-4 h-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
                </DataTable.PageButton>
                <DataTable.PageButton
                  onClick={() => previousPage()}
                  disabled={!canPreviousPage}
                >
                  <span className="sr-only text-gray-400 dark:text-gray-500 dark:bg-gray-700">Previous</span>
                  <ChevronLeftIcon className="w-4 h-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
                </DataTable.PageButton>
                <DataTable.PageButton
                  onClick={() => nextPage()}
                  disabled={!canNextPage}>
                  <span className="sr-only text-gray-400 dark:text-gray-500 dark:bg-gray-700">Next</span>
                  <ChevronRightIcon className="w-4 h-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
                </DataTable.PageButton>
                <DataTable.PageButton
                  className="rounded-r-md"
                  onClick={() => gotoPage(pageCount - 1)}
                  disabled={!canNextPage}
                >
                  <span className="sr-only text-gray-400 dark:text-gray-500 dark:bg-gray-700">Last</span>
                  <ChevronDoubleRightIcon className="w-4 h-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
                </DataTable.PageButton>
              </nav>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
