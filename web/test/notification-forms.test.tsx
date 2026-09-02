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
