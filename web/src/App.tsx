/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { RouterProvider } from "@tanstack/react-router"
import { QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "react-hot-toast";
import { Portal } from "react-portal";
import { Router } from "@app/routes";
import { routerBasePath } from "@utils";
import { queryClient } from "@api/QueryClient";
import { AuthProvider, useAuth } from "@ctx/auth";

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof Router
  }
}

function InnerApp() {
  const auth = useAuth()
  return <RouterProvider basepath={routerBasePath()} router={Router} context={{auth}} />
}

export function App() {
  return (
    <AuthProvider>
      <QueryClientProvider client={queryClient}>
        <Portal>
          <Toaster position="top-right" />
        </Portal>
          <InnerApp />
      </QueryClientProvider>
    </AuthProvider>
  );
}
