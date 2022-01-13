import formatDistanceToNowStrict from 'date-fns/formatDistanceToNowStrict'
import React from 'react'
import App from '../App'
import { useTable, useFilters, useGlobalFilter, useSortBy, usePagination } from 'react-table'
import APIClient from '../api/APIClient'
import { useQuery } from 'react-query'
import { ReleaseFindResponse, ReleaseStats } from '../domain/interfaces'
import { EmptyListState } from '../components/emptystates'
import { ReleaseStatusCell } from './Releases'

export function Dashboard() {
  return (
    <main className="py-10 -mt-48">
      <div className="px-4 pb-8 mx-auto max-w-7xl sm:px-6 lg:px-8">
        <Stats />
        <DataTablee />
      </div>
    </main>
  )
}

const StatsItem = ({ name, stat }: any) => (
  <div
    className="relative px-4 pt-5 pb-2 overflow-hidden bg-white rounded-lg shadow dark:bg-gray-800 sm:pt-6 sm:px-6"
    title="All time"
  >
    <dt>
      <p className="pb-1 text-sm font-medium text-gray-500 truncate dark:text-gray-600">{name}</p>
    </dt>

    <dd className="flex items-baseline pb-6 sm:pb-7">
      <p className="text-2xl font-semibold text-gray-900 dark:text-gray-300">{stat}</p>
    </dd>
  </div>
)

function Stats() {
  const { isLoading, data } = useQuery<ReleaseStats, Error>('dash_release_staats', () => APIClient.release.stats(),
    {
      refetchOnWindowFocus: false
    }
  )

  if (isLoading) {
    return null
  }

  return (
    <div>
      <h3 className="text-lg font-medium leading-6 text-gray-900 dark:text-gray-600">Stats</h3>

      <dl className="grid grid-cols-1 gap-5 mt-5 sm:grid-cols-2 lg:grid-cols-3">
        <StatsItem name="Filtered Releases" stat={data?.filtered_count} />
        {/* <StatsItem name="Filter Rejected Releases" stat={data?.filter_rejected_count} /> */}
        <StatsItem name="Rejected Pushes" stat={data?.push_rejected_count} />
        <StatsItem name="Approved Pushes" stat={data?.push_approved_count} />
      </dl>
    </div>
  )
}

/* function RecentActivity() {
  let data: any[] = [
    {
      id: 1,
      status: "FILTERED",
      created_at: "2021-10-16 20:25:26",
      indexer: "tl",
      title: "That movie 2019 1080p x264-GROUP",
    },
    {
      id: 2,
      status: "PUSH_APPROVED",
      created_at: "2021-10-15 16:16:23",
      indexer: "tl",
      title: "That great movie 2009 1080p x264-1GROUP",
    },
    {
      id: 3,
      status: "FILTER_REJECTED",
      created_at: "2021-10-15 10:16:23",
      indexer: "tl",
      title: "Movie 1 2002 720p x264-1GROUP",
    },
    {
      id: 4,
      status: "PUSH_APPROVED",
      created_at: "2021-10-14 16:16:23",
      indexer: "tl",
      title: "That bad movie 2019 2160p x265-1GROUP",
    },
    {
      id: 5,
      status: "PUSH_REJECTED",
      created_at: "2021-10-13 16:16:23",
      indexer: "tl",
      title: "That really bad movie 20010 1080p x264-GROUP2",
    },
  ]

  return (
    <div className="flex flex-col mt-12">
      <h3 className="text-lg font-medium leading-6 text-gray-900 dark:text-gray-600">Recent activity</h3>

      <div className="mt-3 overflow-x-auto sm:-mx-6 lg:-mx-8">
        <div className="inline-block min-w-full py-2 sm:px-6 lg:px-8">
          <div className="overflow-hidden light:shadow light:border-b light:border-gray-200 sm:rounded-lg">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="light:bg-gray-50 dark:bg-gray-800">
                <tr>
                  <th
                    scope="col"
                    className="px-6 py-3 text-xs font-medium tracking-wider text-left text-gray-500 uppercase dark:text-gray-400"
                  >
                    Age
                  </th>
                  <th
                    scope="col"
                    className="px-6 py-3 text-xs font-medium tracking-wider text-left text-gray-500 uppercase dark:text-gray-400"
                  >
                    Release
                  </th>
                  <th
                    scope="col"
                    className="px-6 py-3 text-xs font-medium tracking-wider text-left text-gray-500 uppercase dark:text-gray-400"
                  >
                    Status
                  </th>
                  <th
                    scope="col"
                    className="px-6 py-3 text-xs font-medium tracking-wider text-left text-gray-500 uppercase dark:text-gray-400"
                  >
                    Indexer
                  </th>
                </tr>
              </thead>
              <tbody className="bg-gray-800 divide-y divide-gray-200 light:bg-white dark:divide-gray-700">
                {data && data.length > 0 ?
                  data.map((release: any, idx) => (
                    <ListItem key={idx} idx={idx} release={release} />
                  ))
                  : <span>No recent activity</span>}
              </tbody>
            </table>
            <nav
              className="flex items-center justify-between px-4 py-3 bg-white border-t border-gray-200 dark:bg-gray-800 dark:border-gray-700 sm:px-6"
              aria-label="Pagination"
            >
              <div className="hidden sm:block">
                <p className="text-sm text-gray-700 dark:text-gray-500">
                  Showing <span className="font-medium">1</span> to <span className="font-medium">10</span> of{' '}
                  <span className="font-medium">20</span> results
                </p>
              </div>
              <div className="flex items-center justify-between flex-1 sm:justify-end">
                <p className="relative text-sm text-gray-700 dark:text-gray-500">
                  Show <span className="font-medium">10</span>
                </p>
                <Menu as="div" className="relative text-left">
                  <Menu.Button className="flex items-center text-sm font-medium text-gray-900 rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-600">
                    <span>Show</span>
                    <ChevronDownIcon className="w-5 h-5 ml-1 text-gray-500" aria-hidden="true" />
                  </Menu.Button>

                  <Transition
                    as={Fragment}
                    enter="transition ease-out duration-100"
                    enterFrom="transform opacity-0 scale-95"
                    enterTo="transform opacity-100 scale-100"
                    leave="transition ease-in duration-75"
                    leaveFrom="transform opacity-100 scale-100"
                    leaveTo="transform opacity-0 scale-95"
                  >
                    <Menu.Items className="absolute right-0 z-30 w-40 mt-2 origin-top-right bg-white rounded-md shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none">
                      <div className="py-1">
                        {[5, 10, 25, 50].map((child) => (
                          <Menu.Item key={child}>
                            {({ active }) => (
                              <a
                                // href={child.href}
                                className={classNames(
                                  active ? 'bg-gray-100' : '',
                                  'block px-4 py-2 text-sm text-gray-700'
                                )}
                              >
                                {child}
                              </a>
                            )}
                          </Menu.Item>
                        ))}
                      </div>
                    </Menu.Items>
                  </Transition>
                </Menu>
                <a
                  href="#"
                  // className="px-4 py-2 mr-4 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md shadow-sm dark:bg-gray-700 dark:border-gray-600 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                  className="relative inline-flex items-center px-4 py-2 ml-5 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md dark:border-gray-600 dark:text-gray-400 dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600"
                >
                  Previous
                </a>
                <a
                  href="#"
                  // className="relative inline-flex items-center px-4 py-2 ml-3 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
                  className="relative inline-flex items-center px-4 py-2 ml-3 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md dark:border-gray-600 dark:text-gray-400 dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600"
                >
                  Next
                </a>

              </div>
            </nav>
          </div>
        </div>
      </div>


    </div>
  )
} */

