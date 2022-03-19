import {baseUrl, sseBaseUrl} from "../utils";

interface ConfigType {
    body?: BodyInit | Record<string, unknown> | null;
    headers?: Record<string, string>;
}

export async function HttpClient<T>(
    endpoint: string,
    method: string,
    { body, ...customConfig }: ConfigType = {}
): Promise<T> {
    const config = {
        method: method,
        body: body ? JSON.stringify(body) : null,
        headers: {
            "Content-Type": "application/json"
        },
        // NOTE: customConfig can override the above defined settings
        ...customConfig
    } as RequestInit;

    return window.fetch(`${baseUrl()}${endpoint}`, config)
        .then(async response => {
            if ([401, 403, 404].includes(response.status))
                return Promise.reject(new Error(response.statusText));

            if ([201, 204].includes(response.status))
                return Promise.resolve(response);

            if (response.ok) {
                return await response.json();
            } else {
                const errorMessage = await response.text();
                return Promise.reject(new Error(errorMessage));
            }
        });
}

const appClient = {
    Get: <T>(endpoint: string) => HttpClient<T>(endpoint, "GET"),
    Post: (endpoint: string, data: any) => HttpClient<void>(endpoint, "POST", { body: data }),
    Put: (endpoint: string, data: any) => HttpClient<void>(endpoint, "PUT", { body: data }),
    Patch: (endpoint: string, data: any) => HttpClient<void>(endpoint, "PATCH", { body: data }),
    Delete: (endpoint: string) => HttpClient<void>(endpoint, "DELETE")
}

export const APIClient = {
    auth: {
        login: (username: string, password: string) => appClient.Post("api/auth/login", { username: username, password: password }),
        logout: () => appClient.Post("api/auth/logout", null),
        test: () => appClient.Get<void>("api/auth/test"),
    },
    actions: {
        create: (action: Action) => appClient.Post("api/actions", action),
        update: (action: Action) => appClient.Put(`api/actions/${action.id}`, action),
        delete: (id: number) => appClient.Delete(`api/actions/${id}`),
        toggleEnable: (id: number) => appClient.Patch(`api/actions/${id}/toggleEnabled`, null),
    },
    config: {
        get: () => appClient.Get<Config>("api/config")
    },
    download_clients: {
        getAll: () => appClient.Get<DownloadClient[]>("api/download_clients"),
        create: (dc: DownloadClient) => appClient.Post("api/download_clients", dc),
        update: (dc: DownloadClient) => appClient.Put("api/download_clients", dc),
        delete: (id: number) => appClient.Delete(`api/download_clients/${id}`),
        test: (dc: DownloadClient) => appClient.Post("api/download_clients/test", dc),
    },
    filters: {
        getAll: () => appClient.Get<Filter[]>("api/filters"),
        getByID: (id: number) => appClient.Get<Filter>(`api/filters/${id}`),
        create: (filter: Filter) => appClient.Post("api/filters", filter),
        update: (filter: Filter) => appClient.Put(`api/filters/${filter.id}`, filter),
        duplicate: (id: number) => appClient.Get<Filter>(`api/filters/${id}/duplicate`),
        toggleEnable: (id: number, enabled: boolean) => appClient.Put(`api/filters/${id}/enabled`, { enabled }),
        delete: (id: number) => appClient.Delete(`api/filters/${id}`),
    },
    indexers: {
        // returns indexer options for all currently present/enabled indexers
        getOptions: () => appClient.Get<Indexer[]>("api/indexer/options"),
        // returns indexer definitions for all currently present/enabled indexers
        getAll: () => appClient.Get<IndexerDefinition[]>("api/indexer"),
        // returns all possible indexer definitions
        getSchema: () => appClient.Get<IndexerDefinition[]>("api/indexer/schema"),
        create: (indexer: Indexer) => appClient.Post("api/indexer", indexer),
        update: (indexer: Indexer) => appClient.Put("api/indexer", indexer),
        delete: (id: number) => appClient.Delete(`api/indexer/${id}`),
    },
    irc: {
        getNetworks: () => appClient.Get<IrcNetworkWithHealth[]>("api/irc"),
        createNetwork: (network: IrcNetworkCreate) => appClient.Post("api/irc", network),
        updateNetwork: (network: IrcNetwork) => appClient.Put(`api/irc/network/${network.id}`, network),
        deleteNetwork: (id: number) => appClient.Delete(`api/irc/network/${id}`),
    },
    events: {
        logs: () => new EventSource(`${sseBaseUrl()}api/events?stream=logs`, { withCredentials: true })
    },
    release: {
        find: (query?: string) => appClient.Get<ReleaseFindResponse>(`api/release${query}`),
        findQuery: (offset?: number, limit?: number, filters?: Array<ReleaseFilter>) => {
            const params = new URLSearchParams();
            if (offset !== undefined)
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
            });

            return appClient.Get<ReleaseFindResponse>(`api/release?${params.toString()}`)
        },
        indexerOptions: () => appClient.Get<string[]>(`api/release/indexers`),
        stats: () => appClient.Get<ReleaseStats>("api/release/stats"),
        delete: () => appClient.Delete(`api/release/all`),
    }
};