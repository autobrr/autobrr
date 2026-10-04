/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { APIClient } from "@api/APIClient";
import { NotificationKeys } from "@api/query_keys";
import { SettingsContext } from "@utils/Context";

export const browserNotificationsSupported = () => typeof window !== "undefined" && "Notification" in window;

const showBrowserNotification = (msg: InboxMessage) => {
  if (!SettingsContext.get().browserNotifications) {
    return;
  }
  if (!browserNotificationsSupported() || Notification.permission !== "granted") {
    return;
  }

  new Notification(msg.title, { body: msg.message, tag: `autobrr-inbox-${msg.id}` });
};

export function useInboxEvents() {
  const queryClient = useQueryClient();

  useEffect(() => {
    let eventSource: EventSource | null = null;
    let reconnectTimer: number | null = null;
    let reconnectAttempt = 0;

    const connect = () => {
      const es = APIClient.events.notifications();
      eventSource = es;

      es.addEventListener("NOTIFICATION", (event) => {
        void queryClient.invalidateQueries({ queryKey: NotificationKeys.inbox.all() });

        try {
          showBrowserNotification(JSON.parse(event.data) as InboxMessage);
        } catch (error) {
          console.error("Failed to parse inbox NOTIFICATION event:", error);
        }
      });

      es.addEventListener("INBOX_CHANGED", () => {
        void queryClient.invalidateQueries({ queryKey: NotificationKeys.inbox.all() });
      });

      // The stream has no replay, so anything published while disconnected is only in the database.
      es.onopen = () => {
        reconnectAttempt = 0;
        void queryClient.invalidateQueries({ queryKey: NotificationKeys.inbox.all() });
      };

      // The browser only retries transport errors. A non-2xx handshake (expired session,
      // proxy restart) closes the stream for good, so reconnect ourselves with backoff.
      es.onerror = () => {
        if (es.readyState !== EventSource.CLOSED) {
          return;
        }

        es.close();
        eventSource = null;

        const delay = Math.min(1000 * 2 ** reconnectAttempt, 30000);
        reconnectAttempt++;
        reconnectTimer = window.setTimeout(connect, delay);
      };
    };

    connect();

    return () => {
      if (reconnectTimer !== null) {
        window.clearTimeout(reconnectTimer);
      }
      eventSource?.close();
    };
  }, [queryClient]);
}
