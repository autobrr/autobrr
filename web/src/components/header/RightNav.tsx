import { Fragment } from "react";
import { Link } from "react-router-dom";
import { UserIcon } from "@heroicons/react/24/solid";
import { Menu, Transition } from "@headlessui/react";

import { classNames } from "@utils";
import { AuthContext } from "@utils/Context";

import { RightNavProps } from "./_shared";

export const RightNav = (props: RightNavProps) => {
  const authContext = AuthContext.useValue();
  return (
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
                        onClick={(e) => {
                          e.preventDefault();
                          props.logoutMutation();
                        }}
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
  );
}
