/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import toast from "react-hot-toast";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Disclosure } from "@headlessui/react";
import { Bars3Icon, XMarkIcon, MegaphoneIcon } from "@heroicons/react/24/outline";

import { APIClient } from "@api/APIClient";
import { AuthContext } from "@utils/Context";
import Toast from "@components/notifications/Toast";

import { LeftNav } from "./LeftNav";
import { RightNav } from "./RightNav";
import { MobileNav } from "./MobileNav";
import { ExternalLink } from "@components/ExternalLink";

export const Header = () => {
  const { data: config } = useQuery({
    queryKey: ["config"],
    queryFn: () => APIClient.config.get(),
    retry: false,
    refetchOnWindowFocus: false,
    onError: err => console.log(err)
  });

  const { data } = useQuery({
    queryKey: ["updates"],
    queryFn: () => APIClient.updates.getLatestRelease(),
    retry: false,
    refetchOnWindowFocus: false,
    enabled: config?.check_for_updates === true,
    onError: err => console.log(err)
  });

  const logoutMutation = useMutation({
    mutationFn: APIClient.auth.logout,
    onSuccess: () => {
      AuthContext.reset();
      toast.custom((t) => (
        <Toast type="success" body="You have been logged out. Goodbye!" t={t} />
      ));
    }
  });

  return (
    <Disclosure
      as="nav"
      className="bg-gradient-to-b from-gray-100 dark:from-[#141414]"
    >
      {({ open }) => (
        <>
          <div className="max-w-screen-xl mx-auto sm:px-6 lg:px-8">
            <div className="border-b border-gray-300 dark:border-gray-700">
              <div className="flex items-center justify-between h-16 px-4 sm:px-0">
                <LeftNav />
                <RightNav logoutMutation={logoutMutation.mutate} />
                <div className="-mr-2 flex sm:hidden">
                  {/* Mobile menu button */}
                  <Disclosure.Button className="bg-gray-200 dark:bg-gray-800 inline-flex items-center justify-center p-2 rounded-md text-gray-600 dark:text-gray-400 hover:text-white hover:bg-gray-700">
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
                  </Disclosure.Button>
                </div>
              </div>
            </div>

            {data?.html_url && (
              <ExternalLink href={data.html_url}>
                <div className="flex mt-4 py-2 bg-blue-500 rounded justify-center">
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
