/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryCache, QueryClient } from "@tanstack/react-query";
import { toast } from "react-hot-toast";
import Toast from "@components/notifications/Toast";
import { baseUrl } from "@utils";

const MAX_RETRIES = 6;
const HTTP_STATUS_TO_NOT_RETRY = [400, 401, 403, 404];

export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error ) => {
      console.error("query client error: ", error);

      toast.custom((t) => <Toast type="error" body={error?.message} t={t}/>);

      // @ts-expect-error TS2339: Property status does not exist on type Error
      if (error?.status === 401 || error?.status === 403) {
        // @ts-expect-error TS2339: Property status does not exist on type Error
        console.error("bad status, redirect to login", error?.status)
        // Redirect to login page
        window.location.href = baseUrl()+"login";

        return
      }
    }
  }),
  defaultOptions: {
    queries: {
      // The retries will have exponential delay.
      // See https://tanstack.com/query/v4/docs/guides/query-retries#retry-delay
      // delay = Math.min(1000 * 2 ** attemptIndex, 30000)
      // retry: false,
      throwOnError: true,
      retry: (failureCount, error) => {
        console.debug("retry count:", failureCount)
        console.error("retry err: ", error)

        // @ts-expect-error TS2339: ignore
        if (HTTP_STATUS_TO_NOT_RETRY.includes(error.status)) {
          // @ts-expect-error TS2339: ignore
          console.log(`retry: Aborting retry due to ${error.status} status`);
          return false;
        }

        return failureCount <= MAX_RETRIES;
      },
    },
    mutations: {
      onError: (error) => {
        console.log("mutation error: ", error)

        if (error instanceof Response) {
          return
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