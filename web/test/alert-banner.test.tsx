/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createMemoryHistory, createRootRoute, createRouter, RouterProvider } from "@tanstack/react-router";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import "@app/i18n";
import { APIClient } from "@api/APIClient";
import { AlertBanner } from "@components/header/AlertBanner";
import { SettingsContext } from "@utils/Context";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  SettingsContext.set((prev) => ({ ...prev, dismissedUpdate: "" }));
});

const alerts: Alert[] = [
  { kind: "APP_UPDATE", severity: "INFO", subject: "v1.90.0", url: "https://github.com/autobrr/autobrr/releases/tag/v1.90.0" },
  { kind: "IRC_UNHEALTHY", severity: "ERROR", subject_id: 3, subject: "PTP", message: "banned from network: K-Lined" },
  { kind: "LIST_REFRESH_ERROR", severity: "ERROR", subject_id: 1, subject: "Sonarr", message: "connection refused" },
  { kind: "LIST_REFRESH_ERROR", severity: "ERROR", subject_id: 2, subject: "Radarr", message: "connection refused" }
];

const renderBanner = async () => {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const router = createRouter({
    routeTree: createRootRoute({ component: AlertBanner }),
    history: createMemoryHistory({ initialEntries: ["/"] })
  });

  await act(async () => {
    render(
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    );
  });
};

test("groups alerts into one banner per kind", async () => {
  vi.spyOn(APIClient.alerts, "list").mockResolvedValue(alerts);

  await renderBanner();

  expect(await screen.findByText("New update available!")).toBeTruthy();
  expect(screen.getByText("IRC network unhealthy!")).toBeTruthy();
  expect(screen.getByText("Multiple list refreshes failed!")).toBeTruthy();
  expect(screen.getByText("PTP").getAttribute("title")).toBe("banned from network: K-Lined");
  expect(screen.getByText("Sonarr").closest("a")?.getAttribute("href")).toBe("/settings/lists");
});

test("dismissing the update hides it until a newer version", async () => {
  const list = vi.spyOn(APIClient.alerts, "list").mockResolvedValue([alerts[0]]);

  await renderBanner();

  await act(async () => {
    fireEvent.click(await screen.findByRole("button", { name: "Dismiss" }));
  });

  await waitFor(() => expect(screen.queryByText("New update available!")).toBeNull());
  expect(SettingsContext.get().dismissedUpdate).toBe("v1.90.0");

  cleanup();
  list.mockResolvedValue([{ ...alerts[0], subject: "v1.91.0" }]);

  await renderBanner();

  expect(await screen.findByText("v1.91.0")).toBeTruthy();
});
