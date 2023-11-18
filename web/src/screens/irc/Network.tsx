/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Disclosure } from "@headlessui/react";
import { ChevronRightIcon, ChevronDownIcon, LockClosedIcon, LockOpenIcon } from "@heroicons/react/24/solid";

import { Channel } from "./Channel";
import type { NetworkProps } from "./Shared";
import { classNames } from "@utils";

export const Network = ({ network, ...rest }: NetworkProps) => {
  let tlsTitleMessage = network.tls ? "Secured using TLS" : "Insecure, not using TLS";
  if (!network.enabled) {
    tlsTitleMessage += " (Disabled)";
  }

  return (
    <>
      <Disclosure as="div" className="w-full my-2" defaultOpen={network.enabled}>
        {({ open }) => (
          <Disclosure.Button
            className={classNames(
              "flex flex-col w-full text-medium",
              network.enabled ? "cursor-pointer" : "cursor-not-allowed"
            )}
          >
            <div className="flex items-center w-full py-1 transition text-gray-800 hover:text-gray-500 dark:text-gray-400 dark:hover:text-gray-300">
              {open ? (
                <ChevronDownIcon className="h-3 w-3 mr-1" />
              ) : (
                <ChevronRightIcon className="h-3 w-3 mr-1" />
              )}
              {network.enabled ? (
                <div
                  className="overflow-x-auto flex items-center mr-1"
                  title={tlsTitleMessage}
                >
                  <div className="min-h-2 min-w-2">
                    {network.tls ? (
                      <LockClosedIcon className="h-4 w-4 text-green-600" />
                    ) : (
                      <LockOpenIcon className="h-4 w-4 text-red-500" />
                    )}
                  </div>
                </div>
              ) : null}
              <span className={classNames(!network.enabled ? "line-through" : "font-semibold")}>
                {network.name}
              </span>
            </div>
            <Disclosure.Panel className="w-full text-medium">
              {network.channels.map((channel) => (
                <Channel
                  key={`channel-${channel.id}`}
                  channel={channel}
                  network={network}
                  {...rest}
                />
              ))}
            </Disclosure.Panel>
          </Disclosure.Button>
        )}
      </Disclosure>
    </>
  )
};
