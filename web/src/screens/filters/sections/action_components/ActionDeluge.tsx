/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { CollapsibleSection, FilterHalfRow, FilterLayout, FilterSection } from "../_components";
import { DownloadClientSelect, NumberField, SwitchGroup, TextAreaAutoResize, TextField } from "@components/inputs/tanstack";
import { ContextField } from "@app/lib/form";

export const Deluge = ({ idx, action, clients }: ClientActionProps) => (
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
              label="Label"
              columns={6}
              placeholder="eg. label1 (must exist in Deluge to work)"
            />
          </ContextField>
        </FilterHalfRow>

        <ContextField name={`actions.${idx}.save_path`}>
          <TextAreaAutoResize
            label="Save path"
            placeholder="eg. /full/path/to/download_folder"
          />
        </ContextField>
      </FilterLayout>

      <FilterLayout className="pb-6">
        <FilterHalfRow>
          <ContextField name={`actions.${idx}.paused`}>
            <SwitchGroup
              label="Add paused"
              description="Add torrent as paused"
            />
          </ContextField>
        </FilterHalfRow>
        <FilterHalfRow>
          <ContextField name={`actions.${idx}.skip_hash_check`}>
            <SwitchGroup
              label="Skip hash check"
              description="Add torrent and skip hash check"
              tooltip={<div>This will only work on Deluge v2.</div>}
            />
          </ContextField>
        </FilterHalfRow>
      </FilterLayout>

      <CollapsibleSection
        noBottomBorder
        title="Limits"
        subtitle="Configure your speed/ratio/seed time limits"
      >
        <ContextField name={`actions.${idx}.limit_download_speed`}>
          <NumberField
            label="Limit download speed (KB/s)"
            placeholder="Takes any number (0 is no limit)"
          />
        </ContextField>
        <ContextField name={`actions.${idx}.limit_upload_speed`}>
          <NumberField
            label="Limit upload speed (KB/s)"
            placeholder="Takes any number (0 is no limit)"
          />
        </ContextField>
      </CollapsibleSection>
    </FilterSection>
  </>
);
