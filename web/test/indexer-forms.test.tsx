/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import { IndexerAddForm, IndexerUpdateForm } from "@forms/settings/IndexerForms";
import "@app/i18n";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const rssIndexer: IndexerDefinition = {
  id: 5,
  name: "DistroWatch",
  identifier: "rss-distrowatch",
  identifier_external: "DistroWatch",
  implementation: "rss",
  base_url: "",
  enabled: true,
  description: "Generic RSS",
  language: "en-us",
  privacy: "private",
  protocol: "torrent",
  urls: [],
  supports: ["rss"],
  use_proxy: false,
  settings: [],
  irc: { network: "", server: "", port: 0, tls: false, settings: [], channels: [] },
  feed: { minInterval: 15, settings: [{ name: "url", type: "text", label: "RSS URL", required: true }] }
};

test.each(
  (["irc", "rss", "torznab", "newznab"] as const).flatMap((implementation) =>
    ["", "DistroWatch (Prowlarr)"].map((identifierExternal) => ({ implementation, identifierExternal }))
  )
)("the $implementation add form submits external identifier '$identifierExternal'", async ({ implementation, identifierExternal }) => {
  const definition: IndexerDefinition = { ...rssIndexer, implementation };
  vi.spyOn(APIClient.indexers, "getSchema").mockResolvedValue([definition]);
  vi.spyOn(APIClient.proxy, "list").mockResolvedValue([]);
  const create = vi.spyOn(APIClient.indexers, "create").mockResolvedValue({ ...definition, id: 9 } as unknown as Indexer);
  vi.spyOn(APIClient.feeds, "create").mockResolvedValue(undefined as never);
  vi.spyOn(APIClient.irc, "createNetwork").mockResolvedValue(undefined as never);

  render(
    <QueryClientProvider client={new QueryClient()}>
      <IndexerAddForm isOpen={true} toggle={() => {}} />
    </QueryClientProvider>
  );

  expect(screen.queryByRole("textbox", { name: "External Identifier" })).toBeNull();

  const picker = document.querySelector("input[id^='react-select'][id$='-input']") as HTMLInputElement;
  fireEvent.keyDown(picker, { key: "ArrowDown", keyCode: 40 });
  fireEvent.click(await screen.findByText("DistroWatch"));

  const externalIdentifier = screen.getByRole("textbox", { name: "External Identifier" }) as HTMLInputElement;
  expect(externalIdentifier.value).toBe("");
  expect(externalIdentifier.required).toBe(false);
  fireEvent.change(externalIdentifier, { target: { value: identifierExternal } });

  if (implementation !== "irc") {
    const urlField = implementation === "newznab" ? "feed.newznab_url" : "feed.url";
    fireEvent.change(document.getElementById(urlField) as HTMLInputElement, { target: { value: "https://example.com/rss" } });
  }
  if (implementation === "torznab" || implementation === "newznab") {
    fireEvent.change(document.getElementById("feed.api_key") as HTMLInputElement, { target: { value: "test-key" } });
  }

  await act(async () => {
    fireEvent.click(screen.getByText("Save"));
  });

  await waitFor(() => expect(create).toHaveBeenCalledTimes(1));
  expect(create.mock.calls[0][0]).toMatchObject({
    identifier: "rss-distrowatch",
    identifier_external: identifierExternal,
    implementation
  });
});

test("changing the selected indexer clears its external identifier", async () => {
  vi.spyOn(APIClient.indexers, "getSchema").mockResolvedValue([
    rssIndexer,
    { ...rssIndexer, name: "Other RSS", identifier: "rss-other" }
  ]);
  vi.spyOn(APIClient.proxy, "list").mockResolvedValue([]);

  render(
    <QueryClientProvider client={new QueryClient()}>
      <IndexerAddForm isOpen={true} toggle={() => {}} />
    </QueryClientProvider>
  );

  const picker = document.querySelector("input[id^='react-select'][id$='-input']") as HTMLInputElement;
  fireEvent.keyDown(picker, { key: "ArrowDown", keyCode: 40 });
  fireEvent.click(await screen.findByText("DistroWatch"));
  fireEvent.change(screen.getByRole("textbox", { name: "External Identifier" }), {
    target: { value: "DistroWatch (Prowlarr)" }
  });

  fireEvent.keyDown(picker, { key: "ArrowDown", keyCode: 40 });
  fireEvent.click(await screen.findByText("Other RSS"));

  expect((screen.getByRole("textbox", { name: "External Identifier" }) as HTMLInputElement).value).toBe("");
});

