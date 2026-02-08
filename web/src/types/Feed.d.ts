/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

interface Feed {
  id: number;
  indexer: IndexerMinimal;
  name: string;
  type: FeedType;
  enabled: boolean;
  url: string;
  interval: number;
  timeout: number;
  max_age: number;
  categories: number[];
  capabilities: FeedCaps | null;
  api_key: string;
  cookie: string;
  tls_skip_verify: boolean;
  last_run: string;
  last_run_data: string;
  next_run: string;
  settings: FeedSettings;
  created_at: Date;
  updated_at: Date;
}

interface FeedSettings {
  download_type: FeedDownloadType;
  // download_type: string;
}

type FeedDownloadType = "MAGNET" | "TORRENT";

type FeedType = "TORZNAB" | "NEWZNAB" | "RSS";

interface FeedCreate {
  name: string;
  type: FeedType;
  enabled: boolean;
  url: string;
  interval: number;
  timeout: number;
  api_key?: string;
  tls_skip_verify?: boolean;
  indexer_id: number;
  categories?: number[];
  capabilities?: FeedCaps | null;
  settings: FeedSettings;
}

interface FeedCapsLimits {
  max: string;
  default: string;
}

interface FeedCapsCategory {
  id: number;
  name: string;
  subcategories: FeedCapsCategory[] | null;
}

interface FeedCaps {
  limits: FeedCapsLimits;
  categories: FeedCapsCategory[];
}

interface FeedCapsRequest {
  type: FeedType;
  url: string;
  api_key?: string;
  timeout?: number;
}
