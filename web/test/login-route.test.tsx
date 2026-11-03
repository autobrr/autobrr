/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { isRedirect } from "@tanstack/react-router";
import { QueryClient } from "@tanstack/react-query";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import { LoginRoute } from "@app/routes";
import { AuthContext } from "@utils/Context";

afterEach(() => {
  vi.restoreAllMocks();
  AuthContext.reset();
});

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

  const { thrown } = await beforeLoad({ redirect });

  expect(isRedirect(thrown)).toBe(true);
  if (isRedirect(thrown)) {
    expect(thrown.options.href).toBe(want);
  }
  expect(AuthContext.get()).toMatchObject({ isLoggedIn: true, authMethod: "disabled" });
});

test("auth disabled: redirect check does not need URL.parse", async () => {
  vi.spyOn(APIClient.auth, "validate").mockResolvedValue({ username: "admin", auth_method: "disabled" });

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
