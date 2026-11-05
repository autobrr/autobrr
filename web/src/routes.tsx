/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { z } from "zod/mini";
import {
  createRootRouteWithContext,
  createRoute,
  createRouter,
  lazyRouteComponent,
  notFound,
  redirect,
} from "@tanstack/react-router";
import { QueryClient } from "@tanstack/react-query";

import { APIClient } from "@api/APIClient";
import { NotFound } from "@components/alerts/NotFound";
import { FilterNotFound } from "@screens/filters/NotFound";
import { Settings } from "@screens/Settings";
import {
  ApikeysQueryOptions,
  CanOnboardQueryOptions,
  ConfigQueryOptions,
  DownloadersQueryOptions,
  FeedsQueryOptions,
  FilterByIdQueryOptions,
  IndexersQueryOptions,
  IrcQueryOptions, ListsQueryOptions,
  NotificationsQueryOptions,
  OIDCConfigQueryOptions,
  ProxiesQueryOptions
} from "@api/queries";
import { RingResizeSpinner } from "@components/Icons";
import { AuthContext, AuthInfo } from "@utils/Context";
import { claimAutoRedirect, consumeLoggedOut } from "@utils/autoLogin";
import { isSameOriginPath } from "@utils";
import { queryClient } from "@api/QueryClient";

import { ErrorPage } from "@components/alerts";
import { AuthenticatedLayout, RootComponent } from "@components/layouts";

const DashboardRoute = createRoute({
  getParentRoute: () => AuthIndexRoute,
  path: '/',
  loader: () => {
    // https://tanstack.com/router/v1/docs/guide/deferred-data-loading#deferred-data-loading-with-defer-and-await
    // TODO load stats

    // TODO load recent releases

    return {}
  },
  component: lazyRouteComponent(() => import("@screens/Dashboard"), "Dashboard"),
});

const FiltersRoute = createRoute({
  getParentRoute: () => AuthIndexRoute,
  path: 'filters'
});

const FilterIndexRoute = createRoute({
  getParentRoute: () => FiltersRoute,
  path: '/',
  component: lazyRouteComponent(() => import("@screens/filters/List"), "Filters"),
});

export const FilterGetByIdRoute = createRoute({
  getParentRoute: () => FiltersRoute,
  path: '$filterId',
  params: {
    parse: (params) => {
      if (!/^[1-9]\d*$/.test(params.filterId)) {
        throw notFound();
      }
      const filterId = z.int().safeParse(Number(params.filterId));
      if (!filterId.success) {
        throw notFound();
      }
      return { filterId: filterId.data };
    },
    stringify: ({ filterId }) => ({ filterId: `${filterId}` }),
  },
  loader: async ({ context, params }) => {
    try {
      const filter = await context.queryClient.query({ ...FilterByIdQueryOptions(params.filterId), staleTime: "static" })
      return { filter }
    } catch {
      throw notFound()
    }
  },
  component: lazyRouteComponent(() => import("@screens/filters/Details"), "FilterDetails"),
  notFoundComponent: () => {
    return <FilterNotFound />
  },
});

export const FilterGeneralRoute = createRoute({
  getParentRoute: () => FilterGetByIdRoute,
  path: '/',
  component: lazyRouteComponent(() => import("@screens/filters/sections/General"), "General")
});

export const FilterMoviesTvRoute = createRoute({
  getParentRoute: () => FilterGetByIdRoute,
  path: 'movies-tv',
  component: lazyRouteComponent(() => import("@screens/filters/sections/MoviesAndTV"), "MoviesTv")
});

export const FilterMusicRoute = createRoute({
  getParentRoute: () => FilterGetByIdRoute,
  path: 'music',
  component: lazyRouteComponent(() => import("@screens/filters/sections/Music"), "Music")
});

export const FilterBooksRoute = createRoute({
  getParentRoute: () => FilterGetByIdRoute,
  path: 'books',
  component: lazyRouteComponent(() => import("@screens/filters/sections/Books"), "Books")
});

export const FilterAdvancedRoute = createRoute({
  getParentRoute: () => FilterGetByIdRoute,
  path: 'advanced',
  component: lazyRouteComponent(() => import("@screens/filters/sections/Advanced"), "Advanced")
});

export const FilterExternalRoute = createRoute({
  getParentRoute: () => FilterGetByIdRoute,
  path: 'external',
  component: lazyRouteComponent(() => import("@screens/filters/sections/External"), "External")
});

