/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useRef } from "react";
import { useSuspenseQuery } from "@tanstack/react-query";
import { ChevronRightIcon, InformationCircleIcon } from "@heroicons/react/24/solid";
import { BellIcon } from "@heroicons/react/24/outline";

import { APIClient } from "@api/APIClient";
import { NotificationKeys } from "@api/query_keys";
import { Checkbox } from "@components/Checkbox";
import { TitleSubtitle } from "@components/headings";
import { EmptyListState } from "@components/emptystates";
import { DeleteModal } from "@components/modals";
import { useToggle } from "@hooks/hooks";
import { classNames } from "@utils";
import { FilterSection, FilterLayout, FilterPage } from "./_components";

import { useFormContext, useStore, ContextField } from "@app/lib/form";
import { Select } from "@components/inputs/tanstack";

const EVENT_OPTIONS = [
  { label: "Push Approved", value: "PUSH_APPROVED" },
  { label: "Push Rejected", value: "PUSH_REJECTED" },
  { label: "Push Error", value: "PUSH_ERROR" }
];

const NOTIFICATION_TYPE_MAP: Record<string, string> = {
  "DISCORD": "Discord",
  "NOTIFIARR": "Notifiarr",
  "TELEGRAM": "Telegram",
  "PUSHBULLET": "Pushbullet",
  "PUSHOVER": "Pushover",
  "GOTIFY": "Gotify",
  "NTFY": "Ntfy",
  "SHOUTRRR": "Shoutrrr",
  "WEBHOOK": "Webhook"
};

export function Notifications() {
  const form = useFormContext();

  const notifications = useStore(form.store, (s: any) => s.values.notifications) as FilterNotification[];

  // Fetch all available notifications
  const { data: availableNotifications = [] } = useSuspenseQuery({
    queryKey: NotificationKeys.lists(),
    queryFn: () => APIClient.notifications.getAll(),
    select: (data) => data.filter(n => n.enabled)
  });

  // Create a new notification object
  const createNewNotification = (): FilterNotification => {
    const firstAvailable = availableNotifications.find(
      n => !notifications?.some(sn => sn.notification_id === n.id)
    );

    return {
      notification_id: firstAvailable?.id || 0,
      notification: firstAvailable,
      events: ["PUSH_APPROVED"]
    };
  };

  const pushNotification = () => {
    (form as any).pushFieldValue("notifications", createNewNotification());
  };

  const removeNotification = (index: number) => {
    (form as any).removeFieldValue("notifications", index);
  };

  const availableToAdd = availableNotifications.filter(
    n => !notifications?.some((sn: FilterNotification) => sn.notification_id === n.id)
  );

  return (
    <div className="mt-5">
      <>
        <div className="-ml-4 -mt-4 mb-6 flex justify-between items-center flex-wrap sm:flex-nowrap">
          <TitleSubtitle
            className="ml-4 mt-4"
            title="Filter Notifications"
            subtitle="Configure which notifications should be sent for this filter. These override global notification settings."
          />
          <div className="ml-4 mt-4 shrink-0">
            {availableToAdd.length > 0 && (
              <button
                type="button"
                className="relative inline-flex items-center px-4 py-2 border border-transparent transition shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                onClick={pushNotification}
              >
                <BellIcon className="w-5 h-5 mr-1" aria-hidden="true" />
                Add notification
              </button>
            )}
          </div>
        </div>

        {notifications && notifications.length > 0 ? (
          <ul className="rounded-md">
            {notifications.map((notification: FilterNotification, index: number) => (
              <NotificationItem
                key={index}
                notification={notification}
                availableNotifications={availableNotifications}
                idx={index}
                remove={removeNotification}
                initialEdit={notifications.length === 1}
              />
            ))}
          </ul>
        ) : (
          <EmptyListState text="No filter-specific notifications configured. Global notifications will be used." />
        )}
      </>
    </div>
  );
}

interface NotificationItemProps {
  notification: FilterNotification;
  availableNotifications: ServiceNotification[];
  idx: number;
  initialEdit: boolean;
  remove: (index: number) => void;
}

