/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { isRedirect } from "@tanstack/react-router";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import { queryClient } from "@api/QueryClient";
import { AuthKeys, SettingsKeys } from "@api/query_keys";
import { AuthRoute } from "@app/routes";
import { AuthContext } from "@utils/Context";

afterEach(() => {
  vi.restoreAllMocks();
  AuthContext.reset();
  queryClient.clear();
});

const capabilities: Record<AuthMode, AuthCapabilities> = {
  builtin: { password_login: true, oidc_login: false, onboarding: true, account_management: true, api_key_management: true, proxy_guards: false },
  oidc: { password_login: true, oidc_login: true, onboarding: false, account_management: true, api_key_management: true, proxy_guards: false },
  forward: { password_login: false, oidc_login: false, onboarding: false, account_management: false, api_key_management: true, proxy_guards: true },
  disabled: { password_login: false, oidc_login: false, onboarding: false, account_management: false, api_key_management: false, proxy_guards: true }
};

const config = (auth_mode: AuthMode, auth_capabilities = capabilities[auth_mode]): Config => ({
  config_dir: "/config",
  application: "/usr/bin/autobrr",
  database: "sqlite",
  host: "127.0.0.1",
  port: 7474,
  log_level: "INFO",
  log_path: "",
  log_max_size: 50,
  log_max_backups: 3,
  base_url: "/",
  check_for_updates: false,
  auth_mode,
  auth_capabilities,
  version: "dev",
  commit: "",
  date: ""
});

const beforeLoad = async () => {
  const opts = { context: { queryClient }, location: { href: "/settings", publicHref: "/settings" } };

  try {
    await AuthRoute.options.beforeLoad?.(opts as never);
  } catch (err) {
    return err;
  }

  return undefined;
};

const disabledAuth = { isLoggedIn: true, username: "admin", authMethod: "disabled", authMode: "disabled", capabilities: capabilities.disabled } as const;

test("logging in resolves the mode and capabilities", async () => {
  vi.spyOn(APIClient.auth, "validate").mockResolvedValue({ username: "ze0s", auth_method: "password" });
  vi.spyOn(APIClient.config, "get").mockResolvedValue(config("builtin"));

  const thrown = await beforeLoad();

  expect(thrown).toBeUndefined();
  expect(AuthContext.get()).toMatchObject({ isLoggedIn: true, username: "ze0s", authMethod: "password", authMode: "builtin", capabilities: capabilities.builtin });
});

test("stale cached disabled mode is revalidated after auth is restored", async () => {
  AuthContext.set(disabledAuth);

  queryClient.setQueryData(SettingsKeys.config(), config("disabled"), { updatedAt: Date.now() - 60 * 1000 });

  const getConfig = vi.spyOn(APIClient.config, "get").mockResolvedValue(config("builtin"));
  vi.spyOn(APIClient.auth, "validate").mockResolvedValue({ username: "admin", auth_method: "password" });

  const thrown = await beforeLoad();

  expect(thrown).toBeUndefined();
  expect(getConfig).toHaveBeenCalledOnce();
  expect(AuthContext.get()).toMatchObject({ isLoggedIn: true, authMethod: "password", authMode: "builtin", capabilities: capabilities.builtin });
});

test("switching to disabled mode replaces the session identity", async () => {
  AuthContext.set({ isLoggedIn: true, username: "ze0s", authMethod: "password", authMode: "builtin", capabilities: capabilities.builtin });

  queryClient.setQueryData(SettingsKeys.config(), config("disabled"));
  vi.spyOn(APIClient.auth, "validate").mockResolvedValue({ username: "admin", auth_method: "disabled" });

  const thrown = await beforeLoad();

  expect(thrown).toBeUndefined();
  expect(AuthContext.get()).toMatchObject({ username: "admin", authMethod: "disabled", authMode: "disabled", capabilities: capabilities.disabled });
});

test("a context stored before modes were tracked is revalidated once", async () => {
  AuthContext.set({ isLoggedIn: true, username: "ze0s", authMethod: "password" });

  queryClient.setQueryData(SettingsKeys.config(), config("builtin"));
  const validate = vi.spyOn(APIClient.auth, "validate").mockResolvedValue({ username: "ze0s", auth_method: "password" });

  expect(await beforeLoad()).toBeUndefined();
  expect(await beforeLoad()).toBeUndefined();

  expect(validate).toHaveBeenCalledOnce();
  expect(AuthContext.get()).toMatchObject({ authMode: "builtin", capabilities: capabilities.builtin });
});

