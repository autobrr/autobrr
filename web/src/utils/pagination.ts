/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

export type PageItem = number | "gap";

// paginationRange lists the zero-based pages to render: always the first and last page plus
// `siblings` pages around the current one, with "gap" where pages are skipped.
export const paginationRange = (current: number, count: number, siblings = 2): PageItem[] => {
  if (count <= siblings * 2 + 5) {
    return Array.from({ length: count }, (_, idx) => idx);
  }

  const start = Math.max(1, Math.min(current - siblings, count - siblings * 2 - 2));
  const end = Math.min(count - 2, Math.max(current + siblings, siblings * 2 + 1));

  const items: PageItem[] = [0];
  if (start > 1) {
    items.push("gap");
  }
  for (let page = start; page <= end; page++) {
    items.push(page);
  }
  if (end < count - 2) {
    items.push("gap");
  }
  items.push(count - 1);

  return items;
};
