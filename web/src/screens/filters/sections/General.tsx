/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useSuspenseQuery } from "@tanstack/react-query";

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
          <Select
            name="max_downloads_unit"
            label="Max downloads per"
            options={downloadsPerUnitOptions}
            optionDefaultText="Select unit"
            tooltip={
              <div>
                <p>The unit of time for counting the maximum downloads per filter.</p>
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
