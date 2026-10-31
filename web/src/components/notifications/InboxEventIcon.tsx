/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { classNames } from "@utils";

import { INBOX_EVENT_STYLES } from "./inboxEvents";

interface InboxEventIconProps {
  event: InboxMessage["event"];
  label: string;
  iconClassName?: string;
  /** showLabel prints the event name beside the icon from the sm breakpoint up. */
  showLabel?: boolean;
}

export const InboxEventIcon = ({ event, label, iconClassName = "h-5 w-5", showLabel = false }: InboxEventIconProps) => {
  const style = INBOX_EVENT_STYLES[event] ?? INBOX_EVENT_STYLES.TEST;
  const Icon = style.icon;

  return (
    <span className="inline-flex items-center gap-2" title={showLabel ? undefined : label}>
      <Icon className={classNames("shrink-0", iconClassName, style.text)} aria-hidden="true" />
      <span
        className={showLabel
          ? "sr-only sm:not-sr-only sm:whitespace-nowrap text-sm text-gray-700 dark:text-gray-300"
          : "sr-only"}
      >
        {label}
      </span>
    </span>
  );
};
