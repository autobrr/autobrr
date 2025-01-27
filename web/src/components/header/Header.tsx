/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQuery } from "@tanstack/react-query";
import { getRouteApi, redirect } from "@tanstack/react-router";
import { Disclosure, DisclosureButton } from "@headlessui/react";
import { Bars3Icon, XMarkIcon, MegaphoneIcon } from "@heroicons/react/24/outline";

import { APIClient } from "@api/APIClient";
import toast from "@components/hot-toast";
import Toast from "@components/notifications/Toast";

import { LeftNav } from "./LeftNav";
import { RightNav } from "./RightNav";
import { MobileNav } from "./MobileNav";
import { ExternalLink } from "@components/ExternalLink";
import { ConfigQueryOptions, UpdatesQueryOptions } from "@api/queries";
import { AuthContext } from "@utils/Context";

export const Header = () => {
  const loginRoute = getRouteApi("/login");

  const { isError:isConfigError, error: configError, data: config } = useQuery(ConfigQueryOptions(true));
  if (isConfigError) {
    console.log(configError);
  }

  const { isError: isUpdateError, error, data } = useQuery(UpdatesQueryOptions(config?.check_for_updates === true));
  if (isUpdateError) {
    console.log("update error", error);
  }

  const logoutMutation = useMutation({
    mutationFn: APIClient.auth.logout,
    onSuccess: () => {
      toast.custom((t) => (
        <Toast type="success" body="You have been logged out. Goodbye!" t={t} />
      ));
      AuthContext.reset();
      throw redirect({
        to: loginRoute.id,
      })
    },
    onError: (err) => {
      console.error("logout error", err)
    }
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
                    <span className="sr-only">Open main menu</span>
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
                  <span className="text-blue-100 font-medium mx-3">New update available!</span>
                  <span className="inline-flex items-center rounded-md bg-blue-100 px-2.5 py-0.5 text-sm font-medium text-blue-800">{data?.name}</span>
                </div>
              </ExternalLink>
            )}
          </div>

          <MobileNav logoutMutation={logoutMutation.mutate} />
        </>
      )}
    </Disclosure>
  );
};