/* const ListItem = ({ idx, release }: any) => {

  const formatDate = formatDistanceToNowStrict(
    new Date(release.created_at),
    { addSuffix: true }
  )

  return (
    <tr key={release.id} className={idx % 2 === 0 ? 'light:bg-white' : 'light:bg-gray-50'}>
      <td className="px-6 py-4 text-sm text-gray-500 whitespace-nowrap dark:text-gray-400" title={release.created_at}>{formatDate}</td>
      <td className="px-6 py-4 text-sm font-medium text-gray-900 whitespace-nowrap dark:text-gray-300">{release.title}</td>
      <td className="px-6 py-4 text-sm text-gray-500 whitespace-nowrap dark:text-gray-300">{statusMap[release.status]}</td>
      <td className="px-6 py-4 text-sm text-gray-500 whitespace-nowrap dark:text-gray-300">{release.indexer}</td>
    </tr>

  )
} */
/* 
const getData = () => {

  const data: any[] = [
    {
      id: 1,
      status: "FILTERED",
      created_at: "2021-10-16 20:25:26",
      indexer: "tl",
      title: "That movie 2019 1080p x264-GROUP",
    },
    {
      id: 2,
      status: "PUSH_APPROVED",
      created_at: "2021-10-15 16:16:23",
      indexer: "tl",
      title: "That great movie 2009 1080p x264-1GROUP",
    },
    {
      id: 3,
      status: "FILTER_REJECTED",
      created_at: "2021-10-15 10:16:23",
      indexer: "tl",
      title: "Movie 1 2002 720p x264-1GROUP",
    },
    {
      id: 4,
      status: "PUSH_APPROVED",
      created_at: "2021-10-14 16:16:23",
      indexer: "tl",
      title: "That bad movie 2019 2160p x265-1GROUP",
    },
    {
      id: 5,
      status: "PUSH_REJECTED",
      created_at: "2021-10-13 16:16:23",
      indexer: "tl",
      title: "That really bad movie 20010 1080p x264-GROUP2",
    },
  ]

  return [...data, ...data, ...data]
} */

