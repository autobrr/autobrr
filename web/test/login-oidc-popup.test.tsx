/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import { Login } from "@screens/auth/Login";
import { OIDCLoginChannel } from "@utils";
import { AuthContext } from "@utils/Context";
import "@app/i18n";

const authorizationUrl = "https://idp.example.invalid/authorize";
let configFetches = 0;
const router = { invalidate: vi.fn(), history: { push: vi.fn() } };

vi.mock("@app/logo.svg?react", () => ({ default: () => null }));
vi.mock("@tanstack/react-router", () => ({
  getRouteApi: () => ({ useSearch: () => ({}) }),
  useRouter: () => router
}));
vi.mock("@api/APIClient", () => ({
  APIClient: {
    auth: {
      getOIDCConfig: vi.fn(async () => ({ enabled: true, authorizationUrl: `${authorizationUrl}?fetch=${++configFetches}`, state: "s", disableBuiltInLogin: true, issuerUrl: "" })),
      canOnboard: vi.fn(async () => { throw new Error("user exists"); }),
      validate: vi.fn(async () => ({ username: "oidc-user", auth_method: "oidc" }))
    }
  }
}));

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const broadcastLogin = async () => {
  const channel = new BroadcastChannel(OIDCLoginChannel);
  channel.postMessage("login");
  channel.close();
  await act(() => new Promise((resolve) => setTimeout(resolve, 50)));
};

test("home screen app logs in through a popup with a fresh state and closes it", async () => {
  const matchMedia = window.matchMedia;
  vi.spyOn(window, "matchMedia").mockImplementation((query) => ({ ...matchMedia(query), matches: query === "(display-mode: standalone)" }));
  const popup = { close: vi.fn(), location: { href: "about:blank" } } as unknown as Window;
  const open = vi.spyOn(window, "open").mockReturnValue(popup);

  render(<QueryClientProvider client={new QueryClient()}><Login /></QueryClientProvider>);
  const button = await screen.findByText("OpenID Connect");

  await broadcastLogin();
  expect(APIClient.auth.validate).not.toHaveBeenCalled();

  fireEvent.click(button);
  expect(open).toHaveBeenCalledWith("about:blank", "autobrr-oidc");
  await waitFor(() => expect(popup.location.href).toBe(`${authorizationUrl}?fetch=2`));

  await broadcastLogin();
  expect(popup.close).toHaveBeenCalledOnce();
  await waitFor(() => expect(AuthContext.get().isLoggedIn).toBe(true));
  expect(APIClient.auth.validate).toHaveBeenCalledOnce();
});
