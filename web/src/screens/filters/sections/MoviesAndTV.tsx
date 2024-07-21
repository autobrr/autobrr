/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { DocsLink } from "@components/ExternalLink";
import { TextAreaAutoResize } from "@components/inputs/input";
import { MultiSelect, SwitchGroup, TextField } from "@components/inputs";

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
      <TextField
        name="seasons"
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
      <TextField
        name="episodes"
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
      <p className="col-span-12 -mb-1 text-sm font-bold text-gray-800 dark:text-gray-100 tracking-wide">Daily Shows</p>
      <TextField
        name="years"
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
      <TextField
        name="months"
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
      <TextField
        name="days"
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
      <div className="col-span-12 sm:col-span-6">
        <SwitchGroup
          name="smart_episode"
          label="Smart Episode"
          description="Do not match episodes older than the last one matched."
        />
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
      <MultiSelect
        name="resolutions"
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
      <MultiSelect
        name="sources"
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
    </FilterLayout>

    <FilterLayout gap={FilterWideGridGapClass}>
      <MultiSelect
        name="codecs"
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
      <MultiSelect
        name="containers"
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
    </FilterLayout>

    <FilterLayout gap={FilterWideGridGapClass}>
      <MultiSelect
        name="match_hdr"
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
      <MultiSelect
        name="except_hdr"
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
    </FilterLayout>

    <FilterLayout gap={FilterWideGridGapClass}>
      <MultiSelect
        name="match_other"
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
      <MultiSelect
        name="except_other"
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
    </FilterLayout>
  </FilterSection>
);

export const MoviesTv = () => (
  <FilterPage>
    <FilterSection>
      <FilterLayout>
        <TextAreaAutoResize
          name="shows"
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
        <TextField
          name="years"
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
      </FilterLayout>
    </FilterSection>

    <SeasonsAndEpisodes />
    <Quality />
  </FilterPage>
);
