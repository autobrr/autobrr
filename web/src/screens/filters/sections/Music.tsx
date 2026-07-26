/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useFormikContext } from "formik";
import { useTranslation } from "react-i18next";

import { DocsLink } from "@components/ExternalLink";
import { FilterLayout, FilterPage, FilterRow, FilterSection } from "./_components";
import { MultiSelect, NumberField, SwitchGroup, TextAreaAutoResize, TextField } from "@components/inputs";

import * as CONSTS from "@domain/constants";


export const Music = () => {
  const { t } = useTranslation("filters");
  const { values } = useFormikContext<Filter>();

  return (
    <FilterPage>
      <FilterSection>
        <FilterLayout>
          <TextAreaAutoResize
            name="artists"
            label={t("music.artists")}
            columns={6}
            placeholder={t("music.artistsPlaceholder")}
            tooltip={
              <div>
                <p>{t("music.wildcardTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters#music" />
              </div>
            }
          />
          <TextAreaAutoResize
            name="albums"
            label={t("music.albums")}
            columns={6}
            placeholder={t("music.albumsPlaceholder")}
            tooltip={
              <div>
                <p>{t("music.wildcardTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters#music" />
              </div>
            }
          />
          <TextAreaAutoResize
            name="match_record_labels"
            label={t("music.matchRecordLabels")}
            columns={6}
            placeholder={t("music.matchRecordLabelsPlaceholder")}
            tooltip={
              <div>
                <p>{t("music.matchRecordLabelsTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters#music" />
              </div>
            }
          />
          <TextAreaAutoResize
            name="except_record_labels"
            label={t("music.exceptRecordLabels")}
            columns={6}
            placeholder={t("music.exceptRecordLabelsPlaceholder")}
            tooltip={
              <div>
                <p>{t("music.exceptRecordLabelsTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters#music" />
              </div>
            }
          />
        </FilterLayout>
      </FilterSection>

      <FilterSection
        title={t("music.releaseDetails.title")}
        subtitle={t("music.releaseDetails.subtitle")}
      >
        <FilterLayout>
          <MultiSelect
            name="match_release_types"
            options={CONSTS.RELEASE_TYPE_MUSIC_OPTIONS}
            label={t("music.releaseDetails.musicType")}
            columns={6}
            tooltip={
              <div>
                <p>{t("music.releaseDetails.musicTypeTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters/music#quality" />
              </div>
            }
          />
          <TextField
            name="years"
            label={t("music.releaseDetails.years")}
            columns={6}
            placeholder={t("music.releaseDetails.yearsPlaceholder")}
            tooltip={
              <div>
                <p>{t("music.releaseDetails.yearsTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters#music" />
              </div>
            }
          />
        </FilterLayout>
      </FilterSection>

      <FilterSection
        title={t("music.quality.title")}
        subtitle={t("music.quality.subtitle")}
      >
        <FilterLayout>
          <FilterLayout>
            <MultiSelect
              name="formats"
              options={CONSTS.FORMATS_OPTIONS}
              label={t("music.quality.format")}
              columns={4}
              disabled={values.perfect_flac}
              tooltip={
                <div>
                  <p>{t("music.quality.formatTooltip")}</p>
                  <DocsLink href="https://autobrr.com/filters/music#quality" />
                </div>
              }
            />
            <MultiSelect
              name="quality"
              options={CONSTS.QUALITY_MUSIC_OPTIONS}
              label={t("music.quality.quality")}
              columns={4}
              disabled={values.perfect_flac}
              tooltip={
                <div>
                  <p>{t("music.quality.qualityTooltip")}</p>
                  <DocsLink href="https://autobrr.com/filters/music#quality" />
                </div>
              }
            />
            <MultiSelect
              name="media"
              options={CONSTS.SOURCES_MUSIC_OPTIONS}
              label={t("music.quality.media")}
              columns={4}
              disabled={values.perfect_flac}
              tooltip={
                <div>
                  <p>{t("music.quality.mediaTooltip")}</p>
                  <DocsLink href="https://autobrr.com/filters/music#quality" />
                </div>
              }
            />
          </FilterLayout>

          <FilterLayout className="items-end sm:gap-x-6!">
            <FilterRow className="sm:col-span-4">
              <SwitchGroup
                name="cue"
                label={t("music.quality.cue")}
                description={t("music.quality.cueDescription")}
                disabled={values.perfect_flac}
                className="sm:col-span-4"
              />
            </FilterRow>

            <FilterRow className="sm:col-span-4">
              <SwitchGroup
                name="log"
                label={t("music.quality.log")}
                description={t("music.quality.logDescription")}
                disabled={values.perfect_flac}
                className="sm:col-span-4"
              />
            </FilterRow>

            <FilterRow className="sm:col-span-4">
              <NumberField
                name="log_score"
                label={t("music.quality.logScore")}
                placeholder={t("music.quality.logScorePlaceholder")}
                min={0}
                max={100}
                disabled={values.perfect_flac || !values.log}
                tooltip={
                  <div>
                    <p>{t("music.quality.logScoreTooltip")}</p>
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
            {t("music.quality.or")}
          </span>
          <span className="border-b border-gray-150 dark:border-gray-750 w-full" />
        </div>

        <FilterLayout className="sm:gap-x-6!">
          <SwitchGroup
            name="perfect_flac"
            label={t("music.quality.perfectFlac")}
            description={t("music.quality.perfectFlacDescription")}
            className="py-2 col-span-12 sm:col-span-6"
            tooltip={
              <div>
                <p>{t("music.quality.perfectFlacTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters/music#quality" />
              </div>
            }
          />

          <span className="col-span-12 sm:col-span-6 self-center ml-0 text-center sm:text-left text-sm text-gray-500 dark:text-gray-425 underline underline-offset-2">
            {t("music.quality.perfectFlacHint")}
          </span>
        </FilterLayout>
      </FilterSection>
    </FilterPage>
  );
}
