/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { CollapsibleSection, FilterHalfRow, FilterLayout, FilterSection } from "../_components";
import { DownloadClientSelect, NumberField, TextAreaAutoResize, TextField } from "@components/inputs/tanstack";
import { ContextField } from "@app/lib/form";

export const Porla = ({ idx, action, clients }: ClientActionProps) => (
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
              label="Preset"
              placeholder="eg. default"
              tooltip={
                <div>A case-sensitive preset name as configured in Porla.</div>
              }
            />
          </ContextField>
        </FilterHalfRow>
      </FilterLayout>

      <ContextField name={`actions.${idx}.save_path`}>
        <TextAreaAutoResize
          label="Save path"
          placeholder="eg. /full/path/to/torrent/data"
          className="pb-6"
        />
      </ContextField>

      <CollapsibleSection
        noBottomBorder
        title="Limits"
        subtitle="Configure your speed/ratio/seed time limits"
      >
        <FilterHalfRow>
          <ContextField name={`actions.${idx}.limit_download_speed`}>
            <NumberField
              label="Limit download speed (KiB/s)"
              placeholder="Takes any number (0 is no limit)"
            />
          </ContextField>
        </FilterHalfRow>
        <FilterHalfRow>
          <ContextField name={`actions.${idx}.limit_upload_speed`}>
            <NumberField
              label="Limit upload speed (KiB/s)"
              placeholder="Takes any number (0 is no limit)"
            />
          </ContextField>
        </FilterHalfRow>
      </CollapsibleSection>
    </FilterSection>
  </>
);
