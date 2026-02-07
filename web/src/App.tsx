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
import { SettingsContext, isDarkTheme } from "@utils/Context";
import { Portal } from "@components/portal";

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof Router
  }
}

export function App() {
  useEffect(() => {
    const themeMediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const handleThemeChange = () => {
      const settings = SettingsContext.get();
      if (settings.theme === "system") {
        // Re-apply theme when OS preference changes
        const dark = isDarkTheme("system");
        document.documentElement.classList.toggle("dark", dark);
      }
    };

    themeMediaQuery.addEventListener("change", handleThemeChange);
    return () => themeMediaQuery.removeEventListener("change", handleThemeChange);
  }, []);

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
