/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useTranslation } from "react-i18next";

import { DocsLink } from "@components/ExternalLink";
import { TextAreaAutoResize } from "@components/inputs/input";
import { MultiSelect, SwitchGroup, TextField } from "@components/inputs";

import * as CONSTS from "@domain/constants";
import {
  FilterLayout,
  FilterPage,
  FilterSection,
  FilterWideGridGapClass
} from "@screens/filters/sections/_components.tsx";

const SeasonsAndEpisodes = () => {
  const { t } = useTranslation("filters");

  return (
    <FilterSection
      title={t("moviesTv.seasonEpisode.title")}
      subtitle={t("moviesTv.seasonEpisode.subtitle")}
    >
    <FilterLayout>
      <TextField
        name="seasons"
        label={t("moviesTv.seasonEpisode.seasons")}
        columns={6}
        placeholder={t("moviesTv.seasonEpisode.seasonsPlaceholder")}
        tooltip={
          <div>
            <p>{t("moviesTv.seasonEpisode.seasonsTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters/examples#only-season-packs" />
          </div>
        }
      />
      <TextField
        name="episodes"
        label={t("moviesTv.seasonEpisode.episodes")}
        columns={6}
        placeholder={t("moviesTv.seasonEpisode.episodesPlaceholder")}
        tooltip={
          <div>
            <p>{t("moviesTv.seasonEpisode.episodesTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters/examples#only-episodes-skip-season-packs"/>
          </div>
        }
      />
      <p className="col-span-12 -mb-1 text-sm font-bold text-gray-800 dark:text-gray-100 tracking-wide">{t("moviesTv.seasonEpisode.dailyShows")}</p>
      <TextField
        name="years"
        label={t("moviesTv.years")}
        columns={4}
        placeholder={t("moviesTv.yearsPlaceholder")}
        tooltip={
          <div>
            <p>{t("moviesTv.yearsTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters#tvmovies"/>
          </div>
        }
      />
      <TextField
        name="months"
        label={t("moviesTv.seasonEpisode.months")}
        columns={4}
        placeholder={t("moviesTv.seasonEpisode.monthsPlaceholder")}
        tooltip={
          <div>
            <p>{t("moviesTv.seasonEpisode.monthsTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters#tvmovies"/>
          </div>
        }
      />
      <TextField
        name="days"
        label={t("moviesTv.seasonEpisode.days")}
        columns={4}
        placeholder={t("moviesTv.seasonEpisode.daysPlaceholder")}
        tooltip={
          <div>
            <p>{t("moviesTv.seasonEpisode.daysTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters#tvmovies"/>
          </div>
        }
      />
      <div className="col-span-12 sm:col-span-6">
        <SwitchGroup
          name="smart_episode"
          label={t("moviesTv.seasonEpisode.smartEpisode")}
          description={t("moviesTv.seasonEpisode.smartEpisodeDescription")}
        />
      </div>
    </FilterLayout>
  </FilterSection>
  );
};

const Quality = () => {
  const { t } = useTranslation("filters");

  return (
    <FilterSection
      title={t("moviesTv.quality.title")}
      subtitle={t("moviesTv.quality.subtitle")}
    >
    <FilterLayout gap={FilterWideGridGapClass}>
      <MultiSelect
        name="resolutions"
        options={CONSTS.RESOLUTION_OPTIONS}
        label={t("moviesTv.quality.resolutions")}
        columns={6}
        tooltip={
          <div>
            <p>{t("moviesTv.quality.resolutionsTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters#quality" />
          </div>
        }
      />
      <MultiSelect
        name="sources"
        options={CONSTS.SOURCES_OPTIONS}
        label={t("moviesTv.quality.sources")}
        columns={6}
        tooltip={
          <div>
            <p>{t("moviesTv.quality.sourcesTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters#quality" />
          </div>
        }
      />
    </FilterLayout>

    <FilterLayout gap={FilterWideGridGapClass}>
      <MultiSelect
        name="codecs"
        options={CONSTS.CODECS_OPTIONS}
        label={t("moviesTv.quality.codecs")}
        columns={6}
        tooltip={
          <div>
            <p>{t("moviesTv.quality.codecsTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters#quality" />
          </div>
        }
      />
      <MultiSelect
        name="containers"
        options={CONSTS.CONTAINER_OPTIONS}
        label={t("moviesTv.quality.containers")}
        columns={6}
        tooltip={
          <div>
            <p>{t("moviesTv.quality.containersTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters#quality" />
          </div>
        }
      />
    </FilterLayout>

    <FilterLayout gap={FilterWideGridGapClass}>
      <MultiSelect
        name="match_hdr"
        options={CONSTS.HDR_OPTIONS}
        label={t("moviesTv.quality.matchHdr")}
        columns={6}
        tooltip={
          <div>
            <p>{t("moviesTv.quality.matchHdrTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters#quality" />
          </div>
        }
      />
      <MultiSelect
        name="except_hdr"
        options={CONSTS.HDR_OPTIONS}
        label={t("moviesTv.quality.exceptHdr")}
        columns={6}
        tooltip={
          <div>
            <p>{t("moviesTv.quality.exceptHdrTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters#quality" />
          </div>
        }
      />
    </FilterLayout>

    <FilterLayout gap={FilterWideGridGapClass}>
      <MultiSelect
        name="match_other"
        options={CONSTS.OTHER_OPTIONS}
        label={t("moviesTv.quality.matchOther")}
        columns={6}
        tooltip={
          <div>
            <p>{t("moviesTv.quality.matchOtherTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters#quality" />
          </div>
        }
      />
      <MultiSelect
        name="except_other"
        options={CONSTS.OTHER_OPTIONS}
        label={t("moviesTv.quality.exceptOther")}
        columns={6}
        tooltip={
          <div>
            <p>{t("moviesTv.quality.exceptOtherTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters#quality" />
          </div>
        }
      />
    </FilterLayout>
  </FilterSection>
  );
};

export const MoviesTv = () => {
  const { t } = useTranslation("filters");

  return (
    <FilterPage>
    <FilterSection>
      <FilterLayout>
        <TextAreaAutoResize
          name="shows"
          label={t("moviesTv.title")}
          columns={8}
          placeholder={t("moviesTv.placeholder")}
          tooltip={
            <div>
              <p>{t("moviesTv.wildcardTooltip")}</p>
              <DocsLink href="https://autobrr.com/filters#tvmovies" />
            </div>
          }
        />
        <TextField
          name="years"
          label={t("moviesTv.years")}
          columns={4}
          placeholder={t("moviesTv.yearsPlaceholder")}
          tooltip={
            <div>
              <p>{t("moviesTv.yearsTooltip")}</p>
              <DocsLink href="https://autobrr.com/filters#tvmovies" />
            </div>
          }
        />
      </FilterLayout>
    </FilterSection>

    <SeasonsAndEpisodes />
    <Quality />
  </FilterPage>
  );
};
