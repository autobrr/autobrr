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
} from "@components/inputs/tanstack";
import { ContextField } from "@app/lib/form";

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
          <ContextField name={`actions.${idx}.client_id`}>
            <DownloadClientSelect
              action={action}
              clients={clients}
            />
          </ContextField>
        </FilterHalfRow>
      </FilterLayout>

      <FilterLayout>
        <ContextField name={`actions.${idx}.category`}>
          <TextField
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
        </ContextField>

        <ContextField name={`actions.${idx}.tags`}>
          <TextField
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
        </ContextField>
      </FilterLayout>

      <FilterLayout className="pb-6">
        <FilterHalfRow>
          <ContextField name={`actions.${idx}.save_path`}>
            <TextAreaAutoResize
              label="Save path"
              placeholder="eg. /full/path/to/save_folder"
              tooltip={
                <div>
                  <p>Set a custom save path for this action. Automatic Torrent Management will take care of this if using qBittorrent with categories.</p>
                  <br />
                  <p>The field can use macros to transform/add values from metadata:</p>
                  <DocsLink href="https://autobrr.com/filters/macros" />
                </div>
              }
            />
          </ContextField>
        </FilterHalfRow>
        <FilterHalfRow>
          <ContextField name={`actions.${idx}.download_path`}>
            <TextAreaAutoResize
              label="Download path"
              placeholder="eg. /full/path/to/download_folder"
              tooltip={
                <div>
                  <p>Set a custom download (incomplete) path for this action. Automatic Torrent Management will take care of this if using qBittorrent with categories.</p>
                  <br />
                  <p>The field can use macros to transform/add values from metadata:</p>
                  <DocsLink href="https://autobrr.com/filters/macros" />
                </div>
              }
            />
          </ContextField>
        </FilterHalfRow>
      </FilterLayout>

      <CollapsibleSection
        title="Rules"
        subtitle="Configure your torrent client rules"
        childClassName={FilterWideGridGapClass}
      >
        <FilterHalfRow>
          <ContextField name={`actions.${idx}.ignore_rules`}>
            <SwitchGroup
              label="Ignore existing client rules"
              description={
                <p>
                  Choose to ignore rules set in <Link className="text-blue-400 visited:text-blue-400" to="/settings/clients">Client Settings</Link>.
                </p>
              }
              className="py-2 pb-4"
            />
          </ContextField>
          <ContextField name={`actions.${idx}.content_layout`}>
            <Select
              label="Content Layout"
              optionDefaultText="Select content layout"
              options={ActionContentLayoutOptions}
              className="py-2 pb-4"
            />
          </ContextField>
          <ContextField name={`actions.${idx}.priority`}>
            <Select
              label="Priority"
              optionDefaultText="Disabled"
              options={ActionPriorityOptions}
              tooltip={
                <div>
                  <p>Torrent Queueing will be enabled for you if it is disabled. Ensure you set your preferred limits for it in your client.</p>
                </div>
              }
            />
          </ContextField>
        </FilterHalfRow>

        <FilterHalfRow>
          <ContextField name={`actions.${idx}.paused`}>
            <SwitchGroup
              label="Add paused"
              description="Add torrent as paused"
            />
          </ContextField>
          <ContextField name={`actions.${idx}.skip_hash_check`}>
            <SwitchGroup
              label="Skip hash check"
              description="Add torrent and skip hash check"
              className="pt-4 sm:pt-4"
            />
          </ContextField>
          <ContextField name={`actions.${idx}.first_last_piece_prio`}>
            <SwitchGroup
              label="Download first and last pieces first"
              description="Add torrent and download first and last pieces first"
              className="pt-6 sm:pt-10"
            />
          </ContextField>
        </FilterHalfRow>
      </CollapsibleSection>

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
