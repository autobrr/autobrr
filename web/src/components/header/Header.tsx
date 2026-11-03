/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { Disclosure, DisclosureButton } from "@headlessui/react";
import { Bars3Icon, XMarkIcon  } from "@heroicons/react/24/outline";
import { useTranslation } from "react-i18next";

import { APIClient } from "@api/APIClient";
import toast from "@components/hot-toast";
import Toast from "@components/notifications/Toast";

import { LeftNav } from "./LeftNav";
import { RightNav } from "./RightNav";
import { MobileNav } from "./MobileNav";
import { useInboxEvents } from "@hooks/useInbox";
import { AuthContext } from "@utils/Context";
import { InboxMenu } from "./InboxMenu";
import { AlertBanner } from "./AlertBanner";

export const Header = () => {
  const { t } = useTranslation("common");
  const navigate = useNavigate();

  useInboxEvents();

  // Navigate before resetting: a reset first makes AuthenticatedLayout send the
  // user to a plain /login, which starts a new OIDC login under oidcAutoRedirect.
  const logout = async () => {
    await navigate({ to: "/login", search: { autoRedirect: false } });
    AuthContext.reset();
  };

  const logoutMutation = useMutation({
    mutationFn: APIClient.auth.logout,
    onSuccess: async () => {
      toast.custom((toastInstance) => (
        <Toast type="success" body={t("header.logoutSuccess")} t={toastInstance} />
      ));
      await logout();
    },
    onError: async (error) => {
      // An expired session is already logged out. Any other failure may have left
      // the session alive, so keep the user signed in to retry.
      if (error.message === "Cookie expired or invalid.") {
        await logout();
        return;
      }

      toast.custom((toastInstance) => (
        <Toast type="error" body={error.message} t={toastInstance} />
      ));
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
                <div className="-mr-2 flex items-center gap-2 sm:hidden">
                  <InboxMenu />
                  {/* Mobile menu button */}
                  <DisclosureButton className="bg-gray-200 dark:bg-gray-800 inline-flex items-center justify-center p-2 rounded-md text-gray-600 dark:text-gray-400 hover:text-white hover:bg-gray-700 cursor-pointer">
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

            <AlertBanner />
          </div>

          <MobileNav logoutMutation={logoutMutation.mutate} />
        </>
      )}
    </Disclosure>
  );
};
