/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createMemoryHistory, createRootRoute, createRoute, createRouter, Outlet, RouterProvider } from "@tanstack/react-router";
import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { z } from "zod/mini";

import "@app/i18n";
import { APIClient } from "@api/APIClient";
import { Login } from "@screens/auth/Login";
import { AuthContext } from "@utils/Context";

vi.mock("@app/logo.svg?react", () => ({ default: () => null }));

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  AuthContext.reset();
});

const renderLogin = async (entry: string) => {
  const rootRoute = createRootRoute({ component: Outlet });
  const loginRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "login",
    validateSearch: z.object({ redirect: z.optional(z.string()) }),
    component: Login
  });
  const indexRoute = createRoute({ getParentRoute: () => rootRoute, path: "/", component: () => <p>dashboard</p> });
  const filtersRoute = createRoute({ getParentRoute: () => rootRoute, path: "filters", component: () => <p>filters</p> });
  const router = createRouter({
    routeTree: rootRoute.addChildren([loginRoute, indexRoute, filtersRoute]),
    basepath: "/autobrr",
    history: createMemoryHistory({ initialEntries: [entry] })
  });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  await act(async () => {
    render(
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    );
  });

  return router;
};

test.each([
  { name: "router path from AuthRoute", redirect: "/filters", want: "/autobrr/filters", page: "filters" },
  { name: "full path from AuthenticatedLayout", redirect: "/autobrr/filters", want: "/autobrr/filters", page: "filters" },
  { name: "no redirect", redirect: undefined, want: "/autobrr/", page: "dashboard" },
  { name: "another origin", redirect: "//evil.example/", want: "/autobrr/", page: "dashboard" }
])("login under a base URL with $name lands on $want", async ({ redirect, want, page }) => {
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue({ enabled: false, autoRedirect: false, authorizationUrl: "", state: "", disableBuiltInLogin: false, issuerUrl: "" });
  vi.spyOn(APIClient.auth, "canOnboard").mockRejectedValue(new Error("onboarding unavailable"));

  const entry = redirect ? `/autobrr/login?redirect=${encodeURIComponent(redirect)}` : "/autobrr/login";
  const router = await renderLogin(entry);

  await act(async () => {
    AuthContext.set({ isLoggedIn: true, username: "admin", authMethod: "password" });
  });

  expect(await screen.findByText(page)).toBeTruthy();
  expect(router.history.location.pathname).toBe(want);
});
