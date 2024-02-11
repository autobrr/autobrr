/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import {
  createRootRouteWithContext,
  createRoute,
  createRouter,
  ErrorComponent, notFound,
  Outlet,
  redirect,
} from "@tanstack/react-router";
import { z } from "zod";
import { QueryClient } from "@tanstack/react-query";

import { Actions, Advanced, External, General, MoviesTv, Music } from "@screens/filters/sections";
import { APIClient } from "@api/APIClient";
import { Login, Onboarding } from "@screens/auth";
import ReleaseSettings from "@screens/settings/Releases";
import { NotFound } from "@components/alerts/NotFound";
import { FilterDetails, Filters } from "@screens/filters";
import { Settings } from "@screens/Settings";
import {
  ApikeysQueryOptions,
  ConfigQueryOptions,
  DownloadClientsQueryOptions,
  FeedsQueryOptions,
  FilterByIdQueryOptions,
  IndexersQueryOptions,
  IrcQueryOptions,
  NotificationsQueryOptions
} from "@api/queries";
import LogSettings from "@screens/settings/Logs";
import NotificationSettings from "@screens/settings/Notifications";
import ApplicationSettings from "@screens/settings/Application";
import { Logs } from "@screens/Logs";
import IrcSettings from "@screens/settings/Irc";
import { Header } from "@components/header";
import { RingResizeSpinner } from "@components/Icons";
import APISettings from "@screens/settings/Api";
import { Releases } from "@screens/Releases";
import IndexerSettings from "@screens/settings/Indexer";
import DownloadClientSettings from "@screens/settings/DownloadClient";
import FeedSettings from "@screens/settings/Feed";
import { Dashboard } from "@screens/Dashboard";
import AccountSettings from "@screens/settings/Account";
import { SettingsContext } from "@utils/Context";
import { TanStackRouterDevtools } from "@tanstack/router-devtools";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { queryClient } from "@api/QueryClient";
import { LogDebug } from "@components/debug";
import { FilterNotFound } from "@screens/filters";

const DashboardRoute = createRoute({
  getParentRoute: () => AuthIndexRoute,
  path: '/',
  loader: () => {
    // https://tanstack.com/router/v1/docs/guide/deferred-data-loading#deferred-data-loading-with-defer-and-await
    // TODO load stats

    // TODO load recent releases

    return {}
  },
  component: Dashboard,
});

const FiltersRoute = createRoute({
  getParentRoute: () => AuthIndexRoute,
  path: 'filters'
});

const FilterIndexRoute = createRoute({
  getParentRoute: () => FiltersRoute,
  path: '/',
  component: Filters,
});

export const FilterGetByIdRoute = createRoute({
  getParentRoute: () => FiltersRoute,
  path: '$filterId',
  parseParams: (params) => ({
    filterId: z.number().int().parse(Number(params.filterId)),
  }),
  stringifyParams: ({filterId}) => ({filterId: `${filterId}`}),
  loader: async ({context, params}) => {
    try {
      const filter = await context.queryClient.ensureQueryData(FilterByIdQueryOptions(params.filterId))
      return { filter }
    } catch (e) {
      throw notFound()
    }
  },
  component: FilterDetails,
  notFoundComponent: () => {
    const { filterId} = FilterGetByIdRoute.useParams()
    return <FilterNotFound filterId={filterId} />
  },
});

export const FilterGeneralRoute = createRoute({
  getParentRoute: () => FilterGetByIdRoute,
  path: '/',
  component: General
});

export const FilterMoviesTvRoute = createRoute({
  getParentRoute: () => FilterGetByIdRoute,
  path: 'movies-tv',
  component: MoviesTv
});

export const FilterMusicRoute = createRoute({
  getParentRoute: () => FilterGetByIdRoute,
  path: 'music',
  component: Music
});

export const FilterAdvancedRoute = createRoute({
  getParentRoute: () => FilterGetByIdRoute,
  path: 'advanced',
  component: Advanced
});

