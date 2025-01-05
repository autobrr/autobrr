/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Link } from "@tanstack/react-router";

import { DocsLink } from "@components/ExternalLink";
import { ActionContentLayoutOptions, ActionPriorityOptions } from "@domain/constants";

import { CollapsibleSection, FilterHalfRow, FilterLayout, FilterSection, FilterWideGridGapClass } from "../_components";
import {
  DownloadClientSelect,
  NumberField,
  Select,
  SwitchGroup,
  TextAreaAutoResize,
  TextField
} from "@components/inputs";

export const QBittorrent = ({ idx, action, clients }: ClientActionProps) => (
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
      </FilterLayout>

      <FilterLayout>
        <TextField
          name={`actions.${idx}.category`}
          label="Category"
          columns={6}
          placeholder="eg. category"
          tooltip={
            <div>
              <p>The field can use macros to transform/add values from metadata:</p>
              <DocsLink href="https://autobrr.com/filters/macros" />
            </div>
          }
        />

        <TextField
          name={`actions.${idx}.tags`}
          label="Tags"
          columns={6}
          placeholder="eg. tag1,tag2"
          tooltip={
            <div>
              <p>The field can use macros to transform/add values from metadata:</p>
              <DocsLink href="https://autobrr.com/filters/macros" />
            </div>
          }
        />
      </FilterLayout>

      <FilterLayout className="pb-6">
        <TextAreaAutoResize
          name={`actions.${idx}.save_path`}
          label="Save path"
          placeholder="eg. /full/path/to/download_folder"
          tooltip={
            <div>
              <p>Set a custom save path for this action. Automatic Torrent Management will take care of this if using qBittorrent with categories.</p>
              <br />
              <p>The field can use macros to transform/add values from metadata:</p>
              <DocsLink href="https://autobrr.com/filters/macros" />
            </div>
          }
        />
      </FilterLayout>

      <CollapsibleSection
        title="Rules"
        subtitle="Configure your torrent client rules"
        childClassName={FilterWideGridGapClass}
      >
        <FilterHalfRow>
          <SwitchGroup
            name={`actions.${idx}.ignore_rules`}
            label="Ignore existing client rules"
            description={
              <p>
                Choose to ignore rules set in <Link className="text-blue-400 visited:text-blue-400" to="/settings/clients">Client Settings</Link>.
              </p>
            }
            className="py-2 pb-4"
          />
          <Select
            name={`actions.${idx}.content_layout`}
            label="Content Layout"
            optionDefaultText="Select content layout"
            options={ActionContentLayoutOptions}
            className="py-2 pb-4"
          />
          <Select
            name={`actions.${idx}.priority`}
            label="Priority"
            optionDefaultText="Disabled"
            options={ActionPriorityOptions}
            tooltip={
              <div>
                <p>Torrent Queueing will be enabled for you if it is disabled. Ensure you set your preferred limits for it in your client.</p>
              </div>
            }
          />
        </FilterHalfRow>

        <FilterHalfRow>
          <SwitchGroup
            name={`actions.${idx}.paused`}
            label="Add paused"
            description="Add torrent as paused"
          />
          <SwitchGroup
            name={`actions.${idx}.skip_hash_check`}
            label="Skip hash check"
            description="Add torrent and skip hash check"
            className="pt-4 sm:pt-4"
          />
          <SwitchGroup
            name={`actions.${idx}.first_last_piece_prio`}
            label="Download first and last pieces first"
            description="Add torrent and download first and last pieces first"
            className="pt-6 sm:pt-10"
          />
        </FilterHalfRow>
      </CollapsibleSection>

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
