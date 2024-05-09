/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { ActionRtorrentRenameOptions } from "@domain/constants";
import * as Input from "@components/inputs";

import * as FilterSection from "../_components";

export const RTorrent = ({ idx, action, clients }: ClientActionProps) => (
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
            placeholder="eg. label1,label2"
          />
        </FilterSection.HalfRow>
      </FilterSection.Layout>

      <Input.TextAreaAutoResize
        name={`actions.${idx}.save_path`}
        label="Save path"
        placeholder="eg. /full/path/to/download_folder"
      />

      <FilterSection.Layout>
        <FilterSection.HalfRow>
          <Input.SwitchGroup
            name={`actions.${idx}.paused`}
            label="Add paused"
            description="Add torrent as paused"
            className="pt-2 pb-4"
          />
          <Input.Select
            name={`actions.${idx}.content_layout`}
            label="Do not add torrent name to path"
            optionDefaultText="No"
            options={ActionRtorrentRenameOptions}
          />
        </FilterSection.HalfRow>
      </FilterSection.Layout>
    </FilterSection.Section>
  </>
);
