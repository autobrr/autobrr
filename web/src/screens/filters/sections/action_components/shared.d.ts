/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

interface ClientActionProps {
  idx: number;
  action: Action;
  clients: DownloadClient[];
}
