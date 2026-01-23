/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useSuspenseQuery } from "@tanstack/react-query";
import { useFormikContext } from "formik";

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
  const indexersQuery = useSuspenseQuery(IndexersOptionsQueryOptions())
  const indexerOptions = indexersQuery.data && indexersQuery.data.map(MapIndexer)

  const duplicateProfilesQuery = useSuspenseQuery(ReleaseProfileDuplicateList())
  const duplicateProfilesOptions = duplicateProfilesQuery.data && duplicateProfilesQuery.data.map(MapReleaseProfile)

  // const indexerOptions = data?.map(MapIndexer) ?? [];

  return (
    <FilterPage>
      <FilterSection>
        <FilterLayout>
          <TextField name="name" label="Filter name" columns={6} placeholder="eg. Filter 1" />

          <MultiSelect
            name="announce_types"
            options={CONSTS.AnnounceTypeOptions}
            label="announce types"
            columns={3}
            tooltip={
              <div>
                <p>NEW! Match releases which contain any of the selected announce types.</p>
                <DocsLink href="https://autobrr.com/filters#announce-type" />
              </div>
            }
          />

          <IndexerMultiSelect name="indexers" options={indexerOptions} label="Indexers" columns={3} />
        </FilterLayout>
      </FilterSection>

      <FilterSection
        title="Rules"
        subtitle="Specify rules on how torrents should be handled/selected."
      >
        <FilterLayout>
          <TextField
            name="min_size"
            label="Min size"
            columns={6}
            placeholder="eg. 100MiB, 80GB"
            tooltip={
              <div>
                <p>Supports units such as MB, MiB, GB, etc.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <TextField
            name="max_size"
            label="Max size"
            columns={6}
            placeholder="eg. 100MiB, 80GB"
            tooltip={
              <div>
                <p>Supports units such as MB, MiB, GB, etc.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <NumberField
            name="delay"
            label="Delay"
            placeholder="Number of seconds to delay actions"
            tooltip={
              <div>
                <p>Number of seconds to wait before running actions.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <NumberField
            name="priority"
            label="Priority"
            placeholder="Higher number = higher priority"
            tooltip={
              <div>
                <p>Filters are checked in order of priority. Higher number = higher priority.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <NumberField
            name="max_downloads"
            label="Max downloads"
            placeholder="Takes any number (0 is infinite)"
            tooltip={
              <div>
                <p>Number of max downloads as specified by the respective unit.</p>
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
            label="Max downloads per"
            columns={3}
            options={downloadsPerUnitOptions}
            optionDefaultText="Select unit"
            tooltip={
              <div>
                <p>The unit of time for counting the maximum downloads per filter.</p>
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
            label="Skip Duplicates profile"
            optionDefaultText="Select profile"
            options={[{label: "Select profile", value: null}, ...duplicateProfilesOptions]}
            tooltip={<div><p>Select the skip duplicate profile.</p></div>}
          />
        </FilterLayout>

        <FilterLayout>
          <SwitchGroup
            name="enabled"
            label="Enabled"
            description="Enable or disable this filter."
            className="pb-2 col-span-12 sm:col-span-6"
          />
        </FilterLayout>
      </FilterSection>
    </FilterPage>
  );
};
