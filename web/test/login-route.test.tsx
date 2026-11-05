/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { isRedirect } from "@tanstack/react-router";
import { QueryClient } from "@tanstack/react-query";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import { SettingsKeys } from "@api/query_keys";
import { LoginRoute } from "@app/routes";
import { AuthContext } from "@utils/Context";

afterEach(() => {
  vi.restoreAllMocks();
  AuthContext.reset();
});

const config = (auth_mode: AuthMode, auth_capabilities: AuthCapabilities): Config => ({
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

const disabledConfig = config("disabled", { password_login: false, oidc_login: false, onboarding: false, account_management: false, api_key_management: false, proxy_guards: true });
const builtinConfig = config("builtin", { password_login: true, oidc_login: false, onboarding: true, account_management: true, api_key_management: true, proxy_guards: false });

const beforeLoad = async (search: { redirect?: string }) => {
  const navigate = vi.fn();
  const opts = { context: { queryClient: new QueryClient() }, navigate, search };

  try {
    await LoginRoute.options.beforeLoad?.(opts as never);
  } catch (err) {
    return { thrown: err, navigate };
  }

  return { thrown: undefined, navigate };
};

test.each([
  [undefined, "/"],
  ["/filters", "/filters"],
  ["/filters/12?tab=actions", "/filters/12?tab=actions"],
  ["https://evil.example.com/x", "/"],
  ["//evil.example/", "/"],
  ["/\\evil.example.com", "/"],
  ["/\t/evil.example", "/"],
  ["//[", "/"]
])("auth disabled: redirect %s enters the app at %s", async (redirect, want) => {
  vi.spyOn(APIClient.auth, "validate").mockResolvedValue({ username: "admin", auth_method: "disabled" });
  vi.spyOn(APIClient.config, "get").mockResolvedValue(disabledConfig);

  const { thrown } = await beforeLoad({ redirect });

  expect(isRedirect(thrown)).toBe(true);
  if (isRedirect(thrown)) {
    expect(thrown.options.href).toBe(want);
  }
  expect(AuthContext.get()).toMatchObject({ isLoggedIn: true, authMethod: "disabled", authMode: "disabled", capabilities: disabledConfig.auth_capabilities });
});

test("auth disabled: redirect check does not need URL.parse", async () => {
  vi.spyOn(APIClient.auth, "validate").mockResolvedValue({ username: "admin", auth_method: "disabled" });
  vi.spyOn(APIClient.config, "get").mockResolvedValue(disabledConfig);

  // Browsers before Chrome 126, Firefox 126 and Safari 17.4 have no static URL.parse.
  const parse = URL.parse;
  Object.defineProperty(URL, "parse", { value: undefined, configurable: true, writable: true });
  try {
    const { thrown } = await beforeLoad({ redirect: "/filters" });

    expect(isRedirect(thrown)).toBe(true);
    if (isRedirect(thrown)) {
      expect(thrown.options.href).toBe("/filters");
    }
  } finally {
    Object.defineProperty(URL, "parse", { value: parse, configurable: true, writable: true });
  }
});

test("auth enabled without a session renders the login form", async () => {
  vi.spyOn(APIClient.auth, "validate").mockRejectedValue(new Error("Forbidden"));
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue({ enabled: false, autoRedirect: false, authorizationUrl: "", state: "", disableBuiltInLogin: false, issuerUrl: "" });
  vi.spyOn(APIClient.auth, "canOnboard").mockRejectedValue(new Error("Forbidden"));

  const { thrown, navigate } = await beforeLoad({ redirect: "/filters" });

  expect(thrown).toBeUndefined();
  expect(navigate).not.toHaveBeenCalled();
  expect(AuthContext.get().isLoggedIn).toBe(false);
});

test("auth disabled: a config error still enters the app", async () => {
  vi.spyOn(APIClient.auth, "validate").mockResolvedValue({ username: "admin", auth_method: "disabled" });
  vi.spyOn(APIClient.config, "get").mockRejectedValue(new Error("Service Unavailable"));

  const { thrown } = await beforeLoad({ redirect: "/filters" });

  expect(isRedirect(thrown)).toBe(true);
  if (isRedirect(thrown)) {
    expect(thrown.options.href).toBe("/filters");
  }
  expect(AuthContext.get()).toMatchObject({ isLoggedIn: true, authMethod: "disabled", authMode: undefined });
});

test("a password session with a config error renders the login form", async () => {
  vi.spyOn(APIClient.auth, "validate").mockResolvedValue({ username: "admin", auth_method: "password" });
  vi.spyOn(APIClient.config, "get").mockRejectedValue(new Error("Service Unavailable"));
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue({ enabled: false, autoRedirect: false, authorizationUrl: "", state: "", disableBuiltInLogin: false, issuerUrl: "" });
  vi.spyOn(APIClient.auth, "canOnboard").mockRejectedValue(new Error("Forbidden"));

  const { thrown } = await beforeLoad({ redirect: "/filters" });

  expect(thrown).toBeUndefined();
  expect(AuthContext.get().isLoggedIn).toBe(false);
});

test("the login redirect does not trust a cached config from before a restart", async () => {
  vi.spyOn(APIClient.auth, "validate").mockResolvedValue({ username: "admin", auth_method: "disabled" });
  vi.spyOn(APIClient.config, "get").mockResolvedValue(disabledConfig);

  // The app's client keeps queries fresh for 30s, so a bare client would hide the bug.
  const client = new QueryClient({ defaultOptions: { queries: { staleTime: 30 * 1000 } } });
  client.setQueryData(SettingsKeys.config(), builtinConfig);
  const opts = { context: { queryClient: client }, navigate: vi.fn(), search: {} };

  let thrown;
  try {
    await LoginRoute.options.beforeLoad?.(opts as never);
  } catch (err) {
    thrown = err;
  }

  expect(isRedirect(thrown)).toBe(true);
});

test("a session in a mode with a login form renders the login form", async () => {
  vi.spyOn(APIClient.auth, "validate").mockResolvedValue({ username: "admin", auth_method: "password" });
  vi.spyOn(APIClient.config, "get").mockResolvedValue(builtinConfig);
  vi.spyOn(APIClient.auth, "getOIDCConfig").mockResolvedValue({ enabled: false, autoRedirect: false, authorizationUrl: "", state: "", disableBuiltInLogin: false, issuerUrl: "" });
  vi.spyOn(APIClient.auth, "canOnboard").mockRejectedValue(new Error("Forbidden"));

  const { thrown } = await beforeLoad({ redirect: "/filters" });

  expect(thrown).toBeUndefined();
  expect(AuthContext.get().isLoggedIn).toBe(false);
});

test("forward mode continues into the app", async () => {
  vi.spyOn(APIClient.auth, "validate").mockResolvedValue({ username: "ze0s", auth_method: "forward" });
  vi.spyOn(APIClient.config, "get").mockResolvedValue(config("forward", { password_login: false, oidc_login: false, onboarding: false, account_management: false, api_key_management: true, proxy_guards: true }));

  const { thrown } = await beforeLoad({ redirect: "/filters" });

  expect(isRedirect(thrown)).toBe(true);
  if (isRedirect(thrown)) {
    expect(thrown.options.href).toBe("/filters");
  }
  expect(AuthContext.get()).toMatchObject({ isLoggedIn: true, username: "ze0s", authMethod: "forward", authMode: "forward" });
});
