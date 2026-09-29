/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { isNotFound } from "@tanstack/react-router";
import { expect, test } from "vitest";

import { FilterGetByIdRoute } from "@app/routes";

const parse = (filterId: string) => FilterGetByIdRoute.options.params?.parse?.({ filterId });

test("numeric filter id parses to a number", () => {
  expect(parse("42")).toEqual({ filterId: 42 });
});

test.each(["does-not-exist", "1.5", "NaN", "Infinity", "1e3", "0x10", " ", "", "-1", "+5", "01", "0", "99999999999999999999"])("filter id %s throws notFound instead of a parse error", (filterId) => {
  let thrown: unknown;
  try {
    parse(filterId);
  } catch (err) {
    thrown = err;
  }
  expect(isNotFound(thrown)).toBe(true);
});
