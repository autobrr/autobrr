/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState } from "react";
import { ChevronUpIcon, ChevronDownIcon } from "@heroicons/react/24/solid";

import { ReleaseTable } from "./releases/ReleaseTable";
import { ExternalLink } from "@components/ExternalLink";

const Code = ({ children }: { children: React.ReactNode }) => (
  <code className="rounded-md inline-block mb-1 px-1 py-0.5 border bg-gray-100 border-gray-300 dark:bg-gray-800 dark:border-gray-700">
    {children}
  </code>
);

export const Releases = () => {
  const [isHintOpen, setIsHintOpen] = useState(false);
  return (
    <main>
      <div className="mt-6 mb-4 mx-auto flex flex-col max-w-(--breakpoint-xl) px-4 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold text-black dark:text-white">Releases</h1>
        <p className="flex-row items-start mt-1 text-sm text-gray-800 dark:text-gray-200">
          The search engine uses a special pattern-matching engine to filter out results.
          Please
          <button
            onClick={(e) => {
              e.preventDefault();
              setIsHintOpen((state) => !state);
            }}
            className="inline-flex whitespace-nowrap items-center shadow-md border rounded-md mx-1 px-1 text-black bg-lime-100 hover:bg-lime-200 border-lime-500 dark:text-white dark:bg-lime-950 dark:hover:bg-lime-900 dark:border-lime-800"

          >
            click here
            {isHintOpen ? (
              <ChevronUpIcon className="ml-1 h-3 w-3" />
            ) : (
              <ChevronDownIcon className="ml-1 h-3 w-3" />
            )}
          </button>
          to get tips on how to get relevant results.
        </p>
        {isHintOpen ? (
          <div className="rounded-md text-sm mt-2 border border-gray-300 text-black shadow-lg dark:text-white dark:border-gray-700 dark:shadow-2xl">
            <div className="flex justify-between items-center text-base font-medium pl-2 py-1 border-b border-gray-300 bg-gray-100 dark:border-gray-700 dark:bg-gray-800 rounded-t-md">
              Search tips
            </div>
            <div className="py-1 px-2 rounded-b-md bg-white dark:bg-gray-900">
              You can use <b>2</b> special <span className="underline decoration-2 underline-offset-2 decoration-amber-500">wildcard characters</span> for the purpose of pattern matching.
              <br />
              - Percent (<Code>%</Code>) - for matching any <i>sequence</i> of characters (equivalent to <Code>.*</Code> in Regex)
              <br />
              - Underscore (<Code>_</Code>) - for matching any <i>single</i> character (equivalent to <Code>.</Code> in Regex)
              <br /><br />

              Additionally, autobrr supports <span className="underline decoration-2 underline-offset-2 decoration-lime-500">keyword faceting</span>.
              The supported keywords are:{" "}
              <b>category</b>, <b>codec</b>, <b>episode</b>, <b>filter</b>, <b>group</b>,{" "}
              <b>hdr</b>, <b>resolution</b>, <b>season</b>, <b>source</b>, <b>title</b> and <b>year</b>.
              <br /><br />

              <b>Examples:</b><br />

              <Code>year:2022 resolution:1080p</Code> (all 1080p from the year 2022)
              <br />
              <Code>group:framestor hdr:DV</Code> (all Dolby Vision releases by a certain group, e.g. Framestor)
              <br />
              <Code>Movie Title resolution:1080p</Code> (all releases starting with "Movie Title" in 1080p)
              <br />
              <Code>The Show season:05 episode:03</Code> (all releases starting with "The Show" related to S05E03)
              <br />
              <Code>%collection hd%</Code> (all releases containing "collection hd" - in the same order - and with a space!)
              <br />
              <Code>%collection_hd%</Code> (all releases containing "collection" <b>AND</b> "hd" - in the same order - but with a wildcard character in between, e.g. a space <b>OR</b> a dot <b>OR</b> any other character)

              <br /><br />

              {"As always, please refer to our "}
              <ExternalLink
                href="https://autobrr.com/usage/search/"
                className="text-gray-700 dark:text-gray-200 underline font-semibold underline-offset-2 decoration-purple-500 decoration-2 hover:text-black dark:hover:text-gray-100"
              >
                Search function usage
              </ExternalLink>
              {" documentation page to keep up with the latest examples and information."}
            </div>
          </div>
        ) : null}
      </div>
      <div className="max-w-(--breakpoint-xl) mx-auto pb-6 px-2 sm:px-6 lg:pb-16 lg:px-8">
        <ReleaseTable />
      </div>
    </main>
  );
};
