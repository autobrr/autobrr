import { NavLink } from "react-router-dom";
import { Disclosure } from "@headlessui/react";

import { classNames } from "@utils";

import { NAV_ROUTES } from "./_shared";
import type { RightNavProps } from "./_shared";

export const MobileNav = (props: RightNavProps) => (
  <Disclosure.Panel className="border-b border-gray-300 dark:border-gray-700 md:hidden">
    <div className="px-2 py-3 space-y-1 sm:px-3">
      {NAV_ROUTES.map((item) => (
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
        onClick={(e) => {
          e.preventDefault();
          props.logoutMutation();
        }}
        className="w-full shadow-sm border bg-gray-100 border-gray-300 dark:border-gray-700 dark:bg-gray-900 dark:text-white block px-3 py-2 rounded-md text-base font-medium text-left"
      >
        Logout
      </button>
    </div>
  </Disclosure.Panel>
);
