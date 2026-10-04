/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { expect, test } from "vitest";

import { paginationRange } from "@utils/pagination";

test("paginationRange lists every page when there are few", () => {
  expect(paginationRange(0, 1)).toEqual([0]);
  expect(paginationRange(3, 9)).toEqual([0, 1, 2, 3, 4, 5, 6, 7, 8]);
});

test("paginationRange keeps a stable window with gaps", () => {
  expect(paginationRange(0, 40)).toEqual([0, 1, 2, 3, 4, 5, "gap", 39]);
  expect(paginationRange(20, 40)).toEqual([0, "gap", 18, 19, 20, 21, 22, "gap", 39]);
  expect(paginationRange(39, 40)).toEqual([0, "gap", 34, 35, 36, 37, 38, 39]);
});
