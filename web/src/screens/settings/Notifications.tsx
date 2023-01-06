import { useQuery } from "react-query";
import { APIClient } from "../../api/APIClient";
import { EmptySimple } from "../../components/emptystates";
import { useToggle } from "../../hooks/hooks";
import { NotificationAddForm, NotificationUpdateForm } from "../../forms/settings/NotificationForms";
import { Switch } from "@headlessui/react";
import { classNames } from "../../utils";
import { componentMapType } from "../../forms/settings/DownloadClientForms";

function NotificationSettings() {
  const [addNotificationsIsOpen, toggleAddNotifications] = useToggle(false);

  const { data } = useQuery(
    "notifications",
    () => APIClient.notifications.getAll(),
    { refetchOnWindowFocus: false }
  );

  return (
    <div className="lg:col-span-9">
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
              className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              Add new
            </button>
          </div>
        </div>

        {data && data.length > 0 ?
          <section className="mt-6 light:bg-white dark:bg-gray-800 light:shadow sm:rounded-md">
            <ol className="min-w-full">
              <li className="grid grid-cols-12 gap-4 border-b border-gray-200 dark:border-gray-700">
                <div className="col-span-3 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Enabled</div>
                <div className="col-span-6 md:col-span-3 lg:col-span-3 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Name</div>
                <div className="hidden md:flex col-span-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Type</div>
                <div className="hidden md:flex col-span-3 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Events</div>
              </li>

              {data && data.map((n: Notification) => (
                <ListItem key={n.id} notification={n} />
              ))}
            </ol>
          </section>
          : <EmptySimple title="No notifications" subtitle="" buttonText="Create new notification" buttonAction={toggleAddNotifications} />}
      </div>
    </div>
  );
}


const DiscordIcon = () => (
  <svg viewBox="0 0 71 71" xmlns="http://www.w3.org/2000/svg" className="mr-2 h-4">
    <path
      d="M60.104 12.927a58.55 58.55 0 0 0-14.452-4.482.22.22 0 0 0-.232.11 40.783 40.783 0 0 0-1.8 3.696c-5.457-.817-10.886-.817-16.232 0-.484-1.164-1.2-2.586-1.827-3.696a.228.228 0 0 0-.233-.11 58.39 58.39 0 0 0-14.452 4.482.207.207 0 0 0-.095.082C1.577 26.759-.945 40.174.292 53.42a.244.244 0 0 0 .093.166c6.073 4.46 11.956 7.167 17.729 8.962a.23.23 0 0 0 .249-.082 42.08 42.08 0 0 0 3.627-5.9.225.225 0 0 0-.123-.312 38.772 38.772 0 0 1-5.539-2.64.228.228 0 0 1-.022-.377c.372-.28.744-.57 1.1-.862a.22.22 0 0 1 .23-.031c11.62 5.305 24.198 5.305 35.681 0a.219.219 0 0 1 .232.028c.356.293.728.586 1.103.865a.228.228 0 0 1-.02.377 36.384 36.384 0 0 1-5.54 2.637.227.227 0 0 0-.12.316 47.249 47.249 0 0 0 3.623 5.897.225.225 0 0 0 .25.084c5.8-1.795 11.683-4.502 17.756-8.962a.228.228 0 0 0 .093-.163c1.48-15.315-2.48-28.618-10.498-40.412a.18.18 0 0 0-.093-.085zM23.725 45.355c-3.498 0-6.38-3.212-6.38-7.156s2.826-7.156 6.38-7.156c3.582 0 6.437 3.24 6.38 7.156 0 3.944-2.826 7.156-6.38 7.156zm23.592 0c-3.498 0-6.38-3.212-6.38-7.156s2.826-7.156 6.38-7.156c3.582 0 6.437 3.24 6.38 7.156 0 3.944-2.798 7.156-6.38 7.156z"
      fill="currentColor"></path>
  </svg>
);

const TelegramIcon = () => (
  <svg viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg" className="mr-2 h-4">
    <path
      d="M0 24c0 13.255 10.745 24 24 24s24-10.745 24-24S37.255 0 24 0 0 10.745 0 24zm19.6 11 .408-6.118 11.129-10.043c.488-.433-.107-.645-.755-.252l-13.735 8.665-5.933-1.851c-1.28-.393-1.29-1.273.288-1.906l23.118-8.914c1.056-.48 2.075.254 1.672 1.87l-3.937 18.553c-.275 1.318-1.072 1.633-2.175 1.024l-5.998-4.43L20.8 34.4l-.027.027c-.323.314-.59.573-1.173.573z"
      clipRule="evenodd" fill="currentColor" fillRule="evenodd"></path>
  </svg>
);


const iconComponentMap: componentMapType = {
  DISCORD: <span className="flex items-center px-2 py-0.5 rounded bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400"><DiscordIcon /> Discord</span>,
  NOTIFIARR: <span className="flex items-center px-2 py-0.5 rounded bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400"><DiscordIcon /> Notifiarr</span>,
  TELEGRAM: <span className="flex items-center px-2 py-0.5 rounded bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400"><TelegramIcon /> Telegram</span>
};

interface ListItemProps {
    notification: Notification;
}

function ListItem({ notification }: ListItemProps) {
  const [updateFormIsOpen, toggleUpdateForm] = useToggle(false);

  return (
    <li key={notification.id} className="text-gray-500 dark:text-gray-400">
      <NotificationUpdateForm isOpen={updateFormIsOpen} toggle={toggleUpdateForm} notification={notification} />

      <div className="grid grid-cols-12 gap-4 items-center py-3">
        <div className="col-span-3 md:col-span-3 lg:col-span-3 px-6 flex items-center sm:px-6">
          <Switch
            checked={notification.enabled}
            onChange={toggleUpdateForm}
            className={classNames(
              notification.enabled ? "bg-blue-500" : "bg-gray-200 dark:bg-gray-600",
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
        <div className="col-span-6 md:col-span-3 lg:col-span-3 px-6 overflow-hidden flex items-center sm:px-6">
          {notification.name}
        </div>
        <div className="hidden md:flex col-span-3 flex items-center">
          {iconComponentMap[notification.type]}
        </div>
        <div className="hidden md:flex col-span-2 px-6 flex items-center sm:px-6">
          <span
            className="mr-2 px-6 inline-flex items-center px-2.5 py-1 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400"
            title={notification.events.join(", ")}
          >
            {notification.events.length}
          </span>
        </div>
        <div className="col-span-1 flex items-center">
          <span className="text-blue-600 dark:text-gray-300 hover:text-blue-900 cursor-pointer" onClick={toggleUpdateForm}>
            Edit
          </span>
        </div>
      </div>
    </li>
  );
}

export default NotificationSettings;