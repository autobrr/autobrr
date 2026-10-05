/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { CheckIcon } from "@heroicons/react/24/outline";
import { useTranslation } from "react-i18next";

import { classNames } from "@utils";

import { InboxAge } from "./InboxAge";
import { InboxEventIcon } from "./InboxEventIcon";
import { inboxEventLabel, inboxHeadline, inboxRejectionClass } from "./inboxEvents";

interface InboxMessageItemProps {
  message: InboxMessage;
  onMarkRead: (id: number) => void;
}

export const InboxMessageItem = ({ message, onMarkRead }: InboxMessageItemProps) => {
  const { t } = useTranslation(["common", "options"]);
  const unread = message.read_at === null;
  const eventLabel = inboxEventLabel(message, t);
  const headline = inboxHeadline(message, eventLabel);

  return (
    <li className="flex items-center gap-3 px-4 py-2.5">
      <InboxEventIcon event={message.event} label={eventLabel} iconClassName="h-7 w-7" />

      <span className={classNames("self-start mt-1.5 -mr-1 h-2 w-2 shrink-0 rounded-full", unread ? "bg-blue-500" : "")}>
        {unread && <span className="sr-only">{t("common:inbox.unread")}</span>}
      </span>

      <div className="min-w-0 flex-1">
        <p
          className={classNames(
            "truncate text-sm",
            unread ? "font-semibold text-gray-900 dark:text-white" : "font-medium text-gray-700 dark:text-gray-300"
          )}
          title={headline}
        >
          {headline}
        </p>
        <p className="flex gap-x-3 min-w-0 text-xs">
          <InboxAge createdAt={message.created_at} className="shrink-0 text-gray-500 dark:text-gray-400" />
          {message.rejections.length > 0 && (
            <span
              className={classNames("truncate", inboxRejectionClass(message.event))}
              title={message.rejections.join("\n")}
            >
              {message.rejections.join(", ")}
            </span>
          )}
        </p>
      </div>

      <div className="flex w-7 shrink-0 justify-end">
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
    </li>
  );
};
