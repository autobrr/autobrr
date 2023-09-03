/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient, QueryClientProvider, useQueryErrorResetBoundary } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { ErrorBoundary } from "react-error-boundary";
import { toast, Toaster } from "react-hot-toast";

import { LocalRouter } from "./domain/routes";
import { AuthContext, SettingsContext } from "./utils/Context";
import { ErrorPage } from "./components/alerts";
import Toast from "./components/notifications/Toast";
import { Portal } from "react-portal";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // The retries will have exponential delay.
      // See https://tanstack.com/query/v4/docs/guides/query-retries#retry-delay
      // delay = Math.min(1000 * 2 ** attemptIndex, 30000)
      retry: true,
      useErrorBoundary: true
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

export function App() {
  const { reset } = useQueryErrorResetBoundary();

  const authContext = AuthContext.useValue();
  const settings = SettingsContext.useValue();

  return (
    <ErrorBoundary
      onReset={reset}
      FallbackComponent={ErrorPage}
    >
      <QueryClientProvider client={queryClient}>
        <Portal>
          <Toaster position="top-right" />
        </Portal>
        <LocalRouter isLoggedIn={authContext.isLoggedIn} />
        {settings.debug ? (
          <ReactQueryDevtools initialIsOpen={false} />
        ) : null}
      </QueryClientProvider>
    </ErrorBoundary>
  );
}
