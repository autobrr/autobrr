/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import { ProxyUpdateForm } from "@forms/settings/ProxyForms";
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

const renderForm = () =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <ProxyUpdateForm isOpen={true} toggle={() => {}} data={proxy} />
    </QueryClientProvider>
  );

test("delete modal lists the indexers, irc networks and feeds using the proxy", async () => {
  vi.spyOn(APIClient.proxy, "usage").mockResolvedValue({
    indexers: [{ id: 1, name: "BeyondHD" }],
    irc_networks: [{ id: 2, name: "TorrentLeech" }],
    feeds: [{ id: 4, name: "BHD Torznab" }]
  });
  renderForm();

  fireEvent.click(screen.getByText("Remove"));

  expect(await screen.findByText("BeyondHD")).toBeTruthy();
  expect(screen.getByText("This proxy is still in use")).toBeTruthy();
  expect(screen.getByText("TorrentLeech")).toBeTruthy();
  expect(screen.getByText("BHD Torznab")).toBeTruthy();
});

test("delete modal shows no usage warning for an unused proxy", async () => {
  const usage = vi.spyOn(APIClient.proxy, "usage").mockResolvedValue({ indexers: [], irc_networks: [], feeds: [] });
  renderForm();

  fireEvent.click(screen.getByText("Remove"));

  expect(await screen.findByText(/Are you sure you want to remove this Proxy/)).toBeTruthy();
  expect(usage).toHaveBeenCalledWith(3);
  expect(screen.queryByText("This proxy is still in use")).toBeNull();
});

test("turning the enabled switch off warns which irc networks will be disabled", async () => {
  vi.spyOn(APIClient.proxy, "usage").mockResolvedValue({
    indexers: [],
    irc_networks: [{ id: 2, name: "TorrentLeech" }],
    feeds: []
  });
  renderForm();

  expect(screen.queryByText("This proxy is still in use")).toBeNull();

  fireEvent.click(document.querySelector("button#enabled") as HTMLButtonElement);

  expect(await screen.findByText("TorrentLeech")).toBeTruthy();
  expect(screen.getByText(/Disabling it will disable the following IRC networks/)).toBeTruthy();

  fireEvent.click(document.querySelector("button#enabled") as HTMLButtonElement);

  expect(screen.queryByText("This proxy is still in use")).toBeNull();
});

test("the delete modal fetches a fresh usage snapshot every time it opens", async () => {
  const usage = vi.spyOn(APIClient.proxy, "usage").mockResolvedValue({ indexers: [], irc_networks: [], feeds: [] });
  renderForm();

  expect(usage).not.toHaveBeenCalled();

  fireEvent.click(screen.getByText("Remove"));
  await waitFor(() => expect(usage).toHaveBeenCalledTimes(1));

  // the slide-over has its own Cancel button, so scope to the modal
  const modal = screen.getByText(/Are you sure you want to remove this Proxy/).closest("[role='dialog']") as HTMLElement;
  fireEvent.click(within(modal).getByText("Cancel"));
  await waitFor(() => expect(screen.queryByText(/Are you sure you want to remove this Proxy/)).toBeNull());

  fireEvent.click(screen.getByText("Remove"));
  await waitFor(() => expect(usage).toHaveBeenCalledTimes(2));
});
