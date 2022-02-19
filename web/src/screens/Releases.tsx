import * as React from "react";
import { useQuery } from "react-query";
import { formatDistanceToNowStrict } from "date-fns";
import { useTable, useSortBy, usePagination, useAsyncDebounce, useFilters, Column } from "react-table";
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
    CheckIcon,
    ChevronDownIcon,
} from "@heroicons/react/solid";

import { APIClient } from "../api/APIClient";
import { EmptyListState } from "../components/emptystates";
import { classNames, simplifyDate } from "../utils";

import { Fragment } from "react";
import { Listbox, Transition } from "@headlessui/react";
import { PushStatusOptions } from "../domain/constants";

export function Releases() {
    return (
        <main>
            <header className="py-10">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex justify-between">
                    <h1 className="text-3xl font-bold text-black dark:text-white capitalize">Releases</h1>
                </div>
            </header>
            <div className="px-4 pb-8 mx-auto max-w-7xl sm:px-6 lg:px-8">
                <Table />
            </div>
        </main>
    )
}

// // Define a default UI for filtering
// function GlobalFilter({
//   preGlobalFilteredRows,
//   globalFilter,
//   setGlobalFilter,
// }: any) {
//   const count = preGlobalFilteredRows.length
//   const [value, setValue] = React.useState(globalFilter)
//   const onChange = useAsyncDebounce(value => {
//     setGlobalFilter(value || undefined)
//   }, 200)

//   return (
//     <span>
//       Search:{' '}
//       <input
//         value={value || ""}
//         onChange={e => {
//           setValue(e.target.value);
//           onChange(e.target.value);
//         }}
//         placeholder={`${count} records...`}
//       />
//     </span>
//   )
// }

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
    
    const opts = ["PUSH_REJECTED"]

    // Render a multi-select box
    return (
        <div className="mb-6">

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
                {opts.map((option, i: number) => (
                    <option key={i} value={option}>
                        {option}
                    </option>
                ))}
            </select>
        </label>
        </div>
    )
}

// This is a custom filter UI for selecting
// a unique option from a list
export function IndexerSelectColumnFilter({
    column: { filterValue, setFilter, id },
}: any) {
    const { data, isSuccess } = useQuery(
        ['release_indexers'],
        () => APIClient.release.indexerOptions(),
        {
            keepPreviousData: true,
            staleTime: Infinity,
        }
    );

    const opts = isSuccess && data?.map(i => ({ value: i, label: i})) as any[]

    // Render a multi-select box
    return (
        <div className="mr-3">
    <div className="w-48">
      <Listbox 
      refName={id} 
      value={filterValue} 
      onChange={setFilter}
      >
        <div className="relative mt-1">
          <Listbox.Button className="relative w-full py-2 pl-3 pr-10 text-left bg-white dark:bg-gray-800 rounded-lg shadow-md cursor-default focus:outline-none focus-visible:ring-2 focus-visible:ring-opacity-75 focus-visible:ring-white focus-visible:ring-offset-orange-300 focus-visible:ring-offset-2 focus-visible:border-indigo-500 dark:text-gray-400 sm:text-sm">
            <span className="block truncate">{filterValue ? filterValue : "Indexer"}</span>
            <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
            <ChevronDownIcon
              className="w-5 h-5 ml-2 -mr-1 text-gray-600 hover:text-gray-600"
              aria-hidden="true"
            />
            </span>
          </Listbox.Button>
          <Transition
            as={Fragment}
            leave="transition ease-in duration-100"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <Listbox.Options className="absolute w-full py-1 mt-1 overflow-auto text-base bg-white dark:bg-gray-800 rounded-md shadow-lg max-h-60 ring-1 ring-black ring-opacity-5 focus:outline-none sm:text-sm">
                <Listbox.Option
                  key={0}
                  className={({ active }) =>
                    `cursor-default select-none relative py-2 pl-10 pr-4 ${
                      active ? 'text-gray-500 dark:text-gray-200 bg-gray-300 dark:bg-gray-900' : 'text-gray-900 dark:text-gray-400'
                    }`
                  }
                  value={undefined}
                >
                  {({ selected }) => (
                    <>
                      <span
                        className={`block truncate ${
                          selected ? 'font-medium' : 'font-normal'
                        }`}
                      >
                    All
                      </span>
                      {selected ? (
                        <span className="absolute inset-y-0 left-0 flex items-center pl-3 text-gray-500 dark:text-gray-400">
                          <CheckIcon className="w-5 h-5" aria-hidden="true" />
                        </span>
                      ) : null}
                    </>
                  )}
                </Listbox.Option>
              {isSuccess && data?.map((indexer, idx) => (
                <Listbox.Option
                  key={idx}
                  className={({ active }) =>
                    `cursor-default select-none relative py-2 pl-10 pr-4 ${
                      active ? 'text-gray-500 dark:text-gray-200 bg-gray-300 dark:bg-gray-900' : 'text-gray-900 dark:text-gray-400'
                    }`
                  }
                  value={indexer}
                >
                  {({ selected }) => (
                    <>
                      <span
                        className={`block truncate ${
                          selected ? 'font-medium' : 'font-normal'
                        }`}
                      >
                        {indexer}
                      </span>
                      {selected ? (
                        <span className="absolute inset-y-0 left-0 flex items-center pl-3 text-gray-500 dark:text-gray-400">
                          <CheckIcon className="w-5 h-5" aria-hidden="true" />
                        </span>
                      ) : null}
                    </>
                  )}
                </Listbox.Option>
              ))}
            </Listbox.Options>
          </Transition>
        </div>
      </Listbox>
    </div>
        </div>
    )
}

