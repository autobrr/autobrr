/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect } from "react";
import { RouterProvider } from "@tanstack/react-router"
import { QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "@components/hot-toast";
import { Router } from "@app/routes";
import { routerBasePath } from "@utils";
import { queryClient } from "@api/QueryClient";
import { SettingsContext } from "@utils/Context";
import { Portal } from "@components/portal";

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof Router
  }
}

export function App() {
  const [ , setSettings] = SettingsContext.use();

  useEffect(() => {
    const themeMediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleThemeChange = (e: MediaQueryListEvent) => {
      setSettings(prevState => ({ ...prevState, darkTheme: e.matches }));
    };

    themeMediaQuery.addEventListener('change', handleThemeChange);
    return () => themeMediaQuery.removeEventListener('change', handleThemeChange);
  }, [setSettings]);

  return (
    <QueryClientProvider client={queryClient}>
      <Portal>
        <Toaster position="top-right" />
      </Portal>
      <RouterProvider
        basepath={routerBasePath()}
        router={Router}
      />
    </QueryClientProvider>
  );
}
