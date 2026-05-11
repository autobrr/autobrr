/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useSuspenseQuery } from "@tanstack/react-query";
import { useFormikContext } from "formik";
import { useTranslation } from "react-i18next";

import { downloadsPerUnitOptions, windowTypeOptions } from "@domain/constants";
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

const MaxDownloadsIndicator = () => {
  const { values } = useFormikContext<Filter>();
  
  const maxDownloads = values.max_downloads;
  const interval = values.max_downloads_interval || 1;
  const unit = values.max_downloads_unit;
  
  // Don't show anything if max_downloads is not set or is 0 (infinite)
  if (!maxDownloads || maxDownloads === 0) {
    return null;
  }
  
  // Don't show if unit is not selected
  if (!unit || unit === "") {
    return null;
  }
  
  // Format the unit to be more readable
  const formatUnit = (unitValue: string, count: number) => {
    const unitMap: Record<string, string> = {
      "MINUTE": "minute",
      "HOUR": "hour",
      "DAY": "day",
      "WEEK": "week",
      "MONTH": "month",
      "EVER": "ever"
    };
    
    const baseUnit = unitMap[unitValue] || unitValue.toLowerCase();
    
    // Handle "ever" specially (no plural)
    if (unitValue === "EVER") {
      return baseUnit;
    }
    
    // Pluralize if interval > 1
    return count > 1 ? `${baseUnit}s` : baseUnit;
  };
  
  const readableUnit = formatUnit(unit, interval);
  const intervalText = interval > 1 ? `${interval} ` : "";
  
  return (
    <div className="col-span-12 -mt-3 text-sm text-gray-600 dark:text-gray-400">
      {maxDownloads} download{maxDownloads > 1 ? "s" : ""} every {intervalText}{readableUnit}
    </div>
  );
};

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
          <NumberField
            name="max_downloads_interval"
            label="Download interval"
            min={1}
            columns={3}
            placeholder="1 (default)"
            tooltip={
              <div>
                <p>Interval multiplier for max downloads. For example: 10 downloads every 2 hours.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <Select
            name="max_downloads_unit"
            label={t("filters:general.maxDownloadsPer")}
            columns={3}
            options={downloadsPerUnitOptions}
            optionDefaultText={t("filters:general.selectUnit")}
            tooltip={
              <div>
                <p>{t("filters:general.maxDownloadsPerTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <MaxDownloadsIndicator />
          <Select
            name="max_downloads_window_type"
            label="Window type"
            columns={6}
            options={windowTypeOptions}
            optionDefaultText="Select window type"
            tooltip={
              <div>
                <p><strong>Fixed:</strong> Resets at calendar boundaries (e.g., midnight, top of hour). Multiple filters may download simultaneously at reset time.</p>
                <p><strong>Rolling:</strong> Sliding window of last X hours/days. Better load distribution across time.</p>
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
