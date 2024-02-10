/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import {
  QueryCache,
  QueryClient,
  QueryClientProvider,
  queryOptions,
} from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { toast, Toaster } from "react-hot-toast";

import { SettingsContext } from "./utils/Context";
import Toast from "./components/notifications/Toast";
import { Portal } from "react-portal";
import {
  Outlet,
  RouterProvider,
  Router,
  Route,
  rootRouteWithContext,
  redirect,
  ErrorComponent,
} from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/router-devtools'
import { Header } from "@components/header";
import { Dashboard } from "@screens/Dashboard.tsx";
import { FilterDetails, Filters } from "@screens/filters";
import { Actions, Advanced, External, General, MoviesTv, Music } from "@screens/filters/sections";
import { Releases } from "@screens/Releases.tsx";
import { z } from "zod";
import { Settings } from "@screens/Settings.tsx";
import LogSettings from "@screens/settings/Logs.tsx";
import IndexerSettings, { indexerKeys } from "@screens/settings/Indexer.tsx";
import IrcSettings, { ircKeys } from "@screens/settings/Irc.tsx";
import FeedSettings, { feedKeys } from "@screens/settings/Feed.tsx";
import DownloadClientSettings, { clientKeys } from "@screens/settings/DownloadClient.tsx";
import NotificationSettings, { notificationKeys } from "@screens/settings/Notifications.tsx";
import APISettings, { apiKeys } from "@screens/settings/Api.tsx";
import ReleaseSettings from "@screens/settings/Releases.tsx";
import AccountSettings from "@screens/settings/Account.tsx";
import ApplicationSettings from "@screens/settings/Application.tsx";
import { Logs } from "@screens/Logs.tsx";
import { Login, Onboarding } from "@screens/auth";
import { APIClient } from "@api/APIClient.ts";
import { routerBasePath } from "@utils";
import { filterKeys } from "@screens/filters/List.tsx";
import { RingResizeSpinner } from "@components/Icons.tsx";

export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error, query) => {
      // check for 401 and redirect here
      console.error("query cache error:", error)
      console.error("query cache query:", query)
      // @ts-expect-error TS2339: Property status does not exist on type Error
      if (error?.status === 401 || error?.status === 403) {
        // @ts-expect-error TS2339: Property status does not exist on type Error
        console.error("bad status, redirect to login", error?.status)
        // Redirect to login page
        window.location.href = "/login";
      }
    }
  }),
  defaultOptions: {
    queries: {
      // The retries will have exponential delay.
      // See https://tanstack.com/query/v4/docs/guides/query-retries#retry-delay
      // delay = Math.min(1000 * 2 ** attemptIndex, 30000)
      // retry: true,
      throwOnError: false,
      retry: ( count) => {
        console.log("retry: ", count)
        return true
      }
    },
    mutations: {
      onError: (error) => {
        // Use a format string to convert the error object to a proper string without much hassle.
        const message = (
          typeof (error) === "object" && typeof ((error as Error).message) ?
            (error as Error).message :
            `${error}`
        );
        toast.custom((t) => <Toast type="error" body={message} t={t} />);
      }
    }
  }
});

// const filtersQueryOptions = () =>
//   queryOptions({
//     queryKey: ['filters'],
//     queryFn: () => APIClient.filters.find([], "")
//   })

export const filterQueryOptions = (filterId: number) =>
  queryOptions({
    queryKey: filterKeys.detail(filterId),
    queryFn: ({queryKey}) => APIClient.filters.getByID(queryKey[2])
  })

export const configQueryOptions = () =>
  queryOptions({
    queryKey: ["config"],
    queryFn: () => APIClient.config.get()
  })

export const indexersQueryOptions = () =>
  queryOptions({
    queryKey: indexerKeys.lists(),
    queryFn: () => APIClient.indexers.getAll()
  })

export const indexersOptionsQueryOptions = () =>
  queryOptions({
    queryKey: ["filters", "indexer_list"],
    queryFn: () => APIClient.indexers.getOptions(),
    refetchOnWindowFocus: false,
  })

