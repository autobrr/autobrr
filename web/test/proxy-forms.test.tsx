/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import { queryClient as appQueryClient } from "@api/QueryClient";
import { Toaster, toast } from "@components/hot-toast";
import { ProxyAddForm, ProxyUpdateForm } from "@forms/settings/ProxyForms";
import "@app/i18n";

afterEach(() => {
  cleanup();
  toast.remove();
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
    <QueryClientProvider client={new QueryClient({ defaultOptions: appQueryClient.getDefaultOptions() })}>
      <Toaster />
      <ProxyUpdateForm isOpen={true} toggle={() => {}} data={proxy} />
    </QueryClientProvider>
  );

const renderAddForm = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: appQueryClient.getDefaultOptions() })}>
      <Toaster />
      <ProxyAddForm isOpen={true} toggle={() => {}} />
    </QueryClientProvider>
  );

const testButton = () => screen.getByRole("button", { name: "Test" });

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

test("a failed test in the update dialog shows the reason and marks the button", async () => {
  vi.spyOn(console, "log").mockImplementation(() => {});
  const proxyTest = vi.spyOn(APIClient.proxy, "test").mockRejectedValue(new Error("could not connect to proxy server: socks5://127.0.0.1:1080"));
  renderForm();

  fireEvent.click(testButton());

  expect(await screen.findByText("could not connect to proxy server: socks5://127.0.0.1:1080")).toBeTruthy();
  expect(screen.getByRole("button", { name: "ERROR" })).toBeTruthy();
  expect(proxyTest).toHaveBeenCalledWith(expect.objectContaining({ id: 3, addr: "socks5://127.0.0.1:1080" }));
});

test("a passing test in the update dialog says so and marks the button", async () => {
  vi.spyOn(APIClient.proxy, "test").mockResolvedValue(undefined);
  renderForm();

  fireEvent.click(testButton());

  expect(await screen.findByText("Test passed")).toBeTruthy();
  expect(screen.getByRole("button", { name: "OK!" })).toBeTruthy();
});

test("the test button shows a spinner and ignores clicks while the test runs", async () => {
  let finish: () => void = () => {};
  const proxyTest = vi.spyOn(APIClient.proxy, "test").mockReturnValue(new Promise<void>((resolve) => { finish = resolve; }));
  renderForm();

  fireEvent.click(testButton());

  const running = await screen.findByRole("button", { name: "Testing" });
  expect((running as HTMLButtonElement).disabled).toBe(true);
  fireEvent.click(running);
  expect(proxyTest).toHaveBeenCalledTimes(1);

  finish();
  expect(await screen.findByRole("button", { name: "OK!" })).toBeTruthy();
});

test("a failed test in the add dialog shows the reason", async () => {
  vi.spyOn(console, "log").mockImplementation(() => {});
  vi.spyOn(APIClient.proxy, "test").mockRejectedValue(new Error("got unexpected status code: 403"));
  renderAddForm();

  fireEvent.click(testButton());

  expect(await screen.findByText("got unexpected status code: 403")).toBeTruthy();
  expect(screen.getByRole("button", { name: "ERROR" })).toBeTruthy();
});

test("a test that passes after the dialog closed shows no popup", async () => {
  let finish: () => void = () => {};
  vi.spyOn(APIClient.proxy, "test").mockReturnValue(new Promise<void>((resolve) => { finish = resolve; }));
  const client = new QueryClient({ defaultOptions: appQueryClient.getDefaultOptions() });
  const dialog = (isOpen: boolean) => (
    <QueryClientProvider client={client}>
      <Toaster />
      <ProxyUpdateForm isOpen={isOpen} toggle={() => {}} data={proxy} />
    </QueryClientProvider>
  );
  const { rerender } = render(dialog(true));

  fireEvent.click(testButton());
  await screen.findByRole("button", { name: "Testing" });
  rerender(dialog(false));
  await waitFor(() => expect(screen.queryByRole("button", { name: "Testing" })).toBeNull());
  finish();

  await new Promise((resolve) => setTimeout(resolve, 50));
  expect(screen.queryByText("Test passed")).toBeNull();
});
