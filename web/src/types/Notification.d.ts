/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

type NotificationType = "DISCORD" | "NOTIFIARR" | "TELEGRAM" | "PUSHOVER" | "GOTIFY" | "NTFY" | "LUNASEA" | "SHOUTRRR";
type NotificationEvent =
  "PUSH_APPROVED"
  | "PUSH_REJECTED"
  | "PUSH_ERROR"
  | "IRC_DISCONNECTED"
  | "IRC_RECONNECTED"
  | "APP_UPDATE_AVAILABLE";

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
}
