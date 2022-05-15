import { useQuery } from "react-query";
import { APIClient } from "../../api/APIClient";
import { EmptySimple } from "../../components/emptystates";
import { useToggle } from "../../hooks/hooks";
import { NotificationAddForm, NotificationUpdateForm } from "../../forms/settings/NotifiactionForms";
import { Switch } from "@headlessui/react";
import { classNames } from "../../utils";

function NotificationSettings() {
    const [addNotificationsIsOpen, toggleAddNotifications] = useToggle(false);

    const { data } = useQuery<Notification[], Error>("notifications", APIClient.notifications.getAll,
        {
            refetchOnWindowFocus: false
        }
    );

    return (
        <div className="divide-y divide-gray-200 lg:col-span-9">
             <NotificationAddForm isOpen={addNotificationsIsOpen} toggle={toggleAddNotifications} />

            <div className="py-6 px-4 sm:p-6 lg:pb-8">
                <div className="-ml-4 -mt-4 flex justify-between items-center flex-wrap sm:flex-nowrap">
                    <div className="ml-4 mt-4">
                        <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">Notifications</h3>
                        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                            Send notifications on events.
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
                                 <div className="col-span-1 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Enabled</div>
                                <div className="col-span-2 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Name</div>
                                <div className="col-span-2 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Type</div>
                                <div className="col-span-4 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Events</div>
                            </li>

                            {data && data.map((n: Notification) => (
                                <ListItem key={n.id} notification={n} />
                            ))}
                        </ol>
                    </section>
                    : <EmptySimple title="No notifications setup" subtitle="Add a new notification" buttonText="New notification" buttonAction={toggleAddNotifications} />}
            </div>
        </div>
    );
}

interface ListItemProps {
    notification: Notification;
}

function ListItem({ notification }: ListItemProps) {
    const [updateFormIsOpen, toggleUpdateForm] = useToggle(false);

    return (
        <li key={notification.id} className="text-gray-500 dark:text-gray-400">
            <NotificationUpdateForm isOpen={updateFormIsOpen} toggle={toggleUpdateForm} notification={notification} />

            <div className="grid grid-cols-12 gap-4 items-center py-4">
                <div className="col-span-1 flex items-center sm:px-6 ">
                    <Switch
                        checked={notification.enabled}
                        onChange={toggleUpdateForm}
                        className={classNames(
                            notification.enabled ? "bg-teal-500 dark:bg-blue-500" : "bg-gray-200 dark:bg-gray-600",
                            "relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                        )}
                    >
                        <span className="sr-only">Use setting</span>
                        <span
                            aria-hidden="true"
                            className={classNames(
                                notification.enabled ? "translate-x-5" : "translate-x-0",
                                "inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200"
                            )}
                        />
                    </Switch>
                </div>
                <div className="col-span-2 flex items-center sm:px-6 ">
                    {notification.name}
                </div>
                <div className="col-span-2 flex items-center sm:px-6 ">
                    {notification.type}
                </div>
                <div className="col-span-5 flex items-center sm:px-6 ">
                    {notification.events.map((n, idx) => (
                        <span
                            key={idx}
                            className="mr-2 inline-flex items-center px-2.5 py-0.5 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400"
                        >
                            {n}
                        </span>
                    ))}
                </div>
                <div className="col-span-1 flex items-center sm:px-6 ">
                    <span className="text-indigo-600 dark:text-gray-300 hover:text-indigo-900 cursor-pointer" onClick={toggleUpdateForm}>
                        Edit
                    </span>
                </div>
            </div>
        </li>
    );
}

export default NotificationSettings;