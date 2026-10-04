/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { ComponentType, SVGProps } from "react";
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
  tile: string;
  text: string;
  label: string;
}

const RED = {
  tile: "bg-red-100 text-red-700 dark:bg-red-500/15 dark:text-red-400",
  text: "text-red-600 dark:text-red-400",
  label: "bg-red-50 text-red-700 ring-red-600/20 dark:bg-red-400/10 dark:text-red-400 dark:ring-red-400/25"
};

const GREEN = {
  tile: "bg-green-100 text-green-700 dark:bg-green-500/15 dark:text-green-400",
  text: "text-green-600 dark:text-green-400",
  label: "bg-green-50 text-green-700 ring-green-600/20 dark:bg-green-400/10 dark:text-green-400 dark:ring-green-400/25"
};

const AMBER = {
  tile: "bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-400",
  text: "text-amber-600 dark:text-amber-400",
  label: "bg-amber-50 text-amber-800 ring-amber-600/20 dark:bg-amber-400/10 dark:text-amber-400 dark:ring-amber-400/25"
};

const BLUE = {
  tile: "bg-blue-100 text-blue-700 dark:bg-blue-500/15 dark:text-blue-400",
  text: "text-blue-600 dark:text-blue-400",
  label: "bg-blue-50 text-blue-700 ring-blue-600/20 dark:bg-blue-400/10 dark:text-blue-400 dark:ring-blue-400/25"
};

const GRAY = {
  tile: "bg-gray-100 text-gray-600 dark:bg-gray-750 dark:text-gray-300",
  text: "text-gray-500 dark:text-gray-400",
  label: "bg-gray-50 text-gray-600 ring-gray-500/20 dark:bg-gray-400/10 dark:text-gray-400 dark:ring-gray-400/25"
};

export const INBOX_EVENT_STYLES: Record<InboxMessage["event"], InboxEventStyle> = {
  PUSH_APPROVED: { icon: CheckCircleIcon, ...GREEN },
  PUSH_REJECTED: { icon: NoSymbolIcon, ...AMBER },
  PUSH_ERROR: { icon: ExclamationTriangleIcon, ...RED },
  IRC_DISCONNECTED: { icon: SignalSlashIcon, ...RED },
  IRC_RECONNECTED: { icon: SignalIcon, ...GREEN },
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
  "LIST_REFRESH_ERROR",
  "LIST_REFRESH_SUCCESS",
  "FEED_REFRESH_ERROR",
  "FEED_REFRESH_SUCCESS",
  "APP_UPDATE_AVAILABLE"
];