export const ircQueryOptions = () =>
  queryOptions({
    queryKey: ircKeys.lists(),
    queryFn: () => APIClient.irc.getNetworks(),
    refetchOnWindowFocus: false,
    refetchInterval: 3000 // Refetch every 3 seconds
  })

export const feedsQueryOptions = () =>
  queryOptions({
    queryKey: feedKeys.lists(),
    queryFn: () => APIClient.feeds.find(),
  })

export const downloadClientsQueryOptions = () =>
  queryOptions({
    queryKey: clientKeys.lists(),
    queryFn: () => APIClient.download_clients.getAll(),
  })

export const notificationsQueryOptions = () =>
  queryOptions({
    queryKey: notificationKeys.lists(),
    queryFn: () => APIClient.notifications.getAll()
  })

export const apikeysQueryOptions = () =>
  queryOptions({
    queryKey: apiKeys.lists(),
    queryFn: () => APIClient.apikeys.getAll()
  })

const dashboardRoute = new Route({
  getParentRoute: () => authIndexRoute,
  path: '/',
  loader: () => {
    // https://tanstack.com/router/v1/docs/guide/deferred-data-loading#deferred-data-loading-with-defer-and-await
    // TODO load stats

    // TODO load recent releases

    return {}
  },
  component: Dashboard,
})

const filtersRoute = new Route({
  getParentRoute: () => authIndexRoute,
  path: 'filters'
})

const filterIndexRoute = new Route({
  getParentRoute: () => filtersRoute,
  path: '/',
  component: Filters
})

// export const filterRoute = new Route({
//   getParentRoute: () => filtersRoute,
//   path: '$filterId',
//   validateSearch: z.object({
//     filterId: z.number(),
//   }),
//   loaderDeps: ({ search }) => ({
//     filterId: search.filterId
//   }),
//   loader: (opts) => opts.context.queryClient.ensureQueryData(filterQueryOptions(opts.deps.filterId)),
//   component: FilterDetails
// })

export const filterRoute = new Route({
  getParentRoute: () => filtersRoute,
  path: '$filterId',
  parseParams: (params) => ({
    filterId: z.number().int().parse(Number(params.filterId)),
  }),
  stringifyParams: ({ filterId }) => ({ filterId: `${filterId}` }),
  // validateSearch: (search) => z.object({
  //   filterId: z.number(),
  // }),
  // loaderDeps: ({ search }) => ({
  //   filterId: search.filterId
  // }),
  // loaderDeps: (opts) => ({
  //   filterId: opts.search
  // }),
  loader: (opts) => {
    console.log("filter route loader")
    return opts.context.queryClient.ensureQueryData(filterQueryOptions(opts.params.filterId))
    // const filterData = opts.context.queryClient.ensureQueryData(filterQueryOptions(opts.params.filterId))
    // const indexersData = opts.context.queryClient.ensureQueryData(indexersOptionsQueryOptions())
    //
    // return {
    //   filterData,
    //   indexersData
    // }
  },
  component: FilterDetails
})

export const filterGeneralRoute = new Route({
  getParentRoute: () => filterRoute,
  path: '/',
  component: General
})

export const filterMoviesTvRoute = new Route({
  getParentRoute: () => filterRoute,
  path: 'movies-tv',
  component: MoviesTv
})

export const filterMusicRoute = new Route({
  getParentRoute: () => filterRoute,
  path: 'music',
  component: Music
})

export const filterAdvancedRoute = new Route({
  getParentRoute: () => filterRoute,
  path: 'advanced',
  component: Advanced
})

export const filterExternalRoute = new Route({
  getParentRoute: () => filterRoute,
  path: 'external',
  component: External
})

export const filterActionsRoute = new Route({
  getParentRoute: () => filterRoute,
  path: 'actions',
  component: Actions
})

const releasesRoute = new Route({
  getParentRoute: () => authIndexRoute,
  path: 'releases'
})

export const releasesSearchSchema = z.object({
  offset: z.number().optional(),
  limit: z.number().optional(),
  filter: z.string().optional(),
  q: z.string().optional(),
  action_status: z.enum(['PUSH_APPROVED', 'PUSH_REJECTED', 'PUSH_ERROR', '']).optional(),
  // filters: z.array().catch(''),
  // sort: z.enum(['newest', 'oldest', 'price']).catch('newest'),
})