test("a mode switch without a surviving session goes to login", async () => {
  AuthContext.set(disabledAuth);

  queryClient.setQueryData(SettingsKeys.config(), config("builtin"));
  vi.spyOn(APIClient.auth, "validate").mockRejectedValue(new Error("Forbidden"));

  const thrown = await beforeLoad();

  expect(isRedirect(thrown)).toBe(true);
  if (isRedirect(thrown)) {
    expect(thrown.options.to).toBe("/login");
    expect(thrown.options.search).toEqual({ redirect: "/settings" });
  }
  expect(AuthContext.get().isLoggedIn).toBe(false);
});

test("an issuer lookup error keeps the valid oidc session", async () => {
  vi.spyOn(APIClient.auth, "validate").mockResolvedValue({ username: "ze0s", auth_method: "oidc" });
  vi.spyOn(APIClient.config, "get").mockResolvedValue(config("oidc"));
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockRejectedValue(new Error("Service Unavailable"));
  // The app client retries with backoff before giving up; skip that wait here.
  queryClient.setQueryDefaults(AuthKeys.oidcConfig(), { retry: false });

  expect(await beforeLoad()).toBeUndefined();
  expect(AuthContext.get()).toMatchObject({ isLoggedIn: true, authMethod: "oidc", authMode: "oidc", issuerUrl: undefined });
});

test("a config error at login keeps the valid session", async () => {
  vi.spyOn(APIClient.auth, "validate").mockResolvedValue({ username: "ze0s", auth_method: "password" });
  const getConfig = vi.spyOn(APIClient.config, "get").mockRejectedValueOnce(new Error("Service Unavailable"));

  expect(await beforeLoad()).toBeUndefined();
  expect(AuthContext.get()).toMatchObject({ isLoggedIn: true, username: "ze0s", authMethod: "password", authMode: undefined });

  getConfig.mockResolvedValue(config("builtin"));

  expect(await beforeLoad()).toBeUndefined();
  expect(AuthContext.get()).toMatchObject({ authMode: "builtin", capabilities: capabilities.builtin });
});

test("capability changes within a mode update the context without validating", async () => {
  AuthContext.set({ isLoggedIn: true, username: "ze0s", authMethod: "oidc", authMode: "oidc", capabilities: capabilities.oidc });

  const oidcOnly = { ...capabilities.oidc, password_login: false };
  queryClient.setQueryData(SettingsKeys.config(), config("oidc", oidcOnly));
  const validate = vi.spyOn(APIClient.auth, "validate");

  const thrown = await beforeLoad();

  expect(thrown).toBeUndefined();
  expect(validate).not.toHaveBeenCalled();
  expect(AuthContext.get()).toMatchObject({ authMethod: "oidc", capabilities: oidcOnly });
});

test("fresh cached config is reused without a request", async () => {
  AuthContext.set(disabledAuth);

  queryClient.setQueryData(SettingsKeys.config(), config("disabled"));

  const getConfig = vi.spyOn(APIClient.config, "get").mockResolvedValue(config("builtin"));

  const thrown = await beforeLoad();

  expect(thrown).toBeUndefined();
  expect(getConfig).not.toHaveBeenCalled();
  expect(AuthContext.get().authMethod).toBe("disabled");
});

test("forward mode resolves the proxy user and logout url", async () => {
  vi.spyOn(APIClient.auth, "validate").mockResolvedValue({ username: "ze0s", auth_method: "forward" });
  vi.spyOn(APIClient.config, "get").mockResolvedValue({ ...config("forward"), auth_logout_url: "https://auth.example.com/logout" });

  const thrown = await beforeLoad();

  expect(thrown).toBeUndefined();
  expect(AuthContext.get()).toMatchObject({ username: "ze0s", authMethod: "forward", authMode: "forward", logoutUrl: "https://auth.example.com/logout" });
});

test("a changed logout url updates the context without validating", async () => {
  AuthContext.set({ isLoggedIn: true, username: "ze0s", authMethod: "forward", authMode: "forward", capabilities: capabilities.forward });

  queryClient.setQueryData(SettingsKeys.config(), { ...config("forward"), auth_logout_url: "https://auth.example.com/logout" });
  const validate = vi.spyOn(APIClient.auth, "validate");

  expect(await beforeLoad()).toBeUndefined();

  expect(validate).not.toHaveBeenCalled();
  expect(AuthContext.get().logoutUrl).toBe("https://auth.example.com/logout");
});
