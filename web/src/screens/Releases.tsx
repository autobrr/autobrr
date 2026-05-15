/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState } from "react";
import { ChevronUpIcon, ChevronDownIcon } from "@heroicons/react/24/solid";
import { Trans, useTranslation } from "react-i18next";

import { ReleaseTable } from "./releases/ReleaseTable";
import { ExternalLink } from "@components/ExternalLink";

const Code = ({ children }: { children: React.ReactNode }) => (
  <code className="rounded-md inline-block mb-1 px-1 py-0.5 border bg-gray-100 border-gray-300 dark:bg-gray-800 dark:border-gray-700">
    {children}
  </code>
);

export const Releases = () => {
  const { t } = useTranslation("common");
  const [isHintOpen, setIsHintOpen] = useState(false);
  return (
    <main>
      <div className="mt-6 mb-4 mx-auto flex flex-col max-w-(--breakpoint-xl) px-4 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold text-black dark:text-white">{t("releaseSearch.title")}</h1>
        <p className="flex-row items-start mt-1 text-sm text-gray-800 dark:text-gray-200">
          {t("releaseSearch.descriptionPrefix")}
          {" "}
          <button
            onClick={(e) => {
              e.preventDefault();
              setIsHintOpen((state) => !state);
            }}
            className="inline-flex whitespace-nowrap items-center shadow-md border rounded-md mx-1 px-1 text-black bg-lime-100 hover:bg-lime-200 border-lime-500 dark:text-white dark:bg-lime-950 dark:hover:bg-lime-900 dark:border-lime-800"

          >
            {t("releaseSearch.clickHere")}
            {isHintOpen ? (
              <ChevronUpIcon className="ml-1 h-3 w-3" />
            ) : (
              <ChevronDownIcon className="ml-1 h-3 w-3" />
            )}
          </button>
          {" "}
          {t("releaseSearch.descriptionSuffix")}
        </p>
        {isHintOpen ? (
          <div className="rounded-md text-sm mt-2 border border-gray-300 text-black shadow-lg dark:text-white dark:border-gray-700 dark:shadow-2xl">
            <div className="flex justify-between items-center text-base font-medium pl-2 py-1 border-b border-gray-300 bg-gray-100 dark:border-gray-700 dark:bg-gray-800 rounded-t-md">
              {t("releaseSearch.searchTips")}
            </div>
            <div className="py-1 px-2 rounded-b-md bg-white dark:bg-gray-900">
              <Trans
                i18nKey="releaseSearch.wildcardIntro"
                ns="common"
                components={{
                  strong: <b />,
                  highlight: <span className="underline decoration-2 underline-offset-2 decoration-amber-500" />
                }}
              />
              <br />
              <Trans
                i18nKey="releaseSearch.percentRule"
                ns="common"
                components={{ code1: <Code>%</Code>, italic: <i />, code2: <Code>.*</Code> }}
              />
              <br />
              <Trans
                i18nKey="releaseSearch.underscoreRule"
                ns="common"
                components={{ code1: <Code>_</Code>, italic: <i />, code2: <Code>.</Code> }}
              />
              <br /><br />

              <Trans
                i18nKey="releaseSearch.keywordFaceting"
                ns="common"
                components={{ highlight: <span className="underline decoration-2 underline-offset-2 decoration-lime-500" /> }}
              />
              {" "}
              <Trans
                i18nKey="releaseSearch.supportedKeywords"
                ns="common"
                components={{
                  category: <b />,
                  codec: <b />,
                  episode: <b />,
                  filter: <b />,
                  group: <b />,
                  hdr: <b />,
                  resolution: <b />,
                  season: <b />,
                  source: <b />,
                  title: <b />,
                  year: <b />
                }}
              />
              <br /><br />

              <b>{t("releaseSearch.examples")}</b><br />

              <Code>year:2022 resolution:1080p</Code> ({t("releaseSearch.exampleYearResolution")})
              <br />
              <Code>group:framestor hdr:DV</Code> ({t("releaseSearch.exampleGroupHdr")})
              <br />
              <Code>Movie Title resolution:1080p</Code> ({t("releaseSearch.exampleMovieTitle")})
              <br />
              <Code>The Show season:05 episode:03</Code> ({t("releaseSearch.exampleShowEpisode")})
              <br />
              <Code>%collection hd%</Code> ({t("releaseSearch.exampleCollectionSpace")})
              <br />
              <Code>%collection_hd%</Code> ({t("releaseSearch.exampleCollectionWildcard")})

              <br /><br />

              {t("releaseSearch.docsPrefix")}
              {" "}
              <ExternalLink
                href="https://autobrr.com/usage/search/"
                className="text-gray-700 dark:text-gray-200 underline font-semibold underline-offset-2 decoration-purple-500 decoration-2 hover:text-black dark:hover:text-gray-100"
              >
                {t("releaseSearch.docsLink")}
              </ExternalLink>
              {" "}
              {t("releaseSearch.docsSuffix")}
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
