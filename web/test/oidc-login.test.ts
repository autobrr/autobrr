/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient } from "@tanstack/react-query";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import { AuthKeys } from "@api/query_keys";
import { LoginRoute, OnboardRoute, Router } from "@app/routes";
import { claimAutoRedirect, consumeLoggedOut, markLoggedOut } from "@utils/autoLogin";

afterEach(() => {
  vi.restoreAllMocks();
  sessionStorage.clear();
});

const config = {
  enabled: true,
  autoRedirect: true,
  authorizationUrl: "https://auth.example.com/authorize?state=fresh-state",
  state: "fresh-state",
  disableBuiltInLogin: false,
  issuerUrl: "https://auth.example.com"
};

const loadLogin = (queryClient = new QueryClient(), preload = false, search: { redirect?: string; autoRedirect?: boolean } = {}) => {
  const beforeLoad = LoginRoute.options.beforeLoad;
  if (typeof beforeLoad !== "function") {
    throw new Error("Login route has no beforeLoad handler");
  }
  const navigate = vi.fn();
  const result = beforeLoad({
    context: { queryClient },
    navigate,
    preload,
    abortController: new AbortController(),
    params: {},
    search,
    location: Router.state.location,
    buildLocation: Router.buildLocation,
    cause: preload ? "preload" : "enter",
    matches: [],
    routeId: LoginRoute.id
  });
  return { result, navigate };
};

test("automatic login redirects the document to the provider", async () => {
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue(config);
  const canOnboard = vi.spyOn(APIClient.auth, "canOnboard");

  await expect(loadLogin().result).rejects.toMatchObject({
    options: { href: config.authorizationUrl, reloadDocument: true }
  });
  expect(canOnboard).not.toHaveBeenCalled();
});

test.each([
  { name: "automatic redirection is disabled", overrides: { autoRedirect: false } },
  { name: "the authorization URL is missing", overrides: { authorizationUrl: "" } }
])("keeps the login page when $name", async ({ overrides }) => {
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue({ ...config, ...overrides });
  const { result, navigate } = loadLogin();

  await expect(result).resolves.toBeUndefined();
  expect(navigate).not.toHaveBeenCalled();
});

test("preloading the login route neither starts OIDC login nor replaces its state", async () => {
  const getConfig = vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue(config);

  await expect(loadLogin(new QueryClient(), true).result).resolves.toBeUndefined();
  expect(getConfig).not.toHaveBeenCalled();
});

test("the login page still renders when the OIDC config fails to load", async () => {
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockRejectedValue(new Error("provider unavailable"));
  vi.spyOn(APIClient.auth, "canOnboard").mockRejectedValue(new Error("onboarding unavailable"));
  const { result, navigate } = loadLogin();

  await expect(result).resolves.toBeUndefined();
  expect(navigate).not.toHaveBeenCalled();
});

test("blocked storage turns automatic login off instead of failing the login page", async () => {
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue(config);
  vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
    throw new DOMException("storage is disabled", "SecurityError");
  });

  await expect(loadLogin().result).resolves.toEqual({ autoLoginFailed: true });
});

test("autoRedirect=false keeps the login page for logout and provider outages", async () => {
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue(config);
  const { result, navigate } = loadLogin(new QueryClient(), false, { autoRedirect: false });

  await expect(result).resolves.toBeUndefined();
  expect(navigate).not.toHaveBeenCalled();
});

test.each([
  { redirect: "/filters/12?tab=actions", want: "/filters/12?tab=actions" },
  { redirect: "https://evil.example/", want: undefined },
  { redirect: undefined, want: undefined }
])("login asks the server to return to $redirect after OIDC login", async ({ redirect, want }) => {
  const getConfig = vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue(config);

  await expect(loadLogin(new QueryClient(), false, { redirect }).result).rejects.toBeTruthy();
  expect(getConfig).toHaveBeenCalledWith(want);
});

test("the login page after a logout drops automatic login and keeps the redirect", async () => {
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue(config);
  markLoggedOut();

  await expect(loadLogin(new QueryClient(), false, { redirect: "/filters" }).result).rejects.toMatchObject({
    options: { to: LoginRoute.to, search: { redirect: "/filters", autoRedirect: false }, replace: true }
  });
  expect(consumeLoggedOut()).toBe(false);
});

test("preloading the login route keeps a pending logout", async () => {
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue(config);
  markLoggedOut();

  await expect(loadLogin(new QueryClient(), true).result).resolves.toBeUndefined();
  expect(consumeLoggedOut()).toBe(true);
});

test("returning without a session right after an automatic login shows the login page", async () => {
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue(config);
  const now = vi.spyOn(Date, "now").mockReturnValue(1_000_000);

  await expect(loadLogin().result).rejects.toMatchObject({ options: { href: config.authorizationUrl } });
  now.mockReturnValue(1_000_000 + 5_000);
  await expect(loadLogin().result).resolves.toEqual({ autoLoginFailed: true });

  now.mockReturnValue(1_000_000 + 60_000);
  await expect(loadLogin().result).rejects.toMatchObject({ options: { href: config.authorizationUrl } });
});

test("an automatic login that never ran does not hold back the next one", async () => {
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue({ ...config, autoRedirect: false });
  await loadLogin().result;

  expect(claimAutoRedirect()).toBe(true);
});

test("login fetches fresh authorization state after a session changes", async () => {
  const queryClient = new QueryClient();
  queryClient.setQueryData(AuthKeys.oidcConfig(), { ...config, autoRedirect: false, state: "old-state" });
  const getConfig = vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue(config);

  await expect(loadLogin(queryClient).result).rejects.toMatchObject({
    options: { href: config.authorizationUrl }
  });
  expect(getConfig).toHaveBeenCalledOnce();
});

test("a fresh authenticated response cancels a cached automatic redirect", async () => {
  const queryClient = new QueryClient();
  queryClient.setQueryData(AuthKeys.oidcConfig(), config);
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue({ ...config, autoRedirect: false });

  await expect(loadLogin(queryClient).result).resolves.toBeUndefined();
});

test("OIDC disabled still allows onboarding", async () => {
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue({ ...config, enabled: false });
  vi.spyOn(APIClient.auth, "canOnboard").mockResolvedValue(undefined);
  const { result, navigate } = loadLogin();

  await result;
  expect(navigate).toHaveBeenCalledWith({ to: OnboardRoute.to });
});

test("OIDC disabled keeps the password login for an existing installation", async () => {
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue({ ...config, enabled: false });
  vi.spyOn(APIClient.auth, "canOnboard").mockRejectedValue(new Error("onboarding unavailable"));
  const { result, navigate } = loadLogin();

  await result;
  expect(navigate).not.toHaveBeenCalled();
});
