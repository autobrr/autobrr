import { Fragment } from "react";
import { NavLink, Link, Route, Switch } from "react-router-dom";
import type { match } from "react-router-dom";
import { Disclosure, Menu, Transition } from "@headlessui/react";
import { ExternalLinkIcon } from "@heroicons/react/solid";
import { ChevronDownIcon, MenuIcon, XIcon } from "@heroicons/react/outline";

import Logs from "./Logs";
import Settings from "./Settings";

import { Releases } from "./Releases";
import { Dashboard } from "./Dashboard";
import { FilterDetails, Filters } from "./filters";
import { AuthContext } from '../utils/Context';

import logo from '../logo.png';

interface NavItem {
  name: string;
  path: string;
}

function classNames(...classes: string[]) {
    return classes.filter(Boolean).join(' ')
}

const isActiveMatcher = (
    match: match<any> | null,
    location: { pathname: string },
    item: NavItem
) => {
  if (!match)
    return false;

  if (match?.url === "/" && item.path === "/" && location.pathname === "/")
      return true

  if (match.url === "/")
      return false;

  return true;
}

export default function Base() {
    const authContext = AuthContext.useValue();
    const nav: Array<NavItem> = [
        { name: 'Dashboard', path: "/" },
        { name: 'Filters', path: "/filters" },
        { name: 'Releases', path: "/releases" },
        { name: "Settings", path: "/settings" },
        { name: "Logs", path: "/logs" }
    ];

    return (
        <div className="min-h-screen">
            <Disclosure
              as="nav"
              className="bg-gradient-to-b from-gray-100 dark:from-[#141414]"
            >
                {({ open }) => (
                    <>
                        <div className="max-w-7xl mx-auto sm:px-6 lg:px-8">
                            <div className="border-b border-gray-300 dark:border-gray-700">
                                <div className="flex items-center justify-between h-16 px-4 sm:px-0">
                                    <div className="flex items-center">
                                        <div className="flex-shrink-0 flex items-center">
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
                                        </div>
                                        <div className="sm:ml-3 hidden sm:block">
                                            <div className="flex items-baseline space-x-4">
                                                {nav.map((item, itemIdx) =>
                                                    <NavLink
                                                        key={item.name + itemIdx}
                                                        to={item.path}
                                                        strict
                                                        className={classNames(
                                                            "text-gray-600 dark:text-gray-500 hover:bg-gray-200 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white px-3 py-2 rounded-2xl text-sm font-medium",
                                                            "transition-colors duration-200"
                                                        )}
                                                        activeClassName="text-black dark:text-gray-50 font-bold"
                                                        isActive={(match, location) => isActiveMatcher(match, location, item)}
                                                    >
                                                        {item.name}
                                                    </NavLink>
                                                )}
                                                <a
                                                    rel="noopener noreferrer"
                                                    target="_blank"
                                                    href="https://autobrr.com/docs/configuration/indexers"
                                                    className={classNames(
                                                        "text-gray-600 dark:text-gray-500 hover:bg-gray-200 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white px-3 py-2 rounded-2xl text-sm font-medium",
                                                        "transition-colors duration-200 flex items-center justify-center"
                                                    )}
                                                >
                                                    Docs
                                                    <ExternalLinkIcon className="inline ml-1 h-5 w-5" aria-hidden="true" />
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
                                                              "text-gray-800 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-800",
                                                              "max-w-xs rounded-full flex items-center text-sm px-3 py-2",
                                                              "transition-colors duration-200"
                                                            )}
                                                        >
                                                            <span className="hidden text-sm font-medium sm:block">
                                                                <span className="sr-only">Open user menu for </span>
                                                                {authContext.username}
                                                            </span>
                                                            <ChevronDownIcon
                                                                className="hidden flex-shrink-0 ml-1 h-5 w-5 text-gray-800 dark:text-gray-300 sm:block"
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
                                                                                active ? 'bg-gray-100 dark:bg-gray-600' : '',
                                                                                'block px-4 py-2 text-sm text-gray-700 dark:text-gray-200'
                                                                            )}
                                                                        >
                                                                            Settings
                                                                        </Link>
                                                                    )}
                                                                </Menu.Item>
                                                                <Menu.Item>
                                                                    {({ active }) => (
                                                                        <Link
                                                                            to="/logout"
                                                                            className={classNames(
                                                                                active ? 'bg-gray-100 dark:bg-gray-600' : '',
                                                                                'block px-4 py-2 text-sm text-gray-700 dark:text-gray-200'
                                                                            )}
                                                                        >
                                                                            Logout
                                                                        </Link>
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
                                        <Disclosure.Button
                                            className="bg-gray-200 dark:bg-gray-800 inline-flex items-center justify-center p-2 rounded-md text-gray-600 dark:text-gray-400 hover:text-white hover:bg-gray-700">
                                            <span className="sr-only">Open main menu</span>
                                            {open ? (
                                                <XIcon className="block h-6 w-6" aria-hidden="true" />
                                            ) : (
                                                <MenuIcon className="block h-6 w-6" aria-hidden="true" />
                                            )}
                                        </Disclosure.Button>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <Disclosure.Panel className="border-b border-gray-300 dark:border-gray-700 md:hidden">
                            <div className="px-2 py-3 space-y-1 sm:px-3">
                                {nav.map((item) =>
                                    <NavLink
                                        key={item.path}
                                        to={item.path}
                                        strict
                                        className="dark:bg-gray-900 dark:text-white block px-3 py-2 rounded-md text-base font-medium"
                                        activeClassName="font-bold bg-gray-300 text-black"
                                        isActive={(match, location) => isActiveMatcher(match, location, item)}
                                    >
                                        {item.name}
                                    </NavLink>
                                )}
                                <Link
                                    to="/logout"
                                    className="dark:bg-gray-900 dark:text-white block px-3 py-2 rounded-md text-base font-medium"
                                >
                                    Logout
                                </Link>
                            </div>

                        </Disclosure.Panel>
                    </>
                )}
            </Disclosure>

            <Switch>
                <Route path="/logs">
                    <Logs />
                </Route>

                <Route path="/settings">
                    <Settings />
                </Route>

                <Route path="/releases">
                    <Releases />
                </Route>

                <Route exact={true} path="/filters">
                    <Filters />
                </Route>

                <Route path="/filters/:filterId">
                    <FilterDetails />
                </Route>

                <Route exact path="/">
                    <Dashboard />
                </Route>
            </Switch>
        </div>
    )
}
