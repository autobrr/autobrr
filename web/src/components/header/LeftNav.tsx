/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Link, NavLink } from "react-router-dom";
import { ArrowTopRightOnSquareIcon } from "@heroicons/react/24/solid";

import { classNames } from "@utils";
import { ReactComponent as Logo } from "@app/logo.svg";

import { NAV_ROUTES } from "./_shared";

export const LeftNav = () => (
  <div className="flex items-center">
    <div className="flex-shrink-0 flex items-center">
      <Link to="/">
        <Logo className="h-10" />
      </Link>
    </div>
    <div className="sm:ml-3 hidden sm:block">
      <div className="flex items-baseline space-x-4">
        {NAV_ROUTES.map((item, itemIdx) => (
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
);
