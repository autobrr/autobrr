interface IrcNetwork {
  id: number;
  name: string;
  enabled: boolean;
  server: string;
  port: number;
  tls: boolean;
  pass: string;
  invite_command: string;
  nickserv?: NickServ; // optional
  channels: IrcChannel[];
  connected: boolean;
  connected_since: string;
}

interface IrcNetworkCreate {
  name: string;
  enabled: boolean;
  server: string;
  port: number;
  tls: boolean;
  pass: string;
  invite_command: string;
  nickserv?: NickServ; // optional
  channels: IrcChannel[];
  connected: boolean;
}

interface IrcChannel {
  id: number;
  enabled: boolean;
  name: string;
  password: string;
  detached: boolean;
  monitoring: boolean;
}

interface IrcChannelWithHealth extends IrcChannel {
  monitoring_since: string;
  last_announce: string;
}

interface IrcNetworkWithHealth {
  id: number;
  name: string;
  enabled: boolean;
  server: string;
  port: number;
  tls: boolean;
  pass: string;
  invite_command: string;
  nickserv?: NickServ; // optional
  channels: IrcChannelWithHealth[];
  connected: boolean;
  connected_since: string;
}

interface NickServ {
  account?: string; // optional
  password?: string; // optional
}

interface Config {
  host: string;
  port: number;
  log_level: string;
  log_path: string;
  base_url: string;
  version: string;
  commit: string;
  date: string;
}