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
import { ColumnFilter } from "@tanstack/react-table";

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

export const ReleasesListQueryOptions = (offset: number, limit: number, filters: ColumnFilter[]) =>
  queryOptions({
    queryKey: ReleaseKeys.list(offset, limit, filters),
    queryFn: () => APIClient.release.findQuery(offset, limit, filters),
    placeholderData: keepPreviousData,
    staleTime: 5000,
    refetchOnWindowFocus: true,
    refetchInterval: 15000 // refetch releases table on releases page every 15s
  });

export const ReleasesLatestQueryOptions = () =>
  queryOptions({
    queryKey: ReleaseKeys.latestActivity(),
    queryFn: () => APIClient.release.findRecent(),
    refetchOnWindowFocus: true,
    refetchInterval: 15000  // refetch recent activity table on dashboard page every 15s
  });

export const ReleasesStatsQueryOptions = () =>
  queryOptions({
    queryKey: ReleaseKeys.stats(),
    queryFn: () => APIClient.release.stats(),
    refetchOnWindowFocus: true,
    refetchInterval: 15000  // refetch stats on dashboard page every 15s
  });

// ReleasesIndexersQueryOptions get basic list of used indexers by identifier
export const ReleasesIndexersQueryOptions = () =>
  queryOptions({
    queryKey: ReleaseKeys.indexers(),
    queryFn: async () => {
      const indexersResponse: IndexerDefinition[] = await APIClient.indexers.getAll();
      const indexerOptionsResponse: string[] = await APIClient.release.indexerOptions();
      
      const indexersMap = new Map(indexersResponse.map((indexer: IndexerDefinition) => [indexer.identifier, indexer.name]));
      
      return indexerOptionsResponse.map((identifier: string) => ({
        name: indexersMap.get(identifier) || identifier,
        identifier: identifier
      }));
    },
    refetchOnWindowFocus: false,
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
