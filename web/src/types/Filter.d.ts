/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

interface Filter {
  id: number;
  name: string;
  enabled: boolean;
  created_at: Date;
  updated_at: Date;
  min_size: string;
  max_size: string;
  delay: number;
  priority: number;
  max_downloads: number;
  max_downloads_unit: string;
  match_releases: string;
  except_releases: string;
  use_regex: boolean;
  match_release_groups: string;
  except_release_groups: string;
  match_release_tags: string;
  except_release_tags: string;
  use_regex_release_tags: boolean;
  match_description: string;
  except_description: string;
  use_regex_description: boolean;
  scene: boolean;
  origins: string[];
  except_origins: string[];
  freeleech: boolean;
  freeleech_percent: string;
  shows: string;
  seasons: string;
  episodes: string;
  smart_episode: boolean;
  unique_download: boolean;
  resolutions: string[];
  codecs: string[];
  sources: string[];
  containers: string[];
  match_hdr: string[];
  except_hdr: string[];
  match_other: string[];
  except_other: string[];
  years: string;
  artists: string;
  albums: string;
  match_release_types: string[];
  except_release_types: string[];
  formats: string[];
  quality: string[];
  media: string[];
  perfect_flac: boolean;
  cue: boolean;
  log: boolean;
  log_score: string;
  match_categories: string;
  except_categories: string;
  match_uploaders: string;
  except_uploaders: string;
  match_language: string[];
  except_language: string[];
  tags: string;
  except_tags: string;
  tags_any: string;
  except_tags_any: string;
  tags_match_logic: string;
  except_tags_match_logic: string;
  actions_count: number;
  actions: Action[];
  indexers: Indexer[];
  external_script_enabled: boolean;
  external_script_cmd: string;
  external_script_args: string;
  external_script_expect_status: number;
  external_webhook_enabled: boolean;
  external_webhook_host: string;
  external_webhook_data: string;
  external_webhook_expect_status: number;
}

interface Action {
  id: number;
  name: string;
  type: ActionType;
  enabled: boolean;
  exec_cmd?: string;
  exec_args?: string;
  watch_folder?: string;
  category?: string;
  tags?: string;
  label?: string;
  save_path?: string;
  paused?: boolean;
  ignore_rules?: boolean;
  skip_hash_check: boolean;
  content_layout?: ActionContentLayout;
  limit_upload_speed?: number;
  limit_download_speed?: number;
  limit_ratio?: number;
  limit_seed_time?: number;
  reannounce_skip: boolean;
  reannounce_delete: boolean;
  reannounce_interval: number;
  reannounce_max_attempts: number;
  webhook_host: string,
  webhook_type: string;
  webhook_method: string;
  webhook_data: string,
  webhook_headers: string[];
  filter_id?: number;
  client_id?: number;
}

type ActionContentLayout = "ORIGINAL" | "SUBFOLDER_CREATE" | "SUBFOLDER_NONE";

type ActionType = "TEST" | "EXEC" | "WATCH_FOLDER" | "WEBHOOK" | DownloadClientType;
