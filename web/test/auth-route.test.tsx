/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import { queryClient } from "@api/QueryClient";
import { SettingsKeys } from "@api/query_keys";
import { AuthRoute } from "@app/routes";
import { AuthContext } from "@utils/Context";

afterEach(() => {
  vi.restoreAllMocks();
  AuthContext.reset();
  queryClient.clear();
});

const config = (auth_mode: AuthMode): Config => ({
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
  version: "dev",
  commit: "",
  date: ""
});

const beforeLoad = async () => {
  const opts = { context: { queryClient }, location: { href: "/settings" } };

  try {
    await AuthRoute.options.beforeLoad?.(opts as never);
  } catch (err) {
    return err;
  }

  return undefined;
};

test("stale cached disabled mode is revalidated after auth is restored", async () => {
  AuthContext.set({ isLoggedIn: true, username: "admin", authMethod: "disabled" });

  queryClient.setQueryData(SettingsKeys.config(), config("disabled"), { updatedAt: Date.now() - 60 * 1000 });

  const getConfig = vi.spyOn(APIClient.config, "get").mockResolvedValue(config("builtin"));
  vi.spyOn(APIClient.auth, "validate").mockResolvedValue({ username: "admin", auth_method: "password" });

  const thrown = await beforeLoad();

  expect(thrown).toBeUndefined();
  expect(getConfig).toHaveBeenCalledOnce();
  expect(AuthContext.get()).toMatchObject({ isLoggedIn: true, authMethod: "password" });
});

test("fresh cached config is reused without a request", async () => {
  AuthContext.set({ isLoggedIn: true, username: "admin", authMethod: "disabled" });

  queryClient.setQueryData(SettingsKeys.config(), config("disabled"));

  const getConfig = vi.spyOn(APIClient.config, "get").mockResolvedValue(config("builtin"));

  const thrown = await beforeLoad();

  expect(thrown).toBeUndefined();
  expect(getConfig).not.toHaveBeenCalled();
  expect(AuthContext.get().authMethod).toBe("disabled");
});
