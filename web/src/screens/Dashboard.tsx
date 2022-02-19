import * as React from "react";
import { useQuery } from "react-query";
import formatDistanceToNowStrict from "date-fns/formatDistanceToNowStrict";
import {
    useTable,
    useFilters,
    useGlobalFilter,
    useSortBy,
    usePagination
} from "react-table";

import { APIClient } from "../api/APIClient";
import { EmptyListState } from "../components/emptystates";
import { ReleaseStatusCell } from "./Releases";

export function Dashboard() {
    return (
        <main className="py-10">
            <div className="px-4 pb-8 mx-auto max-w-7xl sm:px-6 lg:px-8">
                <Stats />
                <DataTable />
            </div>
        </main>
    )
}

const StatsItem = ({ name, stat }: any) => (
    <div
        className="relative px-4 pt-5 pb-2 overflow-hidden bg-white rounded-lg shadow-lg dark:bg-gray-800 sm:pt-6 sm:px-6"
        title="All time"
    >
        <dt>
            <p className="pb-1 text-sm font-medium text-gray-500 truncate">{name}</p>
        </dt>

        <dd className="flex items-baseline pb-6 sm:pb-7">
            <p className="text-2xl font-semibold text-gray-900 dark:text-gray-200">{stat}</p>
        </dd>
    </div>
)

function Stats() {
    const { isLoading, data } = useQuery(
        'dash_release_stats',
        () => APIClient.release.stats(),
        { refetchOnWindowFocus: false }
    );

    if (isLoading)
        return null;

    return (
        <div>
            <h3 className="text-2xl font-medium leading-6 text-gray-900 dark:text-gray-200">
              Stats
            </h3>

            <dl className="grid grid-cols-1 gap-5 mt-5 sm:grid-cols-2 lg:grid-cols-3">
                <StatsItem name="Filtered Releases" stat={data?.filtered_count} />
                {/* <StatsItem name="Filter Rejected Releases" stat={data?.filter_rejected_count} /> */}
                <StatsItem name="Rejected Pushes" stat={data?.push_rejected_count} />
                <StatsItem name="Approved Pushes" stat={data?.push_approved_count} />
            </dl>
        </div>
    )
}

// This is a custom filter UI for selecting
// a unique option from a list
export function SelectColumnFilter({
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

export function StatusPill({ value }: any) {
    const statusMap: any = {
        "FILTER_APPROVED": <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold uppercase bg-blue-100 text-blue-800 ">Approved</span>,
        "FILTER_REJECTED": <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold uppercase bg-red-100 text-red-800">Rejected</span>,
        "PUSH_REJECTED": <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold uppercase bg-pink-100 text-pink-800">Rejected</span>,
        "PUSH_APPROVED": <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold uppercase bg-green-100 text-green-800">Approved</span>,
        "PENDING": <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold uppercase bg-yellow-100 text-yellow-800">PENDING</span>,
        "MIXED": <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold uppercase bg-yellow-100 text-yellow-800">MIXED</span>,
    }

    return statusMap[value];
}

export function AgeCell({ value }: any) {
    const formatDate = formatDistanceToNowStrict(
        new Date(value),
        { addSuffix: true }
    )

    return (
        <div className="text-sm text-gray-500" title={value}>{formatDate}</div>
    )
}

export function ReleaseCell({ value }: any) {
    return (
        <div className="text-sm font-medium text-gray-900 dark:text-gray-300">{value}</div>
    )
}

export function IndexerCell({ value }: any) {
    return (
        <div className="text-sm font-medium text-gray-900 dark:text-gray-500" title={value}>{value}</div>
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
        // which has only the rows for the active page

        // The rest of these things are super handy, too ;)
        // canPreviousPage,
        // canNextPage,
        // pageOptions,
        // pageCount,
        // gotoPage,
        // nextPage,
        // previousPage,
        // setPageSize,

        // state,
        // preGlobalFilteredRows,
        // setGlobalFilter,
    } = useTable({
        columns,
        data,
    },
        useFilters, // useFilters!
        useGlobalFilter,
        useSortBy,
        usePagination,  // new
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
                                                                        <SortDownIcon className="w-4 h-4 text-gray-400" />
                                                                    ) : (
                                                                        <SortUpIcon className="w-4 h-4 text-gray-400" />
                                                                    )
                                                                ) : (
                                                                    <SortIcon className="w-4 h-4 text-gray-400 opacity-0 group-hover:opacity-100" />
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

function SortIcon({ className }: any) {
    return (
        <svg className={className} stroke="currentColor" fill="currentColor" strokeWidth="0" viewBox="0 0 320 512" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg"><path d="M41 288h238c21.4 0 32.1 25.9 17 41L177 448c-9.4 9.4-24.6 9.4-33.9 0L24 329c-15.1-15.1-4.4-41 17-41zm255-105L177 64c-9.4-9.4-24.6-9.4-33.9 0L24 183c-15.1 15.1-4.4 41 17 41h238c21.4 0 32.1-25.9 17-41z"></path></svg>
    )
}

function SortUpIcon({ className }: any) {
    return (
        <svg className={className} stroke="currentColor" fill="currentColor" strokeWidth="0" viewBox="0 0 320 512" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg"><path d="M279 224H41c-21.4 0-32.1-25.9-17-41L143 64c9.4-9.4 24.6-9.4 33.9 0l119 119c15.2 15.1 4.5 41-16.9 41z"></path></svg>
    )
}

function SortDownIcon({ className }: any) {
    return (
        <svg className={className} stroke="currentColor" fill="currentColor" strokeWidth="0" viewBox="0 0 320 512" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg"><path d="M41 288h238c21.4 0 32.1 25.9 17 41L177 448c-9.4 9.4-24.6 9.4-33.9 0L24 329c-15.1-15.1-4.4-41 17-41z"></path></svg>
    )
}

function DataTable() {
    const columns = React.useMemo(() => [
        {
            Header: "Age",
            accessor: 'timestamp',
            Cell: AgeCell,
        },
        {
            Header: "Release",
            accessor: 'torrent_name',
            Cell: ReleaseCell,
        },
        {
            Header: "Actions",
            accessor: 'action_status',
            Cell: ReleaseStatusCell,
        },
        {
            Header: "Indexer",
            accessor: 'indexer',
            Cell: IndexerCell,
            Filter: SelectColumnFilter,  // new
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
