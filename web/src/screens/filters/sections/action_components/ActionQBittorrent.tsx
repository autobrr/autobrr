/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Link } from "@tanstack/react-router";

import { DocsLink } from "@components/ExternalLink";
import { ActionContentLayoutOptions, ActionPriorityOptions } from "@domain/constants";
import * as Input from "@components/inputs";

import { CollapsibleSection } from "../_components";
import * as FilterSection from "../_components";

export const QBittorrent = ({ idx, action, clients }: ClientActionProps) => (
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
      </FilterSection.Layout>

      <FilterSection.Layout>
        <Input.TextField
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

        <Input.TextField
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
      </FilterSection.Layout>

      <FilterSection.Layout className="pb-6">
        <Input.TextAreaAutoResize
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
      </FilterSection.Layout>

      <CollapsibleSection
        title="Rules"
        subtitle="Configure your torrent client rules"
        childClassName={FilterSection.WideGridGapClass}
      >
        <FilterSection.HalfRow>
          <Input.SwitchGroup
            name={`actions.${idx}.ignore_rules`}
            label="Ignore existing client rules"
            description={
              <p>
                Choose to ignore rules set in <Link className="text-blue-400 visited:text-blue-400" to="/settings/clients">Client Settings</Link>.
              </p>
            }
            className="py-2 pb-4"
          />
          <Input.Select
            name={`actions.${idx}.content_layout`}
            label="Content Layout"
            optionDefaultText="Select content layout"
            options={ActionContentLayoutOptions}
          />
        </FilterSection.HalfRow>

        <FilterSection.HalfRow>
          <Input.SwitchGroup
            name={`actions.${idx}.paused`}
            label="Add paused"
            description="Add torrent as paused"
          />
          <Input.SwitchGroup
            name={`actions.${idx}.skip_hash_check`}
            label="Skip hash check"
            description="Add torrent and skip hash check"
          />
          <Input.SwitchGroup
            name={`actions.${idx}.first_last_piece_prio`}
            label="Download first and last pieces first"
            description="Add torrent and download first and last pieces first"
          />
        </FilterSection.HalfRow>
        <FilterSection.HalfRow>
        <Input.Select
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
        </FilterSection.HalfRow>
      </CollapsibleSection>

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