export const FilterActionsRoute = createRoute({
  getParentRoute: () => FilterGetByIdRoute,
  path: 'actions',
  component: lazyRouteComponent(() => import("@screens/filters/sections/Actions"), "Actions")
});

export const FilterNotificationsRoute = createRoute({
  getParentRoute: () => FilterGetByIdRoute,
  path: 'notifications',
  component: lazyRouteComponent(() => import("@screens/filters/sections/Notifications"), "Notifications")
});

export const ReleasesRoute = createRoute({
  getParentRoute: () => AuthIndexRoute,
  path: 'releases',
  component: lazyRouteComponent(() => import("@screens/Releases"), "Releases"),
  validateSearch: (search) => z.object({
    page: z.optional(z.number()),
    pageSize: z.optional(z.number()),
    q: z.optional(z.string()),
    action_status: z.optional(z.enum(['PUSH_APPROVED', 'PUSH_REJECTED', 'PENDING', 'PUSH_ERROR', ''])),
    indexer: z.optional(z.string()),
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
  component: lazyRouteComponent(() => import("@screens/settings/Application"))
});

export const SettingsLogRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'logs',
  loader: (opts) => opts.context.queryClient.query({ ...ConfigQueryOptions(), staleTime: "static" }),
  component: lazyRouteComponent(() => import("@screens/settings/Logs"))
});

export const SettingsIndexersRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'indexers',
  loader: (opts) => opts.context.queryClient.query({ ...IndexersQueryOptions(), staleTime: "static" }),
  component: lazyRouteComponent(() => import("@screens/settings/Indexer"))
});

export const SettingsIrcRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'irc',
  loader: (opts) => opts.context.queryClient.query({ ...IrcQueryOptions(), staleTime: "static" }),
  component: lazyRouteComponent(() => import("@screens/settings/Irc"))
});

export const SettingsListsRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'lists',
  loader: (opts) => opts.context.queryClient.query({ ...ListsQueryOptions(), staleTime: "static" }),
  component: lazyRouteComponent(() => import("@screens/settings/Lists"))
});

export const SettingsFeedsRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'feeds',
  loader: (opts) => opts.context.queryClient.query({ ...FeedsQueryOptions(), staleTime: "static" }),
  component: lazyRouteComponent(() => import("@screens/settings/Feed"))
});

export const SettingsClientsRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'clients',
  loader: (opts) => opts.context.queryClient.query({ ...DownloadersQueryOptions(), staleTime: "static" }),
  component: lazyRouteComponent(() => import("@screens/settings/Downloaders"))
});

export const SettingsNotificationsRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'notifications',
  loader: (opts) => opts.context.queryClient.query({ ...NotificationsQueryOptions(), staleTime: "static" }),
  component: lazyRouteComponent(() => import("@screens/settings/Notifications"))
});

export const SettingsApiRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'api',
  loader: (opts) => {
    // Skip the prefetch in modes without key management so the (403'd) endpoint
    // isn't hit and the page can render its notice.
    if (!AuthContext.get().capabilities?.api_key_management) {
      return;
    }
    return opts.context.queryClient.query({ ...ApikeysQueryOptions(), staleTime: "static" });
  },
  component: lazyRouteComponent(() => import("@screens/settings/Api"))
});

export const SettingsProxiesRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'proxies',
  loader: (opts) => opts.context.queryClient.query({ ...ProxiesQueryOptions(), staleTime: "static" }),
  component: lazyRouteComponent(() => import("@screens/settings/Proxy"))
});

export const SettingsReleasesRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'releases',
  component: lazyRouteComponent(() => import("@screens/settings/Releases"))
});

export const SettingsAccountRoute = createRoute({
  getParentRoute: () => SettingsRoute,
  path: 'account',
  beforeLoad: async () => {
    if (!AuthContext.get().capabilities?.account_management) {
      throw redirect({ to: SettingsRoute.to });
    }
  },
  component: lazyRouteComponent(() => import("@screens/settings/Account"))
});

export const LogsRoute = createRoute({
  getParentRoute: () => AuthIndexRoute,
  path: 'logs',
  component: lazyRouteComponent(() => import("@screens/Logs"), "Logs")
});

