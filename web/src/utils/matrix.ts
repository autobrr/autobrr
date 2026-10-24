/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

export function isMatrixServer(server?: string): boolean {
  const s = (server || "").trim().toLowerCase();
  return (
    s.startsWith("matrix.") ||
    s.startsWith("matrix://") ||
    s.includes(".matrix.") ||
    s.endsWith(".watchtower.pm")
  );
}
