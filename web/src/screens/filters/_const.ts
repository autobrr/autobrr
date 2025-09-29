/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

export const FILTER_FIELDS: Record<string, string> = {
  "id": "number",
  "enabled": "boolean",
  "delay": "number",
  "priority": "number",
  "log_score": "number",
  "max_downloads": "number",
  "use_regex": "boolean",
  "scene": "boolean",
  "smart_episode": "boolean",
  "freeleech": "boolean",
  "perfect_flac": "boolean",
  "download_duplicates": "boolean",
  "cue": "boolean",
  "log": "boolean",
  "match_releases": "string",
  "except_releases": "string",
  "match_release_groups": "string",
  "except_release_groups": "string",
  "shows": "string",
  "seasons": "string",
  "episodes": "string",
  "years": "string",
  "artists": "string",
  "albums": "string",
  "except_release_types": "string",
  "match_categories": "string",
  "except_categories": "string",
  "match_uploaders": "string",
  "except_uploaders": "string",
  "tags": "string",
  "except_tags": "string",
  "match_sites": "string",
  "except_sites": "string",
  "origins": "[]string",
  "except_origins": "[]string",
  "bonus": "[]string",
  "resolutions": "[]string",
  "codecs": "[]string",
  "sources": "[]string",
  "containers": "[]string",
  "match_hdr": "[]string",
  "except_hdr": "[]string",
  "match_other": "[]string",
  "except_other": "[]string",
  "match_release_types": "[]string",
  "tags_any": "boolean",
  "except_tags_any": "boolean",
  "formats": "[]string",
  "quality": "[]string",
  "media": "[]string",
  "min_seeders": "number",
  "max_seeders": "number",
  "min_leechers": "number",
  "max_leechers": "number",
} as const;

export const IRC_FIELDS: Record<string, string> = {
  "enabled": "boolean",
  "port": "number",
  "tls": "boolean"
} as const;

export const IRC_SUBSTITUTION_MAP: Record<string, string> = {
  "ssl": "tls",
  "nick": "nickserv_account",
  "ident_password": "nickserv_password",
  "server-password": "pass"
} as const;

export const FILTER_SUBSTITUTION_MAP: Record<string, string> = {
  "freeleech_percents": "freeleech_percent",
  "encoders": "codecs",
  "bitrates": "quality",
  "max_downloads_per": "max_downloads_unit",
  "log_scores": "log_score",
  "upload_delay_secs": "delay"
} as const;
