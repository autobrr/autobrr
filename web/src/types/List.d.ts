/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

interface List {
  id: number;
  name: string;
  enabled: boolean;
  type: ListType;
  client_id: number;
  url: string;
  headers: string[];
  api_key: string;
  filters: ListFilter[];
  match_release: boolean;
  tags_included: string[];
  tags_excluded: string[];
  include_unmonitored: boolean;
  include_alternate_titles: boolean;
}

interface ListFilter {
  id: number;
  name: string;
}

interface ListCreate {
  name: string;
  enabled: boolean;
  type: ListType;
  client_id: number;
  url: string;
  headers: string[];
  api_key: string;
  filters: number[];
  match_release: boolean;
  tags_include: string[];
  tags_exclude: string[];
  include_unmonitored: boolean;
  include_alternate_titles: boolean;
}

type ListType =
  | "SONARR"
  | "RADARR"
  | "LIDARR"
  | "READARR"
  | "WHISPARR"
  | "MDBLIST"
  | "TRAKT"
  | "METACRITIC"
  | "STEAM"
  | "PLAINTEXT"
  | "ANILIST";
