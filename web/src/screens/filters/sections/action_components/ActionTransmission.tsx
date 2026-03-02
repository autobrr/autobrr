/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { CollapsibleSection, FilterHalfRow, FilterLayout, FilterSection, FilterWideGridGapClass } from "../_components";
import { DownloadClientSelect, NumberField, SwitchGroup, TextAreaAutoResize, TextField } from "@components/inputs/tanstack";
import { ContextField } from "@app/lib/form";

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
          <ContextField name={`actions.${idx}.client_id`}>
            <DownloadClientSelect
              action={action}
              clients={clients}
            />
          </ContextField>
        </FilterHalfRow>
        <FilterHalfRow>
          <ContextField name={`actions.${idx}.label`}>
            <TextField
              label="Torrent Label"
              columns={6}
              placeholder="eg. label1"
            />
          </ContextField>
        </FilterHalfRow>
      </FilterLayout>

      <ContextField name={`actions.${idx}.save_path`}>
        <TextAreaAutoResize
          label="Save path"
          columns={6}
          placeholder="eg. /full/path/to/download_folder"
        />
      </ContextField>

      <FilterLayout className="pb-6">
        <FilterHalfRow>
          <ContextField name={`actions.${idx}.paused`}>
            <SwitchGroup
              label="Add paused"
              description="Add torrent as paused"
            />
          </ContextField>
        </FilterHalfRow>
      </FilterLayout>

      <CollapsibleSection
        title="Limits"
        subtitle="Configure your speed/ratio/seed time limits"
      >
        <FilterLayout>
          <ContextField name={`actions.${idx}.limit_download_speed`}>
            <NumberField
              label="Limit download speed (KiB/s)"
              placeholder="Takes any number (0 is no limit)"
            />
          </ContextField>
          <ContextField name={`actions.${idx}.limit_upload_speed`}>
            <NumberField
              label="Limit upload speed (KiB/s)"
              placeholder="Takes any number (0 is no limit)"
            />
          </ContextField>
        </FilterLayout>

        <FilterLayout>
          <ContextField name={`actions.${idx}.limit_ratio`}>
            <NumberField
              label="Ratio limit"
              placeholder="Takes any number (0 is no limit)"
              step={0.25}
              isDecimal
            />
          </ContextField>
          <ContextField name={`actions.${idx}.limit_seed_time`}>
            <NumberField
              label="Seed time limit (minutes)"
              placeholder="Takes any number (0 is no limit)"
            />
          </ContextField>
        </FilterLayout>
      </CollapsibleSection>

      <CollapsibleSection
        noBottomBorder
        title="Announce"
        subtitle="Set number of reannounces (if needed), delete after Y announce failures, etc."
        childClassName={FilterWideGridGapClass}
      >
        <FilterHalfRow>
          <ContextField name={`actions.${idx}.reannounce_skip`}>
            <SwitchGroup
              label="Disable reannounce"
              description="Reannounce is enabled by default. Disable if it's not needed"
              className="pt-2 pb-4"
            />
          </ContextField>
          <ContextField name={`actions.${idx}.reannounce_interval`}>
            <NumberField
              label="Reannounce interval. Run every X seconds"
              placeholder="7 is default and recommended"
            />
          </ContextField>
        </FilterHalfRow>
        <FilterHalfRow>
          <ContextField name={`actions.${idx}.reannounce_delete`}>
            <SwitchGroup
              label="Delete stalled"
              description="Delete stalled torrents after Y attempts"
              className="pt-2 pb-4"
            />
          </ContextField>
          <ContextField name={`actions.${idx}.reannounce_max_attempts`}>
            <NumberField
              label="Run reannounce Y times"
            />
          </ContextField>
        </FilterHalfRow>
      </CollapsibleSection>
    </FilterSection>
  </>
);
