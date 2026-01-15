/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

type NotificationType = "DISCORD" | "NOTIFIARR" | "TELEGRAM" | "PUSHOVER" | "GOTIFY" | "NTFY" | "LUNASEA" | "SHOUTRRR" | "GENERIC_WEBHOOK";
type NotificationEvent =
  "PUSH_APPROVED"
  | "PUSH_REJECTED"
  | "PUSH_ERROR"
  | "IRC_DISCONNECTED"
  | "IRC_RECONNECTED"
  | "APP_UPDATE_AVAILABLE"
  | "RELEASE_NEW";

interface ServiceNotification {
  id: number;
  name: string;
  enabled: boolean;
  type: NotificationType;
  events: NotificationEvent[];
  webhook?: string;
  token?: string;
  api_key?: string;
  channel?: string;
  priority?: number;
  topic?: string;
  host?: string;
  username?: string;
  password?: string;
  method?: string;
  headers?: string;
  used_by_filters?: NotificationFilter[];
}

interface NotificationFilter {
  filter_name: string;
  filter_id: number;
  notification_id: number;
  notification?: ServiceNotification;
  events: NotificationFilterEvent[];
}

type NotificationFilterEvent = "PUSH_APPROVED" | "PUSH_REJECTED" | "PUSH_ERROR" | "RELEASE_NEW";