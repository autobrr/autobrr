/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Switch } from "@headlessui/react";

import { APIClient } from "@api/APIClient";
import { EmptySimple } from "@components/emptystates";
import { useToggle } from "@hooks/hooks";
import { NotificationAddForm, NotificationUpdateForm } from "@forms/settings/NotificationForms";
import { classNames } from "@utils";
import { componentMapType } from "@forms/settings/DownloadClientForms";
import Toast from "@components/notifications/Toast";
import toast from "react-hot-toast";

export const notificationKeys = {
  all: ["notifications"] as const,
  lists: () => [...notificationKeys.all, "list"] as const,
  details: () => [...notificationKeys.all, "detail"] as const,
  detail: (id: number) => [...notificationKeys.details(), id] as const
};

function NotificationSettings() {
  const [addNotificationsIsOpen, toggleAddNotifications] = useToggle(false);

  const { data } = useQuery({
    queryKey: notificationKeys.lists(),
    queryFn: APIClient.notifications.getAll,
    refetchOnWindowFocus: false
  });

  return (
    <div className="lg:col-span-9">
      <NotificationAddForm isOpen={addNotificationsIsOpen} toggle={toggleAddNotifications} />

      <div className="py-6 px-4 sm:p-6 lg:pb-8">
        <div className="-ml-4 -mt-4 flex justify-between items-center flex-wrap sm:flex-nowrap">
          <div className="ml-4 mt-4">
            <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">Notifications</h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Send notifications on events.</p>
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

        {data && data.length > 0 ? (
          <section className="mt-6 light:bg-white dark:bg-gray-800 light:shadow sm:rounded-md">
            <ol className="min-w-full">
              <li className="grid grid-cols-12 border-b border-gray-200 dark:border-gray-700">
                <div className="col-span-2 sm:col-span-1 pl-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Enabled
                </div>
                <div className="col-span-6 pl-10 md:pl-12 pr-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Name
                </div>
                <div className="hidden md:flex col-span-2 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Type
                </div>
                <div className="hidden md:flex col-span-3 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Events
                </div>
              </li>

              {data && data.map((n: Notification) => <ListItem key={n.id} notification={n} />)}
            </ol>
          </section>
        ) : (
          <EmptySimple title="No notifications" subtitle="" buttonText="Create new notification" buttonAction={toggleAddNotifications} />
        )}
      </div>
    </div>
  );
}

const DiscordIcon = () => (
  <svg viewBox="0 0 71 71" xmlns="http://www.w3.org/2000/svg" className="mr-2 h-4">
    <path
      d="M60.104 12.927a58.55 58.55 0 0 0-14.452-4.482.22.22 0 0 0-.232.11 40.783 40.783 0 0 0-1.8 3.696c-5.457-.817-10.886-.817-16.232 0-.484-1.164-1.2-2.586-1.827-3.696a.228.228 0 0 0-.233-.11 58.39 58.39 0 0 0-14.452 4.482.207.207 0 0 0-.095.082C1.577 26.759-.945 40.174.292 53.42a.244.244 0 0 0 .093.166c6.073 4.46 11.956 7.167 17.729 8.962a.23.23 0 0 0 .249-.082 42.08 42.08 0 0 0 3.627-5.9.225.225 0 0 0-.123-.312 38.772 38.772 0 0 1-5.539-2.64.228.228 0 0 1-.022-.377c.372-.28.744-.57 1.1-.862a.22.22 0 0 1 .23-.031c11.62 5.305 24.198 5.305 35.681 0a.219.219 0 0 1 .232.028c.356.293.728.586 1.103.865a.228.228 0 0 1-.02.377 36.384 36.384 0 0 1-5.54 2.637.227.227 0 0 0-.12.316 47.249 47.249 0 0 0 3.623 5.897.225.225 0 0 0 .25.084c5.8-1.795 11.683-4.502 17.756-8.962a.228.228 0 0 0 .093-.163c1.48-15.315-2.48-28.618-10.498-40.412a.18.18 0 0 0-.093-.085zM23.725 45.355c-3.498 0-6.38-3.212-6.38-7.156s2.826-7.156 6.38-7.156c3.582 0 6.437 3.24 6.38 7.156 0 3.944-2.826 7.156-6.38 7.156zm23.592 0c-3.498 0-6.38-3.212-6.38-7.156s2.826-7.156 6.38-7.156c3.582 0 6.437 3.24 6.38 7.156 0 3.944-2.798 7.156-6.38 7.156z"
      fill="currentColor"
    ></path>
  </svg>
);

const TelegramIcon = () => (
  <svg viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg" className="mr-2 h-4">
    <path
      d="M0 24c0 13.255 10.745 24 24 24s24-10.745 24-24S37.255 0 24 0 0 10.745 0 24zm19.6 11 .408-6.118 11.129-10.043c.488-.433-.107-.645-.755-.252l-13.735 8.665-5.933-1.851c-1.28-.393-1.29-1.273.288-1.906l23.118-8.914c1.056-.48 2.075.254 1.672 1.87l-3.937 18.553c-.275 1.318-1.072 1.633-2.175 1.024l-5.998-4.43L20.8 34.4l-.027.027c-.323.314-.59.573-1.173.573z"
      clipRule="evenodd"
      fill="currentColor"
      fillRule="evenodd"
    ></path>
  </svg>
);

