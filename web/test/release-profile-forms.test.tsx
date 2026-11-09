/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import { ReleaseProfileDuplicateUpdateForm } from "@forms/settings/ReleaseForms";
import "@app/i18n";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const profile: ReleaseProfileDuplicate = {
  id: 2,
  name: "Movie",
  protocol: false,
  release_name: false,
  hash: false,
  title: true,
  sub_title: false,
  year: true,
  month: false,
  day: false,
  source: false,
  resolution: true,
  codec: false,
  container: false,
  dynamic_range: false,
  audio: false,
  group: false,
  season: false,
  episode: false,
  website: false,
  proper: false,
  repack: false,
  edition: false,
  hybrid: false,
  language: false,
  media_processing: true
};

const renderForm = () =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <ReleaseProfileDuplicateUpdateForm isOpen={true} toggle={() => {}} data={profile} />
    </QueryClientProvider>
  );

test("hash is no longer offered and protocol and media processing are saved", async () => {
  const store = vi.spyOn(APIClient.release.profiles.duplicates, "store").mockResolvedValue(undefined as never);
  renderForm();

  expect(document.querySelector("button#hash")).toBeNull();

  fireEvent.click(document.querySelector("button#protocol") as HTMLButtonElement);
  fireEvent.click(screen.getByText("Save"));

  await waitFor(() => expect(store).toHaveBeenCalledTimes(1));
  expect(store.mock.calls[0][0]).toMatchObject({ id: 2, protocol: true, media_processing: true, title: true });
});

test("delete modal warns that filters stop skipping duplicates", async () => {
  renderForm();

  fireEvent.click(screen.getByText("Remove"));

  expect(await screen.findByText("Filters using this profile stop skipping duplicates.")).toBeTruthy();
});
