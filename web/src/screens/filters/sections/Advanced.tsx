/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useTranslation } from "react-i18next";
import { PlusIcon, XMarkIcon } from "@heroicons/react/24/outline";

import { useFormContext, useFormValue } from "@hooks/form";
import { DocsLink } from "@components/ExternalLink";
import { WarningAlert } from "@components/alerts";
import {
  CollapsibleSection,
  FilterHalfRow,
  FilterLayout,
  FilterLayoutClass,
  FilterTightGridGapClass
} from "./_components";
import { classNames } from "@utils";

import * as CONSTS from "@domain/constants";
import {
  MultiSelect, NumberField, RegexField,
  RegexTextAreaField,
  Select,
  SwitchGroup,
  TextAreaAutoResize,
  TextField
} from "@components/inputs";

const Releases = () => {
  const { t } = useTranslation("filters");
  const values = useFormValue((v: Filter) => ({ except_releases: v.except_releases, match_releases: v.match_releases, use_regex: v.use_regex }));

  return (
    <CollapsibleSection
      defaultOpen={values.use_regex || values.match_releases !== undefined || values.except_releases !== undefined}
      title={t("advanced.releases.title")}
      subtitle={t("advanced.releases.subtitle")}
    >
      <FilterLayout>
        <FilterHalfRow>
          <SwitchGroup name="use_regex" label={t("advanced.common.useRegex")} className="pt-2" />
        </FilterHalfRow>
      </FilterLayout>

      <FilterLayout>
        <FilterHalfRow>
          <RegexTextAreaField
            name="match_releases"
            label={t("advanced.releases.match")}
            useRegex={values.use_regex}
            columns={6}
            placeholder={t("advanced.releases.matchPlaceholder")}
            tooltip={
              <div>
                <p>{t("advanced.common.regexTooltip1")}</p>
                <DocsLink href="https://autobrr.com/filters#advanced" />
                <br />
                <br />
                <p>{t("advanced.common.regexTooltip2")}</p>
                <br />
                <p>{t("advanced.common.regexTooltip3")}</p>
              </div>
            }
          />
        </FilterHalfRow>

        <FilterHalfRow>
          <RegexTextAreaField
            name="except_releases"
            label={t("advanced.releases.except")}
            useRegex={values.use_regex}
            columns={6}
            placeholder={t("advanced.releases.exceptPlaceholder")}
            tooltip={
              <div>
                <p>{t("advanced.common.regexTooltip1")}</p>
                <DocsLink href="https://autobrr.com/filters#advanced" />
                <br />
                <br />
                <p>{t("advanced.common.regexTooltip2")}</p>
                <br />
                <p>{t("advanced.common.regexTooltip3")}</p>
              </div>
            }
          />
        </FilterHalfRow>

      </FilterLayout>

      {values.match_releases ? (
        <WarningAlert
          alert={t("advanced.releases.alert")}
          text={
            <>
              {t("advanced.releases.matchWarning")}
            </>
          }
          colors="text-cyan-700 bg-cyan-100 dark:bg-cyan-200 dark:text-cyan-800"
        />
      ) : null}
      {values.except_releases ? (
        <WarningAlert
          alert={t("advanced.releases.alert")}
          text={
            <>
              {t("advanced.releases.exceptWarning")}
            </>
          }
          colors="text-fuchsia-700 bg-fuchsia-100 dark:bg-fuchsia-200 dark:text-fuchsia-800"
        />
      ) : null}
    </CollapsibleSection>
  );
}

