/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createMemoryHistory, createRootRoute, createRoute, createRouter, Outlet, RouterProvider } from "@tanstack/react-router";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { z } from "zod/mini";

import "@app/i18n";
import { APIClient } from "@api/APIClient";
import { AuthenticatedLayout } from "@components/layouts";
import { AuthContext } from "@utils/Context";
import { consumeLoggedOut } from "@utils/autoLogin";

vi.mock("@hooks/useInbox", () => ({ useInboxEvents: () => {} }));
vi.mock("@components/header/InboxMenu", () => ({ InboxMenu: () => null }));
vi.mock("@components/header/AlertBanner", () => ({ AlertBanner: () => null }));
vi.mock("@components/header/LeftNav", () => ({ LeftNav: () => null }));
vi.mock("@components/header/MobileNav", () => ({ MobileNav: () => null }));

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  AuthContext.reset();
});

const renderApp = async () => {
  const rootRoute = createRootRoute({ component: Outlet });
  const indexRoute = createRoute({ getParentRoute: () => rootRoute, path: "/", component: AuthenticatedLayout });
  const loginRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "login",
    validateSearch: z.object({
      redirect: z.optional(z.string()),
      autoRedirect: z.optional(z.boolean())
    }),
    component: () => <p>login page</p>
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([indexRoute, loginRoute]),
    history: createMemoryHistory({ initialEntries: ["/"] })
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

const clickLogout = async () => {
  fireEvent.click(screen.getByRole("button", { name: /open user menu/i }));
  fireEvent.click(await screen.findByText("Log out"));
};

test.each([
  { name: "logout", logout: () => vi.spyOn(APIClient.auth, "logout").mockResolvedValue(undefined as never) },
  { name: "logout with an expired session", logout: () => vi.spyOn(APIClient.auth, "logout").mockRejectedValue(new Error("Cookie expired or invalid.")) }
])("$name tells the login page not to start OIDC login", async ({ logout }) => {
  AuthContext.set({ isLoggedIn: true, username: "admin", authMethod: "password" });
  logout();

  const router = await renderApp();
  await clickLogout();

  expect(await screen.findByText("login page")).toBeTruthy();
  await waitFor(() => expect(AuthContext.get().isLoggedIn).toBe(false));
  expect(router.state.location.pathname).toBe("/login");
  expect(consumeLoggedOut()).toBe(true);
});
