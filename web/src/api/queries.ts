/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { keepPreviousData, queryOptions } from "@tanstack/react-query";
import { notificationKeys } from "@screens/settings/Notifications";
import { APIClient } from "@api/APIClient";
import { clientKeys } from "@screens/settings/DownloadClient";
import { feedKeys } from "@screens/settings/Feed";
import { filterKeys } from "@screens/filters/List";
import { apiKeys } from "@screens/settings/Api";
import { indexerKeys } from "@screens/settings/Indexer";
import { ircKeys } from "@screens/settings/Irc";
import { settingsKeys } from "@screens/Settings";
import { releaseKeys } from "@screens/releases/ReleaseTable";

export const FiltersQueryOptions = (indexers: string[], sortOrder: string) =>
  queryOptions({
    queryKey: filterKeys.list(indexers, sortOrder),
    queryFn: () => APIClient.filters.find(indexers, sortOrder),
    refetchOnWindowFocus: false
  });

export const FilterByIdQueryOptions = (filterId: number) =>
  queryOptions({
    queryKey: filterKeys.detail(filterId),
    queryFn: async ({queryKey}) => await APIClient.filters.getByID(queryKey[2]),
    retry: false,
  });

export const ConfigQueryOptions = (enabled: boolean = true) =>
  queryOptions({
    queryKey: settingsKeys.config(),
    queryFn: () => APIClient.config.get(),
    retry: false,
    refetchOnWindowFocus: false,
    enabled: enabled,
  });

export const UpdatesQueryOptions = (enabled: boolean) =>
  queryOptions({
    queryKey: settingsKeys.updates(),
    queryFn: () => APIClient.updates.getLatestRelease(),
    retry: false,
    refetchOnWindowFocus: false,
    enabled: enabled,
  });

export const IndexersQueryOptions = () =>
  queryOptions({
    queryKey: indexerKeys.lists(),
    queryFn: () => APIClient.indexers.getAll()
  });

export const IndexersOptionsQueryOptions = () =>
  queryOptions({
    queryKey: indexerKeys.options(),
    queryFn: () => APIClient.indexers.getOptions(),
    refetchOnWindowFocus: false,
    staleTime: Infinity
  });

export const IndexersSchemaQueryOptions = (enabled: boolean) =>
  queryOptions({
    queryKey: indexerKeys.schema(),
    queryFn: () => APIClient.indexers.getSchema(),
    refetchOnWindowFocus: false,
    staleTime: Infinity,
    enabled: enabled
  });

export const IrcQueryOptions = () =>
  queryOptions({
    queryKey: ircKeys.lists(),
    queryFn: () => APIClient.irc.getNetworks(),
    refetchOnWindowFocus: false,
    refetchInterval: 3000 // Refetch every 3 seconds
  });

export const FeedsQueryOptions = () =>
  queryOptions({
    queryKey: feedKeys.lists(),
    queryFn: () => APIClient.feeds.find(),
  });

export const DownloadClientsQueryOptions = () =>
  queryOptions({
    queryKey: clientKeys.lists(),
    queryFn: () => APIClient.download_clients.getAll(),
  });

export const NotificationsQueryOptions = () =>
  queryOptions({
    queryKey: notificationKeys.lists(),
    queryFn: () => APIClient.notifications.getAll()
  });

export const ApikeysQueryOptions = () =>
  queryOptions({
    queryKey: apiKeys.lists(),
    queryFn: () => APIClient.apikeys.getAll(),
    refetchOnWindowFocus: false,
  });

export const ReleasesListQueryOptions = (offset: number, limit: number, filters: ReleaseFilter[]) =>
  queryOptions({
    queryKey: releaseKeys.list(offset, limit, filters),
    queryFn: () => APIClient.release.findQuery(offset, limit, filters),
    staleTime: 5000
  });

export const ReleasesLatestQueryOptions = () =>
  queryOptions({
    queryKey: releaseKeys.latestActivity(),
    queryFn: () => APIClient.release.findRecent(),
    refetchOnWindowFocus: false
  });

export const ReleasesStatsQueryOptions = () =>
  queryOptions({
    queryKey: releaseKeys.stats(),
    queryFn: () => APIClient.release.stats(),
    refetchOnWindowFocus: false
  });

// ReleasesIndexersQueryOptions get basic list of used indexers by identifier
export const ReleasesIndexersQueryOptions = () =>
  queryOptions({
    queryKey: releaseKeys.indexers(),
    queryFn: () => APIClient.release.indexerOptions(),
    placeholderData: keepPreviousData,
    staleTime: Infinity
  });
