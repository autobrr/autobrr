/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { LockClosedIcon, LockOpenIcon, PlusIcon } from "@heroicons/react/24/solid";

import { IrcNetworkAddForm } from "@forms/settings/IrcForms";
import { useToggle } from "@hooks/hooks";

import { Panel } from "./Panel";

const Legend = () => (
  <div className="flex flex-row px-4 sm:px-3 py-2 rounded-md bg-white dark:bg-gray-800 border border-gray-400 dark:border-gray-700">
    <p className="text-base text-gray dark:text-gray-400 mr-3">Legend:</p>
    <div className="flex flex-col text-gray-800 dark:text-gray-400">
      <ol className="flex flex-col md:flex-row md:gap-2 md:pb-0 md:divide-x md:divide-gray-400 md:dark:divide-gray-600">
        <li className="flex items-center">
          <span
            className="mr-2 flex h-4 w-4 relative"
            title="Channel healthy"
          >
            <span className="animate-ping inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
            <span className="inline-flex absolute rounded-full h-4 w-4 bg-green-500" />
          </span>
          <span>Channel healthy</span>
        </li>

        <li className="flex items-center md:pl-2">
          <span
            className="mr-2 flex h-4 w-4 rounded-full opacity-75 bg-yellow-400 over:text-yellow-600"
            title="Channel unhealthy"
          />
          <span>Channel unhealthy</span>
        </li>

        <li className="flex items-center md:pl-2">
          <span
            className="mr-2 flex h-4 w-4 rounded-full opacity-75 bg-gray-500"
            title="Channel disabled"
          />
          <span><span className="line-through">Channel</span> disabled</span>
        </li>

        <li className="flex items-center md:pl-2">
          <LockClosedIcon className="h-4 w-4 mr-2 text-green-600" />
          <span>Secured using TLS</span>
        </li>

        <li className="flex items-center md:pl-2">
          <LockOpenIcon className="h-4 w-4 mr-2 text-red-500" />
          <span>Insecure, not using TLS</span>
        </li>
      </ol>
    </div>
  </div>
);

export const Irc = () => {
  const [addNetworkIsOpen, toggleAddNetwork] = useToggle(false);

  return (
    <main>
      <IrcNetworkAddForm isOpen={addNetworkIsOpen} toggle={toggleAddNetwork} />

      <div className="my-3 max-w-screen-xl w-fit mx-auto flex">
        <Legend />
        <button
          className="flex items-center my-auto ml-2 px-2 py-1.5 transition rounded-md shadow border border-lime-600 bg-lime-300 hover:bg-lime-400 dark:bg-lime-900 dark:hover:bg-lime-700"
          onClick={(e) => {
            e.preventDefault();
            toggleAddNetwork();
          }}
        >
          <PlusIcon
            className="h-4 w-4 text-gray-900 dark:text-white"
            aria-hidden="true"
          />
          <span className="ml-2 text-black dark:text-white">Add new network</span>
        </button>
      </div>

      <Panel />
    </main>
  );
}
