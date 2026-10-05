/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test } from "vitest";

import { DashboardGrid } from "@screens/dashboard/DashboardGrid";
import { DASHBOARD_WIDGETS } from "@screens/dashboard/widgets";
import { DashboardConfigContext, SettingsContext } from "@utils/Context";
import "@app/i18n";

afterEach(() => {
  cleanup();
  SettingsContext.set((current) => ({ ...current, incognitoMode: false }));
});

test("incognito toggle label follows the current mode", async () => {
  // Hide every widget so the grid renders only the header controls.
  DashboardConfigContext.set({ version: 1, widgets: DASHBOARD_WIDGETS.map(({ id }) => ({ id, hidden: true })) });
  render(
    <QueryClientProvider client={new QueryClient()}>
      <DashboardGrid />
    </QueryClientProvider>
  );

  fireEvent.click(screen.getByRole("button", { name: "Go incognito" }));

  expect(SettingsContext.get().incognitoMode).toBe(true);
  // react-ridge-state notifies subscribers in a setTimeout, so wait for the re-render.
  expect(await screen.findByRole("button", { name: "Exit incognito" })).toHaveProperty("title", "Exit incognito");

  fireEvent.click(screen.getByRole("button", { name: "Exit incognito" }));

  expect(await screen.findByRole("button", { name: "Go incognito" })).toHaveProperty("title", "Go incognito");
});
