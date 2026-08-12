/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useTranslation } from "react-i18next";

import { DocsLink } from "@components/ExternalLink";
import { FilterLayout, FilterPage, FilterSection } from "./_components";
import { MultiSelect, TextAreaAutoResize, TextField } from "@components/inputs";

import * as CONSTS from "@domain/constants";


export const Books = () => {
  const { t } = useTranslation("filters");

  return (
    <FilterPage>
      <FilterSection>
        <FilterLayout>
          <TextAreaAutoResize
            name="artists"
            label={t("books.authors")}
            columns={6}
            placeholder={t("books.authorsPlaceholder")}
            tooltip={
              <div>
                <p>{t("books.wildcardTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters" />
              </div>
            }
          />
          <TextAreaAutoResize
            name="albums"
            label={t("books.titles")}
            columns={6}
            placeholder={t("books.titlesPlaceholder")}
            tooltip={
              <div>
                <p>{t("books.wildcardTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters" />
              </div>
            }
          />
          <TextField
            name="years"
            label={t("books.years")}
            columns={6}
            placeholder={t("books.yearsPlaceholder")}
            tooltip={
              <div>
                <p>{t("books.yearsTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters" />
              </div>
            }
          />
        </FilterLayout>
      </FilterSection>

      <FilterSection
        title={t("books.options.title")}
        subtitle={t("books.options.subtitle")}
      >
        <FilterLayout>
          <MultiSelect
            name="match_language"
            options={CONSTS.LANGUAGE_OPTIONS}
            label={t("books.options.matchLanguage")}
            columns={6}
          />
          <MultiSelect
            name="except_language"
            options={CONSTS.LANGUAGE_OPTIONS}
            label={t("books.options.exceptLanguage")}
            columns={6}
          />
          <MultiSelect
            name="containers"
            options={CONSTS.FORMATS_BOOKS_OPTIONS}
            label={t("books.options.format")}
            columns={12}
            tooltip={
              <div>
                <p>{t("books.options.formatTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters" />
              </div>
            }
          />
          <TextAreaAutoResize
            name="tags"
            label={t("books.options.matchTags")}
            columns={6}
            placeholder={t("books.options.matchTagsPlaceholder")}
            tooltip={
              <div>
                <p>{t("books.options.matchTagsTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters#advanced" />
              </div>
            }
          />
          <TextAreaAutoResize
            name="except_tags"
            label={t("books.options.exceptTags")}
            columns={6}
            placeholder={t("books.options.exceptTagsPlaceholder")}
            tooltip={
              <div>
                <p>{t("books.options.exceptTagsTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters#advanced" />
              </div>
            }
          />
        </FilterLayout>
      </FilterSection>
    </FilterPage>
  );
}
