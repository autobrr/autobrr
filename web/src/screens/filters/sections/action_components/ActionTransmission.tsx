/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { CollapsibleSection, FilterHalfRow, FilterLayout, FilterSection, FilterWideGridGapClass } from "../_components";
import { DownloadClientSelect, NumberField, SwitchGroup, TextAreaAutoResize, TextField } from "@components/inputs";

export const Transmission = ({ idx, action, clients }: ClientActionProps) => (
  <>
    <FilterSection
      title="Instance"
      subtitle={
        <>Select the <span className="font-bold">specific instance</span> which you want to handle this release filter.</>
      }
    >
      <FilterLayout>
        <FilterHalfRow>
          <DownloadClientSelect
            name={`actions.${idx}.client_id`}
            action={action}
            clients={clients}
          />
        </FilterHalfRow>
        <FilterHalfRow>
          <TextField
            name={`actions.${idx}.label`}
            label="Torrent Label"
            columns={6}
            placeholder="eg. label1"
          />
        </FilterHalfRow>
      </FilterLayout>

      <TextAreaAutoResize
        name={`actions.${idx}.save_path`}
        label="Save path"
        columns={6}
        placeholder="eg. /full/path/to/download_folder"
      />

      <FilterLayout className="pb-6">
        <FilterHalfRow>
          <SwitchGroup
            name={`actions.${idx}.paused`}
            label="Add paused"
            description="Add torrent as paused"
          />
        </FilterHalfRow>
      </FilterLayout>

      <CollapsibleSection
        title="Limits"
        subtitle="Configure your speed/ratio/seed time limits"
      >
        <FilterLayout>
          <NumberField
            name={`actions.${idx}.limit_download_speed`}
            label="Limit download speed (KiB/s)"
            placeholder="Takes any number (0 is no limit)"
          />
          <NumberField
            name={`actions.${idx}.limit_upload_speed`}
            label="Limit upload speed (KiB/s)"
            placeholder="Takes any number (0 is no limit)"
          />
        </FilterLayout>

        <FilterLayout>
          <NumberField
            name={`actions.${idx}.limit_ratio`}
            label="Ratio limit"
            placeholder="Takes any number (0 is no limit)"
            step={0.25}
            isDecimal
          />
          <NumberField
            name={`actions.${idx}.limit_seed_time`}
            label="Seed time limit (minutes)"
            placeholder="Takes any number (0 is no limit)"
          />
        </FilterLayout>
      </CollapsibleSection>

      <CollapsibleSection
        noBottomBorder
        title="Announce"
        subtitle="Set number of reannounces (if needed), delete after Y announce failures, etc."
        childClassName={FilterWideGridGapClass}
      >
        <FilterHalfRow>
          <SwitchGroup
            name={`actions.${idx}.reannounce_skip`}
            label="Disable reannounce"
            description="Reannounce is enabled by default. Disable if it's not needed"
            className="pt-2 pb-4"
          />
          <NumberField
            name={`actions.${idx}.reannounce_interval`}
            label="Reannounce interval. Run every X seconds"
            placeholder="7 is default and recommended"
          />
        </FilterHalfRow>
        <FilterHalfRow>
          <SwitchGroup
            name={`actions.${idx}.reannounce_delete`}
            label="Delete stalled"
            description="Delete stalled torrents after Y attempts"
            className="pt-2 pb-4"
          />
          <NumberField
            name={`actions.${idx}.reannounce_max_attempts`}
            label="Run reannounce Y times"
          />
        </FilterHalfRow>
      </CollapsibleSection>
    </FilterSection>
  </>
);
