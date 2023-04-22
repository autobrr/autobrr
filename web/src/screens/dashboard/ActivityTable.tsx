import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import {
  useTable,
  useFilters,
  useGlobalFilter,
  useSortBy,
  usePagination, FilterProps, Column
} from "react-table";

import { APIClient } from "@api/APIClient";
import { EmptyListState } from "@components/emptystates";
import * as Icons from "@components/Icons";
import * as DataTable from "@components/data-table";

// This is a custom filter UI for selecting
// a unique option from a list
function SelectColumnFilter({
  column: { filterValue, setFilter, preFilteredRows, id, render }
}: FilterProps<object>) {
  // Calculate the options for filtering
  // using the preFilteredRows
  const options = React.useMemo(() => {
    const options = new Set<string>();
    preFilteredRows.forEach((row: { values: { [x: string]: string } }) => {
      options.add(row.values[id]);
    });
    return [...options.values()];
  }, [id, preFilteredRows]);

  // Render a multi-select box
  return (
    <label className="flex items-baseline gap-x-2">
      <span className="text-gray-700"><>{render("Header")}:</></span>
      <select
        className="border-gray-300 rounded-md shadow-sm focus:border-blue-300 focus:ring focus:ring-blue-200 focus:ring-opacity-50"
        name={id}
        id={id}
        value={filterValue}
        onChange={e => {
          setFilter(e.target.value || undefined);
        }}
      >
        <option value="">All</option>
        {options.map((option, i) => (
          <option key={i} value={option}>
            {option}
          </option>
        ))}
      </select>
    </label>
  );
}

interface TableProps {
  columns: Column[];
  data: Release[];
}

function Table({ columns, data }: TableProps) {
  // Use the state and functions returned from useTable to build your UI
  const {
    getTableProps,
    getTableBodyProps,
    headerGroups,
    prepareRow,
    page // Instead of using 'rows', we'll use page,
  } = useTable(
    { columns, data },
    useFilters,
    useGlobalFilter,
    useSortBy,
    usePagination
  );

  if (!page.length) {
    return <EmptyListState text="No recent activity" />;
  }

  // Render the UI for your table
  return (
    <div className="inline-block min-w-full mt-4 mb-2 align-middle">
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
                        className="first:pl-5 first:rounded-tl-md last:rounded-tr-md pl-3 pr-3 py-3 text-xs font-medium tracking-wider text-left text-gray-500 uppercase group"
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
      </div>
    </div>
  );
}

export const ActivityTable = () => {
  const columns = React.useMemo(() => [
    {
      Header: "Age",
      accessor: "timestamp",
      Cell: DataTable.AgeCell
    },
    {
      Header: "Release",
      accessor: "torrent_name",
      Cell: DataTable.TitleCell
    },
    {
      Header: "Actions",
      accessor: "action_status",
      Cell: DataTable.ReleaseStatusCell
    },
    {
      Header: "Indexer",
      accessor: "indexer",
      Cell: DataTable.TitleCell,
      Filter: SelectColumnFilter,
      filter: "includes"
    }
  ], []);

  const { isLoading, data } = useQuery({
    queryKey: ["dash_recent_releases"],
    queryFn: APIClient.release.findRecent,
    refetchOnWindowFocus: false
  });

  if (isLoading) {
    return (
      <div className="flex flex-col mt-12">
        <h3 className="text-2xl font-medium leading-6 text-gray-900 dark:text-gray-200">
          &nbsp;
        </h3>
        <div className="animate-pulse text-black dark:text-white">
          <EmptyListState text="Loading..."/>
        </div>
      </div>
    );
  }
  
  return (
    <div className="flex flex-col mt-12">
      <h3 className="text-2xl font-medium leading-6 text-gray-900 dark:text-gray-200">
        Recent activity
      </h3>

      <Table columns={columns} data={data?.data ?? []} />
    </div>
  );
};