export const NotificationInboxRoute = createRoute({
  getParentRoute: () => AuthIndexRoute,
  path: 'notifications',
  component: lazyRouteComponent(() => import("@screens/NotificationInbox"), "NotificationInbox"),
  validateSearch: (search) => z.object({
    page: z.catch(z.optional(z.int().check(z.nonnegative())), undefined),
    pageSize: z.catch(z.optional(z.literal([10, 25, 50, 100])), undefined),
    unread: z.optional(z.boolean()),
    event: z.optional(z.enum(['PUSH_ERROR', 'PUSH_REJECTED', 'PUSH_APPROVED', 'IRC_DISCONNECTED', 'IRC_RECONNECTED', 'IRC_UNHEALTHY', 'IRC_HEALTHY', 'LIST_REFRESH_ERROR', 'LIST_REFRESH_SUCCESS', 'FEED_REFRESH_ERROR', 'FEED_REFRESH_SUCCESS', 'APP_UPDATE_AVAILABLE'])),
  }).parse(search),
});

export const OnboardRoute = createRoute({
  getParentRoute: () => RootRoute,
  path: 'onboard',
  beforeLoad: async ({ context }) => {
    const canOnboard = await context.queryClient.query({ ...CanOnboardQueryOptions(), staleTime: "static" });
    if (!canOnboard) {
      console.error("onboarding not available, redirect to login")

      throw redirect({
        to: LoginRoute.to,
      })
    }
  },
  component: lazyRouteComponent(() => import("@screens/auth/Onboarding"), "Onboarding")
});

type ValidateResponse = Awaited<ReturnType<typeof APIClient.auth.validate>>;

const toAuthInfo = (validation: ValidateResponse, config?: Config, issuerUrl?: string): AuthInfo => ({
  isLoggedIn: true,
  username: validation.username || "unknown",
  authMethod: validation.auth_method,
  authMode: config?.auth_mode,
  capabilities: config?.auth_capabilities,
  logoutUrl: config?.auth_logout_url,
  profilePicture: validation.profile_picture,
  issuerUrl
});

const resolveAuth = async (client: QueryClient, config?: Config): Promise<AuthInfo> => {
  const validation = await APIClient.auth.validate();

  // Config and issuer errors must not log out a valid session. Without a mode, the
  // next navigation's reconciliation fetches the config again; issuerUrl is optional.
  const serverConfig = config ?? await client.query(ConfigQueryOptions()).catch(() => undefined);

  let issuerUrl;
  if (validation.auth_method === "oidc") {
    const oidcConfig = await client.query({ ...OIDCConfigQueryOptions(), staleTime: "static" }).catch(() => undefined);
    issuerUrl = oidcConfig?.issuerUrl;
  }

  return toAuthInfo(validation, serverConfig, issuerUrl);
};

const sameCapabilities = (a?: AuthCapabilities, b?: AuthCapabilities) => {
  if (!a || !b) {
    return false;
  }

  return (Object.keys(a) as (keyof AuthCapabilities)[]).every((key) => a[key] === b[key]);
};

export const LoginRoute = createRoute({
  getParentRoute: () => RootRoute,
  path: 'login',
  validateSearch: z.object({
    redirect: z.optional(z.string()),
    autoRedirect: z.catch(z.optional(z.boolean()), undefined),
  }),
  beforeLoad: async ({ context, navigate, preload, search }) => {
    // Modes without a login method have nothing to log in to, so bookmarks and stale
    // tabs on /login continue into the app instead of a form that always fails.
    const validation = await APIClient.auth.validate().catch(() => undefined);
    const config = validation && await context.queryClient.query({ ...ConfigQueryOptions(), staleTime: 0 }).catch(() => undefined);

    // Without the config, a caller that is neither a password nor an OIDC session
    // was authenticated by the mode itself, so there is no login form for it either.
    const noLoginMethod = config
      ? !config.auth_capabilities.password_login && !config.auth_capabilities.oidc_login
      : validation?.auth_method !== "password" && validation?.auth_method !== "oidc";
    if (validation && noLoginMethod) {
      AuthContext.set(toAuthInfo(validation, config));

      const target = search.redirect && isSameOriginPath(search.redirect) ? search.redirect : "/";
      throw redirect({ href: target });
    }

    // A preload must not fetch the OIDC config: each fetch starts a login on the
    // server that nobody completes. beforeLoad runs again on the real navigation.
    if (preload) {
      return;
    }

    if (consumeLoggedOut() && search.autoRedirect !== false) {
      throw redirect({ to: LoginRoute.to, search: { ...search, autoRedirect: false }, replace: true });
    }

    let oidcConfig;
    try {
      // Each login needs its own state, so a cached authorization URL is never reused.
      const target = search.redirect && isSameOriginPath(search.redirect) ? search.redirect : undefined;
      oidcConfig = await context.queryClient.query({
        ...OIDCConfigQueryOptions(),
        queryFn: () => APIClient.auth.getOIDCConfig(target),
        staleTime: 0
      });
    } catch (error) {
      console.debug("Failed to get OIDC config, proceeding with onboarding check:", error);
    }

    if (oidcConfig?.enabled) {
      // autoRedirect=false keeps the login page reachable when the provider is down,
      // and lets logout stay logged out while the provider session is still active.
      if (oidcConfig.autoRedirect && search.autoRedirect !== false && oidcConfig.authorizationUrl) {
        if (claimAutoRedirect()) {
          throw redirect({ href: oidcConfig.authorizationUrl, reloadDocument: true });
        }
        return { autoLoginFailed: true };
      }
      return;
    }

    const canOnboard = await context.queryClient.query({ ...CanOnboardQueryOptions(), staleTime: "static" });
    if (canOnboard) {
      console.info("onboarding available, redirecting");
      navigate({ to: OnboardRoute.to });
    } else {
      console.info("onboarding not available, please login");
    }
  },
}).update({ component: lazyRouteComponent(() => import("@screens/auth/Login"), "Login") });