test("a feed indexer offers the proxy section and saves the toggle", async () => {
  vi.spyOn(APIClient.proxy, "list").mockResolvedValue([
    { id: 3, name: "Mullvad", enabled: true, type: "SOCKS5", addr: "socks5://127.0.0.1:1080" }
  ]);
  const update = vi.spyOn(APIClient.indexers, "update").mockResolvedValue(undefined as never);

  render(
    <QueryClientProvider client={new QueryClient()}>
      <IndexerUpdateForm isOpen={true} toggle={() => {}} data={rssIndexer} />
    </QueryClientProvider>
  );

  expect(screen.getByRole("heading", { name: "Proxy" })).toBeTruthy();
  expect(screen.queryByText("Select proxy")).toBeNull();

  fireEvent.click(document.querySelector("button#use_proxy") as HTMLButtonElement);

  expect(await screen.findByText("Select proxy")).toBeTruthy();
  const externalIdentifier = screen.getByRole("textbox", { name: "External Identifier" }) as HTMLInputElement;
  expect(externalIdentifier.value).toBe("DistroWatch");
  fireEvent.change(externalIdentifier, { target: { value: "DistroWatch (Prowlarr)" } });

  await act(async () => {
    fireEvent.click(screen.getByText("Save"));
  });

  expect(update).toHaveBeenCalledTimes(1);
  expect(update.mock.calls[0][0]).toMatchObject({ id: 5, implementation: "rss", use_proxy: true, identifier_external: "DistroWatch (Prowlarr)" });
});

test("the add form offers the proxy section once an indexer is chosen", async () => {
  vi.spyOn(APIClient.indexers, "getSchema").mockResolvedValue([rssIndexer]);
  vi.spyOn(APIClient.proxy, "list").mockResolvedValue([
    { id: 3, name: "Mullvad", enabled: true, type: "SOCKS5", addr: "socks5://127.0.0.1:1080" }
  ]);
  const create = vi.spyOn(APIClient.indexers, "create").mockResolvedValue({ ...rssIndexer, id: 9 } as unknown as Indexer);
  vi.spyOn(APIClient.feeds, "create").mockResolvedValue(undefined as never);

  render(
    <QueryClientProvider client={new QueryClient()}>
      <IndexerAddForm isOpen={true} toggle={() => {}} />
    </QueryClientProvider>
  );

  expect(screen.queryByRole("heading", { name: "Proxy" })).toBeNull();

  const picker = document.querySelector("input[id^='react-select'][id$='-input']") as HTMLInputElement;
  fireEvent.keyDown(picker, { key: "ArrowDown", keyCode: 40 });
  fireEvent.click(await screen.findByText("DistroWatch"));

  expect(await screen.findByRole("heading", { name: "Proxy" })).toBeTruthy();

  fireEvent.click(document.querySelector("button#use_proxy") as HTMLButtonElement);
  expect(await screen.findByText("Select proxy")).toBeTruthy();

  fireEvent.change(document.getElementById("feed.url") as HTMLInputElement, { target: { value: "https://distrowatch.com/news/torrents.xml" } });

  await act(async () => {
    fireEvent.click(screen.getByText("Save"));
  });

  expect(create).toHaveBeenCalledTimes(1);
  expect(create.mock.calls[0][0]).toMatchObject({ identifier: "rss-distrowatch", implementation: "rss", use_proxy: true });
});
