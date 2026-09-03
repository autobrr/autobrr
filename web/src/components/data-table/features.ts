/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { columnFilteringFeature, columnVisibilityFeature, rowPaginationFeature, tableFeatures } from "@tanstack/react-table";

export const dataTableFeatures = tableFeatures({
  columnFilteringFeature,
  columnVisibilityFeature,
  rowPaginationFeature,
  columnMeta: {} as {
    filterVariant?: "text" | "range" | "select" | "search" | "actionPushStatus" | "indexerSelect";
  }
});

export type DataTableFeatures = typeof dataTableFeatures;
