/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useRef, useState, useCallback } from "react";
import { APIClient } from "@api/APIClient";

type IrcEvent = {
  network: number;
  channel: string;
  nick: string;
  msg: string;
  time: string;
};

type IrcHealthEvent = {
  network: number;
  healthy: boolean;
  connected_since?: string;
  connection_errors?: string[];
};

type ChannelKey = `${number}:${string}`;
type EventHandler = (event: IrcEvent) => void;
type HealthEventHandler = (event: IrcHealthEvent) => void;

class IrcEventManager {
  private eventSource: EventSource | null = null;
  private subscribers: Map<ChannelKey, Set<EventHandler>> = new Map();
  private healthSubscribers: Map<number, Set<HealthEventHandler>> = new Map();
  private refCount = 0;

  private getChannelKey(networkId: number, channel: string): ChannelKey {
    return `${networkId}:${channel.toLowerCase()}`;
  }

  public subscribe(networkId: number, channel: string, handler: EventHandler): () => void {
    const key = this.getChannelKey(networkId, channel);

    if (!this.subscribers.has(key)) {
      this.subscribers.set(key, new Set());
    }

    this.subscribers.get(key)!.add(handler);
    this.refCount++;

    // Initialize the SSE connection if this is the first subscriber
    if (this.refCount === 1) {
      this.connect();
    }

    // Return unsubscribe function
    return () => {
      const handlers = this.subscribers.get(key);
      if (handlers) {
        handlers.delete(handler);
        if (handlers.size === 0) {
          this.subscribers.delete(key);
        }
      }

      this.refCount--;

      // Close the connection if there are no more subscribers
      if (this.refCount === 0) {
        this.disconnect();
      }
    };
  }

  public subscribeToHealth(networkId: number, handler: HealthEventHandler): () => void {
    if (!this.healthSubscribers.has(networkId)) {
      this.healthSubscribers.set(networkId, new Set());
    }

    this.healthSubscribers.get(networkId)!.add(handler);
    this.refCount++;

    // Initialize the SSE connection if this is the first subscriber
    if (this.refCount === 1) {
      this.connect();
    }

    // Return unsubscribe function
    return () => {
      const handlers = this.healthSubscribers.get(networkId);
      if (handlers) {
        handlers.delete(handler);
        if (handlers.size === 0) {
          this.healthSubscribers.delete(networkId);
        }
      }

      this.refCount--;

      // Close the connection if there are no more subscribers
      if (this.refCount === 0) {
        this.disconnect();
      }
    };
  }

  private connect() {
    if (this.eventSource) {
      return;
    }

    this.eventSource = APIClient.irc.allEvents();

    // Listen for PRIVMSG events (channel messages)
    this.eventSource.addEventListener("PRIVMSG", (event) => {
      try {
        const ircEvent = JSON.parse(event.data) as IrcEvent;
        const key = this.getChannelKey(ircEvent.network, ircEvent.channel);
        const handlers = this.subscribers.get(key);

        if (handlers) {
          handlers.forEach(handler => handler(ircEvent));
        }
      } catch (error) {
        console.error("Failed to parse IRC PRIVMSG event:", error);
      }
    });

    // Listen for HEALTH events (network health changes)
    this.eventSource.addEventListener("HEALTH", (event) => {
      try {
        const healthEvent = JSON.parse(event.data) as IrcHealthEvent;
        const handlers = this.healthSubscribers.get(healthEvent.network);

        if (handlers) {
          handlers.forEach(handler => handler(healthEvent));
        }
      } catch (error) {
        console.error("Failed to parse IRC HEALTH event:", error);
      }
    });

    // Fallback for events without a type (shouldn't happen with custom events)
    this.eventSource.onmessage = (event) => {
      console.warn("Received untyped IRC event:", event.data);
    };

    this.eventSource.onerror = (error) => {
      console.error("IRC EventSource error:", error);
      // The browser will automatically attempt to reconnect
    };
  }

  private disconnect() {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
  }

  public forceReconnect() {
    this.disconnect();
    if (this.refCount > 0) {
      this.connect();
    }
  }
}

// Singleton instance
const ircEventManager = new IrcEventManager();

/**
 * Hook to subscribe to IRC events for a specific network and channel.
 * Uses a single shared SSE connection for all channels.
 *
 * @param networkId The IRC network ID
 * @param channel The channel name (with or without #)
 * @param enabled Whether to actively subscribe (default: true)
 * @returns Array of IRC events for this channel
 */
