/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { baseUrl, sseBaseUrl } from "../utils";
import { AuthContext } from "../utils/Context";
import { GithubRelease } from "../types/Update";

interface ConfigType {
  body?: BodyInit | Record<string, unknown> | unknown;
  headers?: Record<string, string>;
}

type PostBody = BodyInit | Record<string, unknown> | unknown;

export async function HttpClient<T>(
  endpoint: string,
  method: string,
  { body, ...customConfig }: ConfigType = {}
): Promise<T> {
  const config = {
    method: method,
    body: body ? JSON.stringify(body) : undefined,
    headers: {
      "Content-Type": "application/json"
    },
    // NOTE: customConfig can override the above defined settings
    ...customConfig
  } as RequestInit;

  return window.fetch(`${baseUrl()}${endpoint}`, config)
    .then(async response => {
      if (!response.ok) {
        // if 401 consider the session expired and force logout
        if (response.status === 401) {
          // Remove auth info from localStorage
          AuthContext.reset();

          // Show an error toast to notify the user what occurred
          return Promise.reject(new Error("Unauthorized"));
        } else if (response.status === 404) {
          return Promise.reject(new Error("Not found"));
        }

        return Promise.reject(new Error(await response.text()));
      }

      // Resolve immediately since 204 contains no data
      if (response.status === 204)
        return Promise.resolve(response);

      return await response.json();
    });
}

const appClient = {
  Get: <T>(endpoint: string) => HttpClient<T>(endpoint, "GET"),
  Post: <T = void>(endpoint: string, data: PostBody = undefined) => HttpClient<T>(endpoint, "POST", { body: data }),
  Put: <T = void>(endpoint: string, data: PostBody) => HttpClient<T>(endpoint, "PUT", { body: data }),
  Patch: (endpoint: string, data: PostBody = undefined) => HttpClient<void>(endpoint, "PATCH", { body: data }),
  Delete: (endpoint: string) => HttpClient<void>(endpoint, "DELETE")
};

