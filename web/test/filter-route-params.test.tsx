/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { isNotFound } from "@tanstack/react-router";
import { expect, test } from "vitest";

import { FilterGetByIdRoute } from "@app/routes";

const parse = (filterId: string) => FilterGetByIdRoute.options.parseParams?.({ filterId });

test("numeric filter id parses to a number", () => {
  expect(parse("42")).toEqual({ filterId: 42 });
});

test.each(["does-not-exist", "1.5", "NaN", "Infinity"])("filter id %s throws notFound instead of a parse error", (filterId) => {
  let thrown: unknown;
  try {
    parse(filterId);
  } catch (err) {
    thrown = err;
  }
  expect(isNotFound(thrown)).toBe(true);
});
