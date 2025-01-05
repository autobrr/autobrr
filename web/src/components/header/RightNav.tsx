/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment } from "react";
import { UserIcon } from "@heroicons/react/24/solid";
import { Menu, MenuButton, MenuItem, MenuItems, Transition } from "@headlessui/react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faOpenid } from "@fortawesome/free-brands-svg-icons";

import { classNames } from "@utils";

import { RightNavProps } from "./_shared";

import { Cog6ToothIcon, ArrowLeftOnRectangleIcon, MoonIcon, SunIcon } from "@heroicons/react/24/outline";
import { Link } from "@tanstack/react-router";
import { AuthContext, SettingsContext } from "@utils/Context";

export const RightNav = (props: RightNavProps) => {
  const [settings, setSettings] = SettingsContext.use();

  const auth = AuthContext.get();

  const toggleTheme = () => {
    setSettings(prevState => ({
      ...prevState,
      darkTheme: !prevState.darkTheme
    }));
  };

  return (
    <div className="hidden sm:block">
      <div className="ml-4 flex items-center sm:ml-6">
        <div className="mt-1 items-center">
          <button
            onClick={toggleTheme}
            className="p-1 rounded-full focus:outline-none focus:none transition duration-100 ease-out transform hover:bg-gray-200 dark:hover:bg-gray-800 hover:scale-100"
            title={settings.darkTheme ? "Switch to light mode (currently dark mode)" : "Switch to dark mode (currently light mode)"}
          >
            {settings.darkTheme ? (
              <MoonIcon className="h-4 w-4 text-gray-500 transition duration-100 ease-out transform" aria-hidden="true" />
            ) : (
              <SunIcon className="h-4 w-4 text-gray-600" aria-hidden="true" />
            )}
          </button>
        </div>
        <Menu as="div" className="ml-2 relative">
          {({ open }) => (
            <>
              <MenuButton
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
                  <span className="flex items-center">
                    {auth.username}
                    {auth.authMethod === 'oidc' ? (
                      <FontAwesomeIcon
                        icon={faOpenid}
                        className="inline ml-1 h-4 w-4 text-gray-500 dark:text-gray-500"
                        aria-hidden="true"
                      />
                    ) : (
                      <UserIcon
                        className="inline ml-1 h-5 w-5"
                        aria-hidden="true"
                      />
                    )}
                  </span>
                </span>
              </MenuButton>
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
                <MenuItems
                  static
                  className="origin-top-right absolute right-0 mt-2 w-48 z-10 divide-y divide-gray-100 dark:divide-gray-750 rounded-md shadow-lg bg-white dark:bg-gray-800 border border-gray-250 dark:border-gray-775 focus:outline-none"
                >
                  <MenuItem>
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
                  </MenuItem>
                  <MenuItem>
                    {({ active }) => (
                      <Link
                        to="/settings"
                        className={classNames(
                          active
                            ? "bg-gray-100 dark:bg-gray-600"
                            : "",
                          "flex items-center transition px-2 py-2 text-sm text-gray-900 dark:text-gray-200"
                        )}
                      >
                        <Cog6ToothIcon
                          className="w-5 h-5 mr-1 text-gray-700 dark:text-gray-400"
                          aria-hidden="true"
                        />
                        Settings
                      </Link>
                    )}
                  </MenuItem>
                  <MenuItem>
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
                  </MenuItem>
                </MenuItems>
              </Transition>
            </>
          )}
        </Menu>
      </div>
    </div>
  );
};