export const FilterExternalRoute = createRoute({
  getParentRoute: () => FilterGetByIdRoute,
  path: 'external',
  component: External
});

export const FilterActionsRoute = createRoute({
  getParentRoute: () => FilterGetByIdRoute,
  path: 'actions',
  component: Actions
});

const ReleasesRoute = createRoute({
  getParentRoute: () => AuthIndexRoute,
  path: 'releases'
});

export const releasesSearchSchema = z.object({
  offset: z.number().optional(),
  limit: z.number().optional(),
  filter: z.string().optional(),
  q: z.string().optional(),
  action_status: z.enum(['PUSH_APPROVED', 'PUSH_REJECTED', 'PUSH_ERROR', '']).optional(),
  // filters: z.array().catch(''),
  // sort: z.enum(['newest', 'oldest', 'price']).catch('newest'),
});

// type ReleasesSearch = z.infer<typeof releasesSearchSchema>

export const ReleasesIndexRoute = createRoute({
  getParentRoute: () => ReleasesRoute,
  path: '/',
  component: Releases,
  validateSearch: (search) => releasesSearchSchema.parse(search),
});

export const SettingsRoute = createRoute({
  getParentRoute: () => AuthIndexRoute,
  path: 'settings',
  pendingMs: 3000,
  component: Settings
});

export const SettingsIndexRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: '/',
  component: ApplicationSettings
});

export const SettingsLogRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'logs',
  loader: (opts) => opts.context.queryClient.ensureQueryData(ConfigQueryOptions()),
  component: LogSettings
});

export const SettingsIndexersRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'indexers',
  loader: (opts) => opts.context.queryClient.ensureQueryData(IndexersQueryOptions()),
  component: IndexerSettings
});

export const SettingsIrcRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'irc',
  loader: (opts) => opts.context.queryClient.ensureQueryData(IrcQueryOptions()),
  component: IrcSettings
});

export const SettingsFeedsRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'feeds',
  loader: (opts) => opts.context.queryClient.ensureQueryData(FeedsQueryOptions()),
  component: FeedSettings
});

export const SettingsClientsRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'clients',
  loader: (opts) => opts.context.queryClient.ensureQueryData(DownloadClientsQueryOptions()),
  component: DownloadClientSettings
});

export const SettingsNotificationsRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'notifications',
  loader: (opts) => opts.context.queryClient.ensureQueryData(NotificationsQueryOptions()),
  component: NotificationSettings
});

export const SettingsApiRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'api',
  loader: (opts) => opts.context.queryClient.ensureQueryData(ApikeysQueryOptions()),
  component: APISettings
});

export const SettingsReleasesRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'releases',
  component: ReleaseSettings
});

export const SettingsAccountRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'account',
  component: AccountSettings
});

export const LogsRoute = createRoute({
  getParentRoute: () => AuthIndexRoute,
  path: 'logs',
  component: Logs
});

export const OnboardRoute = createRoute({
  getParentRoute: () => RootRoute,
  path: 'onboard',
  beforeLoad: async () => {
    // Check if onboarding is available for this instance
    // and redirect if needed
    try {
      await APIClient.auth.canOnboard()
    } catch (e) {
      console.error("onboarding not available, redirect to login")

      throw redirect({
        to: LoginRoute.to,
      })
    }
  },
  component: Onboarding
});

export const LoginRoute = createRoute({
  getParentRoute: () => RootRoute,
  path: 'login',
  validateSearch: z.object({
    redirect: z.string().optional(),
  }),
  beforeLoad: async () => {
    // handle canOnboard
    try {
      await APIClient.auth.canOnboard()

      redirect({
        to: OnboardRoute.to,
      })
    } catch (e) {
      console.log("onboarding not available")
    }
  },
}).update({component: Login});

export type AuthCtx = {
  isLoggedIn: boolean
  username?: string
  login: (username: string) => void
  logout: () => void
}

