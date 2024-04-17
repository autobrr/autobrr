/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
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
  api_key: string;
  cookie: string;
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
  indexer_id: number;
  settings: FeedSettings;
}
