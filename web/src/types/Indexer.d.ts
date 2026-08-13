/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

interface Indexer {
  id: number;
  name: string;
  identifier: string;
  identifier_external: string;
  enabled: boolean;
  implementation: string;
  base_url: string;
  use_proxy?: boolean;
  proxy_id?: number;
  settings: Array<IndexerSetting>;
}

interface IndexerMinimal {
  id: number;
  name: string;
  identifier: string;
  identifier_external: string;
}

type IndexerImplementation = "irc" | "torznab" | "newznab" | "rss";

interface IndexerDefinition {
  id: number;
  name: string;
  identifier: string;
  identifier_external: string;
  implementation: IndexerImplementation;
  base_url: string;
  enabled?: boolean;
  description: string;
  language: string;
  privacy: string;
  protocol: string;
  urls: string[];
  supports: string[];
  use_proxy?: boolean;
  proxy_id?: number;
  settings: IndexerSetting[];
  irc: IndexerIRC;
  feed: IndexerFeed;
}

type SettingsFieldType = "text" | "secret";

interface IndexerSetting {
  name: string;
  required?: boolean;
  type: SettingsFieldType;
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
  auth?: IndexerIRCAuth;
  settings: IndexerSetting[];
  channels: IndexerIRCChannel[];
}

interface IndexerIRCAuth {
  mechanism: IrcAuthMechanism;
}

interface IndexerIRCChannel {
  name: string;
  announcers: string[];
  parse: IndexerParse;
}

interface IndexerFeed {
  minInterval: number;
  settings: IndexerSetting[];
}

interface IndexerParse {
  type: string;
  forcesizeunit: boolean;
  skipcleanmessage: boolean;
  lines: IndexerParseLines[];
  match: IndexerParseMatch;
}

interface IndexerParseLines {
  test: string[];
  pattern: string;
  vars: string[];
}

interface IndexerParseMatch {
  downloadurl: string;
  releasename: string;
  magneturi: string;
  infourl: string;
  encode: string[];
}

interface IndexerTestApiReq {
  id?: number;
  identifier?: string;
  api_user?: string;
  api_key: string;
}
