/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { PlusIcon, InformationCircleIcon } from "@heroicons/react/24/solid";

import { APIClient } from "@api/APIClient";
import { NotificationKeys } from "@api/query_keys";
import { NotificationsQueryOptions } from "@api/queries";
import { EmptySimple } from "@components/emptystates";
import { useToggle } from "@hooks/hooks";
import { NotificationAddForm, NotificationUpdateForm } from "@forms/settings/NotificationForms";
import { componentMapType } from "@forms/settings/DownloadClientForms";
import toast from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import {
  DiscordIcon,
  GotifyIcon,
  LunaSeaIcon,
  NotifiarrIcon,
  NtfyIcon,
  PushoverIcon,
  Section,
  TelegramIcon
} from "./_components";
import { Checkbox } from "@components/Checkbox";

function NotificationSettings() {
  const [addNotificationsIsOpen, toggleAddNotifications] = useToggle(false);

  const notificationsQuery = useSuspenseQuery(NotificationsQueryOptions())

  return (
    <Section
      title="Notifications"
      description="Configure notification services and their global event triggers."
      rightSide={
        <button
          type="button"
          onClick={toggleAddNotifications}
          className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        >
          <PlusIcon className="h-5 w-5 mr-1" />
          Add new
        </button>
      }
    >
      <NotificationAddForm isOpen={addNotificationsIsOpen} toggle={toggleAddNotifications} />

      <div className="mb-4 rounded-md bg-blue-50 dark:bg-blue-900/20 p-4">
        <div className="flex">
          <div className="flex-shrink-0">
            <InformationCircleIcon className="h-5 w-5 text-blue-400 dark:text-blue-300" aria-hidden="true" />
          </div>
          <div className="ml-3">
            <p className="text-sm text-blue-800 dark:text-blue-200">
              <strong>Per-filter overrides:</strong> You can override these global settings for specific filters in each filter's Notifications tab. Global events are optional - you can create notification services without any events and configure them only at the filter level. System events (IRC disconnected/reconnected, app updates) always use global settings.
            </p>
          </div>
        </div>
      </div>

      {notificationsQuery.data && notificationsQuery.data.length > 0 ? (
        <ul className="min-w-full">
          <li className="grid grid-cols-12 border-b border-gray-200 dark:border-gray-700">
            <div className="col-span-2 sm:col-span-1 pl-1 sm:pl-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Enabled</div>
            <div className="col-span-9 md:col-span-5 pl-10 sm:pl-12 pr-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Name</div>
            <div className="hidden md:flex col-span-2 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Type</div>
            <div className="hidden md:flex col-span-1 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              <span className="mr-1">Events</span>
            </div>
            <div className="hidden md:flex col-span-1 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              <span className="mr-1">Filters</span>
            </div>
          </li>

          {notificationsQuery.data.map((n) => <ListItem key={n.id} notification={n} />)}
        </ul>
      ) : (
        <EmptySimple title="No notifications" subtitle="" buttonText="Create new notification" buttonAction={toggleAddNotifications} />
      )}
    </Section>
  );
}

const iconStyle = "flex items-center px-2 py-0.5 rounded-sm bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400";
const iconComponentMap: componentMapType = {
  DISCORD: <span className={iconStyle}><DiscordIcon /> Discord</span>,
  NOTIFIARR: <span className={iconStyle}><NotifiarrIcon /> Notifiarr</span>,
  TELEGRAM: <span className={iconStyle}><TelegramIcon /> Telegram</span>,
  PUSHOVER: <span className={iconStyle}><PushoverIcon /> Pushover</span>,
  GOTIFY: <span className={iconStyle}><GotifyIcon /> Gotify</span>,
  NTFY: <span className={iconStyle}><NtfyIcon /> ntfy</span>,
  SHOUTRRR: <span className={iconStyle}><NtfyIcon /> Shoutrrr</span>,
  LUNASEA: <span className={iconStyle}><LunaSeaIcon /> LunaSea</span>
};

interface ListItemProps {
  notification: ServiceNotification;
}

function ListItem({ notification }: ListItemProps) {
  const [updateFormIsOpen, toggleUpdateForm] = useToggle(false);

  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (notification: ServiceNotification) => APIClient.notifications.update(notification).then(() => notification),
    onSuccess: (notification: ServiceNotification) => {
      toast.custom(t => <Toast type="success" body={`${notification.name} was ${notification.enabled ? "enabled" : "disabled"} successfully.`} t={t} />);
      queryClient.invalidateQueries({ queryKey: NotificationKeys.lists() });
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
      <NotificationUpdateForm isOpen={updateFormIsOpen} toggle={toggleUpdateForm} data={notification} />

      <div className="grid grid-cols-12 items-center py-2">
        <div className="col-span-2 sm:col-span-1 pl-1 py-0.5 sm:pl-6 flex items-center">
          <Checkbox
            value={notification.enabled}
            setValue={onToggleMutation}
          />
        </div>
        <div className="col-span-9 md:col-span-5 pl-10 sm:pl-12 pr-2 sm:pr-6 truncate block items-center text-sm font-medium text-gray-900 dark:text-white" title={notification.name}>
          {notification.name}
        </div>
        <div className="hidden md:flex col-span-2 items-center">
          {iconComponentMap[notification.type]}
        </div>
        <div className="hidden md:flex col-span-1 px-6 items-center sm:px-6">
          <span
            className="mr-2 inline-flex items-center px-2.5 py-1 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400"
            title={notification.events.join(", ")}
          >
            {notification.events.length}
          </span>
        </div>
        <div className="hidden md:flex col-span-2 px-6 items-center sm:px-6">
          <span
            className="mr-2 inline-flex items-center px-2.5 py-1 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400"
            title={notification.used_by_filters?.join(", ")}
          >
            {notification.used_by_filters?.length || 0}
          </span>
        </div>
        <div className="col-span-1 flex first-letter:px-6 whitespace-nowrap text-right text-sm font-medium">
          <span
            className="col-span-1 px-0 sm:px-6 text-blue-600 dark:text-gray-300 hover:text-blue-900 dark:hover:text-blue-500 cursor-pointer"
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
