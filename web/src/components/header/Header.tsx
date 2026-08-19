/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQuery } from "@tanstack/react-query";
import { getRouteApi, redirect } from "@tanstack/react-router";
import { Disclosure, DisclosureButton } from "@headlessui/react";
import { Bars3Icon, ExclamationTriangleIcon, MegaphoneIcon, XMarkIcon  } from "@heroicons/react/24/outline";
import { useTranslation } from "react-i18next";

import { APIClient } from "@api/APIClient";
import toast from "@components/hot-toast";
import Toast from "@components/notifications/Toast";

import { LeftNav } from "./LeftNav";
import { RightNav } from "./RightNav";
import { MobileNav } from "./MobileNav";
import { ExternalLink } from "@components/ExternalLink";
import { ConfigQueryOptions, IrcQueryOptions, ListsQueryOptions, UpdatesQueryOptions } from "@api/queries";
import { AuthContext } from "@utils/Context";

import { isUnhealthyIrcNetwork } from "./ircStatus";

export const Header = () => {
  const { t } = useTranslation(["common", "settings"]);
  const loginRoute = getRouteApi("/login");

  const { data: config } = useQuery(ConfigQueryOptions(true));

  const { data } = useQuery(UpdatesQueryOptions(config?.check_for_updates === true));

  const { data: lists } = useQuery(ListsQueryOptions());

  const ircQuery = useQuery({
    ...IrcQueryOptions(),
    throwOnError: false,
  });

  // Check if the last run of any list has errored
  const hasErroredList = lists?.some(list => list.last_refresh_status === "ERROR");
  const erroredLists = lists?.filter(list => list.last_refresh_status === "ERROR");
  const unhealthyIrcNetworks = ircQuery.isError
    ? []
    : ircQuery.data?.filter(isUnhealthyIrcNetwork) ?? [];

  const logoutMutation = useMutation({
    mutationFn: APIClient.auth.logout,
    onSuccess: () => {
      toast.custom((toastInstance) => (
        <Toast type="success" body={t("header.logoutSuccess")} t={toastInstance} />
      ));
      AuthContext.reset();
      redirect({
        to: loginRoute.id,
      })
    },
    onError: () => {}
  });

  return (
    <Disclosure
      as="nav"
      className="bg-linear-to-b from-gray-100 dark:from-gray-925"
    >
      {({ open }) => (
        <>
          <div className="max-w-(--breakpoint-xl) mx-auto sm:px-6 lg:px-8">
            <div className="border-b border-gray-300 dark:border-gray-775">
              <div className="flex items-center justify-between h-16 px-4 sm:px-0">
                <LeftNav />
                <RightNav logoutMutation={logoutMutation.mutate} />
                <div className="-mr-2 flex sm:hidden">
                  {/* Mobile menu button */}
                  <DisclosureButton className="bg-gray-200 dark:bg-gray-800 inline-flex items-center justify-center p-2 rounded-md text-gray-600 dark:text-gray-400 hover:text-white hover:bg-gray-700">
                    <span className="sr-only">{t("header.openMainMenu")}</span>
                    {open ? (
                      <XMarkIcon
                        className="block h-6 w-6"
                        aria-hidden="true"
                      />
                    ) : (
                      <Bars3Icon
                        className="block h-6 w-6"
                        aria-hidden="true"
                      />
                    )}
                  </DisclosureButton>
                </div>
              </div>
            </div>

            {data?.html_url && (
              <ExternalLink href={data.html_url}>
                <div className="flex mt-4 py-2 bg-blue-500 rounded-sm justify-center">
                  <MegaphoneIcon className="h-6 w-6 text-blue-100" />
                  <span className="text-blue-100 font-medium mx-3">{t("header.newUpdateAvailable")}</span>
                  <span className="inline-flex items-center rounded-md bg-blue-100 px-2.5 py-0.5 text-sm font-medium text-blue-800">{data?.name}</span>
                </div>
              </ExternalLink>
            )}

            {unhealthyIrcNetworks.length > 0 && (
              <div className="mt-4 flex flex-wrap items-center justify-center gap-x-3 gap-y-1 rounded-sm bg-red-500 px-3 py-2">
                <span className="flex shrink-0 items-center">
                  <ExclamationTriangleIcon className="h-6 w-6 text-red-100" />
                  <span className="mx-3 font-medium text-red-100">
                    IRC: {t("settings:forms.irc.networkUnhealthy")}
                  </span>
                </span>
                <span className="flex min-w-0 flex-wrap justify-center gap-1">
                  {unhealthyIrcNetworks.map(network => (
                    <span
                      key={network.id}
                      className="inline-flex max-w-full items-center rounded-md bg-red-100 px-2.5 py-0.5 text-sm font-medium text-red-800"
                      title={network.connection_errors.join(", ") || undefined}
                    >
                      <span className="truncate">{network.name}</span>
                    </span>
                  ))}
                </span>
              </div>
            )}

            {hasErroredList && (
              <div className="flex mt-4 py-2 bg-red-500 rounded-sm justify-center">
                <ExclamationTriangleIcon className="h-6 w-6 text-red-100" />
                <span className="text-red-100 font-medium mx-3">
                  {erroredLists?.length === 1 ? t("header.listRefreshFailed") : t("header.multipleListRefreshesFailed")}
                </span>
                {erroredLists?.length === 1 ? (
                  <span className="inline-flex items-center rounded-md bg-red-100 px-2.5 py-0.5 text-sm font-medium text-red-800">
                    {erroredLists[0].name}
                  </span>
                ) : (
                  erroredLists?.map(list => (
                    <span key={list.name} className="inline-flex items-center rounded-md bg-red-100 px-2.5 py-0.5 text-sm font-medium text-red-800 ml-1">
                      {list.name}
                    </span>
                  ))
                )}
              </div>
            )}
          </div>

          <MobileNav logoutMutation={logoutMutation.mutate} />
        </>
      )}
    </Disclosure>
  );
};
