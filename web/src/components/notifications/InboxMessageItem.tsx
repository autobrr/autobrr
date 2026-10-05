/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { CheckIcon } from "@heroicons/react/24/outline";
import { useTranslation } from "react-i18next";

import { classNames, IsEmptyDate, simplifyDate } from "@utils";

import { INBOX_EVENT_STYLES } from "./inboxEvents";

interface InboxMessageItemProps {
  message: InboxMessage;
  onMarkRead: (id: number) => void;
}

export const InboxMessageItem = ({ message, onMarkRead }: InboxMessageItemProps) => {
  const { t } = useTranslation(["common", "options"]);
  const unread = message.read_at === null;
  const style = INBOX_EVENT_STYLES[message.event] ?? INBOX_EVENT_STYLES.TEST;
  const Icon = style.icon;
  const title = message.event === "TEST"
    ? message.title
    : t(`options:event.${message.event}.label`, { defaultValue: message.title });

  return (
    <li className={classNames("flex gap-3 px-4 py-3", unread ? "bg-blue-50/60 dark:bg-blue-400/5" : "")}>
      <span className={classNames("flex size-9 shrink-0 items-center justify-center rounded-full", style.tile)}>
        <Icon className="h-5 w-5" aria-hidden="true" />
      </span>

      <div className="min-w-0 flex-1">
        <div className="flex items-start justify-between gap-3">
          <p className="flex min-w-0 items-center gap-2 pt-0.5">
            <span className={classNames("truncate text-sm text-gray-900 dark:text-gray-100", unread ? "font-semibold" : "font-medium")}>
              {title}
            </span>
            {unread && (
              <span className="h-2 w-2 shrink-0 rounded-full bg-blue-500">
                <span className="sr-only">{t("common:inbox.unread")}</span>
              </span>
            )}
          </p>

          <div className="flex shrink-0 items-center gap-1">
            <time
              className="pt-0.5 text-xs tabular-nums text-gray-500 dark:text-gray-400"
              dateTime={message.created_at}
              title={simplifyDate(message.created_at)}
            >
              {IsEmptyDate(message.created_at)}
            </time>
            <div className="flex w-7 items-center justify-end">
              {unread && (
                <button
                  type="button"
                  onClick={() => onMarkRead(message.id)}
                  title={t("common:inbox.markRead")}
                  className="rounded-md p-1 text-gray-500 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 hover:text-gray-900 dark:hover:text-white focus:outline-hidden focus-visible:ring-2 focus-visible:ring-blue-500 cursor-pointer"
                >
                  <span className="sr-only">{t("common:inbox.markRead")}</span>
                  <CheckIcon className="h-4 w-4" aria-hidden="true" />
                </button>
              )}
            </div>
          </div>
        </div>

        {message.message && (
          <p className="mt-0.5 line-clamp-2 whitespace-pre-line break-words text-sm text-gray-600 dark:text-gray-400">
            {message.message}
          </p>
        )}

        {message.release_name && (
          <p className="mt-0.5 truncate text-sm text-gray-700 dark:text-gray-300" title={message.release_name}>
            {message.release_name}
          </p>
        )}

        {message.rejections.length > 0 && (
          <p
            className={classNames(
              "mt-0.5 truncate text-xs",
              message.event === "PUSH_REJECTED" ? "text-amber-700 dark:text-amber-400" : "text-red-600 dark:text-red-400"
            )}
          >
            {message.rejections.join(", ")}
          </p>
        )}
      </div>
    </li>
  );
};
