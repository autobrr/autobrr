/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import { APIKeyAddForm, APIKeyUpdateForm } from "@forms/settings/APIKeyForms";
import "@app/i18n";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

// Headless UI decides outside clicks from the mousedown target, which a bare fireEvent.click never sends.
const press = (element: HTMLElement) => {
  fireEvent.pointerDown(element);
  fireEvent.mouseDown(element);
  fireEvent.pointerUp(element);
  fireEvent.mouseUp(element);
  fireEvent.click(element);
};

// The drawer moves focus into itself after opening, which would close a popover opened before that.
const renderAddForm = async () => {
  render(
    <QueryClientProvider client={new QueryClient()}>
      <APIKeyAddForm isOpen={true} toggle={() => {}} />
    </QueryClientProvider>
  );

  await waitFor(() => expect(document.activeElement).toBe(screen.getByRole("dialog")));
};

test("new key requires full access or at least one permission", async () => {
  const create = vi.spyOn(APIClient.apikeys, "create").mockResolvedValue(undefined);
  await renderAddForm();

  fireEvent.change(document.querySelector("input#name") as HTMLInputElement, { target: { value: "sonarr" } });
  fireEvent.click(screen.getByText("Create"));

  expect(await screen.findByText("Choose full access or add at least one permission")).toBeTruthy();
  expect(create).not.toHaveBeenCalled();
});

test("added permissions default to the lowest access level and can be raised", async () => {
  const create = vi.spyOn(APIClient.apikeys, "create").mockResolvedValue(undefined);
  await renderAddForm();

  fireEvent.change(document.querySelector("input#name") as HTMLInputElement, { target: { value: "sonarr" } });

  fireEvent.click(screen.getByText("Add permissions"));
  press(await screen.findByLabelText("Filters"));
  press(screen.getByLabelText("Webhooks"));

  expect(screen.getByText("Filters and their actions.")).toBeTruthy();

  press(screen.getByRole("button", { name: /Read-only/ }));
  press(await screen.findByRole("menuitem", { name: "Read and write" }));

  fireEvent.click(screen.getByText("Create"));

  await waitFor(() => expect(create).toHaveBeenCalled());
  expect(create.mock.calls[0][0]).toEqual({ name: "sonarr", scopes: ["filters:write", "webhooks:write"] });
});

test("switching to full access replaces custom permissions", async () => {
  const update = vi.spyOn(APIClient.apikeys, "update").mockResolvedValue(undefined);
  const apikey: APIKey = { name: "omegabrr", key: "mock-key", scopes: ["filters:read"], created_at: new Date() };
  render(
    <QueryClientProvider client={new QueryClient()}>
      <APIKeyUpdateForm isOpen={true} toggle={() => {}} data={apikey} />
    </QueryClientProvider>
  );

  expect(screen.getByText("Filters and their actions.")).toBeTruthy();

  fireEvent.click(screen.getByLabelText(/Full access/));

  expect(screen.queryByText("Filters and their actions.")).toBeNull();

  fireEvent.click(screen.getByText("Save"));

  await waitFor(() => expect(update).toHaveBeenCalled());
  expect(update.mock.calls[0][0]).toMatchObject({ key: "mock-key", scopes: ["*"] });
});
