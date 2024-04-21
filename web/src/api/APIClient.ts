/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { baseUrl, sseBaseUrl } from "@utils";
import { GithubRelease } from "@app/types/Update";

type RequestBody = BodyInit | object | Record<string, unknown> | null;
type Primitive = string | number | boolean | symbol | undefined;

interface HttpConfig {
  method?: string;
  body?: RequestBody;
  queryString?: Record<string, Primitive | Primitive[]>;
}

// See https://stackoverflow.com/a/62969380
function encodeRFC3986URIComponent(str: string): string {
  return encodeURIComponent(str).replace(
    /[!'()*]/g,
    (c) => `%${c.charCodeAt(0).toString(16).toUpperCase()}`
  );
}

export async function HttpClient<T = unknown>(
  endpoint: string,
  config: HttpConfig = {}
): Promise<T> {
  const init: RequestInit = {
    method: config.method,
    headers: { "Accept": "*/*", 'x-requested-with': 'XMLHttpRequest' },
    credentials: "include",
  };

  if (config.body) {
    init.body = JSON.stringify(config.body);

    if (typeof(config.body) === "object") {
      init.headers = {
        ...init.headers,
        "Content-Type": "application/json"
      };
    }
  }

  if (config.queryString) {
    const params: string[] = [];

    for (const [key, value] of Object.entries(config.queryString)) {
      const serializedKey = encodeRFC3986URIComponent(key);

      if (typeof(value) === "undefined") {
        // Skip case when the value is undefined.
        // The solution in this case is to use the request body instead with JSON
        continue;
      } else if (Array.isArray(value)) {
        // Append (don't set) each array member as a query parameter
        // e.g. ?a=1&a=2&a=3
        value.forEach((child) => {
          // Skip undefined member values
          const v = typeof(child) !== "undefined" ? String(child) : "";
          if (v.length) {
            params.push(`${serializedKey}=${encodeRFC3986URIComponent(v)}`);
          }
        });
      } else {
        // This is a primitive value, just add as string
        // e.g. ?a=1
        const v = String(value);
        if (v.length) {
          params.push(`${serializedKey}=${encodeRFC3986URIComponent(v)}`);
        }
      }
    }

    if (params.length) {
      endpoint += `?${params.join("&")}`;
    }
  }

  const response = await window.fetch(`${baseUrl()}${endpoint}`, init);

  const isJson = response.headers.get("Content-Type")?.includes("application/json");
  const json = isJson ? await response.json() : null;

  switch (response.status) {
  case 204: {
    // 204 contains no data, but indicates success
    return Promise.resolve<T>({} as T);
  }
  case 401: {
    return Promise.reject<T>(json as T);
  }
  case 403: {
    return Promise.reject<T>(json as T);
  }
  case 404: {
    return Promise.reject<T>(json as T);
  }
  case 500: {
    const health = await window.fetch(`${baseUrl()}api/healthz/liveness`);
    if (!health.ok) {
      return Promise.reject(
        new Error(`[500] Offline (Internal server error): "${endpoint}"`)
      );
    }
    break;
  }
  case 503: {
    // Show an error toast to notify the user what occurred
    return Promise.reject(new Error(`[503] Service unavailable: "${endpoint}"`));
  }
  default:
    break;
  }

  // Resolve on success
  if (response.status >= 200 && response.status < 300) {
    if (isJson) {
      return Promise.resolve<T>(json as T);
    } else {
      return Promise.resolve<T>(response as T);
    }
  }

  // Otherwise reject, this is most likely an error
  return Promise.reject<T>(json as T);
}

const appClient = {
  Get: <T>(endpoint: string, config: HttpConfig = {}) => HttpClient<T>(endpoint, {
    ...config,
    method: "GET"
  }),
  Post: <T = void>(endpoint: string, config: HttpConfig = {}) => HttpClient<T>(endpoint, {
    ...config,
    method: "POST"
  }),
  Put: <T = void>(endpoint: string, config: HttpConfig = {}) => HttpClient<T>(endpoint, {
    ...config,
    method: "PUT"
  }),
  Patch: (endpoint: string, config: HttpConfig = {}) => HttpClient<void>(endpoint, {
    ...config,
    method: "PATCH"
  }),
  Delete: (endpoint: string, config: HttpConfig = {}) => HttpClient<void>(endpoint, {
    ...config,
    method: "DELETE"
  })
};

export const APIClient = {
  auth: {
    login: (username: string, password: string) => appClient.Post("api/auth/login", {
      body: { username, password }
    }),
    logout: () => appClient.Post("api/auth/logout"),
    validate: () => appClient.Get<void>("api/auth/validate"),
    onboard: (username: string, password: string) => appClient.Post("api/auth/onboard", {
      body: { username, password }
    }),
    canOnboard: () => appClient.Get("api/auth/onboard"),
    updateUser: (req: UserUpdate) => appClient.Patch(`api/auth/user/${req.username_current}`,
      { body: req })
  },
  actions: {
    create: (action: Action) => appClient.Post("api/actions", {
      body: action
    }),
    update: (action: Action) => appClient.Put(`api/actions/${action.id}`, {
      body: action
    }),
    delete: (id: number) => appClient.Delete(`api/actions/${id}`),
    toggleEnable: (id: number) => appClient.Patch(`api/actions/${id}/toggleEnabled`)
  },
  apikeys: {
    getAll: () => appClient.Get<APIKey[]>("api/keys"),
    create: (key: APIKey) => appClient.Post("api/keys", {
      body: key
    }),
    delete: (key: string) => appClient.Delete(`api/keys/${key}`)
  },
  config: {
    get: () => appClient.Get<Config>("api/config"),
    update: (config: ConfigUpdate) => appClient.Patch("api/config", {
      body: config
    })
  },
  download_clients: {
    getAll: () => appClient.Get<DownloadClient[]>("api/download_clients"),
    create: (dc: DownloadClient) => appClient.Post("api/download_clients", {
      body: dc
    }),
    update: (dc: DownloadClient) => appClient.Put("api/download_clients", {
      body: dc
    }),
    delete: (id: number) => appClient.Delete(`api/download_clients/${id}`),
    test: (dc: DownloadClient) => appClient.Post("api/download_clients/test", {
      body: dc
    })
  },
  filters: {
    getAll: () => appClient.Get<Filter[]>("api/filters"),
    find: (indexers: string[], sortOrder: string) => appClient.Get<Filter[]>("api/filters", {
      queryString: {
        sort: sortOrder,
        indexer: indexers
      }
    }),
    getByID: (id: number) => appClient.Get<Filter>(`api/filters/${id}`),
    create: (filter: Filter) => appClient.Post<Filter>("api/filters", {
      body: filter
    }),
    update: (filter: Filter) => appClient.Put<Filter>(`api/filters/${filter.id}`, {
      body: filter
    }),
    duplicate: (id: number) => appClient.Get<Filter>(`api/filters/${id}/duplicate`),
    toggleEnable: (id: number, enabled: boolean) => appClient.Put(`api/filters/${id}/enabled`, {
      body: { enabled }
    }),
    delete: (id: number) => appClient.Delete(`api/filters/${id}`)
  },
  feeds: {
    find: () => appClient.Get<Feed[]>("api/feeds"),
    create: (feed: FeedCreate) => appClient.Post("api/feeds", {
      body: feed
    }),
    toggleEnable: (id: number, enabled: boolean) => appClient.Patch(`api/feeds/${id}/enabled`, {
      body: { enabled }
    }),
    update: (feed: Feed) => appClient.Put(`api/feeds/${feed.id}`, {
      body: feed
    }),
    forceRun: (id: number) => appClient.Post(`api/feeds/${id}/forcerun`),
    delete: (id: number) => appClient.Delete(`api/feeds/${id}`),
    deleteCache: (id: number) => appClient.Delete(`api/feeds/${id}/cache`),
    test: (feed: Feed) => appClient.Post("api/feeds/test", {
      body: feed
    })
  },
  indexers: {
    // returns indexer options for all currently present/enabled indexers
    getOptions: () => appClient.Get<Indexer[]>("api/indexer/options"),
    // returns indexer definitions for all currently present/enabled indexers
    getAll: () => appClient.Get<IndexerDefinition[]>("api/indexer"),
    // returns all possible indexer definitions
    getSchema: () => appClient.Get<IndexerDefinition[]>("api/indexer/schema"),
    create: (indexer: Indexer) => appClient.Post<Indexer>("api/indexer", {
      body: indexer
    }),
    update: (indexer: Indexer) => appClient.Put(`api/indexer/${indexer.id}`, {
      body: indexer
    }),
    delete: (id: number) => appClient.Delete(`api/indexer/${id}`),
    testApi: (req: IndexerTestApiReq) => appClient.Post<IndexerTestApiReq>(`api/indexer/${req.id}/api/test`, {
      body: req
    }),
    toggleEnable: (id: number, enabled: boolean) => appClient.Patch(`api/indexer/${id}/enabled`, {
      body: { enabled }
    })
  },
  irc: {
    getNetworks: () => appClient.Get<IrcNetworkWithHealth[]>("api/irc"),
    createNetwork: (network: IrcNetworkCreate) => appClient.Post("api/irc", {
      body: network
    }),
    updateNetwork: (network: IrcNetwork) => appClient.Put(`api/irc/network/${network.id}`, {
      body: network
    }),
    deleteNetwork: (id: number) => appClient.Delete(`api/irc/network/${id}`),
    restartNetwork: (id: number) => appClient.Get(`api/irc/network/${id}/restart`),
    sendCmd: (cmd: SendIrcCmdRequest) => appClient.Post(`api/irc/network/${cmd.network_id}/cmd`, {
      body: cmd
    }),
    reprocessAnnounce: (networkId: number, channel: string, msg: string) => appClient.Post(`api/irc/network/${networkId}/channel/${channel}/announce/process`, {
      body: { msg: msg }
    }),
    events: (network: string) => new EventSource(
      `${sseBaseUrl()}api/irc/events?stream=${encodeRFC3986URIComponent(network)}`,
      { withCredentials: true }
    )
  },
  logs: {
    files: () => appClient.Get<LogFileResponse>("api/logs/files"),
    getFile: (file: string) => appClient.Get(`api/logs/files/${file}`)
  },
  events: {
    logs: () => new EventSource(`${sseBaseUrl()}api/events?stream=logs`, { withCredentials: true })
  },
  notifications: {
    getAll: () => appClient.Get<ServiceNotification[]>("api/notification"),
    create: (notification: ServiceNotification) => appClient.Post("api/notification", {
      body: notification
    }),
    update: (notification: ServiceNotification) => appClient.Put(
      `api/notification/${notification.id}`,
      { body: notification }
    ),
    delete: (id: number) => appClient.Delete(`api/notification/${id}`),
    test: (notification: ServiceNotification) => appClient.Post("api/notification/test", {
      body: notification
    })
  },
  release: {
    find: (query?: string) => appClient.Get<ReleaseFindResponse>(`api/release${query}`),
    findRecent: () => appClient.Get<ReleaseFindResponse>("api/release/recent"),
    findQuery: (offset?: number, limit?: number, filters?: ReleaseFilter[]) => {
      const params: Record<string, string[]> = {
        indexer: [],
        push_status: [],
        q: []
      };

      filters?.forEach((filter) => {
        if (!filter.value)
          return;

        if (filter.id == "indexer.identifier") {
          params["indexer"].push(filter.value);
        } else if (filter.id === "action_status") {
          params["push_status"].push(filter.value); // push_status is the correct value here otherwise the releases table won't load when filtered by push status
        } else if (filter.id === "push_status") {
          params["push_status"].push(filter.value);
        } else if (filter.id == "name") {
          params["q"].push(filter.value);
        }
      });

      return appClient.Get<ReleaseFindResponse>("api/release", {
        queryString: {
          offset,
          limit,
          ...params
        }
      });
    },
    indexerOptions: () => appClient.Get<string[]>("api/release/indexers"),
    stats: () => appClient.Get<ReleaseStats>("api/release/stats"),
    delete: (olderThan: number) => appClient.Delete("api/release", {
      queryString: { olderThan }
    }),
    replayAction: (releaseId: number, actionId: number) => appClient.Post(
      `api/release/${releaseId}/actions/${actionId}/retry`
    )
  },
  updates: {
    check: () => appClient.Get("api/updates/check"),
    getLatestRelease: () => appClient.Get<GithubRelease>("api/updates/latest")
  }
};
