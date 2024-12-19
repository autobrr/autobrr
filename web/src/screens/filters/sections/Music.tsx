/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useFormikContext } from "formik";

import { DocsLink } from "@components/ExternalLink";
import { FilterLayout, FilterPage, FilterRow, FilterSection } from "./_components";
import { MultiSelect, NumberField, SwitchGroup, TextAreaAutoResize, TextField } from "@components/inputs";

import * as CONSTS from "@domain/constants";


export const Music = () => {
  const { values } = useFormikContext<Filter>();

  return (
    <FilterPage>
      <FilterSection>
        <FilterLayout>
          <TextAreaAutoResize
            name="artists"
            label="Artists"
            columns={6}
            placeholder="eg. Artist One"
            tooltip={
              <div>
                <p>You can use basic filtering like wildcards <code>*</code> or replace single characters with <code>?</code></p>
                <DocsLink href="https://autobrr.com/filters#music" />
              </div>
            }
          />
          <TextAreaAutoResize
            name="albums"
            label="Albums"
            columns={6}
            placeholder="eg. That Album"
            tooltip={
              <div>
                <p>You can use basic filtering like wildcards <code>*</code> or replace single characters with <code>?</code></p>
                <DocsLink href="https://autobrr.com/filters#music" />
              </div>
            }
          />
          <TextAreaAutoResize
            name="match_record_labels"
            label="Match record labels"
            columns={6}
            placeholder="eg. Anjunabeats, Armada"
            tooltip={
              <div>
                <p>Comma separated list of record labels to match. Only Orpheus and Redacted support this.</p>
                <DocsLink href="https://autobrr.com/filters#music" />
              </div>
            }
          />
          <TextAreaAutoResize
            name="except_record_labels"
            label="Except record labels"
            columns={6}
            placeholder="eg. Anjunadeep, Armind"
            tooltip={
              <div>
                <p>Comma separated list of record labels to ignore (takes priority over Match record labels). Only Orpheus and Redacted support this.</p>
                <DocsLink href="https://autobrr.com/filters#music" />
              </div>
            }
          />
        </FilterLayout>
      </FilterSection>

      <FilterSection
        title="Release details"
        subtitle="Type (Album, Single, EP, etc.) and year of release (if announced)"
      >
        <FilterLayout>
          <MultiSelect
            name="match_release_types"
            options={CONSTS.RELEASE_TYPE_MUSIC_OPTIONS}
            label="Music Type"
            columns={6}
            tooltip={
              <div>
                <p>Will only match releases with any of the selected types.</p>
                <DocsLink href="https://autobrr.com/filters/music#quality" />
              </div>
            }
          />
          <TextField
            name="years"
            label="Years"
            columns={6}
            placeholder="eg. 2018,2019-2021"
            tooltip={
              <div>
                <p>This field takes a range of years and/or comma separated single years.</p>
                <DocsLink href="https://autobrr.com/filters#music" />
              </div>
            }
          />
        </FilterLayout>
      </FilterSection>

      <FilterSection
        title="Quality"
        subtitle="Format, source, log, etc."
      >
        <FilterLayout>
          <FilterLayout>
            <MultiSelect
              name="formats"
              options={CONSTS.FORMATS_OPTIONS}
              label="Format"
              columns={4}
              disabled={values.perfect_flac}
              tooltip={
                <div>
                  <p>Will only match releases with any of the selected formats. This is overridden by Perfect FLAC.</p>
                  <DocsLink href="https://autobrr.com/filters/music#quality" />
                </div>
              }
            />
            <MultiSelect
              name="quality"
              options={CONSTS.QUALITY_MUSIC_OPTIONS}
              label="Quality"
              columns={4}
              disabled={values.perfect_flac}
              tooltip={
                <div>
                  <p>Will only match releases with any of the selected qualities. This is overridden by Perfect FLAC.</p>
                  <DocsLink href="https://autobrr.com/filters/music#quality" />
                </div>
              }
            />
            <MultiSelect
              name="media"
              options={CONSTS.SOURCES_MUSIC_OPTIONS}
              label="Media"
              columns={4}
              disabled={values.perfect_flac}
              tooltip={
                <div>
                  <p>Will only match releases with any of the selected sources. This is overridden by Perfect FLAC.</p>
                  <DocsLink href="https://autobrr.com/filters/music#quality" />
                </div>
              }
            />
          </FilterLayout>

          <FilterLayout className="items-end sm:!gap-x-6">
            <FilterRow className="sm:col-span-4">
              <SwitchGroup
                name="cue"
                label="Cue"
                description="Must include CUE info"
                disabled={values.perfect_flac}
                className="sm:col-span-4"
              />
            </FilterRow>

            <FilterRow className="sm:col-span-4">
              <SwitchGroup
                name="log"
                label="Log"
                description="Must include LOG info"
                disabled={values.perfect_flac}
                className="sm:col-span-4"
              />
            </FilterRow>

            <FilterRow className="sm:col-span-4">
              <NumberField
                name="log_score"
                label="Log score"
                placeholder="eg. 100"
                min={0}
                max={100}
                disabled={values.perfect_flac || !values.log}
                tooltip={
                  <div>
                    <p>Log scores go from 0 to 100. This is overridden by Perfect FLAC.</p>
                    <DocsLink href="https://autobrr.com/filters/music#quality" />
                  </div>
                }
              />
            </FilterRow>
          </FilterLayout>
        </FilterLayout>

        <div className="col-span-12 flex items-center justify-center">
          <span className="border-b border-gray-150 dark:border-gray-750 w-full" />
          <span className="flex mx-2 shrink-0 text-lg font-bold uppercase tracking-wide text-gray-700 dark:text-gray-200">
            OR
          </span>
          <span className="border-b border-gray-150 dark:border-gray-750 w-full" />
        </div>

        <FilterLayout className="sm:!gap-x-6">
          <SwitchGroup
            name="perfect_flac"
            label="Perfect FLAC"
            description="Override all options about quality, source, format, and cue/log/log score."
            className="py-2 col-span-12 sm:col-span-6"
            tooltip={
              <div>
                <p>Override all options about quality, source, format, and CUE/LOG/LOG score.</p>
                <DocsLink href="https://autobrr.com/filters/music#quality" />
              </div>
            }
          />

          <span className="col-span-12 sm:col-span-6 self-center ml-0 text-center sm:text-left text-sm text-gray-500 dark:text-gray-425 underline underline-offset-2">
            This is what you want in 90% of cases (instead of options above).
          </span>
        </FilterLayout>
      </FilterSection>
    </FilterPage>
  );
}
