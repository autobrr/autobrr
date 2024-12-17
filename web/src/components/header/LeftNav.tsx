/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

// import { Link, NavLink } from "react-router-dom";

import { Link } from '@tanstack/react-router'

import { ArrowTopRightOnSquareIcon } from "@heroicons/react/24/solid";

import { classNames } from "@utils";

import { NAV_ROUTES } from "./_shared";
import { ExternalLink } from "@components/ExternalLink";

import Logo from "@app/logo.svg?react";

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
          <Link
            key={item.name + itemIdx}
            to={item.path}
            params={{}}
          >
            {({ isActive }) => {
              return (
                <>
                  <span className={
                    classNames(
                      "hover:bg-gray-200 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white px-3 py-2 rounded-2xl text-sm font-medium",
                      "transition-colors duration-200",
                      isActive
                        ? "text-black dark:text-gray-50 font-bold"
                        : "text-gray-600 dark:text-gray-500"
                    )
                  }>{item.name}</span>
                </>
              )
            }}
          </Link>
        ))}
        <ExternalLink
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
        </ExternalLink>
      </div>
    </div>
  </div>
);
