/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { keepPreviousData, queryOptions } from "@tanstack/react-query";
import { APIClient } from "@api/APIClient";
import {
  ApiKeys,
  DownloadClientKeys,
  FeedKeys,
  FilterKeys,
  IndexerKeys,
  IrcKeys, NotificationKeys, ProxyKeys,
  ReleaseKeys,
  SettingsKeys
} from "@api/query_keys";

export const FiltersQueryOptions = (indexers: string[], sortOrder: string) =>
  queryOptions({
    queryKey: FilterKeys.list(indexers, sortOrder),
    queryFn: () => APIClient.filters.find(indexers, sortOrder),
    refetchOnWindowFocus: false
  });

export const FilterByIdQueryOptions = (filterId: number) =>
  queryOptions({
    queryKey: FilterKeys.detail(filterId),
    queryFn: async ({queryKey}) => await APIClient.filters.getByID(queryKey[2]),
    retry: false,
  });

export const ConfigQueryOptions = (enabled: boolean = true) =>
  queryOptions({
    queryKey: SettingsKeys.config(),
    queryFn: () => APIClient.config.get(),
    retry: false,
    refetchOnWindowFocus: false,
    enabled: enabled,
  });

export const UpdatesQueryOptions = (enabled: boolean) =>
  queryOptions({
    queryKey: SettingsKeys.updates(),
    queryFn: () => APIClient.updates.getLatestRelease(),
    retry: false,
    refetchOnWindowFocus: false,
    enabled: enabled,
  });

export const IndexersQueryOptions = () =>
  queryOptions({
    queryKey: IndexerKeys.lists(),
    queryFn: () => APIClient.indexers.getAll()
  });

export const IndexersOptionsQueryOptions = () =>
  queryOptions({
    queryKey: IndexerKeys.options(),
    queryFn: () => APIClient.indexers.getOptions(),
    refetchOnWindowFocus: false,
    staleTime: Infinity
  });

export const IndexersSchemaQueryOptions = (enabled: boolean) =>
  queryOptions({
    queryKey: IndexerKeys.schema(),
    queryFn: () => APIClient.indexers.getSchema(),
    refetchOnWindowFocus: false,
    staleTime: Infinity,
    enabled: enabled
  });

export const IrcQueryOptions = () =>
  queryOptions({
    queryKey: IrcKeys.lists(),
    queryFn: () => APIClient.irc.getNetworks(),
    refetchOnWindowFocus: false,
    refetchInterval: 3000 // Refetch every 3 seconds
  });

export const FeedsQueryOptions = () =>
  queryOptions({
    queryKey: FeedKeys.lists(),
    queryFn: () => APIClient.feeds.find(),
  });

export const DownloadClientsQueryOptions = () =>
  queryOptions({
    queryKey: DownloadClientKeys.lists(),
    queryFn: () => APIClient.download_clients.getAll(),
  });

export const NotificationsQueryOptions = () =>
  queryOptions({
    queryKey: NotificationKeys.lists(),
    queryFn: () => APIClient.notifications.getAll()
  });

export const ApikeysQueryOptions = () =>
  queryOptions({
    queryKey: ApiKeys.lists(),
    queryFn: () => APIClient.apikeys.getAll(),
    refetchOnWindowFocus: false,
  });

export const ReleasesListQueryOptions = (offset: number, limit: number, filters: ReleaseFilter[]) =>
  queryOptions({
    queryKey: ReleaseKeys.list(offset, limit, filters),
    queryFn: () => APIClient.release.findQuery(offset, limit, filters),
    staleTime: 5000
  });

export const ReleasesLatestQueryOptions = () =>
  queryOptions({
    queryKey: ReleaseKeys.latestActivity(),
    queryFn: () => APIClient.release.findRecent(),
    refetchOnWindowFocus: false
  });

export const ReleasesStatsQueryOptions = () =>
  queryOptions({
    queryKey: ReleaseKeys.stats(),
    queryFn: () => APIClient.release.stats(),
    refetchOnWindowFocus: false
  });

// ReleasesIndexersQueryOptions get basic list of used indexers by identifier
export const ReleasesIndexersQueryOptions = () =>
  queryOptions({
    queryKey: ReleaseKeys.indexers(),
    queryFn: () => APIClient.release.indexerOptions(),
    placeholderData: keepPreviousData,
    staleTime: Infinity
  });

export const ProxiesQueryOptions = () =>
  queryOptions({
    queryKey: ProxyKeys.lists(),
    queryFn: () => APIClient.proxy.list(),
    refetchOnWindowFocus: false
  });

export const ProxyByIdQueryOptions = (proxyId: number) =>
  queryOptions({
    queryKey: ProxyKeys.detail(proxyId),
    queryFn: async ({queryKey}) => await APIClient.proxy.getByID(queryKey[2]),
    retry: false,
  });
