import * as React from "react";
import { useQuery } from "react-query";
import { formatDistanceToNowStrict } from "date-fns";
import { useTable, useSortBy, usePagination } from "react-table";
import {
    ClockIcon,
    BanIcon,
    ExclamationCircleIcon
} from "@heroicons/react/outline";
import {
    ChevronDoubleLeftIcon,
    ChevronLeftIcon,
    ChevronRightIcon,
    ChevronDoubleRightIcon,
    CheckIcon
} from "@heroicons/react/solid";

import APIClient from "../api/APIClient";
import { EmptyListState } from "../components/emptystates";
import { classNames, simplifyDate } from "../utils";

export function Releases() {
    return (
        <main className="-mt-48">

            <header className="py-10">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex justify-between">
                    <h1 className="text-3xl font-bold text-white capitalize">Releases</h1>
                </div>
            </header>
            <div className="px-4 pb-8 mx-auto max-w-7xl sm:px-6 lg:px-8">
                <Table />
            </div>
        </main>
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

// export function StatusPill({ value }: any) {

//   const status = value ? value.toLowerCase() : "unknown";

//   return (
//     <span
//       className={
//         classNames(
//           "px-3 py-1 uppercase leading-wide font-bold text-xs rounded-full shadow-sm",
//           status.startsWith("active") ? "bg-green-100 text-green-800" : "",
//           status.startsWith("inactive") ? "bg-yellow-100 text-yellow-800" : "",
//           status.startsWith("offline") ? "bg-red-100 text-red-800" : "",
//         )
//       }
//     >
//       {status}
//     </span>
//   );
// };

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
        <div className="text-sm font-medium text-gray-900 dark:text-gray-300" title={value}>{value}</div>
    )
}

interface ReleaseStatusCellProps {
    value: ReleaseActionStatus[];
    column: any;
    row: any;
}

export function ReleaseStatusCell({ value }: ReleaseStatusCellProps) {
    const statusMap: any = {
        "PUSH_ERROR": <span className="mr-1 inline-flex items-center rounded text-xs font-semibold uppercase bg-pink-100 text-pink-800 hover:bg-pink-300 cursor-pointer">
             <ExclamationCircleIcon className="h-5 w-5" aria-hidden="true" />
        </span>,
        "PUSH_REJECTED": <span className="mr-1 inline-flex items-center rounded text-xs font-semibold uppercase bg-blue-200 dark:bg-blue-100 text-blue-400 dark:text-blue-800 hover:bg-blue-300 dark:hover:bg-blue-400 cursor-pointer">
             <BanIcon className="h-5 w-5" aria-hidden="true" />
        </span>,
        "PUSH_APPROVED": <span className="mr-1 inline-flex items-center rounded text-xs font-semibold uppercase bg-green-100 text-green-800 hover:bg-green-300 cursor-pointer">
             <CheckIcon className="h-5 w-5" aria-hidden="true" />
        </span>,
        "PENDING": <span className="mr-1 inline-flex items-center rounded text-xs font-semibold uppercase bg-yellow-100 text-yellow-800 hover:bg-yellow-200 cursor-pointer">
             <ClockIcon className="h-5 w-5" aria-hidden="true" />
        </span>,
    }
    return (
        <div className="flex text-sm font-medium text-gray-900 dark:text-gray-300">
            {value.map((v, idx) => <div key={idx} title={`action: ${v.action}, type: ${v.type}, status: ${v.status}, time: ${simplifyDate(v.timestamp)}, rejections: ${v?.rejections}`}>{statusMap[v.status]}</div>)}
        </div>
    )
}

export function IndexerCell({ value }: any) {
    return (
        <div className="text-sm font-medium text-gray-900 dark:text-gray-500" title={value}>{value}</div>
    )
}

const initialState = {
    queryPageIndex: 0,
    queryPageSize: 10,
    totalCount: null,
};

const PAGE_CHANGED = 'PAGE_CHANGED';
const PAGE_SIZE_CHANGED = 'PAGE_SIZE_CHANGED';
const TOTAL_COUNT_CHANGED = 'TOTAL_COUNT_CHANGED';

const reducer = (state: any, { type, payload }: any) => {
    switch (type) {
        case PAGE_CHANGED:
            return {
                ...state,
                queryPageIndex: payload,
            };
        case PAGE_SIZE_CHANGED:
            return {
                ...state,
                queryPageSize: payload,
            };
        case TOTAL_COUNT_CHANGED:
            return {
                ...state,
                totalCount: payload,
            };
        default:
            throw new Error(`Unhandled action type: ${type}`);
    }
};