export const APIClient = {
  auth: {
    login: (username: string, password: string) => appClient.Post("api/auth/login", {
      username: username,
      password: password
    }),
    logout: () => appClient.Post("api/auth/logout"),
    validate: () => appClient.Get<void>("api/auth/validate"),
    onboard: (username: string, password: string) => appClient.Post("api/auth/onboard", {
      username: username,
      password: password
    }),
    canOnboard: () => appClient.Get("api/auth/onboard")
  },
  actions: {
    create: (action: Action) => appClient.Post("api/actions", action),
    update: (action: Action) => appClient.Put(`api/actions/${action.id}`, action),
    delete: (id: number) => appClient.Delete(`api/actions/${id}`),
    toggleEnable: (id: number) => appClient.Patch(`api/actions/${id}/toggleEnabled`)
  },
  apikeys: {
    getAll: () => appClient.Get<APIKey[]>("api/keys"),
    create: (key: APIKey) => appClient.Post("api/keys", key),
    delete: (key: string) => appClient.Delete(`api/keys/${key}`)
  },
  config: {
    get: () => appClient.Get<Config>("api/config"),
    update: (config: ConfigUpdate) => appClient.Patch("api/config", config)
  },
  download_clients: {
    getAll: () => appClient.Get<DownloadClient[]>("api/download_clients"),
    create: (dc: DownloadClient) => appClient.Post("api/download_clients", dc),
    update: (dc: DownloadClient) => appClient.Put("api/download_clients", dc),
    delete: (id: number) => appClient.Delete(`api/download_clients/${id}`),
    test: (dc: DownloadClient) => appClient.Post("api/download_clients/test", dc)
  },
  filters: {
    getAll: () => appClient.Get<Filter[]>("api/filters"),
    find: (indexers: string[], sortOrder: string) => {
      const params = new URLSearchParams();

      if (sortOrder.length > 0) {
        params.append("sort", sortOrder);
      }

      indexers?.forEach((i) => {
        if (i !== undefined || i !== "") {
          params.append("indexer", i);
        }
      });

      const p = params.toString();
      const q = p ? `?${p}` : "";

      return appClient.Get<Filter[]>(`api/filters${q}`);
    },
    getByID: (id: number) => appClient.Get<Filter>(`api/filters/${id}`),
    create: (filter: Filter) => appClient.Post<Filter>("api/filters", filter),
    update: (filter: Filter) => appClient.Put<Filter>(`api/filters/${filter.id}`, filter),
    duplicate: (id: number) => appClient.Get<Filter>(`api/filters/${id}/duplicate`),
    toggleEnable: (id: number, enabled: boolean) => appClient.Put(`api/filters/${id}/enabled`, { enabled }),
    delete: (id: number) => appClient.Delete(`api/filters/${id}`)
  },
  feeds: {
    find: () => appClient.Get<Feed[]>("api/feeds"),
    create: (feed: FeedCreate) => appClient.Post("api/feeds", feed),
    toggleEnable: (id: number, enabled: boolean) => appClient.Patch(`api/feeds/${id}/enabled`, { enabled }),
    update: (feed: Feed) => appClient.Put(`api/feeds/${feed.id}`, feed),
    delete: (id: number) => appClient.Delete(`api/feeds/${id}`),
    test: (feed: Feed) => appClient.Post("api/feeds/test", feed)
  },
  indexers: {
    // returns indexer options for all currently present/enabled indexers
    getOptions: () => appClient.Get<Indexer[]>("api/indexer/options"),
    // returns indexer definitions for all currently present/enabled indexers
    getAll: () => appClient.Get<IndexerDefinition[]>("api/indexer"),
    // returns all possible indexer definitions
    getSchema: () => appClient.Get<IndexerDefinition[]>("api/indexer/schema"),
    create: (indexer: Indexer) => appClient.Post<Indexer>("api/indexer", indexer),
    update: (indexer: Indexer) => appClient.Put("api/indexer", indexer),
    delete: (id: number) => appClient.Delete(`api/indexer/${id}`),
    testApi: (req: IndexerTestApiReq) => appClient.Post<IndexerTestApiReq>(`api/indexer/${req.id}/api/test`, req)
  },
  irc: {
    getNetworks: () => appClient.Get<IrcNetworkWithHealth[]>("api/irc"),
    createNetwork: (network: IrcNetworkCreate) => appClient.Post("api/irc", network),
    updateNetwork: (network: IrcNetwork) => appClient.Put(`api/irc/network/${network.id}`, network),
    deleteNetwork: (id: number) => appClient.Delete(`api/irc/network/${id}`),
    restartNetwork: (id: number) => appClient.Get(`api/irc/network/${id}/restart`),
    sendCmd: (cmd: SendIrcCmdRequest) => appClient.Post(`api/irc/network/${cmd.network_id}/cmd`, cmd),
    events: (network: string) => new EventSource(`${sseBaseUrl()}api/irc/events?stream=${network}`, { withCredentials: true })
  },
  logs: {
    files: () => appClient.Get<LogFileResponse>("api/logs/files"),
    getFile: (file: string) => appClient.Get(`api/logs/files/${file}`)
  },
  events: {
    logs: () => new EventSource(`${sseBaseUrl()}api/events?stream=logs`, { withCredentials: true })
  },
  notifications: {
    getAll: () => appClient.Get<Notification[]>("api/notification"),
    create: (notification: Notification) => appClient.Post("api/notification", notification),
    update: (notification: Notification) => appClient.Put(`api/notification/${notification.id}`, notification),
    delete: (id: number) => appClient.Delete(`api/notification/${id}`),
    test: (n: Notification) => appClient.Post("api/notification/test", n)
  },
  release: {
    find: (query?: string) => appClient.Get<ReleaseFindResponse>(`api/release${query}`),
    findRecent: () => appClient.Get<ReleaseFindResponse>("api/release/recent"),
    findQuery: (offset?: number, limit?: number, filters?: Array<ReleaseFilter>) => {
      const params = new URLSearchParams();
      if (offset !== undefined && offset > 0)
        params.append("offset", offset.toString());

      if (limit !== undefined)
        params.append("limit", limit.toString());

      filters?.forEach((filter) => {
        if (!filter.value)
          return;

        if (filter.id == "indexer")
          params.append("indexer", filter.value);
        else if (filter.id === "action_status")
          params.append("push_status", filter.value);
        else if (filter.id == "torrent_name")
          params.append("q", filter.value);
      });

      return appClient.Get<ReleaseFindResponse>(`api/release?${params.toString()}`);
    },
    indexerOptions: () => appClient.Get<string[]>("api/release/indexers"),
    stats: () => appClient.Get<ReleaseStats>("api/release/stats"),
    delete: () => appClient.Delete("api/release/all"),
    deleteOlder: (duration: number) => appClient.Delete(`api/release/older-than/${duration}`),
    replayAction: (releaseId: number, actionId: number) => appClient.Post(`api/release/${releaseId}/actions/${actionId}/retry`)
  },
  updates: {
    check: () => appClient.Get("api/updates/check"),
    getLatestRelease: () => appClient.Get<GithubRelease | undefined>("api/updates/latest")
  }
};
