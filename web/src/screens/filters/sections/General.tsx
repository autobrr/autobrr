/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useSuspenseQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";

import { downloadsPerUnitOptions } from "@domain/constants";
import { IndexersOptionsQueryOptions, ReleaseProfileDuplicateList } from "@api/queries";

import { DocsLink } from "@components/ExternalLink";
import { FilterLayout, FilterPage, FilterSection } from "./_components";
import {
  IndexerMultiSelect,
  MultiSelect,
  MultiSelectOption,
  NumberField,
  Select,
  SelectFieldOption,
  SwitchGroup,
  TextField
} from "@components/inputs";
import * as CONSTS from "@domain/constants.ts";

const MapIndexer = (indexer: Indexer) => (
  { label: indexer.name, value: indexer.id } as MultiSelectOption
);

const MapReleaseProfile = (profile: ReleaseProfileDuplicate) => (
  { label: profile.name, value: profile.id } as SelectFieldOption
);

export const General = () => {
  const { t } = useTranslation(["options", "filters"]);
  const indexersQuery = useSuspenseQuery(IndexersOptionsQueryOptions())
  const indexerOptions = indexersQuery.data && indexersQuery.data.map(MapIndexer)

  const duplicateProfilesQuery = useSuspenseQuery(ReleaseProfileDuplicateList())
  const duplicateProfilesOptions = duplicateProfilesQuery.data && duplicateProfilesQuery.data.map(MapReleaseProfile)

  // const indexerOptions = data?.map(MapIndexer) ?? [];

  return (
    <FilterPage>
      <FilterSection>
        <FilterLayout>
          <TextField name="name" label={t("filters:general.filterName")} columns={6} placeholder={t("filters:general.filterNamePlaceholder")} />

          <MultiSelect
            name="announce_types"
            options={CONSTS.getAnnounceTypeOptions(t)}
            label={t("filters:general.announceTypes")}
            columns={3}
            tooltip={
              <div>
                <p>{t("filters:general.announceTypesTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters#announce-type" />
              </div>
            }
          />

          <IndexerMultiSelect name="indexers" options={indexerOptions} label={t("filters:general.indexers")} columns={3} />
        </FilterLayout>
      </FilterSection>

      <FilterSection
        title={t("filters:general.rulesTitle")}
        subtitle={t("filters:general.rulesSubtitle")}
      >
        <FilterLayout>
          <TextField
            name="min_size"
            label={t("filters:general.minSize")}
            columns={6}
            placeholder={t("filters:general.sizePlaceholder")}
            tooltip={
              <div>
                <p>{t("filters:general.sizeTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <TextField
            name="max_size"
            label={t("filters:general.maxSize")}
            columns={6}
            placeholder={t("filters:general.sizePlaceholder")}
            tooltip={
              <div>
                <p>{t("filters:general.sizeTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <NumberField
            name="delay"
            label={t("filters:general.delay")}
            placeholder={t("filters:general.delayPlaceholder")}
            tooltip={
              <div>
                <p>{t("filters:general.delayTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <NumberField
            name="priority"
            label={t("filters:general.priority")}
            placeholder={t("filters:general.priorityPlaceholder")}
            tooltip={
              <div>
                <p>{t("filters:general.priorityTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <NumberField
            name="max_downloads"
            label={t("filters:general.maxDownloads")}
            placeholder={t("filters:general.maxDownloadsPlaceholder")}
            tooltip={
              <div>
                <p>{t("filters:general.maxDownloadsTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <Select
            name="max_downloads_unit"
            label={t("filters:general.maxDownloadsPer")}
            options={downloadsPerUnitOptions}
            optionDefaultText={t("filters:general.selectUnit")}
            tooltip={
              <div>
                <p>{t("filters:general.maxDownloadsPerTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <Select
            name={`release_profile_duplicate_id`}
            label={t("filters:general.skipDuplicatesProfile")}
            optionDefaultText={t("filters:general.selectProfile")}
            options={[{label: t("filters:general.selectProfile"), value: null}, ...duplicateProfilesOptions]}
            tooltip={<div><p>{t("filters:general.skipDuplicatesProfileTooltip")}</p></div>}
          />
        </FilterLayout>

        <FilterLayout>
          <SwitchGroup
            name="enabled"
            label={t("filters:general.enabled")}
            description={t("filters:general.enabledDescription")}
            className="pb-2 col-span-12 sm:col-span-6"
          />
        </FilterLayout>
      </FilterSection>
    </FilterPage>
  );
};
