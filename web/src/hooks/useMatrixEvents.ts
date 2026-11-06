/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useMemo, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import { APIClient } from "@api/APIClient";
import { MatrixKeys } from "@api/query_keys";
import { MatrixRoomHistoryQueryOptions } from "@api/queries";

type MessageHandler = (message: MatrixMessage) => void;
type HealthHandler = (event: MatrixHealthEvent) => void;
type OpenHandler = () => void;

const roomKey = (networkId: number, room: string) => `${networkId}:${room}`;

// One EventSource shared by every subscriber, opened with the first and closed with the last.
class MatrixEventManager {
  private eventSource: EventSource | null = null;
  private messageHandlers = new Map<string, Set<MessageHandler>>();
  private healthHandlers = new Set<HealthHandler>();
  private openHandlers = new Set<OpenHandler>();
  private refCount = 0;
  private reconnectTimer: number | null = null;
  private reconnectAttempt = 0;

  subscribeToRoom(networkId: number, room: string, handler: MessageHandler): () => void {
    const key = roomKey(networkId, room);
    const handlers = this.messageHandlers.get(key) ?? new Set<MessageHandler>();
    handlers.add(handler);
    this.messageHandlers.set(key, handlers);

    return this.retain(() => {
      handlers.delete(handler);
      if (handlers.size === 0) {
        this.messageHandlers.delete(key);
      }
    });
  }

  subscribeToHealth(handler: HealthHandler): () => void {
    this.healthHandlers.add(handler);

    return this.retain(() => this.healthHandlers.delete(handler));
  }

  // Called every time the stream opens. Events sent before that, including while a first
  // connect was still being set up after a history snapshot, are not replayed.
  subscribeToOpen(handler: OpenHandler): () => void {
    this.openHandlers.add(handler);

    return this.retain(() => this.openHandlers.delete(handler));
  }

  private retain(release: () => void): () => void {
    this.refCount++;
    if (this.refCount === 1) {
      this.connect();
    }

    return () => {
      release();
      this.refCount--;
      if (this.refCount === 0) {
        this.disconnect();
      }
    };
  }

  private connect() {
    if (this.eventSource) {
      return;
    }

    this.eventSource = APIClient.matrix.events();

    this.eventSource.addEventListener("MESSAGE", (event) => {
      const message = JSON.parse(event.data) as MatrixMessage;
      this.messageHandlers.get(roomKey(message.network, message.room))?.forEach((handler) => handler(message));
    });

    this.eventSource.addEventListener("HEALTH", (event) => {
      const health = JSON.parse(event.data) as MatrixHealthEvent;
      this.healthHandlers.forEach((handler) => handler(health));
    });

    this.eventSource.onopen = () => {
      this.reconnectAttempt = 0;
      this.openHandlers.forEach((handler) => handler());
    };

    // The browser only retries transport errors. A non-2xx handshake (expired session,
    // proxy restart) closes the stream for good, so reconnect ourselves with backoff.
    this.eventSource.onerror = () => {
      if (this.eventSource?.readyState !== EventSource.CLOSED) {
        return;
      }

      this.disconnect();
      if (this.refCount === 0) {
        return;
      }

      const delay = Math.min(1000 * 2 ** this.reconnectAttempt, 30000);
      this.reconnectAttempt++;
      this.reconnectTimer = window.setTimeout(() => {
        this.reconnectTimer = null;
        if (this.refCount > 0) {
          this.connect();
        }
      }, delay);
    };
  }

  private disconnect() {
    if (this.reconnectTimer !== null) {
      window.clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    this.eventSource?.close();
    this.eventSource = null;
  }
}

const matrixEventManager = new MatrixEventManager();

// mergeRoomMessages combines the server history with live messages, dropping messages
// that are in both and keeping the newest limit in time order.
export const mergeRoomMessages = (history: MatrixMessage[], live: MatrixMessage[], limit: number) => {
  const seen = new Set(history.map((message) => message.id));
  const merged = [...history, ...live.filter((message) => !seen.has(message.id))];

  merged.sort((a, b) => Date.parse(a.time) - Date.parse(b.time));

  return merged.slice(-limit);
};

export function useMatrixRoomWithHistory(networkId: number, room: string, limit: number) {
  const queryClient = useQueryClient();
  const [liveMessages, setLiveMessages] = useState<MatrixMessage[]>([]);

  const historyQuery = useQuery(MatrixRoomHistoryQueryOptions(networkId, room));

  // Subscribed before history arrives so nothing falls between the two; the ids drop the overlap.
  useEffect(() => matrixEventManager.subscribeToRoom(networkId, room, (message) => {
    setLiveMessages((prev) => [...prev, message].slice(-limit));
  }), [networkId, room, limit]);

  useEffect(() => matrixEventManager.subscribeToOpen(() => {
    queryClient.invalidateQueries({ queryKey: MatrixKeys.roomHistory(networkId, room) });
  }), [queryClient, networkId, room]);

  const messages = useMemo(
    () => mergeRoomMessages(historyQuery.data ?? [], liveMessages, limit),
    [historyQuery.data, liveMessages, limit]
  );

  return { messages, isLoading: historyQuery.isLoading };
}

export function useMatrixHealthSync(onHealth: HealthHandler) {
  const handlerRef = useRef(onHealth);

  useEffect(() => {
    handlerRef.current = onHealth;
  }, [onHealth]);

  useEffect(() => matrixEventManager.subscribeToHealth((event) => handlerRef.current(event)), []);
}
