/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { CloseButton, Popover, PopoverButton, PopoverPanel } from "@headlessui/react";
import { BellIcon } from "@heroicons/react/24/outline";
import { useTranslation } from "react-i18next";

import { APIClient } from "@api/APIClient";
import { NotificationKeys } from "@api/query_keys";
import { NotificationInboxQueryOptions } from "@api/queries";
import { ConfirmModal } from "@components/modals";
import { InboxMessageItem } from "@components/notifications/InboxMessageItem";
import { classNames } from "@utils";

const MENU_QUERY: InboxQueryParams = { limit: 10, offset: 0, unread: false };

export const InboxMenu = () => {
  const { t } = useTranslation("common");
  const queryClient = useQueryClient();

  const { data } = useQuery(NotificationInboxQueryOptions(MENU_QUERY));
  const unreadCount = data?.unread_count ?? 0;

  const markReadMutation = useMutation({
    mutationFn: (id: number) => APIClient.notifications.inbox.markRead([id]),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NotificationKeys.inbox.all() });
    }
  });

  const markAllReadMutation = useMutation({
    mutationFn: () => APIClient.notifications.inbox.markRead([]),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NotificationKeys.inbox.all() });
    }
  });

  const cancelModalButtonRef = useRef<HTMLInputElement | null>(null);
  const [pending, setPending] = useState<number | "all" | null>(null);
  const [confirmIsOpen, setConfirmIsOpen] = useState(false);

  const confirmMarkRead = (target: number | "all") => {
    setPending(target);
    setConfirmIsOpen(true);
  };

  return (
    <Popover className="relative">
      {pending !== null && (
        <ConfirmModal
          isOpen={confirmIsOpen}
          isLoading={false}
          toggle={() => setConfirmIsOpen(false)}
          buttonRef={cancelModalButtonRef}
          onConfirm={() => pending === "all" ? markAllReadMutation.mutate() : markReadMutation.mutate(pending)}
          title={pending === "all" ? t("inbox.markAllRead") : t("inbox.markRead")}
          text={pending === "all" ? t("inbox.confirmMarkAllRead") : t("inbox.confirmMarkReadOne")}
          confirmLabel={pending === "all" ? t("inbox.markAllRead") : t("inbox.markRead")}
          tone="primary"
        />
      )}

      <PopoverButton
        className="relative p-1.5 rounded-full text-gray-600 dark:text-gray-500 hover:text-gray-900 dark:hover:text-white hover:bg-gray-200 dark:hover:bg-gray-800 focus:outline-hidden transition duration-100 cursor-pointer"
        title={t("inbox.title")}
      >
        <span className="sr-only">{t("inbox.title")}</span>
        <BellIcon className="h-5 w-5" aria-hidden="true" />
        {unreadCount > 0 && (
          <span
            className="absolute -top-0.5 -right-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-semibold leading-none text-white"
            aria-label={t("inbox.unreadCount", { unread: unreadCount })}
          >
            {unreadCount > 99 ? "99+" : unreadCount}
          </span>
        )}
      </PopoverButton>

      <PopoverPanel
        anchor={{ to: "bottom end", gap: 8 }}
        className="z-20 w-96 max-w-[calc(100vw-2rem)] rounded-md border border-gray-250 dark:border-gray-775 bg-white dark:bg-gray-825 shadow-lg focus:outline-hidden"
      >
        {({ close }) => (
          <>
            <div className="flex items-center justify-between border-b border-gray-200 dark:border-gray-750 px-4 py-2">
              <h3 className="text-sm font-medium text-gray-900 dark:text-white">{t("inbox.title")}</h3>
              <button
                type="button"
                onClick={() => {
                  close();
                  confirmMarkRead("all");
                }}
                disabled={unreadCount === 0 || markAllReadMutation.isPending}
                className="text-xs font-medium text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 disabled:text-gray-400 dark:disabled:text-gray-600 cursor-pointer disabled:cursor-not-allowed"
              >
                {t("inbox.markAllRead")}
              </button>
            </div>

            {data && data.data.length > 0 ? (
              <ul className="max-h-96 overflow-y-auto divide-y divide-gray-100 dark:divide-gray-750">
                {data.data.map((message) => (
                  <InboxMessageItem
                    key={message.id}
                    message={message}
                    onMarkRead={(id) => {
                      close();
                      confirmMarkRead(id);
                    }}
                  />
                ))}
              </ul>
            ) : (
              <p className="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400">{t("inbox.empty")}</p>
            )}

            <CloseButton
              as={Link}
              to="/notifications"
              className={classNames(
                "block rounded-b-md border-t border-gray-200 dark:border-gray-750 px-4 py-2 text-center text-sm font-medium",
                "text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-750 cursor-pointer"
              )}
            >
              {t("inbox.viewAll")}
            </CloseButton>
          </>
        )}
      </PopoverPanel>
    </Popover>
  );
};
