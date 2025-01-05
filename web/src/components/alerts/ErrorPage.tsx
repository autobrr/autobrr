/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import StackTracey from "stacktracey";
import { ArrowPathIcon } from "@heroicons/react/24/solid";
import { ExternalLink } from "@components/ExternalLink";

type ErrorPageProps = {
  error: unknown;
  reset: () => void;
}

export const ErrorPage = ({ error, reset }: ErrorPageProps) => {
  let pageTitle = "We caught an unrecoverable error!";
  let errorLine: string, summary ="";

  if (error instanceof Error) {
    const stack = new StackTracey(error);
    summary = stack.clean().asTable({
      maxColumnWidths: {
        callee: 48,
        file: 48,
        sourceLine: 384
      }
    });

    if (error.cause === "OFFLINE") {
      pageTitle = "Connection to Autobrr failed! Check the application state and verify your connectivity.";
    }

    errorLine = error.toString();
  } else {
    errorLine = String(error);
    // Leave summary blank?
  }

  return (
    <div className="min-h-screen flex flex-col justify-center py-12 px-2 sm:px-6 lg:px-8">
      <div className="sm:mx-auto sm:w-full sm:max-w-screen-md md:max-w-screen-lg lg:max-w-screen-xl">
        <h1 className="text-3xl font-bold leading-6 text-gray-900 dark:text-gray-200 mt-4 mb-3">
          {pageTitle}
        </h1>
        <h3 className="text-xl leading-6 text-gray-700 dark:text-gray-400 mb-4">
          Please consider reporting this error to our
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
        <div
          id="alert-additional-content-2"
          className="px-4 pt-4 pb-3 m-auto bg-red-100 rounded-lg dark:bg-red-200 shadow-lg"
          role="alert"
        >
          <div className="flex items-center">
            <svg className="mr-2 w-5 h-5 text-red-700 dark:text-red-800" fill="currentColor" viewBox="0 0 20 20"
                 xmlns="http://www.w3.org/2000/svg">
              <path
                fillRule="evenodd"
                d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                clipRule="evenodd"
              />
            </svg>
            <h3 className="text-lg font-medium text-red-700 dark:text-red-800">{errorLine}</h3>
          </div>
          {summary ? (
            <pre className="mt-2 mb-4 text-sm text-red-700 dark:text-red-800 overflow-x-auto">
              {summary}
            </pre>
          ) : null}
          <span className="block text-gray-800 mb-2 text-md">
            You can try resetting the page state using the button provided below or restarting your autobrr application.
            However, this is not guaranteed to fix the error.
          </span>
          <button
            type="button"
            className="text-white bg-red-700 hover:bg-red-800 focus:ring-4 focus:outline-none focus:ring-red-300 font-medium rounded-lg text-sm px-3 py-1.5 mr-2 text-center inline-flex items-center dark:bg-red-800 dark:hover:bg-red-900"
            onClick={(event) => {
              event.preventDefault();
              reset();
            }}
          >
            <ArrowPathIcon className="-ml-0.5 mr-2 h-5 w-5"/>
            Reset page state
          </button>
        </div>
      </div>
    </div>
  );
};
