/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

interface Indexer {
  id: number;
  name: string;
  identifier: string;
  enabled: boolean;
  implementation: string;
  base_url: string;
  settings: Array<IndexerSetting>;
}

interface IndexerDefinition {
  id: number;
  name: string;
  identifier: string;
  implementation: string;
  base_url: string;
  enabled?: boolean;
  description: string;
  language: string;
  privacy: string;
  protocol: string;
  urls: string[];
  supports: string[];
  settings: IndexerSetting[];
  irc: IndexerIRC;
  torznab: IndexerTorznab;
  newznab?: IndexerTorznab;
  rss: IndexerFeed;
  parse: IndexerParse;
}

interface IndexerSetting {
  name: string;
  required?: boolean;
  type: string;
  value?: string;
  label: string;
  default?: string;
  description?: string;
  help?: string;
  regex?: string;
}

interface IndexerIRC {
  network: string;
  server: string;
  port: number;
  tls: boolean;
  nickserv: boolean;
  channels: string[];
  announcers: string[];
  settings: IndexerSetting[];
}

interface IndexerTorznab {
  minInterval: number;
  settings: IndexerSetting[];
}

interface IndexerFeed {
  minInterval: number;
  settings: IndexerSetting[];
}

interface IndexerParse {
  type: string;
  lines: IndexerParseLines[];
  match: IndexerParseMatch;
}

interface IndexerParseLines {
  test: string[];
  pattern: string;
  vars: string[];
}

interface IndexerParseMatch {
  torrentUrl: string;
  encode: string[];
}

interface IndexerTestApiReq {
  id?: number;
  identifier?: string;
  api_user?: string;
  api_key: string;
}
