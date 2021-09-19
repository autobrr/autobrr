import { Fragment } from 'react'
import { Disclosure, Menu, Transition } from '@headlessui/react'
import { ChevronDownIcon, MenuIcon, XIcon } from '@heroicons/react/outline'
import { NavLink, Link, Route, Switch } from "react-router-dom";
import Settings from "./Settings";
import { Dashboard } from "./Dashboard";
import { FilterDetails, Filters } from "./filters";
import Logs from './Logs';
import logo from '../logo.png';

function classNames(...classes: string[]) {
    return classes.filter(Boolean).join(' ')
}

export default function Base() {
    const nav = [{ name: 'Dashboard', path: "/" }, { name: 'Filters', path: "/filters" }, { name: "Settings", path: "/settings" }, { name: "Logs", path: "/logs" }]

    return (
        <div className="">
            <Disclosure as="nav" className="bg-gray-800 pb-48">
                {({ open }) => (
                    <>
                        <div className="max-w-7xl mx-auto sm:px-6 lg:px-8">
                            <div className="border-b border-gray-700">
                                <div className="flex items-center justify-between h-16 px-4 sm:px-0">
                                    <div className="flex items-center">
                                        <div className="flex-shrink-0 flex items-center">
                                            <img
                                                className="block lg:hidden h-8 w-auto"
                                                src={logo}
                                                alt="Logo"
                                            />
                                            <img
                                                className="hidden lg:block h-8 w-auto"
                                                src={logo}
                                                alt="Logo"
                                            />
                                        </div>
                                        <div className="sm:ml-6 hidden sm:block">
                                            <div className="flex items-baseline space-x-4">
                                                {nav.map((item, itemIdx) =>
                                                    <NavLink
                                                        key={itemIdx}
                                                        to={item.path}
                                                        exact={true}
                                                        activeClassName="bg-gray-900 text-white "
                                                        className="text-gray-300 hover:bg-gray-700 hover:text-white px-3 py-2 rounded-md text-sm font-medium"
                                                    >
                                                        {item.name}
                                                    </NavLink>
                                                )}
                                            </div>
                                        </div>
                                    </div>
                                    <div className="hidden sm:block">
                                        <div className="ml-4 flex items-center sm:ml-6">
                                            <Menu as="div" className="ml-3 relative">
                                                {({ open }) => (
                                                    <>
                                                        <div>
                                                            <Menu.Button
                                                                className="max-w-xs bg-gray-800 rounded-full flex items-center text-sm focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-gray-800 focus:ring-white">
                                                                <span
                                                                    className="hidden text-gray-300 text-sm font-medium sm:block">
                                                                    <span className="sr-only">Open user menu for </span>User
                                                                </span>
                                                                <ChevronDownIcon
                                                                    className="hidden flex-shrink-0 ml-1 h-5 w-5 text-gray-400 sm:block"
                                                                    aria-hidden="true"
                                                                />
                                                            </Menu.Button>
                                                        </div>
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
                                                                className="origin-top-right absolute right-0 mt-2 w-48 z-10 rounded-md shadow-lg py-1 bg-white ring-1 ring-black ring-opacity-5 focus:outline-none"
                                                            >
                                                                <Menu.Item>
                                                                    {({ active }) => (
                                                                        <Link
                                                                            to="settings"
                                                                            className={classNames(
                                                                                active ? 'bg-gray-100' : '',
                                                                                'block px-4 py-2 text-sm text-gray-700'
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
                                                                                active ? 'bg-gray-100' : '',
                                                                                'block px-4 py-2 text-sm text-gray-700'
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
                                            className="bg-gray-800 inline-flex items-center justify-center p-2 rounded-md text-gray-400 hover:text-white hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-gray-800 focus:ring-white">
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

                        <Disclosure.Panel className="border-b border-gray-700 md:hidden">
                            <div className="px-2 py-3 space-y-1 sm:px-3">
                                {nav.map((item, itemIdx) =>
                                    itemIdx === 0 ? (
                                        <Fragment key={item.path}>
                                            {/* Current: "bg-gray-900 text-white", Default: "text-gray-300 hover:bg-gray-700 hover:text-white" */}
                                            <Link to={item.path}
                                                className="bg-gray-900 text-white block px-3 py-2 rounded-md text-base font-medium">
                                                {item.name}
                                            </Link>
                                        </Fragment>
                                    ) : (
                                        <Link
                                            key={item.path}
                                            to={item.path}
                                            className="text-gray-300 hover:bg-gray-700 hover:text-white block px-3 py-2 rounded-md text-base font-medium"
                                        >
                                            {item.name}
                                        </Link>
                                    )
                                )}
                            </div>
                            <div className="pt-4 pb-3 border-t border-gray-700">
                                <div className="flex items-center px-5">
                                    <div>
                                        <div className="text-base font-medium leading-none text-white">User</div>
                                    </div>
                                </div>
                                <div className="mt-3 px-2 space-y-1">
                                    <Link
                                        to="settings"
                                        className="block px-3 py-2 rounded-md text-base font-medium text-gray-400 hover:text-white hover:bg-gray-700"
                                    >
                                        Settings
                                    </Link>
                                    <Link
                                        to="/logout"
                                        className="block px-3 py-2 rounded-md text-base font-medium text-gray-400 hover:text-white hover:bg-gray-700"
                                    >
                                        Logout
                                    </Link>
                                </div>
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