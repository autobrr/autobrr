/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { DocsLink } from "@components/ExternalLink";
import { TextAreaAutoResize } from "@components/inputs/tanstack";
import { MultiSelect, SwitchGroup, TextField } from "@components/inputs/tanstack";
import { ContextField } from "@app/lib/form";

import * as CONSTS from "@domain/constants";
import {
  FilterLayout,
  FilterPage,
  FilterSection,
  FilterWideGridGapClass
} from "@screens/filters/sections/_components.tsx";

const SeasonsAndEpisodes = () => (
  <FilterSection
    title="Seasons, Episodes and Date"
    subtitle="Set season, episode, year, months and day match constraints."
  >
    <FilterLayout>
      <ContextField name="seasons">
        <TextField
          label="Seasons"
          columns={6}
          placeholder="eg. 1,3,2-6"
          tooltip={
            <div>
              <p>See docs for information about how to <b>only</b> grab season packs:</p>
              <DocsLink href="https://autobrr.com/filters/examples#only-season-packs" />
            </div>
          }
        />
      </ContextField>
      <ContextField name="episodes">
        <TextField
          label="Episodes"
          columns={6}
          placeholder="eg. 2,4,10-20"
          tooltip={
            <div>
              <p>See docs for information about how to <b>only</b> grab episodes:</p>
              <DocsLink href="https://autobrr.com/filters/examples#only-episodes-skip-season-packs"/>
            </div>
          }
        />
      </ContextField>
      <p className="col-span-12 -mb-1 text-sm font-bold text-gray-800 dark:text-gray-100 tracking-wide">Daily Shows</p>
      <ContextField name="years">
        <TextField
          label="Years"
          columns={4}
          placeholder="eg. 2018,2019-2021"
          tooltip={
            <div>
              <p>This field takes a range of years and/or comma separated single years.</p>
              <DocsLink href="https://autobrr.com/filters#tvmovies"/>
            </div>
          }
        />
      </ContextField>
      <ContextField name="months">
        <TextField
          label="Months"
          columns={4}
          placeholder="eg. 4,2-9"
          tooltip={
            <div>
              <p>This field takes a range of years and/or comma separated single months.</p>
              <DocsLink href="https://autobrr.com/filters#tvmovies"/>
            </div>
          }
        />
      </ContextField>
      <ContextField name="days">
        <TextField
          label="Days"
          columns={4}
          placeholder="eg. 1,15-30"
          tooltip={
            <div>
              <p>This field takes a range of years and/or comma separated single days.</p>
              <DocsLink href="https://autobrr.com/filters#tvmovies"/>
            </div>
          }
        />
      </ContextField>
      <div className="col-span-12 sm:col-span-6">
        <ContextField name="smart_episode">
          <SwitchGroup
            label="Smart Episode"
            description="Do not match episodes older than the last one matched."
          />
        </ContextField>
      </div>
    </FilterLayout>
  </FilterSection>
);

const Quality = () => (
  <FilterSection
    title="Quality"
    subtitle="Set resolution, source, codec and related match constraints."
  >
    <FilterLayout gap={FilterWideGridGapClass}>
      <ContextField name="resolutions">
        <MultiSelect
          options={CONSTS.RESOLUTION_OPTIONS}
          label="resolutions"
          columns={6}
          tooltip={
            <div>
              <p>Will match releases which contain any of the selected resolutions.</p>
              <DocsLink href="https://autobrr.com/filters#quality" />
            </div>
          }
        />
      </ContextField>
      <ContextField name="sources">
        <MultiSelect
          options={CONSTS.SOURCES_OPTIONS}
          label="sources"
          columns={6}
          tooltip={
            <div>
              <p>Will match releases which contain any of the selected sources.</p>
              <DocsLink href="https://autobrr.com/filters#quality" />
            </div>
          }
        />
      </ContextField>
    </FilterLayout>

    <FilterLayout gap={FilterWideGridGapClass}>
      <ContextField name="codecs">
        <MultiSelect
          options={CONSTS.CODECS_OPTIONS}
          label="codecs"
          columns={6}
          tooltip={
            <div>
              <p>Will match releases which contain any of the selected codecs.</p>
              <DocsLink href="https://autobrr.com/filters#quality" />
            </div>
          }
        />
      </ContextField>
      <ContextField name="containers">
        <MultiSelect
          options={CONSTS.CONTAINER_OPTIONS}
          label="containers"
          columns={6}
          tooltip={
            <div>
              <p>Will match releases which contain any of the selected containers.</p>
              <DocsLink href="https://autobrr.com/filters#quality" />
            </div>
          }
        />
      </ContextField>
    </FilterLayout>

    <FilterLayout gap={FilterWideGridGapClass}>
      <ContextField name="match_hdr">
        <MultiSelect
          options={CONSTS.HDR_OPTIONS}
          label="Match HDR"
          columns={6}
          tooltip={
            <div>
              <p>Will match releases which contain any of the selected HDR designations.</p>
              <DocsLink href="https://autobrr.com/filters#quality" />
            </div>
          }
        />
      </ContextField>
      <ContextField name="except_hdr">
        <MultiSelect
          options={CONSTS.HDR_OPTIONS}
          label="Except HDR"
          columns={6}
          tooltip={
            <div>
              <p>Won't match releases which contain any of the selected HDR designations (takes priority over Match HDR).</p>
              <DocsLink href="https://autobrr.com/filters#quality" />
            </div>
          }
        />
      </ContextField>
    </FilterLayout>

    <FilterLayout gap={FilterWideGridGapClass}>
      <ContextField name="match_other">
        <MultiSelect
          options={CONSTS.OTHER_OPTIONS}
          label="Match Other"
          columns={6}
          tooltip={
            <div>
              <p>Will match releases which contain any of the selected designations.</p>
              <DocsLink href="https://autobrr.com/filters#quality" />
            </div>
          }
        />
      </ContextField>
      <ContextField name="except_other">
        <MultiSelect
          options={CONSTS.OTHER_OPTIONS}
          label="Except Other"
          columns={6}
          tooltip={
            <div>
              <p>Won't match releases which contain any of the selected Other designations (takes priority over Match Other).</p>
              <DocsLink href="https://autobrr.com/filters#quality" />
            </div>
          }
        />
      </ContextField>
    </FilterLayout>
  </FilterSection>
);

export const MoviesTv = () => (
  <FilterPage>
    <FilterSection>
      <FilterLayout>
        <ContextField name="shows">
          <TextAreaAutoResize
            label="Movies / Shows"
            columns={8}
            placeholder="eg. Movie,Show 1,Show?2"
            tooltip={
              <div>
                <p>You can use basic filtering like wildcards <code>*</code> or replace single characters with <code>?</code></p>
                <DocsLink href="https://autobrr.com/filters#tvmovies" />
              </div>
            }
          />
        </ContextField>
        <ContextField name="years">
          <TextField
            label="Years"
            columns={4}
            placeholder="eg. 2018,2019-2021"
            tooltip={
              <div>
                <p>This field takes a range of years and/or comma separated single years.</p>
                <DocsLink href="https://autobrr.com/filters#tvmovies" />
              </div>
            }
          />
        </ContextField>
      </FilterLayout>
    </FilterSection>

    <SeasonsAndEpisodes />
    <Quality />
  </FilterPage>
);
