import React, { Fragment } from "react";
import { Link, NavLink, Outlet } from "react-router-dom";
import { Disclosure, Menu, Transition } from "@headlessui/react";
import { ArrowTopRightOnSquareIcon, UserIcon } from "@heroicons/react/24/solid";
import { Bars3Icon, XMarkIcon, MegaphoneIcon } from "@heroicons/react/24/outline";

import { AuthContext } from "../utils/Context";

import logo from "../logo.png";
import { useMutation, useQuery } from "@tanstack/react-query";
import { APIClient } from "../api/APIClient";
import toast from "react-hot-toast";
import Toast from "@/components/notifications/Toast";
import { classNames } from "@utils";
import { filterKeys } from "@screens/filters/list";

interface NavItem {
  name: string;
  path: string;
}

const nav: Array<NavItem> = [
  { name: "Dashboard", path: "/" },
  { name: "Filters", path: "/filters" },
  { name: "Releases", path: "/releases" },
  { name: "Settings", path: "/settings" },
  { name: "Logs", path: "/logs" }
];

export default function Base() {
  const authContext = AuthContext.useValue();

  const { data } = useQuery({
    queryKey: ["updates"],
    queryFn: () => APIClient.updates.getLatestRelease(),
    retry: false,
    refetchOnWindowFocus: false,
    onError: err => console.log(err)
  });

  const logoutMutation = useMutation( {
    mutationFn: APIClient.auth.logout,
    onSuccess: () => {
      AuthContext.reset();

      toast.custom((t) => (
        <Toast type="success" body="You have been logged out. Goodbye!" t={t} />
      ));
    }
  });

  const logoutAction = () => {
    logoutMutation.mutate();
  };

  return (
    <div className="min-h-screen">
      <Disclosure
        as="nav"
        className="bg-gradient-to-b from-gray-100 dark:from-[#141414]"
      >
        {({ open }) => (
          <>
            <div className="max-w-screen-xl mx-auto sm:px-6 lg:px-8">
              <div className="border-b border-gray-300 dark:border-gray-700">
                <div className="flex items-center justify-between h-16 px-4 sm:px-0">
                  <div className="flex items-center">
                    <div className="flex-shrink-0 flex items-center">
                      <Link to="/">
                        <img
                          className="block lg:hidden h-10 w-auto"
                          src={logo}
                          alt="Logo"
                        />
                        <img
                          className="hidden lg:block h-10 w-auto"
                          src={logo}
                          alt="Logo"
                        />
                      </Link>
                    </div>
                    <div className="sm:ml-3 hidden sm:block">
                      <div className="flex items-baseline space-x-4">
                        {nav.map((item, itemIdx) => (
                          <NavLink
                            key={item.name + itemIdx}
                            to={item.path}
                            className={({ isActive }) =>
                              classNames(
                                "hover:bg-gray-200 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white px-3 py-2 rounded-2xl text-sm font-medium",
                                "transition-colors duration-200",
                                isActive
                                  ? "text-black dark:text-gray-50 font-bold"
                                  : "text-gray-600 dark:text-gray-500"
                              )
                            }
                            end={item.path === "/"}
                          >
                            {item.name}
                          </NavLink>
                        ))}
                        <a
                          rel="noopener noreferrer"
                          target="_blank"
                          href="https://autobrr.com"
                          className={classNames(
                            "text-gray-600 dark:text-gray-500 hover:bg-gray-200 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white px-3 py-2 rounded-2xl text-sm font-medium",
                            "transition-colors duration-200 flex items-center justify-center"
                          )}
                        >
                          Docs
                          <ArrowTopRightOnSquareIcon
                            className="inline ml-1 h-5 w-5"
                            aria-hidden="true"
                          />
                        </a>
                      </div>
                    </div>
                  </div>
                  <div className="hidden sm:block">
                    <div className="ml-4 flex items-center sm:ml-6">
                      <Menu as="div" className="ml-3 relative">
                        {({ open }) => (
                          <>
                            <Menu.Button
                              className={classNames(
                                open ? "bg-gray-200 dark:bg-gray-800" : "",
                                "text-gray-600 dark:text-gray-500 hover:bg-gray-200 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white px-3 py-2 rounded-2xl text-sm font-medium",
                                "max-w-xs rounded-full flex items-center text-sm px-3 py-2",
                                "transition-colors duration-200"
                              )}
                            >
                              <span className="hidden text-sm font-medium sm:block">
                                <span className="sr-only">
                                  Open user menu for{" "}
                                </span>
                                {authContext.username}
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
                                className="origin-top-right absolute right-0 mt-2 w-48 z-10 rounded-md shadow-lg py-1 bg-white dark:bg-gray-800 ring-1 ring-black ring-opacity-5 focus:outline-none"
                              >
                                <Menu.Item>
                                  {({ active }) => (
                                    <Link
                                      to="/settings"
                                      className={classNames(
                                        active
                                          ? "bg-gray-100 dark:bg-gray-600"
                                          : "",
                                        "block px-4 py-2 text-sm text-gray-900 dark:text-gray-200"
                                      )}
                                    >
                                      Settings
                                    </Link>
                                  )}
                                </Menu.Item>
                                <Menu.Item>
                                  {({ active }) => (
                                    <button
                                      onClick={logoutAction}
                                      className={classNames(
                                        active
                                          ? "bg-gray-100 dark:bg-gray-600"
                                          : "",
                                        "block w-full px-4 py-2 text-sm text-gray-900 dark:text-gray-200 text-left"
                                      )}
                                    >
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

              {data && data.html_url && (
                <a href={data.html_url} target="_blank">
                  <div className="flex mt-4 py-2 bg-blue-500 rounded justify-center">
                    <MegaphoneIcon className="h-6 w-6 text-blue-100"/>
                    <span className="text-blue-100 font-medium mx-3">New update available!</span>
                    <span className="inline-flex items-center rounded-md bg-blue-100 px-2.5 py-0.5 text-sm font-medium text-blue-800">{data?.name}</span>
                  </div>
                </a>
              )}
            </div>

            <Disclosure.Panel className="border-b border-gray-300 dark:border-gray-700 md:hidden">
              <div className="px-2 py-3 space-y-1 sm:px-3">
                {nav.map((item) => (
                  <NavLink
                    key={item.path}
                    to={item.path}
                    className={({ isActive }) =>
                      classNames(
                        "shadow-sm border bg-gray-100 border-gray-300 dark:border-gray-700 dark:bg-gray-900 dark:text-white block px-3 py-2 rounded-md text-base",
                        isActive
                          ? "underline underline-offset-2 decoration-2 decoration-sky-500 font-bold text-black"
                          : "font-medium"
                      )
                    }
                    end={item.path === "/"}
                  >
                    {item.name}
                  </NavLink>
                ))}
                <button
                  onClick={logoutAction}
                  className="w-full shadow-sm border bg-gray-100 border-gray-300 dark:border-gray-700 dark:bg-gray-900 dark:text-white block px-3 py-2 rounded-md text-base font-medium text-left"
                >
                  Logout
                </button>
              </div>
            </Disclosure.Panel>
          </>
        )}
      </Disclosure>
      <Outlet />
    </div>
  );
}
