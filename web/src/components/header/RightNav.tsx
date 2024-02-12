/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment } from "react";
import { UserIcon } from "@heroicons/react/24/solid";
import { Menu, Transition } from "@headlessui/react";

import { classNames } from "@utils";

import { RightNavProps } from "./_shared";
import { Cog6ToothIcon, ArrowLeftOnRectangleIcon } from "@heroicons/react/24/outline";
import {Link} from "@tanstack/react-router";

export const RightNav = (props: RightNavProps) => {
  return (
    <div className="hidden sm:block">
      <div className="ml-4 flex items-center sm:ml-6">
        <Menu as="div" className="ml-3 relative">
          {({ open }) => (
            <>
              <Menu.Button
                className={classNames(
                  open ? "bg-gray-200 dark:bg-gray-800 text-gray-900 dark:text-white" : "hover:text-gray-900 dark:hover:text-white",
                  "text-gray-600 dark:text-gray-500 hover:bg-gray-200 dark:hover:bg-gray-800 px-3 py-2 rounded-2xl text-sm font-medium",
                  "max-w-xs rounded-full flex items-center text-sm px-3 py-2",
                  "transition duration-200"
                )}
              >
                <span className="hidden text-sm font-medium sm:block">
                  <span className="sr-only">
                    Open user menu for{" "}
                  </span>
                  {props.auth.username}
                </span>
                <UserIcon
                  className="inline ml-1 h-5 w-5"
                  aria-hidden="true"
                />
              </Menu.Button>
              <Transition
                show={open}
                as={Fragment}
                enter="transition ease-out duration-100"
                enterFrom="transform opacity-0 scale-95"
                enterTo="transform opacity-100 scale-100"
                leave="transition ease-in duration-75"
                leaveFrom="transform opacity-100 scale-100"
                leaveTo="transform opacity-0 scale-95"
              >
                <Menu.Items
                  static
                  className="origin-top-right absolute right-0 mt-2 w-48 z-10 divide-y divide-gray-100 dark:divide-gray-750 rounded-md shadow-lg bg-white dark:bg-gray-800 border border-gray-250 dark:border-gray-775 focus:outline-none"
                >
                  <Menu.Item>
                    {({ active }) => (
                      <Link
                        to="/settings/account"
                        className={classNames(
                          active
                            ? "bg-gray-100 dark:bg-gray-600"
                            : "",
                          "flex items-center transition rounded-t-md px-2 py-2 text-sm text-gray-900 dark:text-gray-200"
                        )}
                      >
                        <UserIcon
                          className="w-5 h-5 mr-1 text-gray-700 dark:text-gray-400"
                          aria-hidden="true"
                        />
                        Account
                      </Link>
                    )}
                  </Menu.Item>
                  <Menu.Item>
                    {({ active }) => (
                      <Link
                        to="/settings"
                        className={classNames(
                          active
                            ? "bg-gray-100 dark:bg-gray-600"
                            : "",
                          "flex items-center transition rounded-t-md px-2 py-2 text-sm text-gray-900 dark:text-gray-200"
                        )}
                      >
                        <Cog6ToothIcon
                          className="w-5 h-5 mr-1 text-gray-700 dark:text-gray-400"
                          aria-hidden="true"
                        />
                        Settings
                      </Link>
                    )}
                  </Menu.Item>
                  <Menu.Item>
                    {({ active }) => (
                      <button
                        onClick={(e) => {
                          e.preventDefault();
                          props.logoutMutation();
                        }}
                        className={classNames(
                          active
                            ? "bg-gray-100 dark:bg-gray-600"
                            : "",
                          "flex items-center transition rounded-b-md w-full px-2 py-2 text-sm text-gray-900 dark:text-gray-200 text-left"
                        )}
                      >
                        <ArrowLeftOnRectangleIcon
                          className="w-5 h-5 mr-1 text-gray-700 dark:text-gray-400"
                          aria-hidden="true"
                        />
                        Log out
                      </button>
                    )}
                  </Menu.Item>
                </Menu.Items>
              </Transition>
            </>
          )}
        </Menu>
      </div>
    </div>
  );
};
