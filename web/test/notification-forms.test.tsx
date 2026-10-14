/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import { NotificationUpdateForm } from "@forms/settings/NotificationForms";
import "@app/i18n";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const webhook: ServiceNotification = {
  id: 7,
  name: "Hook",
  enabled: true,
  type: "WEBHOOK",
  events: ["PUSH_APPROVED"],
  webhook: "http://localhost:9999/hook",
  method: "POST",
  headers: "Authorization=Bearer placeholder"
};

test("saving an unrelated change keeps the webhook method and custom headers", async () => {
  const update = vi.spyOn(APIClient.notifications, "update").mockResolvedValue(undefined as never);
  render(
    <QueryClientProvider client={new QueryClient()}>
      <NotificationUpdateForm isOpen={true} toggle={() => {}} data={webhook} />
    </QueryClientProvider>
  );

  expect((screen.getByLabelText("Custom Headers") as HTMLInputElement).value).toBe("Authorization=Bearer placeholder");

  fireEvent.change(screen.getByLabelText(/^Name/), { target: { value: "Renamed" } });
  await act(async () => {
    fireEvent.click(screen.getByText("Save"));
  });

  expect(update).toHaveBeenCalledTimes(1);
  expect(update.mock.calls[0][0]).toMatchObject({
    id: 7,
    name: "Renamed",
    method: "POST",
    headers: "Authorization=Bearer placeholder"
  });
});

test("changing the type keeps the name, enabled flag and selected events", async () => {
  const update = vi.spyOn(APIClient.notifications, "update").mockResolvedValue(undefined as never);
  render(
    <QueryClientProvider client={new QueryClient()}>
      <NotificationUpdateForm isOpen={true} toggle={() => {}} data={webhook} />
    </QueryClientProvider>
  );

  fireEvent.change(screen.getByLabelText(/^Name/), { target: { value: "Renamed" } });

  const picker = document.querySelector("input[id^='react-select'][id$='-input']") as HTMLInputElement;
  fireEvent.keyDown(picker, { key: "ArrowDown", keyCode: 40 });
  await act(async () => {
    fireEvent.click(await screen.findByText("Discord"));
  });

  expect((screen.getByLabelText(/^Name/) as HTMLInputElement).value).toBe("Renamed");

  await act(async () => {
    fireEvent.click(screen.getByText("Save"));
  });

  expect(update).toHaveBeenCalledTimes(1);
  expect(update.mock.calls[0][0]).toMatchObject({
    id: 7,
    type: "DISCORD",
    name: "Renamed",
    enabled: true,
    events: ["PUSH_APPROVED"]
  });
});

const notifiarr: ServiceNotification = {
  id: 8,
  name: "Notifiarr",
  enabled: true,
  type: "NOTIFIARR",
  events: ["PUSH_APPROVED"],
  api_key: "<redacted>"
};

test("a stored notifiarr api key saves without being re-entered", async () => {
  const update = vi.spyOn(APIClient.notifications, "update").mockResolvedValue(undefined as never);
  render(
    <QueryClientProvider client={new QueryClient()}>
      <NotificationUpdateForm isOpen={true} toggle={() => {}} data={notifiarr} />
    </QueryClientProvider>
  );

  await act(async () => {
    fireEvent.click(screen.getByText("Save"));
  });

  expect(update).toHaveBeenCalledTimes(1);
  expect(update.mock.calls[0][0]).toMatchObject({ api_key: "<redacted>" });
});

test.each([
  ["", "Required"],
  ["not-a-uuid", "Must be a 36 character API key of a-z, 0-9 and dashes"]
])("notifiarr api key %j blocks saving", async (apiKey, message) => {
  const update = vi.spyOn(APIClient.notifications, "update").mockResolvedValue(undefined as never);
  render(
    <QueryClientProvider client={new QueryClient()}>
      <NotificationUpdateForm isOpen={true} toggle={() => {}} data={notifiarr} />
    </QueryClientProvider>
  );

  fireEvent.change(screen.getByLabelText(/^API Key/), { target: { value: apiKey } });
  await act(async () => {
    fireEvent.click(screen.getByText("Save"));
  });

  expect(update).not.toHaveBeenCalled();
  expect(await screen.findByText(message)).toBeTruthy();
});

const bark: ServiceNotification = {
  id: 9,
  name: "Bark",
  enabled: true,
  type: "BARK",
  events: ["PUSH_APPROVED"],
  host: "https://api.day.app",
  token: "<redacted>",
  sound: "minuet",
  topic: "autobrr",
  priority: 1
};

test("a stored bark notification saves its device key, sound, group and level", async () => {
  const update = vi.spyOn(APIClient.notifications, "update").mockResolvedValue(undefined as never);
  render(
    <QueryClientProvider client={new QueryClient()}>
      <NotificationUpdateForm isOpen={true} toggle={() => {}} data={bark} />
    </QueryClientProvider>
  );

  expect((screen.getByLabelText("Sound") as HTMLInputElement).value).toBe("minuet");
  expect((screen.getByLabelText("Group") as HTMLInputElement).value).toBe("autobrr");
  expect(screen.getByText("Time Sensitive")).toBeTruthy();

  await act(async () => {
    fireEvent.click(screen.getByText("Save"));
  });

  expect(update).toHaveBeenCalledTimes(1);
  expect(update.mock.calls[0][0]).toMatchObject({
    type: "BARK",
    host: "https://api.day.app",
    token: "<redacted>",
    sound: "minuet",
    topic: "autobrr",
    priority: 1
  });
});

test("picking a bark interruption level saves it as priority", async () => {
  const update = vi.spyOn(APIClient.notifications, "update").mockResolvedValue(undefined as never);
  render(
    <QueryClientProvider client={new QueryClient()}>
      <NotificationUpdateForm isOpen={true} toggle={() => {}} data={bark} />
    </QueryClientProvider>
  );

  const picker = document.querySelector("#priority input") as HTMLInputElement;
  fireEvent.keyDown(picker, { key: "ArrowDown", keyCode: 40 });
  await act(async () => {
    fireEvent.click(await screen.findByText("Critical"));
  });

  await act(async () => {
    fireEvent.click(screen.getByText("Save"));
  });

  expect(update).toHaveBeenCalledTimes(1);
  expect(update.mock.calls[0][0]).toMatchObject({ type: "BARK", priority: 2 });
});

test("bark requires a server url before saving", async () => {
  const update = vi.spyOn(APIClient.notifications, "update").mockResolvedValue(undefined as never);
  render(
    <QueryClientProvider client={new QueryClient()}>
      <NotificationUpdateForm isOpen={true} toggle={() => {}} data={bark} />
    </QueryClientProvider>
  );

  fireEvent.change(screen.getByLabelText(/^Server URL/), { target: { value: "" } });
  await act(async () => {
    fireEvent.click(screen.getByText("Save"));
  });

  expect(update).not.toHaveBeenCalled();
  expect(await screen.findByText("Required")).toBeTruthy();
});
