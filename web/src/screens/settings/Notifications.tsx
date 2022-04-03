import { useQuery } from "react-query";
import { APIClient } from "../../api/APIClient";
import { EmptySimple } from "../../components/emptystates";
import { useToggle } from "../../hooks/hooks";
import {NotificationAddForm} from "../../forms/settings/NotifiactionForms";


function NotificationSettings() {
    const [addNotificationsIsOpen, toggleAddNotifications] = useToggle(false)

    const { data } = useQuery<Notification[], Error>('notifications', APIClient.notifications.getAll,
        {
            refetchOnWindowFocus: false
        }
    )

    return (
        <div className="divide-y divide-gray-200 lg:col-span-9">
             <NotificationAddForm isOpen={addNotificationsIsOpen} toggle={toggleAddNotifications} />

            <div className="py-6 px-4 sm:p-6 lg:pb-8">
                <div className="-ml-4 -mt-4 flex justify-between items-center flex-wrap sm:flex-nowrap">
                    <div className="ml-4 mt-4">
                        <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">Notifications</h3>
                        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                            Notification settings
                        </p>
                    </div>
                    <div className="ml-4 mt-4 flex-shrink-0">
                        <button
                            type="button"
                            onClick={toggleAddNotifications}
                            className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 dark:bg-blue-600 hover:bg-indigo-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                        >
                            Add new
                        </button>
                    </div>
                </div>

                {data && data.length > 0 ?
                    <section className="mt-6 light:bg-white dark:bg-gray-800 light:shadow sm:rounded-md">
                        <ol className="min-w-full">
                            <li className="grid grid-cols-12 gap-4 border-b border-gray-200 dark:border-gray-700">
                                {/* <div className="col-span-1 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Enabled</div> */}
                                <div className="col-span-3 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Name</div>
                                <div className="col-span-4 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Type</div>
                                <div className="col-span-4 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Events</div>
                                {/* <div className="col-span-4 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Nick</div> */}
                            </li>

                            {data && data.map((notif: any, idx) => (
                                // <LiItem key={idx} idx={idx} network={network} />
                                <span key={idx}>{notif.name} {notif.type}</span>
                            ))}
                        </ol>
                    </section>
                    : <EmptySimple title="No networks" subtitle="Add a new network" buttonText="New network" buttonAction={toggleAddNotifications} />}

            </div>
        </div>
    )
}

export default NotificationSettings;