/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import {QueryCache, QueryClient, QueryClientProvider, useQueryErrorResetBoundary} from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { ErrorBoundary } from "react-error-boundary";
import { toast, Toaster } from "react-hot-toast";

import { AuthContext, SettingsContext } from "./utils/Context";
import { ErrorPage } from "./components/alerts";
import Toast from "./components/notifications/Toast";
import { Portal } from "react-portal";
import {
  Outlet,
  RouterProvider,
  Link,
  Router,
  Route,
  RootRoute, rootRouteWithContext, redirect,
} from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/router-devtools'
import {Header} from "@components/header";
import {Suspense} from "react";
import {SectionLoader} from "@components/SectionLoader.tsx";
import {Dashboard} from "@screens/Dashboard.tsx";
import {FilterDetails, Filters} from "@screens/filters";
import {Section} from "@screens/filters/sections/_components.tsx";
import {Actions, Advanced, External, General, MoviesTv, Music} from "@screens/filters/sections";
import {Releases} from "@screens/Releases.tsx";
import {z} from "zod";
import {Settings} from "@screens/Settings.tsx";
import LogSettings from "@screens/settings/Logs.tsx";
import IndexerSettings from "@screens/settings/Indexer.tsx";
import IrcSettings from "@screens/settings/Irc.tsx";
import FeedSettings from "@screens/settings/Feed.tsx";
import DownloadClientSettings from "@screens/settings/DownloadClient.tsx";
import NotificationSettings from "@screens/settings/Notifications.tsx";
import APISettings from "@screens/settings/Api.tsx";
import ReleaseSettings from "@screens/settings/Releases.tsx";
import AccountSettings from "@screens/settings/Account.tsx";
import ApplicationSettings from "@screens/settings/Application.tsx";
import {Logs} from "@screens/Logs.tsx";
import {Login} from "@screens/auth";
import {APIClient} from "@api/APIClient.ts";
import {baseUrl} from "@utils";

export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error, query) => {
      // check for 401 and redirect here
      console.error("query cache error:", error)
      console.error("query cache query:", query)
      // @ts-ignore
      if (error?.status === 401 || error?.status === 403) {
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

const dashboardRoute = new Route({
  getParentRoute: () => authIndexRoute,
  path: '/',
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

export const filterRoute = new Route({
  getParentRoute: () => filtersRoute,
  path: '$filterId',
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
  // page: z.number().catch(1),
  filter: z.string().catch(''),
  // sort: z.enum(['newest', 'oldest', 'price']).catch('newest'),
})

type ReleasesSearch = z.infer<typeof releasesSearchSchema>

export const releasesIndexRoute = new Route({
  getParentRoute: () => releasesRoute,
  path: '/',
  component: Releases,
  validateSearch: (search) => releasesSearchSchema.parse(search),
})

const settingsRoute = new Route({
  getParentRoute: () => authIndexRoute,
  path: 'settings',
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
  component: LogSettings
})

export const settingsIndexersRoute = new Route({
  getParentRoute: () => settingsRoute,
  path: 'indexers',
  component: IndexerSettings
})

export const settingsIrcRoute = new Route({
  getParentRoute: () => settingsRoute,
  path: 'irc',
  component: IrcSettings
})

export const settingsFeedsRoute = new Route({
  getParentRoute: () => settingsRoute,
  path: 'feeds',
  component: FeedSettings
})

export const settingsClientsRoute = new Route({
  getParentRoute: () => settingsRoute,
  path: 'clients',
  component: DownloadClientSettings
})

export const settingsNotificationsRoute = new Route({
  getParentRoute: () => settingsRoute,
  path: 'notifications',
  component: NotificationSettings
})

export const settingsApiRoute = new Route({
  getParentRoute: () => settingsRoute,
  path: 'api',
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

export const loginRoute = new Route({
  getParentRoute: () => rootRoute,
  path: 'login',
  validateSearch: z.object({
    redirect: z.string().optional(),
  }),
}).update({component: Login})

const RootComponent = () => {
  return (
    <div className="min-h-screen">
      <Outlet />
      <TanStackRouterDevtools />
    </div>
  )
}

export type Auth = {
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
    console.log("before load")

    // If the user is not logged in, check for item in localStorage
    if (!context.auth.isLoggedIn) {
      console.log("before load: not logged in")
      const key = "user_auth"
      const storage = localStorage.getItem(key);
      if (storage) {
        try {
          const json = JSON.parse(storage);
          if (json === null) {
            console.warn(`JSON localStorage value for '${key}' context state is null`);
          } else {
            console.log("local storage found", json)
            console.log("ctx", context.auth)
            context.auth.isLoggedIn = json.isLoggedIn
            context.auth.username = json.username
            // context.auth = { ...json };
            console.log("ctx", context.auth)
          }
        } catch (e) {
          console.error(`Failed to merge ${key} context state: ${e}`);
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
      username: auth.username,
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
  auth: Auth
}>()({
  component: RootComponent,
})

const filterRouteTree = filtersRoute.addChildren([filterIndexRoute, filterRoute.addChildren([filterGeneralRoute, filterMoviesTvRoute, filterMusicRoute, filterAdvancedRoute, filterExternalRoute, filterActionsRoute])])
const settingsRouteTree = settingsRoute.addChildren([settingsIndexRoute, settingsLogRoute, settingsIndexersRoute, settingsIrcRoute, settingsFeedsRoute, settingsClientsRoute, settingsNotificationsRoute, settingsApiRoute, settingsReleasesRoute, settingsAccountRoute])

const authenticatedTree = authRoute.addChildren([authIndexRoute.addChildren([dashboardRoute, filterRouteTree, releasesRoute.addChildren([releasesIndexRoute]), settingsRouteTree, logsRoute])])

const routeTree = rootRoute.addChildren([
  authenticatedTree,
  loginRoute
])

const router = new Router({
  routeTree,
  context: {
    auth: undefined!, // We'll inject this when we render
  },
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

const auth: Auth = {
  isLoggedIn: false,
  // status: 'loggedOut',
  username: undefined,
  login: (username: string) => {
    auth.isLoggedIn = true
    auth.username = username

    localStorage.setItem("user_auth", JSON.stringify(auth));
  },
  logout: () => {
    auth.isLoggedIn = false
    auth.username = undefined

    localStorage.removeItem("user_auth");
  },
}

export function App() {
  // const { reset } = useQueryErrorResetBoundary();

  // const authContext = AuthContext.useValue();
  const settings = SettingsContext.useValue();

  return (
    // <ErrorBoundary
    //   onReset={reset}
    //   FallbackComponent={ErrorPage}
    // >
      <QueryClientProvider client={queryClient}>
        <Portal>
          <Toaster position="top-right" />
        </Portal>
        {/*<LocalRouter isLoggedIn={authContext.isLoggedIn} />*/}
        <RouterProvider
          basepath={baseUrl()}
          router={router}
          context={{
            auth,
          }}        />
        {settings.debug ? (
          <>
          <ReactQueryDevtools initialIsOpen={false} />
          </>
        ) : null}
      </QueryClientProvider>
    // </ErrorBoundary>
  );
}
