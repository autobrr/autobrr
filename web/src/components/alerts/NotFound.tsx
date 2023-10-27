/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Link } from "react-router-dom";
import { ExternalLink } from "@components/ExternalLink";

import Logo from "@app/logo.svg?react";

export const NotFound = () => {
  return (
    <div className="min-h-screen flex flex-col justify-center ">
      <div className="flex justify-center">
        <Logo className="h-24 sm:h-48" />
      </div>
      <h1 className="text-3xl text-center font-bold text-gray-900 dark:text-gray-200 my-8 px-2">
        Oops, looks like there was a little too much brr!
      </h1>
      <h3 className="text-xl text-center text-gray-700 dark:text-gray-400 mb-1 px-2">
        In case you think this is a bug rather than too much brr,
      </h3>
      <h3 className="text-xl text-center text-gray-700 dark:text-gray-400 mb-1 px-2">
        feel free to report this to our
        {" "}
        <ExternalLink
          href="https://github.com/autobrr/autobrr"
          className="text-gray-700 dark:text-gray-200 underline font-semibold underline-offset-2 decoration-sky-500 hover:decoration-2 hover:text-black hover:dark:text-gray-100"
        >
          GitHub page
        </ExternalLink>
        {" or to "}
        <ExternalLink
          href="https://discord.gg/WQ2eUycxyT"
          className="text-gray-700 dark:text-gray-200 underline font-semibold underline-offset-2 decoration-purple-500 hover:decoration-2 hover:text-black hover:dark:text-gray-100"
        >
          our official Discord channel
        </ExternalLink>
        .
      </h3>
      <h3 className="text-xl text-center leading-6 text-gray-700 dark:text-gray-400 mb-8 px-2">
        Otherwise, let us help you to get you back on track for more brr!
      </h3>
      <div className="flex justify-center">
        <Link to="/">
          <button
            className="w-48 flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
          >
            Back to Dashboard
          </button>
        </Link>
      </div>
    </div>
  );
};
