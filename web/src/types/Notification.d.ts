/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

type NotificationType = "DISCORD" | "NOTIFIARR" | "TELEGRAM" | "PUSHOVER" | "GOTIFY" | "NTFY" | "LUNASEA" | "SHOUTRRR" | "WEBHOOK" | "BUILTIN";
type NotificationEvent =
  "PUSH_APPROVED"
  | "PUSH_REJECTED"
  | "PUSH_ERROR"
  | "IRC_DISCONNECTED"
  | "IRC_RECONNECTED"
  | "IRC_UNHEALTHY"
  | "IRC_HEALTHY"
  | "LIST_REFRESH_SUCCESS"
  | "LIST_REFRESH_ERROR"
  | "FEED_REFRESH_SUCCESS"
  | "FEED_REFRESH_ERROR"
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
  sound?: string;
  event_sounds?: Record<string, string>;
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

interface InboxMessage {
  id: number;
  event: NotificationEvent | "TEST";
  title: string;
  message: string;
  release_name: string;
  indexer: string;
  filter_name: string;
  filter_id: number;
  action: string;
  action_client: string;
  rejections: string[];
  url: string;
  read_at: string | null;
  created_at: string;
}

interface InboxResponse {
  data: InboxMessage[];
  count: number;
  all_count: number;
  unread_count: number;
}

interface InboxQueryParams {
  limit: number;
  offset: number;
  unread: boolean;
  event?: NotificationEvent;
}