function Table() {
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
        // {
        //   Header: "Filter Status",
        //   accessor: 'filter_status',
        //   Cell: StatusPill,
        // },
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

    const [{ queryPageIndex, queryPageSize, totalCount }, dispatch] =
        React.useReducer(reducer, initialState);

    const { isLoading, error, data, isSuccess } = useQuery(
        ['releases', queryPageIndex, queryPageSize],
        () => APIClient.release.find(`?offset=${queryPageIndex * queryPageSize}&limit=${queryPageSize}`),
        {
            keepPreviousData: true,
            staleTime: Infinity,
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

        state: { pageIndex, pageSize },
        // preGlobalFilteredRows,
        // setGlobalFilter,
    } = useTable({
        columns,
        data: isSuccess ? data.data : [],
        initialState: {
            pageIndex: queryPageIndex,
            pageSize: queryPageSize,
        },
        manualPagination: true,
        manualSortBy: true,
        pageCount: isSuccess ? Math.ceil(totalCount / queryPageSize) : 0,
    },
        // useFilters, // useFilters!
        // useGlobalFilter,
        useSortBy,
        usePagination,  // new
    )

    React.useEffect(() => {
        dispatch({ type: PAGE_CHANGED, payload: pageIndex });
    }, [pageIndex]);

    React.useEffect(() => {
        dispatch({ type: PAGE_SIZE_CHANGED, payload: pageSize });
        gotoPage(0);
    }, [pageSize, gotoPage]);

    React.useEffect(() => {
        if (data?.count) {
            dispatch({
                type: TOTAL_COUNT_CHANGED,
                payload: data.count,
            });
        }
    }, [data?.count]);

    if (error) {
        return <p>Error</p>;
    }

    if (isLoading) {
        return <p>Loading...</p>;
    }

    // Render the UI for your table
    return (
        <>
            {isSuccess && data ? (
                <div className="flex flex-col mt-4">
                    {/* <GlobalFilter
                        preGlobalFilteredRows={preGlobalFilteredRows}
                        globalFilter={state.globalFilter}
                        setGlobalFilter={setGlobalFilter}
                    /> */}
                    {/* {headerGroups.map((headerGroup: { headers: any[] }) =>
                        headerGroup.headers.map((column) =>
                        column.Filter ? (
                        <div className="mt-2 sm:mt-0" key={column.id}>
                            {column.render("Filter")}
                        </div>
                        ) : null
                    )
                    )} */}
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
                                        {page.map((row: any) => {  // new
                                            prepareRow(row)
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
                                        <Button onClick={() => previousPage()} disabled={!canPreviousPage}>Previous</Button>
                                        <Button onClick={() => nextPage()} disabled={!canNextPage}>Next</Button>
                                    </div>
                                    <div className="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
                                        <div className="flex items-baseline gap-x-2">
                                            <span className="text-sm text-gray-700">
                                                Page <span className="font-medium">{pageIndex + 1}</span> of <span className="font-medium">{pageOptions.length}</span>
                                            </span>
                                            <label>
                                                <span className="sr-only">Items Per Page</span>
                                                <select
                                                    className="block w-full border-gray-300 rounded-md shadow-sm cursor-pointer dark:bg-gray-800 dark:border-gray-800 dark:text-gray-600 dark:hover:text-gray-500 focus:border-blue-300 focus:ring focus:ring-blue-200 focus:ring-opacity-50"
                                                    value={pageSize}
                                                    onChange={e => {
                                                        setPageSize(Number(e.target.value))
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
                                            <nav className="relative z-0 inline-flex -space-x-px rounded-md shadow-sm" aria-label="Pagination">
                                                <PageButton
                                                    className="rounded-l-md"
                                                    onClick={() => gotoPage(0)}
                                                    disabled={!canPreviousPage}
                                                >
                                                    <span className="sr-only">First</span>
                                                    <ChevronDoubleLeftIcon className="w-5 h-5 text-gray-400" aria-hidden="true" />
                                                </PageButton>
                                                <PageButton
                                                    onClick={() => previousPage()}
                                                    disabled={!canPreviousPage}
                                                >
                                                    <span className="sr-only">Previous</span>
                                                    <ChevronLeftIcon className="w-5 h-5 text-gray-400" aria-hidden="true" />
                                                </PageButton>
                                                <PageButton
                                                    onClick={() => nextPage()}
                                                    disabled={!canNextPage}>
                                                    <span className="sr-only">Next</span>
                                                    <ChevronRightIcon className="w-5 h-5 text-gray-400" aria-hidden="true" />
                                                </PageButton>
                                                <PageButton
                                                    className="rounded-r-md"
                                                    onClick={() => gotoPage(pageCount - 1)}
                                                    disabled={!canNextPage}
                                                >
                                                    <span className="sr-only">Last</span>
                                                    <ChevronDoubleRightIcon className="w-5 h-5 text-gray-400" aria-hidden="true" />
                                                </PageButton>
                                            </nav>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            ) : <EmptyListState text="No recent activity" />}
        </>
    )
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

function Button({ children, className, ...rest }: any) {
    return (
        <button
            type="button"
            className={
                classNames(
                    "relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50",
                    className
                )}
            {...rest}
        >
            {children}
        </button>
    )
}

function PageButton({ children, className, ...rest }: any) {
    return (
        <button
            type="button"
            className={
                classNames(
                    "relative inline-flex items-center px-2 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-sm font-medium text-gray-500 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-600",
                    className
                )}
            {...rest}
        >
            {children}
        </button>
    )
}