const Groups = () => {
  const { t } = useTranslation("filters");
  const values = useFormValue((v: Filter) => ({ except_release_groups: v.except_release_groups, match_release_groups: v.match_release_groups }));

  return (
    <CollapsibleSection
      defaultOpen={values.match_release_groups !== undefined || values.except_release_groups !== undefined}
      title={t("advanced.groups.title")}
      subtitle={t("advanced.groups.subtitle")}
    >
      <TextAreaAutoResize
        name="match_release_groups"
        label={t("advanced.groups.match")}
        columns={6}
        placeholder={t("advanced.groups.matchPlaceholder")}
        tooltip={
          <div>
            <p>{t("advanced.groups.matchTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters#advanced" />
          </div>
        }
      />
      <TextAreaAutoResize
        name="except_release_groups"
        label={t("advanced.groups.except")}
        columns={6}
        placeholder={t("advanced.groups.exceptPlaceholder")}
        tooltip={
          <div>
            <p>{t("advanced.groups.exceptTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters#advanced" />
          </div>
        }
      />
    </CollapsibleSection>
  );
}

const Categories = () => {
  const { t } = useTranslation("filters");
  const values = useFormValue((v: Filter) => ({ except_categories: v.except_categories, match_categories: v.match_categories }));

  return (
    <CollapsibleSection
      defaultOpen={values.match_categories !== undefined || values.except_categories !== undefined}
      title={t("advanced.categories.title")}
      subtitle={t("advanced.categories.subtitle")}
    >
      <TextAreaAutoResize
        name="match_categories"
        label={t("advanced.categories.match")}
        columns={6}
        placeholder={t("advanced.categories.matchPlaceholder")}
        tooltip={
          <div>
            <p>{t("advanced.categories.matchTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters/categories" />
          </div>
        }
      />
      <TextAreaAutoResize
        name="except_categories"
        label={t("advanced.categories.except")}
        columns={6}
        placeholder={t("advanced.categories.exceptPlaceholder")}
        tooltip={
          <div>
            <p>{t("advanced.categories.exceptTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters/categories" />
          </div>
        }
      />
    </CollapsibleSection>
  );
}

const Tags = () => {
  const { t } = useTranslation("filters");
  const values = useFormValue((v: Filter) => ({ except_tags: v.except_tags, tags: v.tags }));

  return (
    <CollapsibleSection
      defaultOpen={values.tags !== undefined || values.except_tags !== undefined}
      title={t("advanced.tags.title")}
      subtitle={t("advanced.tags.subtitle")}
    >
      <div className={classNames("sm:col-span-6", FilterLayoutClass, FilterTightGridGapClass)}>
        <TextAreaAutoResize
          name="tags"
          label={t("advanced.tags.match")}
          columns={8}
          placeholder={t("advanced.tags.matchPlaceholder")}
          tooltip={
            <div>
              <p>{t("advanced.tags.matchTooltip")}</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
            </div>
          }
        />
        <Select
          name="tags_match_logic"
          label={t("advanced.tags.matchLogic")}
          columns={4}
          options={CONSTS.tagsMatchLogicOptions}
          optionDefaultText={t("advanced.tags.matchLogicDefault")}
          tooltip={
            <div>
              <p>{t("advanced.tags.matchLogicTooltip")}</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
            </div>
          }
        />
      </div>
      <div className={classNames("sm:col-span-6", FilterLayoutClass, FilterTightGridGapClass)}>
        <TextAreaAutoResize
          name="except_tags"
          label={t("advanced.tags.except")}
          columns={8}
          placeholder={t("advanced.tags.exceptPlaceholder")}
          tooltip={
            <div>
              <p>{t("advanced.tags.exceptTooltip")}</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
            </div>
          }
        />
        <Select
          name="except_tags_match_logic"
          label={t("advanced.tags.exceptLogic")}
          columns={4}
          options={CONSTS.tagsMatchLogicOptions}
          optionDefaultText={t("advanced.tags.exceptLogicDefault")}
          tooltip={
            <div>
              <p>{t("advanced.tags.exceptLogicTooltip")}</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
            </div>
          }
        />
      </div>
    </CollapsibleSection>
  );
}

const Uploaders = () => {
  const { t } = useTranslation("filters");
  const values = useFormValue((v: Filter) => ({ except_uploaders: v.except_uploaders, match_uploaders: v.match_uploaders }));

  return (
    <CollapsibleSection
      defaultOpen={values.match_uploaders !== undefined || values.except_uploaders !== undefined}
      title={t("advanced.uploaders.title")}
      subtitle={t("advanced.uploaders.subtitle")}
    >
      <TextAreaAutoResize
        name="match_uploaders"
        label={t("advanced.uploaders.match")}
        columns={6}
        placeholder={t("advanced.uploaders.matchPlaceholder")}
        tooltip={
          <div>
            <p>{t("advanced.uploaders.matchTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters#advanced" />
          </div>
        }
      />
      <TextAreaAutoResize
        name="except_uploaders"
        label={t("advanced.uploaders.except")}
        columns={6}
        placeholder={t("advanced.uploaders.exceptPlaceholder")}
        tooltip={
          <div>
            <p>{t("advanced.uploaders.exceptTooltip")}</p>
            <DocsLink href="https://autobrr.com/filters#advanced" />
          </div>
        }
      />
    </CollapsibleSection>
  );
}

const Language = () => {
  const { t } = useTranslation("filters");
  const values = useFormValue((v: Filter) => ({ except_language: v.except_language, match_language: v.match_language }));

  return (
    <CollapsibleSection
      defaultOpen={values.match_language?.length > 0 || values.except_language?.length > 0}
      title={t("advanced.language.title")}
      subtitle={t("advanced.language.subtitle")}
    >
      <MultiSelect
        name="match_language"
        options={CONSTS.LANGUAGE_OPTIONS}
        label={t("advanced.language.match")}
        columns={6}
      />
      <MultiSelect
        name="except_language"
        options={CONSTS.LANGUAGE_OPTIONS}
        label={t("advanced.language.except")}
        columns={6}
      />
    </CollapsibleSection>
  );
}

const Origins = () => {
  const { t } = useTranslation("filters");
  const values = useFormValue((v: Filter) => ({ except_origins: v.except_origins, origins: v.origins }));

  return (
    <CollapsibleSection
      defaultOpen={values.origins?.length > 0 || values.except_origins?.length > 0}
      title={t("advanced.origins.title")}
      subtitle={t("advanced.origins.subtitle")}
    >
      <MultiSelect
        name="origins"
        options={CONSTS.ORIGIN_OPTIONS}
        label={t("advanced.origins.match")}
        columns={6}
      />
      <MultiSelect
        name="except_origins"
        options={CONSTS.ORIGIN_OPTIONS}
        label={t("advanced.origins.except")}
        columns={6}
      />
    </CollapsibleSection>
  );
}

const Freeleech = () => {
  const { t } = useTranslation("filters");
  const values = useFormValue((v: Filter) => ({ freeleech: v.freeleech, freeleech_percent: v.freeleech_percent }));

  return (
    <CollapsibleSection
      defaultOpen={values.freeleech || values.freeleech_percent !== undefined}
      title={t("advanced.freeleech.title")}
      subtitle={t("advanced.freeleech.subtitle")}
    >
      <TextField
        name="freeleech_percent"
        label={t("advanced.freeleech.percent")}
        disabled={values.freeleech}
        tooltip={
          <div>
            <p>{t("advanced.freeleech.percentTooltip1")}</p>
            <br />
            <p>{t("advanced.freeleech.percentTooltip2")}</p>
            <br />
            <p>
              {t("advanced.freeleech.percentTooltip3")}{" "}
              <DocsLink href="https://autobrr.com/filters/freeleech" />
            </p>
          </div>
        }
        columns={6}
        placeholder={t("advanced.freeleech.percentPlaceholder")}
      />
      <FilterHalfRow>
        <SwitchGroup
          name="freeleech"
          label={t("advanced.freeleech.toggle")}
          className="py-0"
          description={t("advanced.freeleech.toggleDescription")}
          tooltip={
            <div>
              <p>{t("advanced.freeleech.toggleTooltip1")}</p>
              <br />
              <p>{t("advanced.freeleech.toggleTooltip2")}</p>
              <br />
              <p>
                {t("advanced.freeleech.toggleTooltip3")}{" "}
                <DocsLink href="https://autobrr.com/filters/freeleech" />
              </p>
            </div>
          }
        />
      </FilterHalfRow>
    </CollapsibleSection>
  );
}

const FeedSpecific = () => {
  const { t } = useTranslation("filters");
  const values = useFormValue((v: Filter) => ({ except_description: v.except_description, match_description: v.match_description, max_leechers: v.max_leechers, max_seeders: v.max_seeders, min_leechers: v.min_leechers, min_seeders: v.min_seeders, use_regex_description: v.use_regex_description }));

  return (
    <CollapsibleSection
      defaultOpen={
        values.use_regex_description ||
        values.match_description !== undefined ||
        values.except_description !== undefined ||
        values.min_seeders !== undefined ||
        values.max_seeders !== undefined ||
        values.min_leechers !== undefined ||
        values.max_leechers !== undefined
      }
      title={t("advanced.feedSpecific.title")}
      subtitle={
        <>{t("advanced.feedSpecific.subtitle")}</>
      }
    >
      <FilterLayout>
        <SwitchGroup
          name="use_regex_description"
          label={t("advanced.common.useRegex")}
          className="col-span-12 sm:col-span-6"
        />
      </FilterLayout>

      <RegexTextAreaField
        name="match_description"
        label={t("advanced.feedSpecific.matchDescription")}
        useRegex={values.use_regex_description}
        columns={6}
        placeholder={t("advanced.feedSpecific.matchDescriptionPlaceholder")}
        tooltip={
          <div>
            <p>{t("advanced.common.regexTooltip1")}</p>
            <DocsLink href="https://autobrr.com/filters#advanced" />
            <br />
            <br />
            <p>{t("advanced.common.regexTooltip2")}</p>
            <br />
            <p>{t("advanced.common.regexTooltip3")}</p>
          </div>
        }
      />
      <RegexTextAreaField
        name="except_description"
        label={t("advanced.feedSpecific.exceptDescription")}
        useRegex={values.use_regex_description}
        columns={6}
        placeholder={t("advanced.feedSpecific.exceptDescriptionPlaceholder")}
        tooltip={
          <div>
            <p>{t("advanced.common.regexTooltip1")}</p>
            <DocsLink href="https://autobrr.com/filters#advanced" />
            <br />
            <br />
            <p>{t("advanced.common.regexTooltip2")}</p>
            <br />
            <p>{t("advanced.common.regexTooltip3")}</p>
          </div>
        }
      />
      <NumberField
        name="min_seeders"
        label={t("advanced.feedSpecific.minSeeders")}
        placeholder={t("advanced.feedSpecific.countPlaceholder")}
        tooltip={
          <div>
            <p>{t("advanced.feedSpecific.seedersTooltip", { type: t("advanced.feedSpecific.minSeeders").toLowerCase() })}</p>
            <DocsLink href="https://autobrr.com/filters#rules" />
          </div>
        }
      />
      <NumberField
        name="max_seeders"
        label={t("advanced.feedSpecific.maxSeeders")}
        placeholder={t("advanced.feedSpecific.countPlaceholder")}
        tooltip={
          <div>
            <p>{t("advanced.feedSpecific.seedersTooltip", { type: t("advanced.feedSpecific.maxSeeders").toLowerCase() })}</p>
            <DocsLink href="https://autobrr.com/filters#rules" />
          </div>
        }
      />
      <NumberField
        name="min_leechers"
        label={t("advanced.feedSpecific.minLeechers")}
        placeholder={t("advanced.feedSpecific.countPlaceholder")}
        tooltip={
          <div>
            <p>{t("advanced.feedSpecific.seedersTooltip", { type: t("advanced.feedSpecific.minLeechers").toLowerCase() })}</p>
            <DocsLink href="https://autobrr.com/filters#rules" />
          </div>
        }
      />
      <NumberField
        name="max_leechers"
        label={t("advanced.feedSpecific.maxLeechers")}
        placeholder={t("advanced.feedSpecific.countPlaceholder")}
        tooltip={
          <div>
            <p>{t("advanced.feedSpecific.seedersTooltip", { type: t("advanced.feedSpecific.maxLeechers").toLowerCase() })}</p>
            <DocsLink href="https://autobrr.com/filters#rules" />
          </div>
        }
      />
    </CollapsibleSection>
  );
}

const CustomFields = () => {
  const { t } = useTranslation("filters");
  const form = useFormContext();
  const values = useFormValue((v: Filter) => ({
    custom_fields: v.custom_fields || [],
    custom_fields_match_logic: v.custom_fields_match_logic || "ALL"
  }));

  const operatorOptions = ["=", "!=", "<", ">", "<=", ">="].map((operator) => ({
    label: operator,
    value: operator
  }));

  const logicOptions = [
    { label: t("advanced.customFields.logic.all"), value: "ALL" },
    { label: t("advanced.customFields.logic.any"), value: "ANY" }
  ];

  const addRule = (field = "") => {
    form.pushFieldValue("custom_fields", {
      field,
      operator: "=",
      value: ""
    } satisfies FilterCustomFieldRule);
  };

  return (
    <CollapsibleSection
      defaultOpen={values.custom_fields.length > 0}
      title={t("advanced.customFields.title")}
      subtitle={t("advanced.customFields.subtitle")}
    >
      <div className="sm:col-span-12">
        <p className="mb-4 text-sm text-gray-500 dark:text-gray-400">
          {t("advanced.customFields.description")}
        </p>
      </div>


      <FilterLayout>
        <Select
          name="custom_fields_match_logic"
          label={t("advanced.customFields.matchLogic")}
          optionDefaultText={t("advanced.customFields.matchLogicDefault")}
          options={logicOptions}
          columns={4}
        />
        <div className="col-span-12 sm:col-span-8 flex items-end justify-end">
          <button
            type="button"
            className="inline-flex items-center px-3 py-2 rounded-md text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-hidden cursor-pointer"
            onClick={() => addRule()}
          >
            <PlusIcon className="w-4 h-4 mr-1" aria-hidden="true" />
            {t("advanced.customFields.add")}
          </button>
        </div>
      </FilterLayout>

      {values.custom_fields.map((_rule, index) => {
        return (
          <div
            key={index}
            className={classNames("sm:col-span-12", FilterLayoutClass, FilterTightGridGapClass)}
          >
            <TextField
              name={`custom_fields[${index}].field`}
              label={t("advanced.customFields.field")}
              placeholder={t("advanced.customFields.fieldPlaceholder")}
              columns={4}
            />
            <Select
              name={`custom_fields[${index}].operator`}
              label={t("advanced.customFields.operator")}
              optionDefaultText={t("advanced.customFields.operatorDefault")}
              options={operatorOptions}
              columns={3}
            />
            <TextField
              name={`custom_fields[${index}].value`}
              label={t("advanced.customFields.value")}
              placeholder={t("advanced.customFields.valuePlaceholder")}
              columns={4}
            />
            <div className="col-span-12 sm:col-span-1 flex items-end">
              <button
                type="button"
                className="h-10 w-full inline-flex items-center justify-center rounded-md border border-red-300 dark:border-red-800 text-red-700 dark:text-red-300 hover:bg-red-50 dark:hover:bg-red-950/30 cursor-pointer"
                aria-label={t("advanced.customFields.remove")}
                onClick={() => form.removeFieldValue("custom_fields", index)}
              >
                <XMarkIcon className="w-5 h-5" aria-hidden="true" />
              </button>
            </div>
          </div>
        );
      })}
    </CollapsibleSection>
  );
}

const RawReleaseTags = () => {
  const { t } = useTranslation("filters");
  const values = useFormValue((v: Filter) => ({ except_release_tags: v.except_release_tags, match_release_tags: v.match_release_tags, use_regex_release_tags: v.use_regex_release_tags }));

  return (
    <CollapsibleSection
      defaultOpen={values.use_regex_release_tags || values.match_release_tags !== undefined || values.except_release_tags !== undefined}
      title={t("advanced.rawReleaseTags.title")}
      subtitle={
        <>
          <span className="underline underline-offset-2">{t("advanced.rawReleaseTags.subtitlePrefix")}</span>
          {": "}{t("advanced.rawReleaseTags.subtitleSuffix")}
        </>
      }
    >
      <WarningAlert
        text={
          <>{t("advanced.rawReleaseTags.warning")}</>
        }
      />

      <FilterLayout>
        <SwitchGroup
          name="use_regex_release_tags"
          label={t("advanced.common.useRegex")}
          className="col-span-12 sm:col-span-6"
        />
      </FilterLayout>

      <RegexField
        name="match_release_tags"
        label={t("advanced.rawReleaseTags.match")}
        useRegex={values.use_regex_release_tags}
        columns={6}
        placeholder={t("advanced.rawReleaseTags.placeholder")}
        tooltip={
          <div>
            <p>{t("advanced.common.regexTooltip1")}</p>
            <DocsLink href="https://autobrr.com/filters#advanced" />
            <br />
            <br />
            <p>{t("advanced.common.regexTooltip2")}</p>
            <br />
            <p>{t("advanced.common.regexTooltip3")}</p>
          </div>
        }
      />
      <RegexField
        name="except_release_tags"
        label={t("advanced.rawReleaseTags.except")}
        useRegex={values.use_regex_release_tags}
        columns={6}
        placeholder={t("advanced.rawReleaseTags.placeholder")}
        tooltip={
          <div>
            <p>{t("advanced.common.regexTooltip1")}</p>
            <DocsLink href="https://autobrr.com/filters#advanced" />
            <br />
            <br />
            <p>{t("advanced.common.regexTooltip2")}</p>
            <br />
            <p>{t("advanced.common.regexTooltip3")}</p>
          </div>
        }
      />
    </CollapsibleSection>
  );
}

export const Advanced = () => {
  return (
    <div className="flex flex-col w-full gap-y-4 py-2 sm:-mx-1">
      <Releases />
      <Groups />
      <Categories />
      <Freeleech />
      <Tags />
      <Uploaders />
      <Language />
      <Origins />
      <FeedSpecific />
      <CustomFields />
      <RawReleaseTags />
    </div>
  );
}