// type ReleasesSearch = z.infer<typeof releasesSearchSchema>

export const releasesIndexRoute = new Route({
  getParentRoute: () => releasesRoute,
  path: '/',
  component: Releases,
  validateSearch: (search) => releasesSearchSchema.parse(search),
})

export const settingsRoute = new Route({
  getParentRoute: () => authIndexRoute,
  path: 'settings',
  pendingMs: 3000,
  component: Settings
})

export const settingsIndexRoute = new Route({
  getParentRoute: () => settingsRoute,
  path: '/',
  component: ApplicationSettings
})

export const settingsLogRoute = new Route({
  getParentRoute: () => settingsRoute,
  path: 'logs',
  loader: (opts) => opts.context.queryClient.ensureQueryData(configQueryOptions()),
  component: LogSettings
})

export const settingsIndexersRoute = new Route({
  getParentRoute: () => settingsRoute,
  path: 'indexers',
  loader: (opts) => opts.context.queryClient.ensureQueryData(indexersQueryOptions()),
  component: IndexerSettings
})

export const settingsIrcRoute = new Route({
  getParentRoute: () => settingsRoute,
  path: 'irc',
  loader: (opts) => opts.context.queryClient.ensureQueryData(ircQueryOptions()),
  component: IrcSettings
})

export const settingsFeedsRoute = new Route({
  getParentRoute: () => settingsRoute,
  path: 'feeds',
  loader: (opts) => opts.context.queryClient.ensureQueryData(feedsQueryOptions()),
  component: FeedSettings
})

export const settingsClientsRoute = new Route({
  getParentRoute: () => settingsRoute,
  path: 'clients',
  loader: (opts) => opts.context.queryClient.ensureQueryData(downloadClientsQueryOptions()),
  component: DownloadClientSettings
})

export const settingsNotificationsRoute = new Route({
  getParentRoute: () => settingsRoute,
  path: 'notifications',
  loader: (opts) => opts.context.queryClient.ensureQueryData(notificationsQueryOptions()),
  component: NotificationSettings
})

export const settingsApiRoute = new Route({
  getParentRoute: () => settingsRoute,
  path: 'api',
  loader: (opts) => opts.context.queryClient.ensureQueryData(apikeysQueryOptions()),
  component: APISettings
})

export const settingsReleasesRoute = new Route({
  getParentRoute: () => settingsRoute,
  path: 'releases',
  component: ReleaseSettings
})

export const settingsAccountRoute = new Route({
  getParentRoute: () => settingsRoute,
  path: 'account',
  component: AccountSettings
})

export const logsRoute = new Route({
  getParentRoute: () => authIndexRoute,
  path: 'logs',
  component: Logs
})

export const onboardRoute = new Route({
  getParentRoute: () => rootRoute,
  path: 'onboard',
  beforeLoad: async () => {
    // Check if onboarding is available for this instance
    // and redirect if needed
    try {
      await APIClient.auth.canOnboard()
    } catch (e) {
      console.error("onboarding not available, redirect to login")

      throw redirect({
        to: loginRoute.to,
      })
    }
  },
  component: Onboarding
})

export const loginRoute = new Route({
  getParentRoute: () => rootRoute,
  path: 'login',
  validateSearch: z.object({
    redirect: z.string().optional(),
  }),
  beforeLoad: async () => {
    console.log("login beforeLoad")

    // handle canOnboard
    try {
      await APIClient.auth.canOnboard()

      redirect({
        to: onboardRoute.to,
      })
    } catch (e) {
      console.log("onboarding not available")
    }
  },
}).update({component: Login})

/* COMMENT(martylukyy): This can probably be removed since the spinner works with pendingComponent?

  export function RouterSpinner() {
  const isLoading = useRouterState({ select: (s) => s.status === 'pending' })
  return <Spinner show={isLoading} />

}*/

const RootComponent = () => {
  const settings = SettingsContext.useValue();
  return (
    <div className="min-h-screen">
      <Outlet />
      {settings.debug ? (
        <>
          <TanStackRouterDevtools />
          <ReactQueryDevtools initialIsOpen={false} />
        </>
      ) : null}
    </div>
  )
}

