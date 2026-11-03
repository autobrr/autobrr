/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { ComponentType, SVGProps } from "react";
import type { TFunction } from "i18next";
import {
  ArrowPathIcon,
  ArrowUpCircleIcon,
  BeakerIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  InboxArrowDownIcon,
  NoSymbolIcon,
  RssIcon,
  SignalIcon,
  SignalSlashIcon
} from "@heroicons/react/24/outline";

interface InboxEventStyle {
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  text: string;
}

const RED = {
  text: "text-red-600 dark:text-red-400"
};

const GREEN = {
  text: "text-green-600 dark:text-green-400"
};

const AMBER = {
  text: "text-amber-600 dark:text-amber-400"
};

const BLUE = {
  text: "text-blue-600 dark:text-blue-400"
};

const GRAY = {
  text: "text-gray-500 dark:text-gray-400"
};

export const INBOX_EVENT_STYLES: Record<InboxMessage["event"], InboxEventStyle> = {
  PUSH_APPROVED: { icon: CheckCircleIcon, ...GREEN },
  PUSH_REJECTED: { icon: NoSymbolIcon, ...AMBER },
  PUSH_ERROR: { icon: ExclamationTriangleIcon, ...RED },
  IRC_DISCONNECTED: { icon: SignalSlashIcon, ...RED },
  IRC_RECONNECTED: { icon: SignalIcon, ...GREEN },
  IRC_UNHEALTHY: { icon: ExclamationTriangleIcon, ...AMBER },
  IRC_HEALTHY: { icon: SignalIcon, ...GREEN },
  LIST_REFRESH_SUCCESS: { icon: ArrowPathIcon, ...GREEN },
  LIST_REFRESH_ERROR: { icon: ArrowPathIcon, ...RED },
  FEED_REFRESH_SUCCESS: { icon: RssIcon, ...GREEN },
  FEED_REFRESH_ERROR: { icon: RssIcon, ...RED },
  APP_UPDATE_AVAILABLE: { icon: ArrowUpCircleIcon, ...BLUE },
  RELEASE_NEW: { icon: InboxArrowDownIcon, ...GRAY },
  TEST: { icon: BeakerIcon, ...GRAY }
};

// The built-in notification can not subscribe to RELEASE_NEW, so the inbox never holds one.
export type InboxFilterEvent = Exclude<NotificationEvent, "RELEASE_NEW">;

export const INBOX_FILTER_EVENTS: InboxFilterEvent[] = [
  "PUSH_ERROR",
  "PUSH_REJECTED",
  "PUSH_APPROVED",
  "IRC_DISCONNECTED",
  "IRC_RECONNECTED",
  "IRC_UNHEALTHY",
  "IRC_HEALTHY",
  "LIST_REFRESH_ERROR",
  "LIST_REFRESH_SUCCESS",
  "FEED_REFRESH_ERROR",
  "FEED_REFRESH_SUCCESS",
  "APP_UPDATE_AVAILABLE"
];

/** inboxEventLabel returns the translated event name, or the stored title for test messages. */
export const inboxEventLabel = (message: InboxMessage, t: TFunction) => message.event === "TEST"
  ? message.title
  : t(`options:event.${message.event}.label`, { defaultValue: message.title });

/** inboxHeadline picks the most specific text a message has. */
export const inboxHeadline = (message: InboxMessage, eventLabel: string) =>
  message.release_name || message.message || eventLabel;

/** inboxRejectionClass colors rejection text amber for rejected pushes and red for errors. */
export const inboxRejectionClass = (event: InboxMessage["event"]) => event === "PUSH_REJECTED"
  ? "text-amber-700 dark:text-amber-400"
  : "text-red-600 dark:text-red-400";
