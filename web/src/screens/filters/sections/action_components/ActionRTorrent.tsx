/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { ActionRtorrentRenameOptions } from "@domain/constants";
import { FilterHalfRow, FilterLayout, FilterSection } from "@screens/filters/sections/_components.tsx";
import { DownloadClientSelect, Select, SwitchGroup, TextAreaAutoResize, TextField } from "@components/inputs/tanstack";
import { ContextField } from "@app/lib/form";


export const RTorrent = ({ idx, action, clients }: ClientActionProps) => (
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
              placeholder="eg. label1,label2"
            />
          </ContextField>
        </FilterHalfRow>
      </FilterLayout>

      <ContextField name={`actions.${idx}.save_path`}>
        <TextAreaAutoResize
          label="Save path"
          placeholder="eg. /full/path/to/download_folder"
        />
      </ContextField>

      <FilterLayout>
        <FilterHalfRow>
          <ContextField name={`actions.${idx}.paused`}>
            <SwitchGroup
              label="Add paused"
              description="Add torrent as paused"
              className="pt-2 pb-4"
            />
          </ContextField>
          <ContextField name={`actions.${idx}.content_layout`}>
            <Select
              label="Do not add torrent name to path"
              optionDefaultText="No"
              options={ActionRtorrentRenameOptions}
            />
          </ContextField>
        </FilterHalfRow>
      </FilterLayout>
    </FilterSection>
  </>
);