export function useIrcEvents(
  networkId: number,
  channel: string,
  enabled: boolean = true
): IrcEvent[] {
  const [events, setEvents] = useState<IrcEvent[]>([]);

  useEffect(() => {
    if (!enabled || !networkId || !channel) {
      return;
    }

    const handleEvent = (event: IrcEvent) => {
      setEvents(prev => [...prev, event]);
    };

    const unsubscribe = ircEventManager.subscribe(networkId, channel, handleEvent);

    return () => {
      unsubscribe();
    };
  }, [networkId, channel, enabled]);

  return events;
}

/**
 * Hook to load channel history and subscribe to new events.
 * Combines historical data with real-time SSE updates.
 *
 * @param networkId The IRC network ID
 * @param channel The channel name (with or without #)
 * @param limit Maximum number of historical events to load (default: 100)
 * @param enabled Whether to actively load and subscribe (default: true)
 * @returns Object with events array, loading state, and error
 */
export function useIrcChannelWithHistory(
  networkId: number,
  channel: string,
  limit: number = 100,
  enabled: boolean = true
): {
  events: IrcEvent[];
  isLoading: boolean;
  error: Error | null;
  clearEvents: () => void;
} {
  const [events, setEvents] = useState<IrcEvent[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const historyLoadedRef = useRef(false);

  // Load historical events
  useEffect(() => {
    if (!enabled || !networkId || !channel || historyLoadedRef.current) {
      return;
    }

    const loadHistory = async () => {
      setIsLoading(true);
      setError(null);

      try {
        // Strip # from channel if present for API call
        const cleanChannel = channel.startsWith("#") ? channel.substring(1) : channel;
        const history = await APIClient.irc.getChannelHistory(networkId, cleanChannel);
        setEvents(history);
        historyLoadedRef.current = true;
      } catch (err) {
        setError(err instanceof Error ? err : new Error("Failed to load channel history"));
        console.error("Failed to load IRC channel history:", err);
      } finally {
        setIsLoading(false);
      }
    };

    loadHistory();
  }, [networkId, channel, limit, enabled]);

  // Subscribe to new events
  useEffect(() => {
    if (!enabled || !networkId || !channel || !historyLoadedRef.current) {
      return;
    }

    const handleEvent = (event: IrcEvent) => {
      setEvents(prev => [...prev, event]);
    };

    const unsubscribe = ircEventManager.subscribe(networkId, channel, handleEvent);

    return () => {
      unsubscribe();
    };
  }, [networkId, channel, enabled, historyLoadedRef.current]);

  // Reset when network or channel changes
  useEffect(() => {
    historyLoadedRef.current = false;
    setEvents([]);
    setError(null);
  }, [networkId, channel]);

  const clearEvents = useCallback(() => {
    setEvents([]);
  }, []);

  return { events, isLoading, error, clearEvents };
}

/**
 * Hook to subscribe to IRC network health events.
 * Receives real-time updates when network health changes.
 *
 * @param networkId The IRC network ID to monitor
 * @param enabled Whether to actively subscribe (default: true)
 * @returns The latest health event for this network, or null if none received
 */
export function useIrcNetworkHealth(
  networkId: number,
  enabled: boolean = true
): IrcHealthEvent | null {
  const [healthEvent, setHealthEvent] = useState<IrcHealthEvent | null>(null);

  useEffect(() => {
    if (!enabled || !networkId) {
      return;
    }

    const handleHealthEvent = (event: IrcHealthEvent) => {
      setHealthEvent(event);
    };

    const unsubscribe = ircEventManager.subscribeToHealth(networkId, handleHealthEvent);

    return () => {
      unsubscribe();
    };
  }, [networkId, enabled]);

  return healthEvent;
}

/**
 * Hook to subscribe to all IRC network health events and update React Query cache.
 * This keeps the network list data in sync with real-time health changes.
 *
 * @param networkIds Array of network IDs to monitor
 * @param onHealthChange Optional callback when health changes
 */
export function useIrcNetworkHealthSync(
  networkIds: number[],
  onHealthChange?: (event: IrcHealthEvent) => void
): void {
  useEffect(() => {
    if (!networkIds || networkIds.length === 0) {
      return;
    }

    const unsubscribers = networkIds.map(networkId => {
      return ircEventManager.subscribeToHealth(networkId, (healthEvent) => {
        if (onHealthChange) {
          onHealthChange(healthEvent);
        }
      });
    });

    return () => {
      unsubscribers.forEach(unsubscribe => unsubscribe());
    };
  }, [networkIds, onHealthChange]);
}

/**
 * Force reconnect the shared SSE connection.
 * Useful when you need to manually trigger a reconnection.
 */
export function reconnectIrcEvents() {
  ircEventManager.forceReconnect();
}

// Export types for use in other files
export type { IrcEvent, IrcHealthEvent };