// Define a default UI for filtering
/* function GlobalFilter({
  preGlobalFilteredRows,
  globalFilter,
  setGlobalFilter,
}: any) {
  const count = preGlobalFilteredRows.length
  const [value, setValue] = React.useState(globalFilter)
  const onChange = useAsyncDebounce((value: any) => {
    setGlobalFilter(value || undefined)
  }, 200)

  return (
    <label className="flex items-baseline gap-x-2">
      <span className="text-gray-700">Search: </span>
      <input
        type="text"
        className="border-gray-300 rounded-md shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"
        value={value || ""}
        onChange={e => {
          setValue(e.target.value);
          onChange(e.target.value);
        }}
        placeholder={`${count} records...`}
      />
    </label>
  )
} */

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

  return (
    statusMap[value]
  );
};

export function AgeCell({ value, column, row }: any) {

  const formatDate = formatDistanceToNowStrict(
    new Date(value),
    { addSuffix: true }
  )

  return (
    <div className="text-sm text-gray-500" title={value}>{formatDate}</div>
  )
}

export function ReleaseCell({ value, column, row }: any) {
  return (
    <div className="text-sm font-medium text-gray-900 dark:text-gray-300">{value}</div>
  )
}

export function IndexerCell({ value, column, row }: any) {
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
  )

  // Render the UI for your table
  return (
    <>
      <div className="sm:flex sm:gap-x-2">
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
      </div>
      {page.length > 0 ?
      <div className="flex flex-col mt-4">
        <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
          <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
            <div className="overflow-hidden bg-white shadow dark:bg-gray-800 sm:rounded-lg">
              <table {...getTableProps()} className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                <thead className="bg-gray-50 dark:bg-gray-800">
                  {headerGroups.map((headerGroup: { getHeaderGroupProps: () => JSX.IntrinsicAttributes & React.ClassAttributes<HTMLTableRowElement> & React.HTMLAttributes<HTMLTableRowElement>; headers: any[] }) => (
                    <tr {...headerGroup.getHeaderGroupProps()}>
                      {headerGroup.headers.map(column => (
                        // Add the sorting props to control sorting. For this example
                        // we can add them into the header props
                        <th
                          scope="col"
                          className="px-6 py-3 text-xs font-medium tracking-wider text-left text-gray-500 uppercase group"
                          {...column.getHeaderProps(column.getSortByToggleProps())}
                        >
                          <div className="flex items-center justify-between">
                            {column.render('Header')}
                            {/* Add a sort direction indicator */}
                            <span>
                              {column.isSorted
                                ? column.isSortedDesc
                                  ? <SortDownIcon className="w-4 h-4 text-gray-400" />
                                  : <SortUpIcon className="w-4 h-4 text-gray-400" />
                                : (
                                  <SortIcon className="w-4 h-4 text-gray-400 opacity-0 group-hover:opacity-100" />
                                )}
                            </span>
                          </div>
                        </th>
                      ))}
                    </tr>
                  ))}
                </thead>
                <tbody
                  {...getTableBodyProps()}
                  className="divide-y divide-gray-200 dark:divide-gray-700"
                >
                  {page.map((row: any, i: any) => {  // new
                    prepareRow(row)
                    return (
                      <tr {...row.getRowProps()}>
                        {row.cells.map((cell: any) => {
                          return (
                            <td
                              {...cell.getCellProps()}
                              className="px-6 py-4 whitespace-nowrap"
                              role="cell"
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


              {/* Pagination */}
              {/* <div className="flex items-center justify-between px-6 py-3 border-t border-gray-200 dark:border-gray-700">
        <div className="flex justify-between flex-1 sm:hidden">
          <Button onClick={() => previousPage()} disabled={!canPreviousPage}>Previous</Button>
          <Button onClick={() => nextPage()} disabled={!canNextPage}>Next</Button>
        </div>
        <div className="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
          <div className="flex items-baseline gap-x-2">
            <span className="text-sm text-gray-700">
              Page <span className="font-medium">{state.pageIndex + 1}</span> of <span className="font-medium">{pageOptions.length}</span>
            </span>
            <label>
              <span className="sr-only">Items Per Page</span>
              <select
                className="block w-full border-gray-300 rounded-md shadow-sm cursor-pointer dark:bg-gray-800 dark:border-gray-800 dark:text-gray-600 dark:hover:text-gray-500 focus:border-blue-300 focus:ring focus:ring-blue-200 focus:ring-opacity-50"
                value={state.pageSize}
                onChange={e => {
                  setPageSize(Number(e.target.value))
                }}
              >
                {[5, 10, 20].map(pageSize => (
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
                disabled={!canNextPage
                }>
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
      </div> */}


            </div>
          </div>
        </div>
      </div>
      : <EmptyListState text="No recent activity"/>}
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

/* function Button({ children, className, ...rest }: any) {
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
} */

function DataTablee() {

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

  // const data = React.useMemo(() => getData(), [])

  const { isLoading, data } = useQuery<ReleaseFindResponse, Error>('dash_release', () => APIClient.release.find("?limit=10"),
    {
      refetchOnWindowFocus: false
    }
  )

  if (isLoading) {
    return null
  }

  return (
    <div className="flex flex-col mt-12">
      <h3 className="text-lg font-medium leading-6 text-gray-900 dark:text-gray-600">Recent activity</h3>

      <Table columns={columns} data={data?.data} />
    </div>
  )
}

export default App;