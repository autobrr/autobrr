/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { ColumnFilter } from "@tanstack/react-table";

export const SettingsKeys = {
  all: ["settings"] as const,
  updates: () => [...SettingsKeys.all, "updates"] as const,
  config: () => [...SettingsKeys.all, "config"] as const,
  lists: () => [...SettingsKeys.all, "list"] as const,
};

export const FilterKeys = {
  all: ["filters"] as const,
  lists: () => [...FilterKeys.all, "list"] as const,
  list: (indexers: string[], sortOrder: string) => [...FilterKeys.lists(), {indexers, sortOrder}] as const,
  details: () => [...FilterKeys.all, "detail"] as const,
  detail: (id: number) => [...FilterKeys.details(), id] as const
};

export const ReleaseKeys = {
  all: ["releases"] as const,
  lists: () => [...ReleaseKeys.all, "list"] as const,
  list: (pageIndex: number, pageSize: number, filters: ColumnFilter[]) => [...ReleaseKeys.lists(), {
    pageIndex,
    pageSize,
    filters
  }] as const,
  details: () => [...ReleaseKeys.all, "detail"] as const,
  detail: (id: number) => [...ReleaseKeys.details(), id] as const,
  indexers: () => [...ReleaseKeys.all, "indexers"] as const,
  stats: () => [...ReleaseKeys.all, "stats"] as const,
  latestActivity: () => [...ReleaseKeys.all, "latest-activity"] as const,
};

export const ApiKeys = {
  all: ["api_keys"] as const,
  lists: () => [...ApiKeys.all, "list"] as const,
  details: () => [...ApiKeys.all, "detail"] as const,
  detail: (id: string) => [...ApiKeys.details(), id] as const
};

export const DownloadClientKeys = {
  all: ["download_clients"] as const,
  lists: () => [...DownloadClientKeys.all, "list"] as const,
  // list: (indexers: string[], sortOrder: string) => [...clientKeys.lists(), { indexers, sortOrder }] as const,
  details: () => [...DownloadClientKeys.all, "detail"] as const,
  detail: (id: number) => [...DownloadClientKeys.details(), id] as const
};

export const FeedKeys = {
  all: ["feeds"] as const,
  lists: () => [...FeedKeys.all, "list"] as const,
  // list: (indexers: string[], sortOrder: string) => [...feedKeys.lists(), { indexers, sortOrder }] as const,
  details: () => [...FeedKeys.all, "detail"] as const,
  detail: (id: number) => [...FeedKeys.details(), id] as const
};

export const IndexerKeys = {
  all: ["indexers"] as const,
  schema: () => [...IndexerKeys.all, "indexer-definitions"] as const,
  options: () => [...IndexerKeys.all, "options"] as const,
  lists: () => [...IndexerKeys.all, "list"] as const,
  // list: (indexers: string[], sortOrder: string) => [...indexerKeys.lists(), { indexers, sortOrder }] as const,
  details: () => [...IndexerKeys.all, "detail"] as const,
  detail: (id: number) => [...IndexerKeys.details(), id] as const
};

export const IrcKeys = {
  all: ["irc_networks"] as const,
  lists: () => [...IrcKeys.all, "list"] as const,
  // list: (indexers: string[], sortOrder: string) => [...ircKeys.lists(), { indexers, sortOrder }] as const,
  details: () => [...IrcKeys.all, "detail"] as const,
  detail: (id: number) => [...IrcKeys.details(), id] as const
};

export const NotificationKeys = {
  all: ["notifications"] as const,
  lists: () => [...NotificationKeys.all, "list"] as const,
  details: () => [...NotificationKeys.all, "detail"] as const,
  detail: (id: number) => [...NotificationKeys.details(), id] as const
};

export const ProxyKeys = {
  all: ["proxy"] as const,
  lists: () => [...ProxyKeys.all, "list"] as const,
  details: () => [...ProxyKeys.all, "detail"] as const,
  detail: (id: number) => [...ProxyKeys.details(), id] as const
};