const PushoverIcon = () => (
  <svg viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg" className="mr-2 h-4">
    <path
      d="m495.6 319.4 104-13.7-101.3 228.6c17.8-1.4 35.2-7.4 52.3-18.1 17.1-10.7 32.9-24.2 47.2-40.4 14.4-16.2 26.8-34.2 37.3-54.1 10.5-19.8 18-39.4 22.6-58.5 2.7-11.9 4-23.3 3.8-34.2-.2-10.9-3.1-20.5-8.6-28.7s-13.8-14.8-25-19.8-26.3-7.5-45.5-7.5c-22.4 0-44.4 3.6-66 10.9-21.7 7.3-41.7 17.9-60.2 31.8-18.5 13.9-34.5 31.2-48.2 52-13.7 20.8-23.5 44.4-29.4 70.8-2.3 8.7-3.6 15.6-4.1 20.9-.5 5.3-.6 9.6-.3 13 .2 3.4.7 6.1 1.4 7.9.7 1.8 1.3 3.6 1.7 5.5-23.3 0-40.3-4.7-51-14-10.7-9.3-13.3-25.7-7.9-48.9 5.5-24.2 17.9-47.2 37.3-69.1 19.4-21.9 42.4-41.2 69.1-57.8 26.7-16.6 55.9-29.9 87.6-39.7 31.7-9.8 62.6-14.7 92.7-14.7 26.5 0 48.7 3.8 66.7 11.3 18 7.5 32.1 17.5 42.1 29.8s16.3 26.7 18.8 43.1c2.5 16.4 1.7 33.5-2.4 51.3-5 21.4-14.5 43-28.4 64.7-13.9 21.7-31.4 41.3-52.3 58.8-21 17.6-45 31.8-72.2 42.8-27.1 10.9-56 16.4-86.6 16.4h-3.4l-86.9 195H302l193.6-435.4z"
      clipRule="evenodd"
      fill="currentColor"
      fillRule="evenodd"
    />
  </svg>
);

const EmailIcon = () => (
  <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" className="mr-2 h-4">
    <path
      d="M20.5 4H3.5C2.7 4 2 4.7 2 5.5v13C2 19.3 2.7 20 3.5 20h17c.8 0 1.5-.7 1.5-1.5v-13C22 4.7 21.3 4 20.5 4zM20 18H4V6l8 5 8-5v12z"
      fill="currentColor"
    />
  </svg>
);

const iconComponentMap: componentMapType = {
  DISCORD: (
    <span className="flex items-center px-2 py-0.5 rounded bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400">
      <DiscordIcon /> Discord
    </span>
  ),
  NOTIFIARR: (
    <span className="flex items-center px-2 py-0.5 rounded bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400">
      <DiscordIcon /> Notifiarr
    </span>
  ),
  TELEGRAM: (
    <span className="flex items-center px-2 py-0.5 rounded bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400">
      <TelegramIcon /> Telegram
    </span>
  ),
  PUSHOVER: (
    <span className="flex items-center px-2 py-0.5 rounded bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400">
      <PushoverIcon /> Pushover
    </span>
  ),
  EMAIL: (
    <span className="flex items-center px-2 py-0.5 rounded bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400">
      <EmailIcon /> Email
    </span>
  )
};

interface ListItemProps {
  notification: Notification;
}

function ListItem({ notification }: ListItemProps) {
  const [updateFormIsOpen, toggleUpdateForm] = useToggle(false);

  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (notification: Notification) => APIClient.notifications.update(notification).then(() => notification),
    onSuccess: (notification: Notification) => {
      toast.custom(t => (
        <Toast type="success" body={`${notification.name} was ${notification.enabled ? "enabled" : "disabled"} successfully.`} t={t} />
      ));
      queryClient.invalidateQueries({ queryKey: notificationKeys.lists() });
    }
  });

  const onToggleMutation = (newState: boolean) => {
    mutation.mutate({
      ...notification,
      enabled: newState
    });
  };

  return (
    <li key={notification.id} className="text-gray-500 dark:text-gray-400">
      <NotificationUpdateForm isOpen={updateFormIsOpen} toggle={toggleUpdateForm} notification={notification} />

      <div className="grid grid-cols-12 items-center py-4">
        <div className="col-span-2 sm:col-span-1 px-6 flex items-center ">
          <Switch
            checked={notification.enabled}
            onChange={onToggleMutation}
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
        <div
          className="col-span-8 md:col-span-6 pl-10 md:pl-12 pr-2 sm:pr-6 truncate block items-center text-sm font-medium text-gray-900 dark:text-white"
          title={notification.name}
        >
          {notification.name}
        </div>
        <div className="hidden md:flex col-span-2 items-center">{iconComponentMap[notification.type]}</div>
        <div className="hidden md:flex col-span-2 px-6 items-center sm:px-6">
          <span
            className="mr-2 inline-flex items-center px-2.5 py-1 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400"
            title={notification.events.join(", ")}
          >
            {notification.events.length}
          </span>
        </div>
        <div className="col-span-1 flex first-letter:px-6 whitespace-nowrap text-right text-sm font-medium">
          <span
            className="col-span-1 px-6 text-blue-600 dark:text-gray-300 hover:text-blue-900 dark:hover:text-blue-500 cursor-pointer"
            onClick={toggleUpdateForm}
          >
            Edit
          </span>
        </div>
      </div>
    </li>
  );
}

export default NotificationSettings;
