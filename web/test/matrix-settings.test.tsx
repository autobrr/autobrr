/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Suspense } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import MatrixSettings from "@screens/settings/Matrix";
import "@app/i18n";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const room = "#blu-announce:matrix.watchtower.pm";

const network: MatrixNetwork = {
  id: 5,
  name: "Watchtower",
  enabled: true,
  homeserver: "https://matrix.watchtower.pm",
  auth_method: "OAUTH",
  user_id: "@autobrr:matrix.watchtower.pm",
  device_id: "DEVICE",
  access_token: "<redacted>",
  rooms: [{ id: 1, enabled: true, name: room, room_id: "!room:matrix.watchtower.pm", health: { joined: true } }],
  health: { connected: true, healthy: true }
};

class FakeEventSource {
  listeners = new Map<string, (event: MessageEvent) => void>();
  readyState = 1;
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;

  addEventListener(type: string, listener: (event: MessageEvent) => void) {
    this.listeners.set(type, listener);
  }

  close() {
    this.readyState = 2;
  }

  emit(type: string, data: unknown) {
    act(() => {
      this.listeners.get(type)?.(new MessageEvent(type, { data: JSON.stringify(data) }));
    });
  }

  reopen() {
    act(() => {
      this.onopen?.();
    });
  }
}

const renderScreen = () => {
  const stream = new FakeEventSource();
  vi.spyOn(APIClient.matrix, "events").mockReturnValue(stream as unknown as EventSource);
  vi.spyOn(APIClient.matrix, "list").mockResolvedValue([network]);

  render(
    <QueryClientProvider client={new QueryClient()}>
      <Suspense fallback={null}>
        <MatrixSettings />
      </Suspense>
    </QueryClientProvider>
  );

  return stream;
};

const openRoomLog = async () => {
  fireEvent.click(await screen.findByText("Watchtower"));
  fireEvent.click(await screen.findByText("View"));
};

test("room log shows history, then live messages from the stream, and replays a line", async () => {
  const history: MatrixMessage = { id: "$first", network: 5, room, sender: "@blubot:matrix.watchtower.pm", msg: "[Blutopia] [NEW] first", time: "2026-11-06T00:00:00Z" };
  const getRoomHistory = vi.spyOn(APIClient.matrix, "getRoomHistory").mockResolvedValue([history]);
  const reprocess = vi.spyOn(APIClient.matrix, "reprocessAnnounce").mockResolvedValue();

  const stream = renderScreen();
  await openRoomLog();

  expect(await screen.findByText(/\[Blutopia\] \[NEW\] first/)).toBeTruthy();
  expect(getRoomHistory).toHaveBeenCalledWith(5, room);

  stream.emit("MESSAGE", { ...history, id: "$second", msg: "[Blutopia] [NEW] second", time: "2026-11-06T00:01:00Z" });
  stream.emit("MESSAGE", { ...history, id: "$other", room: "#other:matrix.watchtower.pm", msg: "other room" });

  expect(await screen.findByText(/\[Blutopia\] \[NEW\] second/)).toBeTruthy();
  expect(screen.queryByText(/other room/)).toBeNull();

  fireEvent.click(screen.getAllByTitle("Re-process announce")[1]);

  await waitFor(() => expect(reprocess).toHaveBeenCalledWith(5, room, "[Blutopia] [NEW] second"));
});

test("health from the stream updates the network without waiting for the poll", async () => {
  vi.spyOn(APIClient.matrix, "getRoomHistory").mockResolvedValue([]);

  const stream = renderScreen();
  expect(await screen.findByText("Watchtower")).toBeTruthy();
  expect(screen.queryByText("access token was rejected, log in again")).toBeNull();

  stream.emit("HEALTH", {
    network: 5,
    health: { connected: false, healthy: false, error: "access token was rejected, log in again" },
    rooms: { [room]: { joined: false } }
  });

  expect(await screen.findByText("access token was rejected, log in again")).toBeTruthy();
});

test("a message sent while history is loading is shown once", async () => {
  const first: MatrixMessage = { id: "$first", network: 5, room, sender: "@blubot:matrix.watchtower.pm", msg: "first", time: "2026-11-06T00:00:00Z" };
  const during: MatrixMessage = { ...first, id: "$during", msg: "during load", time: "2026-11-06T00:00:30Z" };

  let resolveHistory: (messages: MatrixMessage[]) => void = () => {};
  vi.spyOn(APIClient.matrix, "getRoomHistory").mockReturnValue(new Promise((resolve) => {
    resolveHistory = resolve;
  }));

  const stream = renderScreen();
  await openRoomLog();

  // the server snapshot already contains the message the stream delivered first
  stream.emit("MESSAGE", during);
  await act(async () => resolveHistory([first, during]));

  expect(await screen.findByText(/first/)).toBeTruthy();
  expect(screen.getAllByText(/during load/)).toHaveLength(1);
});

test("history is fetched again whenever the stream opens", async () => {
  const getRoomHistory = vi.spyOn(APIClient.matrix, "getRoomHistory").mockResolvedValue([]);

  const stream = renderScreen();
  await openRoomLog();
  await waitFor(() => expect(getRoomHistory).toHaveBeenCalledTimes(1));

  // the stream can open after the first snapshot, or come back after a drop
  getRoomHistory.mockResolvedValue([{ id: "$missed", network: 5, room, sender: "@blubot:matrix.watchtower.pm", msg: "missed before the stream opened", time: "2026-11-06T00:02:00Z" }]);
  stream.reopen();

  expect(await screen.findByText(/missed before the stream opened/)).toBeTruthy();
  expect(getRoomHistory).toHaveBeenCalledTimes(2);
});

test("a network whose login was rejected shows a log in button that opens the login panel", async () => {
  vi.spyOn(APIClient.matrix, "getRoomHistory").mockResolvedValue([]);
  vi.spyOn(APIClient.matrix, "getByID").mockResolvedValue(network);

  const stream = renderScreen();
  expect(await screen.findByText("Watchtower")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Log in" })).toBeNull();

  stream.emit("HEALTH", {
    network: 5,
    health: { connected: false, healthy: false, error: "access token was rejected, log in again", login_required: true },
    rooms: { [room]: { joined: false } }
  });

  const loginButtons = await screen.findAllByRole("button", { name: "Log in" });
  fireEvent.click(loginButtons[0]);

  expect(await screen.findByText("Log in: Watchtower")).toBeTruthy();
  // the click opens the login panel without also expanding the rooms
  expect(screen.queryByText("View")).toBeNull();
});
