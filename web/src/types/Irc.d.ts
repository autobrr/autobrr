/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
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
  bot_mode: boolean;
  channels: IrcChannel[];
  connected: boolean;
  connected_since: string;
  use_proxy: boolean;
  proxy_id: number;
  connection_errors: string[];
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
  bot_mode?: boolean;
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

type IrcChannelState = "Idle" | "AwaitingInvite" | "AwaitingInviteBot" | "InviteFailed" | "InviteFailedNoSuchNick" | "Joining" | "Monitoring" | "Kicked" | "Parted" | "Disabled" | "Error" | "Unknown";

interface IrcChannelWithHealth extends IrcChannel {
  state: IrcChannelState;
  monitoring_since: string;
  last_announce: string;
  connection_errors: string[];
}

interface IrcNetworkWithHealth extends IrcNetwork {
  channels: IrcChannelWithHealth[];
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

interface IrcProcessManualRequest {
  network_id: number;
  channel: string;
  nick?: string;
  msg: string;
}
