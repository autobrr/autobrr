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
            className="p-1 rounded-full hover:cursor-pointer focus:outline-hidden focus:none transition duration-100 ease-out transform hover:bg-gray-200 dark:hover:bg-gray-800 hover:scale-100"
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
                <span className="hidden hover:cursor-pointer text-sm font-medium sm:flex items-center space-x-2">
                  <span className="sr-only">
                    Open user menu for{" "}
                  </span>
                  <span className="mr-1">{auth.username}</span>
                  {auth.authMethod === 'oidc' ? (
                    auth.profilePicture ? (
                      <div className="relative flex-shrink-0 w-6 h-6 overflow-hidden rounded-full ring-2 ring-white dark:ring-gray-700">
                        <img
                          src={auth.profilePicture}
                          alt={`${auth.username}'s profile`}
                          className="object-cover w-full h-full transition-opacity duration-200"
                          onError={(e) => {
                            const target = e.target as HTMLImageElement;
                            target.style.display = 'none';
                            const parent = target.parentElement;
                            if (parent) {
                              parent.className = "inline-flex items-center justify-center w-6 h-6 rounded-full bg-gray-100 dark:bg-gray-700 ring-2 ring-white dark:ring-gray-700";
                              const icon = document.createElement('span');
                              icon.innerHTML = '<svg aria-hidden="true" focusable="false" data-prefix="fab" data-icon="openid" class="h-3.5 w-3.5 text-gray-500 dark:text-gray-400" role="img" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 448 512"><path fill="currentColor" d="M271.5 432l-68 32C88.5 453.7 0 392.5 0 318.2c0-71.5 82.5-131 191.7-144.3v43c-71.5 12.5-124 53-124 101.3 0 51 58.5 93.3 135.5 103v-340l68-33.2v384zM448 291l-131.3-28.5 36.8-20.7c-19.5-11.5-43.5-20-70-24.8v-43c46.2 5.5 87.7 19.5 120.3 39.3l35-19.8L448 291z"></path></svg>';
                              parent.appendChild(icon);
                            }
                          }}
                        />
                      </div>
                    ) : (
                      <div className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-gray-100 dark:bg-gray-700 ring-2 ring-white dark:ring-gray-700">
                        <FontAwesomeIcon
                          icon={faOpenid}
                          className="h-3.5 w-3.5 text-gray-500 dark:text-gray-400"
                          aria-hidden="true"
                        />
                      </div>
                    )
                  ) : (
                    <div className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-gray-100 dark:bg-gray-700 ring-2 ring-white dark:ring-gray-700">
                      <UserIcon
                        className="h-3.5 w-3.5 text-gray-500 dark:text-gray-400"
                        aria-hidden="true"
                      />
                    </div>
                  )}
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
                  className="origin-top-right absolute right-0 mt-2 w-48 z-10 divide-y divide-gray-100 dark:divide-gray-750 rounded-md shadow-lg bg-white dark:bg-gray-800 border border-gray-250 dark:border-gray-775 focus:outline-hidden"
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
