/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import * as Input from "@components/inputs";

import { CollapsibleSection } from "../_components";
import * as FilterSection from "../_components";

export const Transmission = ({ idx, action, clients }: ClientActionProps) => (
  <>
    <FilterSection.Section
      title="Instance"
      subtitle={
        <>Select the <span className="font-bold">specific instance</span> which you want to handle this release filter.</>
      }
    >
      <FilterSection.Layout>
        <FilterSection.HalfRow>
          <Input.DownloadClientSelect
            name={`actions.${idx}.client_id`}
            action={action}
            clients={clients}
          />
        </FilterSection.HalfRow>
        <FilterSection.HalfRow>
          <Input.TextField
            name={`actions.${idx}.label`}
            label="Torrent Label"
            columns={6}
            placeholder="eg. label1"
          />
        </FilterSection.HalfRow>
      </FilterSection.Layout>

      <Input.TextAreaAutoResize
        name={`actions.${idx}.save_path`}
        label="Save path"
        columns={6}
        placeholder="eg. /full/path/to/download_folder"
      />

      <FilterSection.Layout className="pb-6">
        <FilterSection.HalfRow>
          <Input.SwitchGroup
            name={`actions.${idx}.paused`}
            label="Add paused"
            description="Add torrent as paused"
          />
        </FilterSection.HalfRow>
      </FilterSection.Layout>

      <CollapsibleSection
        title="Limits"
        subtitle="Configure your speed/ratio/seed time limits"
      >
        <FilterSection.Layout>
          <Input.NumberField
            name={`actions.${idx}.limit_download_speed`}
            label="Limit download speed (KiB/s)"
            placeholder="Takes any number (0 is no limit)"
          />
          <Input.NumberField
            name={`actions.${idx}.limit_upload_speed`}
            label="Limit upload speed (KiB/s)"
            placeholder="Takes any number (0 is no limit)"
          />
        </FilterSection.Layout>

        <FilterSection.Layout>
          <Input.NumberField
            name={`actions.${idx}.limit_ratio`}
            label="Ratio limit"
            placeholder="Takes any number (0 is no limit)"
            step={0.25}
            isDecimal
          />
          <Input.NumberField
            name={`actions.${idx}.limit_seed_time`}
            label="Seed time limit (minutes)"
            placeholder="Takes any number (0 is no limit)"
          />
        </FilterSection.Layout>
      </CollapsibleSection>

      <CollapsibleSection
        noBottomBorder
        title="Announce"
        subtitle="Set number of reannounces (if needed), delete after Y announce failures, etc."
        childClassName={FilterSection.WideGridGapClass}
      >
        <FilterSection.HalfRow>
          <Input.SwitchGroup
            name={`actions.${idx}.reannounce_skip`}
            label="Skip reannounce"
            description="If reannounce is not needed, skip it completely"
            className="pt-2 pb-4"
          />
          <Input.NumberField
            name={`actions.${idx}.reannounce_interval`}
            label="Reannounce interval. Run every X seconds"
            placeholder="7 is default and recommended"
          />
        </FilterSection.HalfRow>
        <FilterSection.HalfRow>
          <Input.SwitchGroup
            name={`actions.${idx}.reannounce_delete`}
            label="Delete stalled"
            description="Delete stalled torrents after Y attempts"
            className="pt-2 pb-4"
          />
          <Input.NumberField
            name={`actions.${idx}.reannounce_max_attempts`}
            label="Run reannounce Y times"
          />
        </FilterSection.HalfRow>
      </CollapsibleSection>
    </FilterSection.Section>
  </>
);
