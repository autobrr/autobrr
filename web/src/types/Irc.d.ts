/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

interface IrcNetwork {
  id: number;
  name: string;
  enabled: boolean;
  server: string;
  port: number;
  tls: boolean;
  nick: string;
  pass: string;
  auth: IrcAuth; // optional
  invite_command: string;
  use_bouncer: boolean;
  bouncer_addr: string;
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
  nick: string;
  auth: IrcAuth; // optional
  invite_command: string;
  use_bouncer?: boolean;
  bouncer_addr?: string;
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
  nick: string;
  auth: IrcAuth; // optional
  invite_command: string;
  use_bouncer: boolean;
  bouncer_addr: string;
  channels: IrcChannelWithHealth[];
  connected: boolean;
  connected_since: string;
  connection_errors: string[];
  healthy: boolean;
}

type IrcAuthMechanism = "NONE" | "SASL_PLAIN" | "NICKSERV";

interface IrcAuth {
  mechanism: IrcAuthMechanism; // optional
  account?: string; // optional
  password?: string; // optional
}

interface SendIrcCmdRequest {
  network_id: number;
  server: string;
  channel: string;
  nick: string;
  msg: string;
}
