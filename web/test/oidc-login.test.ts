/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient } from "@tanstack/react-query";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import { AuthKeys } from "@api/query_keys";
import { LoginRoute, OnboardRoute, Router } from "@app/routes";

afterEach(() => {
  vi.restoreAllMocks();
});

const config = {
  enabled: true,
  autoRedirect: true,
  authorizationUrl: "https://auth.example.com/authorize?state=fresh-state",
  state: "fresh-state",
  disableBuiltInLogin: false,
  issuerUrl: "https://auth.example.com"
};

const loadLogin = (queryClient = new QueryClient(), preload = false) => {
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
    search: {},
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

test("preloading the login route does not start OIDC login", async () => {
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue(config);

  await expect(loadLogin(new QueryClient(), true).result).resolves.toBeUndefined();
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
