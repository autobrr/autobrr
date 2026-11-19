/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Disclosure } from "@headlessui/react";
import { createMemoryHistory, createRootRoute, createRouter, RouterProvider } from "@tanstack/react-router";
import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import "@app/i18n";
import { MobileNav } from "@components/header/MobileNav";
import { AuthContext } from "@utils/Context";

afterEach(() => {
  cleanup();
  AuthContext.reset();
});

const forwardCapabilities: AuthCapabilities = { password_login: false, oidc_login: false, onboarding: false, account_management: false, api_key_management: true, proxy_guards: true };
const builtinCapabilities: AuthCapabilities = { password_login: true, oidc_login: false, onboarding: true, account_management: true, api_key_management: true, proxy_guards: false };

const renderMobileNav = async (logoutMutation: () => void) => {
  const rootRoute = createRootRoute({
    component: () => (
      <Disclosure defaultOpen>
        <MobileNav logoutMutation={logoutMutation} />
      </Disclosure>
    )
  });
  const router = createRouter({ routeTree: rootRoute, history: createMemoryHistory({ initialEntries: ["/"] }) });

  await act(async () => {
    render(<RouterProvider router={router} />);
  });
};

test("forward mode logs out through the proxy logout url", async () => {
  AuthContext.set({ isLoggedIn: true, username: "ze0s", authMethod: "forward", authMode: "forward", capabilities: forwardCapabilities, logoutUrl: "https://auth.example.com/logout" });
  const logout = vi.fn();

  await renderMobileNav(logout);

  const link = screen.getByRole("link", { name: /log out/i });
  expect(link.getAttribute("href")).toBe("https://auth.example.com/logout");
  expect(screen.queryByRole("button", { name: /log out/i })).toBeNull();
  expect(logout).not.toHaveBeenCalled();
});

test("forward mode without a logout url hides log out", async () => {
  AuthContext.set({ isLoggedIn: true, username: "ze0s", authMethod: "forward", authMode: "forward", capabilities: forwardCapabilities });

  await renderMobileNav(vi.fn());

  expect(screen.queryByText("Log out")).toBeNull();
});

test("a session mode logs out with the logout button", async () => {
  AuthContext.set({ isLoggedIn: true, username: "ze0s", authMethod: "password", authMode: "builtin", capabilities: builtinCapabilities });

  await renderMobileNav(vi.fn());

  expect(screen.getByRole("button", { name: /log out/i })).toBeTruthy();
  expect(screen.queryByRole("link", { name: /log out/i })).toBeNull();
});
