/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryCache, QueryClient } from "@tanstack/react-query";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { APIClient } from "@api/APIClient";
import { AuthContext } from "@utils/Context";

const MAX_RETRIES = 6;

export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error, query) => {
      console.error(`Caught error for query '${query.queryKey}': `, error);

      if (error.message === "Cookie expired or invalid.") {
        // several queries can fail at once; only the first one acts
        if (AuthContext.get().isLoggedIn) {
          // best effort: destroys the server-side session and clears the
          // cookie in case the session is still alive on the server
          APIClient.auth.logout().catch(() => {
            // the session was already dead server-side
          });
          // navigation to /login happens in AuthenticatedLayout, which
          // reacts to the reset while the location still points at the
          // page to return to; navigating from here too would race it
          AuthContext.reset();
        }
        return;
      } else {
        toast.custom((t) => <Toast type="error" body={ error?.message } t={ t }/>);
      }
    }
  }),
  defaultOptions: {
    queries: {
      // The retries will have exponential delay.
      // See https://tanstack.com/query/v4/docs/guides/query-retries#retry-delay
      // delay = Math.min(1000 * 2 ** attemptIndex, 30000)
      // retry: false,
      throwOnError: (error) => {
        return error.message !== "Cookie expired or invalid.";

      },
      retry: (failureCount, error) => {
        /*
        console.debug("retry count:", failureCount)
        console.error("retry err: ", error)

        // @ts-expect-error TS2339: ignore
        if (HTTP_STATUS_TO_NOT_RETRY.includes(error.status)) {
          // @ts-expect-error TS2339: ignore
          console.log(`retry: Aborting retry due to ${error.status} status`);
          return false;
        }
        */
        if (error.message === "Cookie expired or invalid.") {
          return false;
        }

        console.error(`Retrying query (N=${failureCount}): `, error);
        return failureCount <= MAX_RETRIES;
      },
    },
    mutations: {
      onError: (error) => {
        console.log("mutation error: ", error)

        // TODO: Maybe unneeded with our little HttpClient refactor.
        if (error instanceof Response) {
          return;
        }

        // Use a format string to convert the error object to a proper string without much hassle.
        const message = (
          typeof (error) === "object" && typeof ((error as Error).message) ?
            (error as Error).message :
            `${error}`
        );
        toast.custom((t) => <Toast type="error" body={message} t={t}/>);
      }
    }
  }
});
