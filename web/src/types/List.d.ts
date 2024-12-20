/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
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
  filters: number[];
  match_release: boolean;
  tags_included: string[];
  tags_excluded: string[];
  include_unmonitored: boolean;
  exclude_alternate_titles: boolean;
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
  exclude_alternate_titles: boolean;
}

type ListType = "SONARR" | "RADARR" | "LIDARR" | "READARR" | "WHISPARR" | "MDBLIST" | "TRAKT" | "METACRITIC" | "STEAM" | "PLAINTEXT";
