import {Action, DownloadClient, Filter, Indexer, Network} from "../domain/interfaces";
import {baseUrl} from "../utils/utils";

function baseClient(endpoint: string, method: string, { body, ...customConfig}: any = {}) {
    let baseURL = baseUrl()

    const headers = {'content-type': 'application/json'}
    const config = {
        method: method,
        ...customConfig,
        headers: {
            ...headers,
            ...customConfig.headers,
        },
    }

    if (body) {
        config.body = JSON.stringify(body)
    }

    return window.fetch(`${baseURL}${endpoint}`, config)
        .then(async response => {
            if (response.status === 401) {
                // unauthorized
                // window.location.assign(window.location)

                return
            }

            if (response.status === 403) {
                // window.location.assign("/login")
                return Promise.reject(new Error(response.statusText))
                // return
            }

            if (response.status === 404) {
                return Promise.reject(new Error(response.statusText))
            }

            if (response.status === 201) {
                return ""
            }

            if (response.status === 204) {
                return ""
            }

            if (response.ok) {
                return await response.json()
            } else {
                const errorMessage = await response.text()

                return Promise.reject(new Error(errorMessage))
            }
        })
}

const appClient = {
    Get: (endpoint: string) => baseClient(endpoint, "GET"),
    Post: (endpoint: string, data: any) => baseClient(endpoint, "POST", { body: data }),
    Put: (endpoint: string, data: any) => baseClient(endpoint, "PUT", { body: data }),
    Patch: (endpoint: string, data: any) => baseClient(endpoint, "PATCH", { body: data }),
    Delete: (endpoint: string) => baseClient(endpoint, "DELETE"),
}

const APIClient = {
    auth: {
        login: (username: string, password: string) => appClient.Post("api/auth/login", {username: username, password: password}),
        logout: () => appClient.Post(`api/auth/logout`, null),
        test: () => appClient.Get(`api/auth/test`),
    },
    actions: {
        create: (action: Action) => appClient.Post("api/actions", action),
        update: (action: Action) => appClient.Put(`api/actions/${action.id}`, action),
        delete: (id: number) => appClient.Delete(`api/actions/${id}`),
        toggleEnable: (id: number) => appClient.Patch(`api/actions/${id}/toggleEnabled`, null),
    },
    config: {
        get: () => appClient.Get("api/config")
    },
    download_clients: {
        getAll: () => appClient.Get("api/download_clients"),
        create: (dc: DownloadClient) => appClient.Post(`api/download_clients`, dc),
        update: (dc: DownloadClient) => appClient.Put(`api/download_clients`, dc),
        delete: (id: number) => appClient.Delete(`api/download_clients/${id}`),
        test: (dc: DownloadClient) => appClient.Post(`api/download_clients/test`, dc),
    },
    filters: {
        getAll: () => appClient.Get("api/filters"),
        getByID: (id: number) => appClient.Get(`api/filters/${id}`),
        create: (filter: Filter) => appClient.Post(`api/filters`, filter),
        update: (filter: Filter) => appClient.Put(`api/filters/${filter.id}`, filter),
        delete: (id: number) => appClient.Delete(`api/filters/${id}`),
    },
    indexers: {
        getOptions: () => appClient.Get("api/indexer/options"),
        getAll: () => appClient.Get("api/indexer"),
        getSchema: () => appClient.Get("api/indexer/schema"),
        create: (indexer: Indexer) => appClient.Post(`api/indexer`, indexer),
        update: (indexer: Indexer) => appClient.Put(`api/indexer`, indexer),
        delete: (id: number) => appClient.Delete(`api/indexer/${id}`),
    },
    irc: {
        getNetworks: () => appClient.Get("api/irc"),
        createNetwork: (network: Network) => appClient.Post(`api/irc`, network),
        updateNetwork: (network: Network) => appClient.Put(`api/irc/network/${network.id}`, network),
        deleteNetwork: (id: number) => appClient.Delete(`api/irc/network/${id}`),
    },
}

export default APIClient;