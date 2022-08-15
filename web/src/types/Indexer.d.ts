interface Indexer {
  id: number;
  name: string;
  identifier: string;
  enabled: boolean;
  implementation: string;
  settings: Array<IndexerSetting>;
}

interface IndexerDefinition {
  id: number;
  name: string;
  identifier: string;
  implementation: string;
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