const localStorageUserKey = "autobrr_user_auth"

export const AuthRoute = createRoute({
  getParentRoute: () => RootRoute,
  id: 'auth',
  // Before loading, authenticate the user via our auth context
  // This will also happen during prefetching (e.g. hovering over links, etc)
  beforeLoad: ({context, location}) => {
    // If the user is not logged in, check for item in localStorage
    if (!context.auth.isLoggedIn) {
      const storage = localStorage.getItem(localStorageUserKey);
      if (storage) {
        try {
          const json = JSON.parse(storage);
          if (json === null) {
            console.warn(`JSON localStorage value for '${localStorageUserKey}' context state is null`);
          } else {
            LogDebug("auth local storage found", json)

            context.auth.isLoggedIn = json.isLoggedIn
            context.auth.username = json.username

            LogDebug("auth ctx", context.auth)
          }
        } catch (e) {
          console.error(`auth Failed to merge ${localStorageUserKey} context state: ${e}`);
        }
      } else {
        // If the user is logged out, redirect them to the login page
        throw redirect({
          to: LoginRoute.to,
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
      username: AuthContext.username,
    }
  },
})

function AuthenticatedLayout() {
  return (
    <div className="min-h-screen">
      <Header/>
      <Outlet/>
    </div>
  )
}

export const AuthIndexRoute = createRoute({
  getParentRoute: () => AuthRoute,
  component: AuthenticatedLayout,
  id: 'authenticated-routes',
});

export const RootComponent = () => {
  const settings = SettingsContext.useValue();
  return (
    <div className="min-h-screen">
      <Outlet/>
      {settings.debug ? (
        <>
          <TanStackRouterDevtools/>
          <ReactQueryDevtools initialIsOpen={false}/>
        </>
      ) : null}
    </div>
  )
}

export const RootRoute = createRootRouteWithContext<{
  auth: AuthCtx,
  queryClient: QueryClient
}>()({
  component: RootComponent,
  notFoundComponent: NotFound,
});

const filterRouteTree = FiltersRoute.addChildren([FilterIndexRoute, FilterGetByIdRoute.addChildren([FilterGeneralRoute, FilterMoviesTvRoute, FilterMusicRoute, FilterAdvancedRoute, FilterExternalRoute, FilterActionsRoute])])
const settingsRouteTree = SettingsRoute.addChildren([SettingsIndexRoute, SettingsLogRoute, SettingsIndexersRoute, SettingsIrcRoute, SettingsFeedsRoute, SettingsClientsRoute, SettingsNotificationsRoute, SettingsApiRoute, SettingsReleasesRoute, SettingsAccountRoute])
const authenticatedTree = AuthRoute.addChildren([AuthIndexRoute.addChildren([DashboardRoute, filterRouteTree, ReleasesRoute.addChildren([ReleasesIndexRoute]), settingsRouteTree, LogsRoute])])
const routeTree = RootRoute.addChildren([
  authenticatedTree,
  LoginRoute,
  OnboardRoute
]);

export const Router = createRouter({
  routeTree,
  defaultPendingComponent: () => (
    <div className="absolute top-1/4 left-1/2 !border-0">
      <RingResizeSpinner className="text-blue-500 size-24"/>
    </div>
  ),
  defaultErrorComponent: ({error}) => <ErrorComponent error={error}/>,
  context: {
    auth: undefined!, // We'll inject this when we render
    queryClient
  },
});

export const AuthContext: AuthCtx = {
  isLoggedIn: false,
  username: undefined,
  login: (username: string) => {
    AuthContext.isLoggedIn = true
    AuthContext.username = username

    localStorage.setItem(localStorageUserKey, JSON.stringify(AuthContext));
  },
  logout: () => {
    AuthContext.isLoggedIn = false
    AuthContext.username = undefined

    localStorage.removeItem(localStorageUserKey);
  },
}