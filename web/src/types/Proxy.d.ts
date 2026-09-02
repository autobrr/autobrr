/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

interface Proxy {
  id: number;
  name: string;
  enabled: boolean;
  type: ProxyType;
  addr: string;
  user?: string;
  pass?: string;
  timeout?: number;
}

interface ProxyCreate {
  name: string;
  enabled: boolean;
  type: ProxyType;
  addr: string;
  user?: string;
  pass?: string;
  timeout?: number;
}

interface ProxyUsageItem {
  id: number;
  name: string;
}

interface ProxyUsage {
  indexers: ProxyUsageItem[];
  irc_networks: ProxyUsageItem[];
  feeds: ProxyUsageItem[];
}

type ProxyType = "SOCKS5" | "HTTP";
