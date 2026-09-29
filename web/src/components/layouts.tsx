/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Navigate, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";

import { Header } from "@components/header";
import { AuthContext, SettingsContext } from "@utils/Context";

export function AuthenticatedLayout() {
  const isLoggedIn = AuthContext.useSelector((s) => s.isLoggedIn);
  if (!isLoggedIn) {
    return <Navigate to="/login" search={{ redirect: location.pathname + location.search }} />;
  }

  return (
    <div className="flex flex-col min-h-screen">
      <Header />
      <Outlet />
    </div>
  )
}

export const RootComponent = () => {
  const debug = SettingsContext.useSelector((s) => s.debug);
  return (
    <div className="flex flex-col min-h-screen">
      <Outlet />
      {debug ? (
        <>
          {import.meta.env.DEV && <TanStackRouterDevtools />}
          {import.meta.env.DEV && <ReactQueryDevtools initialIsOpen={false} />}
        </>
      ) : null}
    </div>
  )
}