export function PushStatusSelectColumnFilter({
    column: { filterValue, setFilter, id },
}: any) {
    return (
        <div className="mr-3">

    <div className="w-48">
        <Listbox 
            refName={id} 
            value={filterValue} 
            onChange={setFilter}
        >
        <div className="relative mt-1">
          <Listbox.Button className="relative w-full py-2 pl-3 pr-10 text-left bg-white dark:bg-gray-800 rounded-lg shadow-md cursor-default focus:outline-none focus-visible:ring-2 focus-visible:ring-opacity-75 focus-visible:ring-white focus-visible:ring-offset-orange-300 focus-visible:ring-offset-2 focus-visible:border-indigo-500 dark:text-gray-400 sm:text-sm">
            <span className="block truncate">{filterValue ? PushStatusOptions.find((o) => o.value === filterValue && o.value)!.label : "Push status"}</span>
            <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
            <ChevronDownIcon
              className="w-5 h-5 ml-2 -mr-1 text-gray-600 hover:text-gray-600"
              aria-hidden="true"
            />
            </span>
          </Listbox.Button>
          <Transition
            as={Fragment}
            leave="transition ease-in duration-100"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <Listbox.Options className="absolute w-full py-1 mt-1 overflow-auto text-base bg-white dark:bg-gray-800 rounded-md shadow-lg max-h-60 ring-1 ring-black ring-opacity-5 focus:outline-none sm:text-sm">
                <Listbox.Option
                  key={0}
                  className={({ active }) =>
                    `cursor-default select-none relative py-2 pl-10 pr-4 ${
                      active ? 'text-gray-500 dark:text-gray-200 bg-gray-300 dark:bg-gray-900' : 'text-gray-900 dark:text-gray-400'
                    }`
                  }
                  value={undefined}
                >
                  {({ selected }) => (
                    <>
                      <span
                        className={`block truncate ${
                          selected ? 'font-medium' : 'font-normal'
                        }`}
                      >
                    All
                      </span>
                      {selected ? (
                        <span className="absolute inset-y-0 left-0 flex items-center pl-3 text-gray-500 dark:text-gray-400">
                          <CheckIcon className="w-5 h-5" aria-hidden="true" />
                        </span>
                      ) : null}
                    </>
                  )}
                </Listbox.Option>
              {PushStatusOptions.map((status, idx) => (
                <Listbox.Option
                  key={idx}
                  className={({ active }) =>
                    `cursor-default select-none relative py-2 pl-10 pr-4 ${
                      active ? 'text-gray-500 dark:text-gray-200 bg-gray-300 dark:bg-gray-900' : 'text-gray-900 dark:text-gray-400'
                    }`
                  }
                  value={status.value}
                >
                  {({ selected }) => (
                    <>
                      <span
                        className={`block truncate ${
                          selected ? 'font-medium' : 'font-normal'
                        }`}
                      >
                        {status.label}
                      </span>
                      {selected ? (
                        <span className="absolute inset-y-0 left-0 flex items-center pl-3 text-gray-500 dark:text-gray-400">
                          <CheckIcon className="w-5 h-5" aria-hidden="true" />
                        </span>
                      ) : null}
                    </>
                  )}
                </Listbox.Option>
              ))}
            </Listbox.Options>
          </Transition>
        </div>
      </Listbox>
    </div>
        </div>
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
    queryFilters: []
};

const PAGE_CHANGED = 'PAGE_CHANGED';
const PAGE_SIZE_CHANGED = 'PAGE_SIZE_CHANGED';
const TOTAL_COUNT_CHANGED = 'TOTAL_COUNT_CHANGED';
const FILTER_CHANGED = 'FILTER_CHANGED';

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
            
        case FILTER_CHANGED:
            return {
                ...state,
                queryFilters: payload,
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
            Filter: PushStatusSelectColumnFilter,  // new
        },
        {
            Header: "Indexer",
            accessor: 'indexer',
            Cell: IndexerCell,
            Filter: IndexerSelectColumnFilter,  // new
            filter: 'equal',
            // filter: 'includes',
        },
    ] as Column<Release>[], [])

    const [{ queryPageIndex, queryPageSize, totalCount, queryFilters }, dispatch] =
        React.useReducer(reducer, initialState);

    const { isLoading, error, data, isSuccess } = useQuery(
        ['releases', queryPageIndex, queryPageSize, queryFilters],
        // () => APIClient.release.find(`?offset=${queryPageIndex * queryPageSize}&limit=${queryPageSize}${filterIndexer && `&indexer=${filterIndexer}`}`),
        () => APIClient.release.findQuery(queryPageIndex * queryPageSize, queryPageSize, queryFilters),
        {
            keepPreviousData: true,
            staleTime: Infinity,
        }
    );

    // const initialFilters = React.useMemo(() => [
    //     {
    //         id: "indexer",
    //         value: "",
    //     }
    // ], [])

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

        state: { pageIndex, pageSize, globalFilter, filters },
        // preGlobalFilteredRows,
        // setGlobalFilter,
        // preFilteredRows,
    } = useTable({
        columns,
        data: data && isSuccess ? data.data : [],
        initialState: {
            pageIndex: queryPageIndex,
            pageSize: queryPageSize,
            filters: []
            // filters: initialFilters
        },
        manualPagination: true,
        manualFilters: true,
        manualSortBy: true,
        pageCount: isSuccess ? Math.ceil(totalCount / queryPageSize) : 0,
        autoResetSortBy: false,
        autoResetExpanded: false,
        autoResetPage: false
    },
        useFilters, // useFilters!
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

    React.useEffect(() => {
        dispatch({ type: FILTER_CHANGED, payload: filters });
    }, [filters]);


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
                <div className="flex flex-col">
                    {/* <GlobalFilter
                        preGlobalFilteredRows={preGlobalFilteredRows}
                        globalFilter={globalFilter}
                        setGlobalFilter={setGlobalFilter}
                        preFilteredRows={preFilteredRows}
                    /> */}
                    <div className="flex mb-6">

                    {headerGroups.map((headerGroup: { headers: any[] }) =>
                        headerGroup.headers.map((column) =>
                        column.Filter ? (
                        <div className="mt-2 sm:mt-0" key={column.id}>
                            {column.render("Filter")}
                        </div>
                        ) : null
                    )
                    )}
                    </div>

                    <div className="overflow-hidden bg-white shadow-lg dark:bg-gray-800 sm:rounded-lg">
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
                                        <span className="sr-only bg-gray-700">Items Per Page</span>
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
                                            <span className="sr-only text-gray-400 dark:text-gray-500 dark:bg-gray-700">First</span>
                                            <ChevronDoubleLeftIcon className="w-4 h-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
                                        </PageButton>
                                        <PageButton
                                            onClick={() => previousPage()}
                                            disabled={!canPreviousPage}
                                        >
                                            <span className="sr-only text-gray-400 dark:text-gray-500 dark:bg-gray-700">Previous</span>
                                            <ChevronLeftIcon className="w-4 h-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
                                        </PageButton>
                                        <PageButton
                                            onClick={() => nextPage()}
                                            disabled={!canNextPage}>
                                            <span className="sr-only text-gray-400 dark:text-gray-500 dark:bg-gray-700">Next</span>
                                            <ChevronRightIcon className="w-4 h-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
                                        </PageButton>
                                        <PageButton
                                            className="rounded-r-md"
                                            onClick={() => gotoPage(pageCount - 1)}
                                            disabled={!canNextPage}
                                        >
                                            <span className="sr-only text-gray-400 dark:text-gray-500 dark:bg-gray-700">Last</span>
                                            <ChevronDoubleRightIcon className="w-4 h-4 text-gray-400 dark:text-gray-500" aria-hidden="true" />
                                        </PageButton>
                                    </nav>
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
                    "relative inline-flex items-center px-4 py-2 border border-gray-300 dark:border-gray-800 text-sm font-medium rounded-md text-gray-700 dark:text-gray-500 bg-white dark:bg-gray-800 hover:bg-gray-50",
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
                    "relative inline-flex items-center px-2 py-2 border border-gray-300 dark:border-gray-700 text-sm font-medium text-gray-500 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-600",
                    className
                )}
            {...rest}
        >
            {children}
        </button>
    )
}