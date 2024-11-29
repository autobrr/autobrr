/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import {
  createRootRouteWithContext,
  createRoute,
  createRouter,
  Navigate,
  notFound,
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
import { FilterDetails, FilterNotFound, Filters } from "@screens/filters";
import { Settings } from "@screens/Settings";
import {
  ApikeysQueryOptions,
  ConfigQueryOptions,
  DownloadClientsQueryOptions,
  FeedsQueryOptions,
  FilterByIdQueryOptions,
  IndexersQueryOptions,
  IrcQueryOptions,
  NotificationsQueryOptions,
  ProxiesQueryOptions
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
import { AuthContext, SettingsContext } from "@utils/Context";
import { TanStackRouterDevtools } from "@tanstack/router-devtools";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { queryClient } from "@api/QueryClient";
import ProxySettings from "@screens/settings/Proxy";

import { ErrorPage } from "@components/alerts";

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
  stringifyParams: ({ filterId }) => ({ filterId: `${filterId}` }),
  loader: async ({ context, params }) => {
    try {
      const filter = await context.queryClient.ensureQueryData(FilterByIdQueryOptions(params.filterId))
      return { filter }
    } catch (e) {
      throw notFound()
    }
  },
  component: FilterDetails,
  notFoundComponent: () => {
    return <FilterNotFound />
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

export const ReleasesRoute = createRoute({
  getParentRoute: () => AuthIndexRoute,
  path: 'releases',
  component: Releases,
  validateSearch: (search) => z.object({
    offset: z.number().optional(),
    limit: z.number().optional(),
    filter: z.string().optional(),
    q: z.string().optional(),
    action_status: z.enum(['PUSH_APPROVED', 'PUSH_REJECTED', 'PUSH_ERROR', '']).optional(),
    // filters: z.array().catch(''),
    // sort: z.enum(['newest', 'oldest', 'price']).catch('newest'),
  }).parse(search),
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

export const SettingsProxiesRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'proxies',
  loader: (opts) => opts.context.queryClient.ensureQueryData(ProxiesQueryOptions()),
  component: ProxySettings
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
  beforeLoad: async ({ navigate }) => {
    // First check if OIDC is enabled
    try {
      const oidcConfig = await APIClient.auth.getOIDCConfig();
      if (oidcConfig.enabled) {
        // Skip onboarding check if OIDC is enabled
        return;
      }
    } catch (error) {
      console.debug("Failed to get OIDC config, proceeding with onboarding check");
    }

    // Only check onboarding if OIDC is not enabled
    try {
      await APIClient.auth.canOnboard();
      console.info("onboarding available, redirecting");
      navigate({ to: OnboardRoute.to });
    } catch (error) {
      console.info("onboarding not available, please login");
    }
  },
}).update({ component: Login });

export const AuthRoute = createRoute({
  getParentRoute: () => RootRoute,
  id: 'auth',
  // Before loading, authenticate the user via our auth context
  // This will also happen during prefetching (e.g. hovering over links, etc.)
  beforeLoad: async ({ context, location }) => {
    // If the user is not logged in, validate the session
    if (!AuthContext.get().isLoggedIn) {
      try {
        const response = await APIClient.auth.validate();
        // If validation succeeds, set the user as logged in
        AuthContext.set({
          isLoggedIn: true,
          username: response.username || 'unknown'
        });
      } catch (error) {
        throw redirect({
          to: LoginRoute.to,
          search: {
            // Use the current location to power a redirect after login
            // (Do not use `router.state.resolvedLocation` as it can
            // potentially lag behind the actual current location)
            redirect: location.href,
          },
        });
      }
    }

    // Otherwise, return the user in context
    return context;
  },
})

function AuthenticatedLayout() {
  const isLoggedIn = AuthContext.useSelector((s) => s.isLoggedIn);
  if (!isLoggedIn) {
    const redirect = (
      location.pathname.length > 1
        ? { redirect: location.pathname }
        : undefined
    );
    return <Navigate to="/login" search={redirect} />;
  }

  return (
    <div className="flex flex-col min-h-screen">
      <Header />
      <Outlet />
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
    <div className="flex flex-col min-h-screen">
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

export const RootRoute = createRootRouteWithContext<{
  queryClient: QueryClient
}>()({
  component: RootComponent,
  notFoundComponent: NotFound,
});

const filterRouteTree = FiltersRoute.addChildren([FilterIndexRoute, FilterGetByIdRoute.addChildren([FilterGeneralRoute, FilterMoviesTvRoute, FilterMusicRoute, FilterAdvancedRoute, FilterExternalRoute, FilterActionsRoute])])
const settingsRouteTree = SettingsRoute.addChildren([SettingsIndexRoute, SettingsLogRoute, SettingsIndexersRoute, SettingsIrcRoute, SettingsFeedsRoute, SettingsClientsRoute, SettingsNotificationsRoute, SettingsApiRoute, SettingsProxiesRoute, SettingsReleasesRoute, SettingsAccountRoute])
const authenticatedTree = AuthRoute.addChildren([AuthIndexRoute.addChildren([DashboardRoute, filterRouteTree, ReleasesRoute, settingsRouteTree, LogsRoute])])
const routeTree = RootRoute.addChildren([
  authenticatedTree,
  LoginRoute,
  OnboardRoute
]);

export const Router = createRouter({
  routeTree,
  defaultPendingComponent: () => (
    <div className="flex flex-grow items-center justify-center col-span-9">
      <RingResizeSpinner className="text-blue-500 size-24" />
    </div>
  ),
  defaultErrorComponent: (ctx) => (
    <ErrorPage error={ctx.error} reset={ctx.reset} />
  ),
  context: {
    queryClient
  },
});

