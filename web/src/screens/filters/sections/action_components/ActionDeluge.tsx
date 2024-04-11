/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import * as Input from "@components/inputs";

import { CollapsibleSection } from "../_components";
import * as FilterSection from "../_components";

export const Deluge = ({ idx, action, clients }: ClientActionProps) => (
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
            label="Label"
            columns={6}
            placeholder="eg. label1 (must exist in Deluge to work)"
          />
        </FilterSection.HalfRow>

        <Input.TextAreaAutoResize
          name={`actions.${idx}.save_path`}
          label="Save path"
          placeholder="eg. /full/path/to/download_folder"
        />
      </FilterSection.Layout>

      <FilterSection.Layout className="pb-6">
        <FilterSection.HalfRow>
          <Input.SwitchGroup
            name={`actions.${idx}.paused`}
            label="Add paused"
            description="Add torrent as paused"
          />
        </FilterSection.HalfRow>
        <FilterSection.HalfRow>
        <Input.SwitchGroup
            name={`actions.${idx}.skip_hash_check`}
            label="Skip hash check"
            description="Add torrent and skip hash check"
            tooltip={<div>This will only work on Deluge v2.</div>}
          />
        </FilterSection.HalfRow>
      </FilterSection.Layout>

      <CollapsibleSection
        noBottomBorder
        title="Limits"
        subtitle="Configure your speed/ratio/seed time limits"
      >
        <Input.NumberField
          name={`actions.${idx}.limit_download_speed`}
          label="Limit download speed (KB/s)"
          placeholder="Takes any number (0 is no limit)"
        />
        <Input.NumberField
          name={`actions.${idx}.limit_upload_speed`}
          label="Limit upload speed (KB/s)"
          placeholder="Takes any number (0 is no limit)"
        />
      </CollapsibleSection>
    </FilterSection.Section>
  </>
);
