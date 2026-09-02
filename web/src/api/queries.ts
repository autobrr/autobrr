/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { keepPreviousData, queryOptions } from "@tanstack/react-query";
import { APIClient } from "@api/APIClient";
import {
  ApiKeys,
  DownloaderKeys,
  FeedKeys,
  FilterKeys,
  IndexerKeys,
  IrcKeys, ListKeys, NotificationKeys, ProxyKeys,
  ReleaseKeys, ReleaseProfileDuplicateKeys,
  SettingsKeys
} from "@api/query_keys";
import { ColumnFilter } from "@tanstack/react-table";

export const FiltersGetAllQueryOptions = () =>
  queryOptions({
    queryKey: FilterKeys.lists(),
    queryFn: () => APIClient.filters.getAll(),
    refetchOnWindowFocus: false
  });

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

export const IndexerDeprecationsQueryOptions = () =>
  queryOptions({
    queryKey: IndexerKeys.deprecations(),
    queryFn: () => APIClient.indexers.getDeprecations(),
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
    // SSE STATE/HEALTH events drive instant updates; poll as a fallback so the
    // list self-heals if an event is missed (e.g. during an SSE reconnect).
    refetchInterval: 5000
  });

export const FeedsQueryOptions = () =>
  queryOptions({
    queryKey: FeedKeys.lists(),
    queryFn: () => APIClient.feeds.find(),
  });

export const DownloadersQueryOptions = () =>
  queryOptions({
    queryKey: DownloaderKeys.lists(),
    queryFn: () => APIClient.downloaders.getAll(),
  });

export const DownloadersArrTagsQueryOptions = (clientID: number) =>
  queryOptions({
    queryKey: DownloaderKeys.arrTags(clientID),
    queryFn: () => APIClient.downloaders.getArrTags(clientID),
    retry: false,
    enabled: clientID > 0,
  });

export const NotificationsQueryOptions = () =>
  queryOptions({
    queryKey: NotificationKeys.lists(),
    queryFn: () => APIClient.notifications.getAll()
  });

export const PushoverSoundsQueryOptions = (apiToken: string) =>
  queryOptions({
    queryKey: NotificationKeys.pushoverSounds(apiToken),
    queryFn: () => {
      // Double-check before making the request
      if (!apiToken || apiToken === "<redacted>" || apiToken === "") {
        throw new Error("API token is required");
      }
      return APIClient.notifications.getPushoverSounds(apiToken);
    },
    enabled: apiToken !== undefined && apiToken !== "" && apiToken !== "<redacted>",
    retry: false,
    refetchOnWindowFocus: false,
    staleTime: 5 * 60 * 1000 // 5 minutes
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

// Dashboard widget queries contain failures in their own card: no throw to
// the route error boundary (the global default) and only quick retries so
// the widget's error state appears in seconds instead of minutes. The
// expired-cookie error must not retry so the login redirect stays prompt.
const widgetQueryDefaults = {
  placeholderData: keepPreviousData,
  staleTime: 5000,
  refetchOnWindowFocus: true,
  refetchInterval: 15000,
  throwOnError: false,
  retry: (failureCount: number, error: Error) =>
    error.message !== "Cookie expired or invalid." && failureCount < 2
};

export const ReleasesStatsQueryOptions = () =>
  queryOptions({
    queryKey: ReleaseKeys.stats(),
    queryFn: () => APIClient.release.stats(),
    ...widgetQueryDefaults
  });

export const ReleasesActivityQueryOptions = (days: number = 30) =>
  queryOptions({
    queryKey: ReleaseKeys.statsActivity(days),
    queryFn: () => APIClient.release.statsActivity(days),
    ...widgetQueryDefaults
  });

export const ReleasesVolumeQueryOptions = (days: number = 30) =>
  queryOptions({
    queryKey: ReleaseKeys.statsVolume(days),
    queryFn: () => APIClient.release.statsVolume(days),
    ...widgetQueryDefaults
  });

export const ReleasesHeatmapQueryOptions = (days: number = 30) =>
  queryOptions({
    queryKey: ReleaseKeys.statsHeatmap(days),
    queryFn: () => APIClient.release.statsHeatmap(days),
    ...widgetQueryDefaults
  });

export const ReleasesTopIndexersQueryOptions = (days: number = 30) =>
  queryOptions({
    queryKey: ReleaseKeys.statsTopIndexers(days),
    queryFn: () => APIClient.release.statsTopIndexers(days),
    ...widgetQueryDefaults
  });

export const ReleasesTopFiltersQueryOptions = (days: number = 30) =>
  queryOptions({
    queryKey: ReleaseKeys.statsTopFilters(days),
    queryFn: () => APIClient.release.statsTopFilters(days),
    ...widgetQueryDefaults
  });

// ReleasesIndexersQueryOptions get basic list of used indexers by identifier
export const ReleasesIndexersQueryOptions = () =>
  queryOptions({
    queryKey: ReleaseKeys.indexers(),
    queryFn: async () => {
      const indexersResponse: IndexerDefinition[] = await APIClient.indexers.getAll();
      const deprecationsResponse: IndexerDeprecation[] = await APIClient.indexers.getDeprecations();
      const indexerOptionsResponse: string[] = await APIClient.release.indexerOptions();

      const indexersMap = new Map(indexersResponse.map((indexer: IndexerDefinition) => [indexer.identifier, indexer.name]));
      // fall back to deprecation metadata so removed indexers still show a friendly name
      const deprecationsMap = new Map(deprecationsResponse.map((d: IndexerDeprecation) => [d.identifier, d.name]));

      return indexerOptionsResponse.map((identifier: string) => ({
        name: indexersMap.get(identifier) || deprecationsMap.get(identifier) || identifier,
        identifier: identifier
      }));
    },
    refetchOnWindowFocus: false,
    staleTime: Infinity
  });

export const ReleaseProfileDuplicateList = () =>
  queryOptions({
    queryKey: ReleaseProfileDuplicateKeys.lists(),
    queryFn: () => APIClient.release.profiles.duplicates.list(),
    staleTime: 5000,
    refetchOnWindowFocus: true,
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

export const ProxyUsageQueryOptions = (proxyId: number) =>
  queryOptions({
    queryKey: ProxyKeys.usage(proxyId),
    queryFn: () => APIClient.proxy.usage(proxyId),
    // the warning that owns this query is mounted only while shown, so every open refetches
    staleTime: 0,
    refetchOnWindowFocus: false
  });

export const ListsQueryOptions = () =>
  queryOptions({
    queryKey: ListKeys.lists(),
    queryFn: () => APIClient.lists.list(),
    refetchOnWindowFocus: false
  });