function NotificationItem({ notification, availableNotifications, idx, initialEdit, remove }: NotificationItemProps) {
  const form = useFormContext();
  const cancelButtonRef = useRef(null);
  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);
  const [edit, toggleEdit] = useToggle(initialEdit);

  const notifications = useStore(form.store, (s: any) => s.values.notifications) as FilterNotification[];
  const currentNotificationId = notifications?.[idx]?.notification_id;
  const currentEvents = notifications?.[idx]?.events || [];

  const removeNotification = () => {
    remove(idx);
  };

  const handleEventToggle = (event: string, checked: boolean) => {
    const newEvents = checked
      ? [...currentEvents, event]
      : currentEvents.filter((e: string) => e !== event);
    (form as any).setFieldValue(`notifications[${idx}].events`, newEvents);
  };

  // Update notification object when ID changes
  useEffect(() => {
    if (currentNotificationId) {
      const notif = availableNotifications.find(n => n.id === currentNotificationId);
      if (notif) {
        (form as any).setFieldValue(`notifications[${idx}].notification`, notif);
      }
    }
  }, [currentNotificationId, availableNotifications, idx, form]);

  const selectedNotification = availableNotifications.find(
    n => n.id === notification.notification_id
  );

  const availableOptions = availableNotifications
    .filter(n => n.id === notification.notification_id ||
      !notifications?.some((sn: FilterNotification) => sn.notification_id === n.id))
    .map(n => ({ label: `${n.name} (${NOTIFICATION_TYPE_MAP[n.type] || n.type})`, value: n.id }));

  return (
    <li>
      <div
        className={classNames(
          idx % 2 === 0
            ? "bg-white dark:bg-gray-775"
            : "bg-gray-100 dark:bg-gray-815",
          "flex items-center transition px-2 sm:px-6 rounded-md my-1 border border-gray-150 dark:border-gray-750 hover:bg-gray-200 dark:hover:bg-gray-850"
        )}
      >
        <button className="px-4 py-4 w-full flex items-center" type="button" onClick={toggleEdit}>
          <div className="min-w-0 flex-1 sm:flex sm:items-center sm:justify-between">
            <div className="flex text-sm truncate">
              <p className="font-medium text-dark-600 dark:text-gray-100 truncate">
                {selectedNotification?.name || "Select notification"}
              </p>
            </div>
            <div className="shrink-0 sm:mt-0 sm:ml-5">
              <div className="flex overflow-hidden -space-x-1">
                <span className="text-sm font-normal text-gray-500 dark:text-gray-400">
                  {NOTIFICATION_TYPE_MAP[selectedNotification?.type || ""] || selectedNotification?.type}
                  {notification.events?.length === 0 ? " • Muted" : notification.events?.length > 0 ? ` • ${notification.events.length} event${notification.events.length > 1 ? 's' : ''}` : ""}
                </span>
              </div>
            </div>
          </div>
          <div className="ml-5 shrink-0">
            <ChevronRightIcon className="h-5 w-5 text-gray-400" aria-hidden="true" />
          </div>
        </button>
      </div>

      {edit && (
        <div className="flex items-center mt-1 px-3 sm:px-5 rounded-md border border-gray-150 dark:border-gray-750">
          <DeleteModal
            isOpen={deleteModalIsOpen}
            isLoading={false}
            buttonRef={cancelButtonRef}
            toggle={toggleDeleteModal}
            deleteAction={removeNotification}
            title="Remove notification"
            text="Are you sure you want to remove this notification? This action cannot be undone."
          />

          <FilterPage gap="sm:gap-y-6">
            <FilterSection
              title="Notification"
              subtitle="Select the notification service and events to trigger"
            >
              <FilterLayout>
                <div className="col-span-12">
                  <ContextField name={`notifications[${idx}].notification_id`}>
                    <Select
                      label="Notification service"
                      optionDefaultText="Select a notification"
                      options={availableOptions}
                      tooltip={<div><p>Select the notification service to use for this filter.</p></div>}
                    />
                  </ContextField>
                </div>

                <div className="col-span-12">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-4">
                    Notification Settings
                  </label>

                  {/* Mute Switch */}
                  <div className="mb-6 p-4 rounded-lg bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700">
                    <Checkbox
                      value={notification.events?.length === 0}
                      setValue={(muted) => {
                        if (muted) {
                          // Clear all events to mute
                          (form as any).setFieldValue(`notifications[${idx}].events`, []);
                        } else {
                          // Enable Push Approved by default when unmuting
                          (form as any).setFieldValue(`notifications[${idx}].events`, ["PUSH_APPROVED"]);
                        }
                      }}
                      label="Mute filter"
                      description="Disable all notifications for this filter. When muted, no notifications will be sent for releases matched by this filter."
                    />

                    {notification.events?.length === 0 && (
                      <div className="mt-3 flex items-start">
                        <InformationCircleIcon className="h-5 w-5 text-yellow-400 flex-shrink-0 mt-0.5" />
                        <p className="ml-2 text-sm text-yellow-700 dark:text-yellow-300">
                          Filter muted - overrides all global notification settings
                        </p>
                      </div>
                    )}
                  </div>

                  {/* Event Triggers - disabled when muted */}
                  <div className={notification.events?.length === 0 ? "opacity-50 pointer-events-none" : ""}>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Trigger events
                    </label>
                    <div className="space-y-3">
                      {EVENT_OPTIONS.map((event) => (
                        <Checkbox
                          key={event.value}
                          value={notification.events?.includes(event.value) || false}
                          setValue={(checked) => handleEventToggle(event.value, checked)}
                          label={event.label}
                          description={
                            event.value === "PUSH_APPROVED" ? "Send notification when release is successfully sent to client" :
                            event.value === "PUSH_REJECTED" ? "Send notification when release is rejected" :
                            "Send notification when an error occurs while processing"
                          }
                          disabled={notification.events?.length === 0}
                        />
                      ))}
                    </div>
                  </div>
                </div>
              </FilterLayout>
            </FilterSection>

            <div className="pt-6 pb-4 flex space-x-2 justify-between">
              <button
                type="button"
                className="inline-flex items-center justify-center px-4 py-2 rounded-md sm:text-sm bg-red-700 dark:bg-red-900 dark:hover:bg-red-700 hover:bg-red-800 text-white focus:outline-hidden"
                onClick={toggleDeleteModal}
              >
                Remove Notification
              </button>

              <button
                type="button"
                className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden"
                onClick={toggleEdit}
              >
                Close
              </button>
            </div>
          </FilterPage>
        </div>
      )}
    </li>
  );
}
