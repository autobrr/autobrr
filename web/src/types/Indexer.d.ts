interface Indexer {
  id: number;
  name: string;
  identifier: string;
  enabled: boolean;
  type?: string;
  settings: Array<IndexerSetting>;
}

interface IndexerDefinition {
  id?: number;
  name: string;
  identifier: string;
  enabled?: boolean;
  description: string;
  language: string;
  privacy: string;
  protocol: string;
  urls: string[];
  supports: string[];
  settings: IndexerSetting[];
  irc: IndexerIRC;
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
