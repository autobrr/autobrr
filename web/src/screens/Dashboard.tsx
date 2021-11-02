import { CursorClickIcon, MailOpenIcon, UsersIcon } from '@heroicons/react/outline'
import formatDistanceToNowStrict from 'date-fns/formatDistanceToNowStrict'

const stats = [
    { id: 1, name: 'Total Releases', stat: '11897', icon: UsersIcon, change: '122', changeType: 'increase' },
    { id: 2, name: 'Filtered Releases', stat: '6770', icon: MailOpenIcon, change: '5.4%', changeType: 'increase' },
    { id: 3, name: 'Approved Pushes', stat: '4301', icon: CursorClickIcon, change: '3.2%', changeType: 'decrease' },
]

export function Dashboard() {
    return (
        <main className="-mt-48 py-10">
            {/* <header className="py-10">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <h1 className="text-3xl font-bold text-white capitalize">Dashboard</h1>
                </div>
            </header> */}
            <div className="max-w-7xl mx-auto pb-8 px-4 sm:px-6 lg:px-8">
                <Stats />
                {/* <div className="bg-white dark:bg-gray-800 rounded-lg shadow px-5 py-6 sm:px-6">
                    <div className="border-4 border-dashed border-gray-200 dark:border-gray-700 rounded-lg h-96" />
                </div> */}
                <RecentActivity />
            </div>
        </main>
    )
}

function Stats() {
    return (
        <div>
            <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-gray-600">Last 30 days</h3>

            <dl className="mt-5 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
                {stats.map((item) => (
                    <div
                        key={item.id}
                        className="relative bg-white dark:bg-gray-800 pt-5 px-4 pb-2 sm:pt-6 sm:px-6 shadow rounded-lg overflow-hidden"
                    >
                        <dt>
                            <p className="text-sm pb-1 font-medium text-gray-500 dark:text-gray-600 truncate">{item.name}</p>
                        </dt>

                        <dd className="pb-6 flex items-baseline sm:pb-7">
                            <p className="text-2xl font-semibold text-gray-900 dark:text-gray-300">{item.stat}</p>
                        </dd>
                    </div>
                ))}
            </dl>
        </div>
    )
}

const statusMap: any = {
    "FILTERED": <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold uppercase bg-blue-100 text-blue-800 ">Filtered</span>,
    "FILTER_REJECTED": <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold uppercase bg-red-100 text-red-800">Filter rejected</span>,
    "PUSH_REJECTED": <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold uppercase bg-pink-100 text-pink-800">Push rejected</span>,
    "PUSH_APPROVED": <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold uppercase bg-green-100 text-green-800">Push approved</span>,
}

function RecentActivity() {
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
            <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-gray-600">Recent activity</h3>

            <div className="mt-3 overflow-x-auto sm:-mx-6 lg:-mx-8">
                <div className="py-2 inline-block min-w-full sm:px-6 lg:px-8">
                    <div className="light:shadow overflow-hidden light:border-b light:border-gray-200 sm:rounded-lg">
                        <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                            <thead className="light:bg-gray-50 dark:bg-gray-800">
                                <tr>
                                    <th
                                        scope="col"
                                        className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
                                    >
                                        Age
                                    </th>
                                    <th
                                        scope="col"
                                        className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
                                    >
                                        Release
                                    </th>
                                    <th
                                        scope="col"
                                        className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
                                    >
                                        Status
                                    </th>
                                    <th
                                        scope="col"
                                        className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
                                    >
                                        Indexer
                                    </th>
                                </tr>
                            </thead>
                            <tbody className="light:bg-white bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                                {data && data.length > 0 ?
                                    data.map((release: any, idx) => (
                                        <ListItem key={idx} idx={idx} release={release} />
                                    ))
                                    : <span>No recent activity</span>}
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>


        </div>
    )
}

const ListItem = ({ idx, release }: any) => {

    const formatDate = formatDistanceToNowStrict(
        new Date(release.created_at),
        { addSuffix: true }
    )

    return (
        <tr key={release.id} className={idx % 2 === 0 ? 'light:bg-white' : 'light:bg-gray-50'}>
            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400" title={release.created_at}>{formatDate}</td>
            <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-gray-300">{release.title}</td>
            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">{statusMap[release.status]}</td>
            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">{release.indexer}</td>
        </tr>

    )
}