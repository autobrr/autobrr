/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Suspense } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import ProxySettings from "@screens/settings/Proxy";
import "@app/i18n";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const proxy: Proxy = {
  id: 3,
  name: "Mullvad",
  enabled: true,
  type: "SOCKS5",
  addr: "socks5://127.0.0.1:1080"
};

const renderScreen = () =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <Suspense fallback={null}>
        <ProxySettings />
      </Suspense>
    </QueryClientProvider>
  );

const enabledSwitch = async () => {
  await screen.findByText("Mullvad");
  return document.querySelector("button#enabled") as HTMLButtonElement;
};

test("disabling a proxy from the list asks for confirmation and lists its irc networks", async () => {
  vi.spyOn(APIClient.proxy, "list").mockResolvedValue([proxy]);
  vi.spyOn(APIClient.proxy, "usage").mockResolvedValue({
    indexers: [],
    irc_networks: [{ id: 2, name: "TorrentLeech" }],
    feeds: []
  });
  const update = vi.spyOn(APIClient.proxy, "update").mockResolvedValue(undefined as never);

  renderScreen();

  fireEvent.click(await enabledSwitch());

  expect(await screen.findByText("Disable Mullvad")).toBeTruthy();
  expect(await screen.findByText("TorrentLeech")).toBeTruthy();
  expect(update).not.toHaveBeenCalled();

  await act(async () => {
    fireEvent.click(screen.getByText("Disable"));
  });

  expect(update).toHaveBeenCalledTimes(1);
  expect(update.mock.calls[0][0]).toMatchObject({ id: 3, enabled: false });
});

test("enabling a proxy from the list needs no confirmation", async () => {
  vi.spyOn(APIClient.proxy, "list").mockResolvedValue([{ ...proxy, enabled: false }]);
  const usage = vi.spyOn(APIClient.proxy, "usage");
  const update = vi.spyOn(APIClient.proxy, "update").mockResolvedValue(undefined as never);

  renderScreen();

  const toggle = await enabledSwitch();
  await act(async () => {
    fireEvent.click(toggle);
  });

  expect(update).toHaveBeenCalledTimes(1);
  expect(update.mock.calls[0][0]).toMatchObject({ id: 3, enabled: true });
  expect(usage).not.toHaveBeenCalled();
  expect(screen.queryByText("Disable Mullvad")).toBeNull();
});
