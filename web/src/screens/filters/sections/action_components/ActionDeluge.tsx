/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { CollapsibleSection, FilterHalfRow, FilterLayout, FilterSection } from "../_components";
import { DownloadClientSelect, NumberField, SwitchGroup, TextAreaAutoResize, TextField } from "@components/inputs";

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
          <DownloadClientSelect
            name={`actions.${idx}.client_id`}
            action={action}
            clients={clients}
          />
        </FilterHalfRow>
        <FilterHalfRow>
          <TextField
            name={`actions.${idx}.label`}
            label="Label"
            columns={6}
            placeholder="eg. label1 (must exist in Deluge to work)"
          />
        </FilterHalfRow>

        <TextAreaAutoResize
          name={`actions.${idx}.save_path`}
          label="Save path"
          placeholder="eg. /full/path/to/download_folder"
        />
      </FilterLayout>

      <FilterLayout className="pb-6">
        <FilterHalfRow>
          <SwitchGroup
            name={`actions.${idx}.paused`}
            label="Add paused"
            description="Add torrent as paused"
          />
        </FilterHalfRow>
        <FilterHalfRow>
        <SwitchGroup
            name={`actions.${idx}.skip_hash_check`}
            label="Skip hash check"
            description="Add torrent and skip hash check"
            tooltip={<div>This will only work on Deluge v2.</div>}
          />
        </FilterHalfRow>
      </FilterLayout>

      <CollapsibleSection
        noBottomBorder
        title="Limits"
        subtitle="Configure your speed/ratio/seed time limits"
      >
        <NumberField
          name={`actions.${idx}.limit_download_speed`}
          label="Limit download speed (KB/s)"
          placeholder="Takes any number (0 is no limit)"
        />
        <NumberField
          name={`actions.${idx}.limit_upload_speed`}
          label="Limit upload speed (KB/s)"
          placeholder="Takes any number (0 is no limit)"
        />
      </CollapsibleSection>
    </FilterSection>
  </>
);
