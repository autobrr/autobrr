/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Link } from "@tanstack/react-router";
import { DisclosurePanel } from "@headlessui/react";

import { classNames } from "@utils";

import { NAV_ROUTES } from "./_shared";
import type { RightNavProps } from "./_shared";
import { AuthContext } from "@utils/Context";

export const MobileNav = (props: RightNavProps) => {
  const auth = AuthContext.get();
  const isNoAuth = auth.authMethod === 'none';

  return (
    <DisclosurePanel className="border-b border-gray-300 dark:border-gray-700 md:hidden">
      <div className="px-2 py-3 space-y-1 sm:px-3">
        {NAV_ROUTES.map((item) => (
          <Link
            key={item.path}
            activeOptions={{ exact: item.exact }}
            to={item.path}
            search={{}}
            params={{}}
          >
            {({ isActive }) => {
              return (
                <span className={
                  classNames(
                    "shadow-xs border bg-gray-100 border-gray-300 dark:border-gray-700 dark:bg-gray-900 dark:text-white block px-3 py-2 rounded-md text-base",
                    isActive
                      ? "underline underline-offset-2 decoration-2 decoration-sky-500 font-bold text-black"
                      : "font-medium"
                  )
                }>
                  {item.name}
                </span>
              )
            }}
          </Link>
        ))}
        {!isNoAuth && (
          <button
            onClick={(e) => {
              e.preventDefault();
              props.logoutMutation();
            }}
            className="w-full shadow-xs border bg-gray-100 border-gray-300 dark:border-gray-700 dark:bg-gray-900 dark:text-white block px-3 py-2 rounded-md text-base font-medium text-left"
          >
            Logout
          </button>
        )}
      </div>
    </DisclosurePanel>
  );
};
