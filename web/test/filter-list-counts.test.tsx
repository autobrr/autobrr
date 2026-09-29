/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createMemoryHistory, createRootRouteWithContext, createRoute, createRouter, RouterProvider } from "@tanstack/react-router";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import { queryClient as appQueryClient } from "@api/QueryClient";
import { Toaster, toast } from "@components/hot-toast";
import { FilterDetails } from "@screens/filters/Details";
import { Filters } from "@screens/filters/List";
import { Actions } from "@screens/filters/sections/Actions";
import { External } from "@screens/filters/sections/External";
import "@app/i18n";

afterEach(() => {
  cleanup();
  toast.remove();
  vi.restoreAllMocks();
});

function renderFilters() {
  vi.spyOn(window, "scrollTo").mockImplementation(() => {});
  const queryClient = new QueryClient({
    defaultOptions: { queries: { ...appQueryClient.getDefaultOptions().queries, retry: false } }
  });
  const root = createRootRouteWithContext<{ queryClient: QueryClient }>()();
  const auth = createRoute({ getParentRoute: () => root, id: "auth" });
  const authenticated = createRoute({ getParentRoute: () => auth, id: "authenticated-routes" });
  const filters = createRoute({ getParentRoute: () => authenticated, path: "filters" });
  const list = createRoute({ getParentRoute: () => filters, path: "/", component: Filters });
  const details = createRoute({
    getParentRoute: () => filters,
    path: "$filterId",
    params: { parse: ({ filterId }) => ({ filterId: Number(filterId) }) },
    component: FilterDetails
  });
  const actions = createRoute({ getParentRoute: () => details, path: "actions", component: Actions });
  const external = createRoute({ getParentRoute: () => details, path: "external", component: External });
  const router = createRouter({
    routeTree: root.addChildren([auth.addChildren([authenticated.addChildren([
      filters.addChildren([list, details.addChildren([actions, external])])
    ])])]),
    history: createMemoryHistory({ initialEntries: ["/filters"] }),
    context: { queryClient }
  });

  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
      <Toaster />
    </QueryClientProvider>
  );
}

test.each([
  { enabled: false, section: "Actions", count: 2, expected: "Actions: 2/2" },
  { enabled: true, section: "Actions", count: 2, expected: "Actions: 2/2" },
  { enabled: false, section: "External", count: 1, expected: "Actions: 1/1" },
  { enabled: true, section: "External", count: 1, expected: "Actions: 1/1" }
])("saving $section (filter enabled: $enabled) shows correct list counts without a reload", async ({ enabled, section, count, expected }) => {
  const filter = {
    id: 1,
    name: "Test filter",
    enabled,
    priority: 0,
    max_downloads_unit: "",
    actions: [{ id: 1, name: "Existing action", enabled: true, type: "TEST" } as Action],
    external: [] as ExternalFilter[],
    indexers: [{ id: 1, name: "Test indexer", enabled: true } as Indexer],
    actions_count: 0,
    actions_enabled_count: 0
  } as Filter;

  vi.spyOn(APIClient.indexers, "getOptions").mockResolvedValue([]);
  vi.spyOn(APIClient.downloaders, "getAll").mockResolvedValue([]);
  vi.spyOn(APIClient.filters, "getByID").mockResolvedValue(filter);
  vi.spyOn(APIClient.filters, "find")
    .mockResolvedValueOnce([{ ...filter, actions_count: 1, actions_enabled_count: 1 }])
    .mockResolvedValue([{ ...filter, actions_count: count, actions_enabled_count: count }]);
  vi.spyOn(APIClient.filters, "update").mockImplementation(async (updated) => ({
    ...updated,
    actions_count: 0,
    actions_enabled_count: 0
  }));

  renderFilters();

  fireEvent.click(await screen.findByRole("link", { name: "Actions: 1/1" }));
  await screen.findByRole("button", { name: "Add new" });
  if (section === "External") {
    fireEvent.click(screen.getByRole("link", { name: "External" }));
    await screen.findByRole("heading", { name: "External filters" });
  }
  fireEvent.click(await screen.findByRole("button", { name: "Add new" }));
  if (section === "External") {
    fireEvent.change(await screen.findByRole("textbox", { name: /Path to Executable/ }), { target: { value: "true" } });
  }
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
  });
  await screen.findByText("Test filter was updated successfully");
  fireEvent.click(screen.getByRole("link", { name: "Filters" }));

  expect(await screen.findByRole("link", { name: expected })).toBeTruthy();
});
