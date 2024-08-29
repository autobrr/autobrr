/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { ActionRtorrentRenameOptions } from "@domain/constants";
import { FilterHalfRow, FilterLayout, FilterSection } from "@screens/filters/sections/_components.tsx";
import { DownloadClientSelect, Select, SwitchGroup, TextAreaAutoResize, TextField } from "@components/inputs";


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
            placeholder="eg. label1,label2"
          />
        </FilterHalfRow>
      </FilterLayout>

      <TextAreaAutoResize
        name={`actions.${idx}.save_path`}
        label="Save path"
        placeholder="eg. /full/path/to/download_folder"
      />

      <FilterLayout>
        <FilterHalfRow>
          <SwitchGroup
            name={`actions.${idx}.paused`}
            label="Add paused"
            description="Add torrent as paused"
            className="pt-2 pb-4"
          />
          <Select
            name={`actions.${idx}.content_layout`}
            label="Do not add torrent name to path"
            optionDefaultText="No"
            options={ActionRtorrentRenameOptions}
          />
        </FilterHalfRow>
      </FilterLayout>
    </FilterSection>
  </>
);