export type AuthCtx = {
  isLoggedIn: boolean
  username?: string
  login: (username: string) => void
  logout: () => void
}

export const authRoute = new Route({
  getParentRoute: () => rootRoute,
  id: 'auth',
  // Before loading, authenticate the user via our auth context
  // This will also happen during prefetching (e.g. hovering over links, etc)
  beforeLoad: ({ context, location }) => {
    // If the user is not logged in, check for item in localStorage
    if (!context.auth.isLoggedIn) {
      const key = "user_auth"
      const storage = localStorage.getItem(key);
      if (storage) {
        try {
          const json = JSON.parse(storage);
          if (json === null) {
            console.warn(`JSON localStorage value for '${key}' context state is null`);
          } else {
            console.log("auth local storage found", json)
            console.log("auth ctx", context.auth)
            context.auth.isLoggedIn = json.isLoggedIn
            context.auth.username = json.username
            // context.auth = { ...json };
            console.log("auth ctx", context.auth)
          }
        } catch (e) {
          console.error(`auth Failed to merge ${key} context state: ${e}`);
        }
      } else {
        // If the user is logged out, redirect them to the login page
        throw redirect({
          to: loginRoute.to,
          search: {
            // Use the current location to power a redirect after login
            // (Do not use `router.state.resolvedLocation` as it can
            // potentially lag behind the actual current location)
            redirect: location.href,
          },
        })
      }
    }

    // Otherwise, return the user in context
    return {
      username: authCtx.username,
    }
  },
})

function AuthenticatedLayout() {
  return (
    <div className="min-h-screen">
      <Header />
      <Outlet />
    </div>
  )
}

export const authIndexRoute = new Route({
  getParentRoute: () => authRoute,
  component: AuthenticatedLayout,
  id: 'authenticated-routes',
})

export const rootRoute = rootRouteWithContext<{
  auth: AuthCtx,
  queryClient: QueryClient
}>()({
  component: RootComponent,
})

const filterRouteTree = filtersRoute.addChildren([filterIndexRoute, filterRoute.addChildren([filterGeneralRoute, filterMoviesTvRoute, filterMusicRoute, filterAdvancedRoute, filterExternalRoute, filterActionsRoute])])
const settingsRouteTree = settingsRoute.addChildren([settingsIndexRoute, settingsLogRoute, settingsIndexersRoute, settingsIrcRoute, settingsFeedsRoute, settingsClientsRoute, settingsNotificationsRoute, settingsApiRoute, settingsReleasesRoute, settingsAccountRoute])

const authenticatedTree = authRoute.addChildren([authIndexRoute.addChildren([dashboardRoute, filterRouteTree, releasesRoute.addChildren([releasesIndexRoute]), settingsRouteTree, logsRoute])])

const routeTree = rootRoute.addChildren([
  authenticatedTree,
  loginRoute,
  onboardRoute
])

const router = new Router({
  routeTree,
  defaultPendingComponent: () => (
    <div className="absolute top-1/4 left-1/2 !border-0">
      <RingResizeSpinner className="text-blue-500 size-24" />
    </div>
  ),
  defaultErrorComponent: ({ error }) => <ErrorComponent error={error} />,
  context: {
    auth: undefined!, // We'll inject this when we render
    queryClient
  },
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

export const authCtx: AuthCtx = {
  isLoggedIn: false,
  // status: 'loggedOut',
  username: undefined,
  login: (username: string) => {
    authCtx.isLoggedIn = true
    authCtx.username = username

    localStorage.setItem("user_auth", JSON.stringify(authCtx));
  },
  logout: () => {
    authCtx.isLoggedIn = false
    authCtx.username = undefined

    localStorage.removeItem("user_auth");
  },
}

export function App() {
  return (
      <QueryClientProvider client={queryClient}>
        <Portal>
          <Toaster position="top-right" />
        </Portal>
        <RouterProvider
          basepath={routerBasePath()}
          router={router}
          context={{
            auth: authCtx,
          }}        />
      </QueryClientProvider>
  );
}
