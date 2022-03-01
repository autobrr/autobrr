import * as React from "react";
import { useQuery } from "react-query";
import {
    useTable,
    useFilters,
    useGlobalFilter,
    useSortBy,
    usePagination
} from "react-table";

import { APIClient } from "../../api/APIClient";
import { EmptyListState } from "../../components/emptystates";

import * as Icons from "../../components/Icons";
import * as DataTable from "../../components/data-table";

// This is a custom filter UI for selecting
// a unique option from a list
function SelectColumnFilter({
    column: { filterValue, setFilter, preFilteredRows, id, render },
}: any) {
    // Calculate the options for filtering
    // using the preFilteredRows
    const options = React.useMemo(() => {
        const options: any = new Set()
        preFilteredRows.forEach((row: { values: { [x: string]: unknown } }) => {
            options.add(row.values[id])
        })
        return [...options.values()]
    }, [id, preFilteredRows])

    // Render a multi-select box
    return (
        <label className="flex items-baseline gap-x-2">
            <span className="text-gray-700">{render("Header")}: </span>
            <select
                className="border-gray-300 rounded-md shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"
                name={id}
                id={id}
                value={filterValue}
                onChange={e => {
                    setFilter(e.target.value || undefined)
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
    )
}

function Table({ columns, data }: any) {
    // Use the state and functions returned from useTable to build your UI
    const {
        getTableProps,
        getTableBodyProps,
        headerGroups,
        prepareRow,
        page, // Instead of using 'rows', we'll use page,
    } = useTable(
        { columns, data },
        useFilters,
        useGlobalFilter,
        useSortBy,
        usePagination
    );

    if (!page.length)
        return <EmptyListState text="No recent activity" />;

    // Render the UI for your table
    return (
        <div className="flex flex-col mt-4">
            <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
                <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
                    <div className="overflow-hidden bg-white shadow dark:bg-gray-800 sm:rounded-lg">
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
                                                        className="px-6 py-3 text-xs font-medium tracking-wider text-left text-gray-500 uppercase group"
                                                        {...columnRest}
                                                    >
                                                        <div className="flex items-center justify-between">
                                                            {column.render('Header')}
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
                                {page.map((row: any) => {
                                    prepareRow(row);
                                    const { key: bodyRowKey, ...bodyRowRest } = row.getRowProps();
                                    return (
                                        <tr key={bodyRowKey} {...bodyRowRest}>
                                            {row.cells.map((cell: any) => {
                                                const { key: cellRowKey, ...cellRowRest } = cell.getCellProps();
                                                return (
                                                    <td
                                                        key={cellRowKey}
                                                        className="px-6 py-4 whitespace-nowrap"
                                                        role="cell"
                                                        {...cellRowRest}
                                                    >
                                                        {cell.column.Cell.name === "defaultRenderer"
                                                            ? <div className="text-sm text-gray-500">{cell.render('Cell')}</div>
                                                            : cell.render('Cell')
                                                        }
                                                    </td>
                                                )
                                            })}
                                        </tr>
                                    )
                                })}
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    );
}

export const ActivityTable = () => {
    const columns = React.useMemo(() => [
        {
            Header: "Age",
            accessor: 'timestamp',
            Cell: DataTable.AgeCell,
        },
        {
            Header: "Release",
            accessor: 'torrent_name',
            Cell: DataTable.ReleaseCell,
        },
        {
            Header: "Actions",
            accessor: 'action_status',
            Cell: DataTable.ReleaseStatusCell,
        },
        {
            Header: "Indexer",
            accessor: 'indexer',
            Cell: DataTable.IndexerCell,
            Filter: SelectColumnFilter,
            filter: 'includes',
        },
    ], [])

    const { isLoading, data } = useQuery(
        'dash_release',
        () => APIClient.release.find("?limit=10"),
        { refetchOnWindowFocus: false }
    );

    if (isLoading)
        return null;

    return (
        <div className="flex flex-col mt-12">
            <h3 className="text-2xl font-medium leading-6 text-gray-900 dark:text-gray-200">
                Recent activity
            </h3>

            <Table columns={columns} data={data?.data} />
        </div>
    );
}