export const AuthRoute = createRoute({
  getParentRoute: () => RootRoute,
  id: 'auth',
  // Before loading, authenticate the user via our auth context
  // This will also happen during prefetching (e.g. hovering over links, etc.)
  beforeLoad: async ({ context, location }) => {
    // If the user is not logged in, check for item in localStorage
    if (!AuthContext.get().isLoggedIn) {
      try {
        AuthContext.set(await resolveAuth(context.queryClient));
      } catch (error) {
        console.debug("Authentication validation failed:", error);
        throw redirect({
          to: LoginRoute.to,
          search: {
            redirect: location.publicHref,
          },
        });
      }
    } else {
      // Reconcile the persisted auth context with the server so switching auth
      // modes can't leave stale UI behind for a session that was already marked
      // logged in. The mode only changes across a restart, so an open tab must
      // refetch once its cached config goes stale.
      let config: Config | undefined;
      try {
        config = await context.queryClient.query(ConfigQueryOptions());
      } catch (error) {
        console.debug("auth mode reconciliation failed:", error);
      }

      const auth = AuthContext.get();
      if (config && config.auth_mode !== auth.authMode) {
        // The stored identity belongs to another mode; replace it with the server's,
        // or fall back to login if no session survives the switch.
        try {
          AuthContext.set(await resolveAuth(context.queryClient, config));
        } catch {
          AuthContext.reset();
          throw redirect({
            to: LoginRoute.to,
            search: {
              redirect: location.publicHref,
            },
          });
        }
      } else if (config && (!sameCapabilities(config.auth_capabilities, auth.capabilities) || config.auth_logout_url !== auth.logoutUrl)) {
        AuthContext.set({ ...auth, capabilities: config.auth_capabilities, logoutUrl: config.auth_logout_url });
      }
    }

    return context;
  },
})

export const AuthIndexRoute = createRoute({
  getParentRoute: () => AuthRoute,
  component: AuthenticatedLayout,
  id: 'authenticated-routes',
});

export const RootRoute = createRootRouteWithContext<{
  queryClient: QueryClient
}>()({
  component: RootComponent,
  notFoundComponent: NotFound,
});

const filterRouteTree = FiltersRoute.addChildren([FilterIndexRoute, FilterGetByIdRoute.addChildren([FilterGeneralRoute, FilterMoviesTvRoute, FilterMusicRoute, FilterBooksRoute, FilterAdvancedRoute, FilterExternalRoute, FilterActionsRoute, FilterNotificationsRoute])])
const settingsRouteTree = SettingsRoute.addChildren([SettingsIndexRoute, SettingsLogRoute, SettingsIndexersRoute, SettingsIrcRoute, SettingsListsRoute, SettingsFeedsRoute, SettingsClientsRoute, SettingsNotificationsRoute, SettingsApiRoute, SettingsProxiesRoute, SettingsReleasesRoute, SettingsAccountRoute])
const authenticatedTree = AuthRoute.addChildren([AuthIndexRoute.addChildren([DashboardRoute, filterRouteTree, ReleasesRoute, settingsRouteTree, LogsRoute, NotificationInboxRoute])])
const routeTree = RootRoute.addChildren([
  authenticatedTree,
  LoginRoute,
  OnboardRoute
]);

export const Router = createRouter({
  routeTree,
  defaultPreload: "intent",
  defaultPendingComponent: () => (
    <div className="flex grow items-center justify-center col-span-9">
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

