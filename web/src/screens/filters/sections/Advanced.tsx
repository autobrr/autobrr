/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useFormContext, useStore, ContextField } from "@app/lib/form";

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
} from "@components/inputs/tanstack";


const Releases = () => {

  const form = useFormContext();

  const use_regex = useStore(form.store, (s: any) => s.values.use_regex);
  const match_releases = useStore(form.store, (s: any) => s.values.match_releases);
  const except_releases = useStore(form.store, (s: any) => s.values.except_releases);

  return (
    <CollapsibleSection
      defaultOpen={use_regex || match_releases !== undefined || except_releases !== undefined}
      title="Release Names"
      subtitle="Match only certain release names and/or ignore other release names."
    >
      <FilterLayout>
        <FilterHalfRow>
          <ContextField name="use_regex">
            <SwitchGroup label="Use Regex" className="pt-2" />
          </ContextField>
        </FilterHalfRow>
      </FilterLayout>

      <FilterLayout>
        <FilterHalfRow>
          <ContextField name="match_releases">
            <RegexTextAreaField
              label="Match releases"
              useRegex={use_regex}
              columns={6}
              placeholder="eg. *some?movie*,*some?show*s01*"
              tooltip={
                <div>
                  <p>This field has full regex support (Golang flavour).</p>
                  <DocsLink href="https://autobrr.com/filters#advanced" />
                  <br />
                  <br />
                  <p>Remember to tick <b>Use Regex</b> if using more than <code>*</code> and <code>?</code>.</p>
                  <br />
                  <p>Mode: <code>(?i)</code> <b>case-insensitive</b></p>
                </div>
              }
            />
          </ContextField>
        </FilterHalfRow>

        <FilterHalfRow>
          <ContextField name="except_releases">
            <RegexTextAreaField
              label="Except releases"
              useRegex={use_regex}
              columns={6}
              placeholder="eg. *bad?movie*,*bad?show*s03*"
              tooltip={
                <div>
                  <p>This field has full regex support (Golang flavour).</p>
                  <DocsLink href="https://autobrr.com/filters#advanced" />
                  <br />
                  <br />
                  <p>Remember to tick <b>Use Regex</b> if using more than <code>*</code> and <code>?</code>.</p>
                  <br />
                  <p>Mode: <code>(?i)</code> <b>case-insensitive</b></p>
                </div>
              }
            />
          </ContextField>
        </FilterHalfRow>

      </FilterLayout>

      {match_releases ? (
        <WarningAlert
          alert="Ask yourself:"
          text={
            <>
              Do you have a good reason to use <strong>Match releases</strong> instead of one of the other tabs?
            </>
          }
          colors="text-cyan-700 bg-cyan-100 dark:bg-cyan-200 dark:text-cyan-800"
        />
      ) : null}
      {except_releases ? (
        <WarningAlert
          alert="Ask yourself:"
          text={
            <>
              Do you have a good reason to use <strong>Except releases</strong> instead of one of the other tabs?
            </>
          }
          colors="text-fuchsia-700 bg-fuchsia-100 dark:bg-fuchsia-200 dark:text-fuchsia-800"
        />
      ) : null}
    </CollapsibleSection>
  );
}

const Groups = () => {

  const form = useFormContext();

  const match_release_groups = useStore(form.store, (s: any) => s.values.match_release_groups);
  const except_release_groups = useStore(form.store, (s: any) => s.values.except_release_groups);

  return (
    <CollapsibleSection
      defaultOpen={match_release_groups !== undefined || except_release_groups !== undefined}
      title="Groups"
      subtitle="Match only certain groups and/or ignore other groups."
    >
      <ContextField name="match_release_groups">
        <TextAreaAutoResize
          label="Match release groups"
          columns={6}
          placeholder="eg. group1,group2"
          tooltip={
            <div>
              <p>Comma separated list of release groups to match.</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
            </div>
          }
        />
      </ContextField>
      <ContextField name="except_release_groups">
        <TextAreaAutoResize
          label="Except release groups"
          columns={6}
          placeholder="eg. badgroup1,badgroup2"
          tooltip={
            <div>
              <p>Comma separated list of release groups to ignore (takes priority over Match releases).</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
            </div>
          }
        />
      </ContextField>
    </CollapsibleSection>
  );
}

const Categories = () => {

  const form = useFormContext();

  const match_categories = useStore(form.store, (s: any) => s.values.match_categories);
  const except_categories = useStore(form.store, (s: any) => s.values.except_categories);

  return (
    <CollapsibleSection
      defaultOpen={match_categories !== undefined || except_categories !== undefined}
      title="Categories"
      subtitle="Match or exclude categories (if announced)"
    >
      <ContextField name="match_categories">
        <TextAreaAutoResize
          label="Match categories"
          columns={6}
          placeholder="eg. *category*,category1"
          tooltip={
            <div>
              <p>Comma separated list of categories to match.</p>
              <DocsLink href="https://autobrr.com/filters/categories" />
            </div>
          }
        />
      </ContextField>
      <ContextField name="except_categories">
        <TextAreaAutoResize
          label="Except categories"
          columns={6}
          placeholder="eg. *category*"
          tooltip={
            <div>
              <p>Comma separated list of categories to ignore (takes priority over Match releases).</p>
              <DocsLink href="https://autobrr.com/filters/categories" />
            </div>
          }
        />
      </ContextField>
    </CollapsibleSection>
  );
}

const Tags = () => {

  const form = useFormContext();

  const tags = useStore(form.store, (s: any) => s.values.tags);
  const except_tags = useStore(form.store, (s: any) => s.values.except_tags);

  return (
    <CollapsibleSection
      defaultOpen={tags !== undefined || except_tags !== undefined}
      title="Tags"
      subtitle="Match or exclude tags (if announced)"
    >
      <div className={classNames("sm:col-span-6", FilterLayoutClass, FilterTightGridGapClass)}>
        <ContextField name="tags">
          <TextAreaAutoResize
            label="Match tags"
            columns={8}
            placeholder="eg. tag1,tag2"
            tooltip={
              <div>
                <p>Comma separated list of tags to match.</p>
                <DocsLink href="https://autobrr.com/filters#advanced" />
              </div>
            }
          />
        </ContextField>
        <ContextField name="tags_match_logic">
          <Select
            label="Match logic"
            columns={4}
            options={CONSTS.tagsMatchLogicOptions}
            optionDefaultText="any"
            tooltip={
              <div>
                <p>Logic used to match filter tags.</p>
                <DocsLink href="https://autobrr.com/filters#advanced" />
              </div>
            }
          />
        </ContextField>
      </div>
      <div className={classNames("sm:col-span-6", FilterLayoutClass, FilterTightGridGapClass)}>
        <ContextField name="except_tags">
          <TextAreaAutoResize
            label="Except tags"
            columns={8}
            placeholder="eg. tag1,tag2"
            tooltip={
              <div>
                <p>Comma separated list of tags to ignore (takes priority over Match releases).</p>
                <DocsLink href="https://autobrr.com/filters#advanced" />
              </div>
            }
          />
        </ContextField>
        <ContextField name="except_tags_match_logic">
          <Select
            label="Except logic"
            columns={4}
            options={CONSTS.tagsMatchLogicOptions}
            optionDefaultText="any"
            tooltip={
              <div>
                <p>Logic used to match except tags.</p>
                <DocsLink href="https://autobrr.com/filters#advanced" />
              </div>
            }
          />
        </ContextField>
      </div>
    </CollapsibleSection>
  );
}

const Uploaders = () => {

  const form = useFormContext();

  const match_uploaders = useStore(form.store, (s: any) => s.values.match_uploaders);
  const except_uploaders = useStore(form.store, (s: any) => s.values.except_uploaders);

  return (
    <CollapsibleSection
      defaultOpen={match_uploaders !== undefined || except_uploaders !== undefined}
      title="Uploaders"
      subtitle="Match or ignore uploaders (if announced)"
    >
      <ContextField name="match_uploaders">
        <TextAreaAutoResize
          label="Match uploaders"
          columns={6}
          placeholder="eg. uploader1,uploader2"
          tooltip={
            <div>
              <p>Comma separated list of uploaders to match.</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
            </div>
          }
        />
      </ContextField>
      <ContextField name="except_uploaders">
        <TextAreaAutoResize
          label="Except uploaders"
          columns={6}
          placeholder="eg. anonymous1,anonymous2"
          tooltip={
            <div>
              <p>Comma separated list of uploaders to ignore (takes priority over Match releases).
              </p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
            </div>
          }
        />
      </ContextField>
    </CollapsibleSection>
  );
}

const Language = () => {

  const form = useFormContext();

  const match_language = useStore(form.store, (s: any) => s.values.match_language);
  const except_language = useStore(form.store, (s: any) => s.values.except_language);

  return (
    <CollapsibleSection
      defaultOpen={match_language?.length > 0 || except_language?.length > 0}
      title="Language"
      subtitle="Match or ignore languages (if announced)"
    >
      <ContextField name="match_language">
        <MultiSelect
          options={CONSTS.LANGUAGE_OPTIONS}
          label="Match Language"
          columns={6}
        />
      </ContextField>
      <ContextField name="except_language">
        <MultiSelect
          options={CONSTS.LANGUAGE_OPTIONS}
          label="Except Language"
          columns={6}
        />
      </ContextField>
    </CollapsibleSection>
  );
}

const Origins = () => {

  const form = useFormContext();

  const origins = useStore(form.store, (s: any) => s.values.origins);
  const except_origins = useStore(form.store, (s: any) => s.values.except_origins);

  return (
    <CollapsibleSection
      defaultOpen={origins?.length > 0 || except_origins?.length > 0}
      title="Origins"
      subtitle="Match Internals, Scene, P2P, etc. (if announced)"
    >
      <ContextField name="origins">
        <MultiSelect
          options={CONSTS.ORIGIN_OPTIONS}
          label="Match Origins"
          columns={6}
        />
      </ContextField>
      <ContextField name="except_origins">
        <MultiSelect
          options={CONSTS.ORIGIN_OPTIONS}
          label="Except Origins"
          columns={6}
        />
      </ContextField>
    </CollapsibleSection>
  );
}

const Freeleech = () => {

  const form = useFormContext();

  const freeleech = useStore(form.store, (s: any) => s.values.freeleech);
  const freeleech_percent = useStore(form.store, (s: any) => s.values.freeleech_percent);

  return (
    <CollapsibleSection
      defaultOpen={freeleech || freeleech_percent !== undefined}
      title="Freeleech"
      subtitle="Match based off freeleech (if announced)"
    >
      <ContextField name="freeleech_percent">
        <TextField
          label="Freeleech percent"
          disabled={freeleech}
          tooltip={
            <div>
              <p>
                Freeleech may be announced as a binary true/false value or as a
                percentage (less likely), depending on the indexer. Use one <span className="font-bold">or</span> the other.
                The Freeleech toggle overrides this field if it is toggled/true.
              </p>
              <br />
              <p>
                Refer to our documentation for more details:{" "}
                <DocsLink href="https://autobrr.com/filters/freeleech" />
              </p>
            </div>
          }
          columns={6}
          placeholder="eg. 50,75-100"
        />
      </ContextField>
      <FilterHalfRow>
        <ContextField name="freeleech">
          <SwitchGroup
            label="Freeleech"
            className="py-0"
            description="Cannot be used with Freeleech percent. Overrides Freeleech percent if toggled/true."
            tooltip={
              <div>
                <p>
                  Freeleech may be announced as a binary true/false value (more likely) or as a
                  percentage, depending on the indexer. Use one <span className="font-bold">or</span> the other.
                  This field overrides Freeleech percent if it is toggled/true.
                </p>
                <br />
                <p>
                  See who uses what in the documentation:{" "}
                  <DocsLink href="https://autobrr.com/filters/freeleech" />
                </p>
              </div>
            }
          />
        </ContextField>
      </FilterHalfRow>
    </CollapsibleSection>
  );
}

const FeedSpecific = () => {

  const form = useFormContext();

  const use_regex_description = useStore(form.store, (s: any) => s.values.use_regex_description);
  const match_description = useStore(form.store, (s: any) => s.values.match_description);
  const except_description = useStore(form.store, (s: any) => s.values.except_description);
  const min_seeders = useStore(form.store, (s: any) => s.values.min_seeders);
  const max_seeders = useStore(form.store, (s: any) => s.values.max_seeders);
  const min_leechers = useStore(form.store, (s: any) => s.values.min_leechers);
  const max_leechers = useStore(form.store, (s: any) => s.values.max_leechers);

  return (
    <CollapsibleSection
      defaultOpen={
        use_regex_description ||
        match_description !== undefined ||
        except_description !== undefined ||
        min_seeders !== undefined ||
        max_seeders !== undefined ||
        min_leechers !== undefined ||
        max_leechers !== undefined
      }
      title="RSS/Torznab/Newznab-specific"
      subtitle={
        <>These options are <span className="font-bold">only</span> for Feeds such as RSS, Torznab and Newznab</>
      }
    >
      <FilterLayout>
        <ContextField name="use_regex_description">
          <SwitchGroup
            label="Use Regex"
            className="col-span-12 sm:col-span-6"
          />
        </ContextField>
      </FilterLayout>

      <ContextField name="match_description">
        <RegexTextAreaField
          label="Match description"
          useRegex={use_regex_description}
          columns={6}
          placeholder="eg. *some?movie*,*some?show*s01*"
          tooltip={
            <div>
              <p>This field has full regex support (Golang flavour).</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
              <br />
              <br />
              <p>Remember to tick <b>Use Regex</b> if using more than <code>*</code> and <code>?</code>.</p>
              <br />
              <p>Mode: <code>(?i)</code> <b>case-insensitive</b></p>
            </div>
          }
        />
      </ContextField>
      <ContextField name="except_description">
        <RegexTextAreaField
          label="Except description"
          useRegex={use_regex_description}
          columns={6}
          placeholder="eg. *bad?movie*,*bad?show*s03*"
          tooltip={
            <div>
              <p>This field has full regex support (Golang flavour).</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
              <br />
              <br />
              <p>Remember to tick <b>Use Regex</b> if using more than <code>*</code> and <code>?</code>.</p>
              <br />
              <p>Mode: <code>(?i)</code> <b>case-insensitive</b></p>
            </div>
          }
        />
      </ContextField>
      <ContextField name="min_seeders">
        <NumberField
          label="Min Seeders"
          placeholder="Takes any number (0 is infinite)"
          tooltip={
            <div>
              <p>Number of min seeders as specified by the respective unit. Only for Torznab</p>
              <DocsLink href="https://autobrr.com/filters#rules" />
            </div>
          }
        />
      </ContextField>
      <ContextField name="max_seeders">
        <NumberField
          label="Max Seeders"
          placeholder="Takes any number (0 is infinite)"
          tooltip={
            <div>
              <p>Number of max seeders as specified by the respective unit. Only for Torznab</p>
              <DocsLink href="https://autobrr.com/filters#rules" />
            </div>
          }
        />
      </ContextField>
      <ContextField name="min_leechers">
        <NumberField
          label="Min Leechers"
          placeholder="Takes any number (0 is infinite)"
          tooltip={
            <div>
              <p>Number of min leechers as specified by the respective unit. Only for Torznab</p>
              <DocsLink href="https://autobrr.com/filters#rules" />
            </div>
          }
        />
      </ContextField>
      <ContextField name="max_leechers">
        <NumberField
          label="Max Leechers"
          placeholder="Takes any number (0 is infinite)"
          tooltip={
            <div>
              <p>Number of max leechers as specified by the respective unit. Only for Torznab</p>
              <DocsLink href="https://autobrr.com/filters#rules" />
            </div>
          }
        />
      </ContextField>
    </CollapsibleSection>
  );
}
const RawReleaseTags = () => {

  const form = useFormContext();

  const use_regex_release_tags = useStore(form.store, (s: any) => s.values.use_regex_release_tags);
  const match_release_tags = useStore(form.store, (s: any) => s.values.match_release_tags);
  const except_release_tags = useStore(form.store, (s: any) => s.values.except_release_tags);

  return (
    <CollapsibleSection
      defaultOpen={use_regex_release_tags || match_release_tags !== undefined || except_release_tags !== undefined}
      title="Raw Release Tags"
      subtitle={
        <>
          <span className="underline underline-offset-2">Advanced users only</span>
          {": "}This is the <span className="font-bold">raw</span> releaseTags string from the announce.
        </>
      }
    >
      <WarningAlert
        text={
          <>These might not be what you think they are. For <span className="underline font-bold">very advanced</span> users who know how things are parsed.</>
        }
      />

      <FilterLayout>
        <ContextField name="use_regex_release_tags">
          <SwitchGroup
            label="Use Regex"
            className="col-span-12 sm:col-span-6"
          />
        </ContextField>
      </FilterLayout>

      <ContextField name="match_release_tags">
        <RegexField
          label="Match release tags"
          useRegex={use_regex_release_tags}
          columns={6}
          placeholder="eg. *mkv*,*foreign*"
          tooltip={
            <div>
              <p>This field has full regex support (Golang flavour).</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
              <br />
              <br />
              <p>Remember to tick <b>Use Regex</b> if using more than <code>*</code> and <code>?</code>.</p>
              <br />
              <p>Mode: <code>(?i)</code> <b>case-insensitive</b></p>
            </div>
          }
        />
      </ContextField>
      <ContextField name="except_release_tags">
        <RegexField
          label="Except release tags"
          useRegex={use_regex_release_tags}
          columns={6}
          placeholder="eg. *mkv*,*foreign*"
          tooltip={
            <div>
              <p>This field has full regex support (Golang flavour).</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
              <br />
              <br />
              <p>Remember to tick <b>Use Regex</b> if using more than <code>*</code> and <code>?</code>.</p>
              <br />
              <p>Mode: <code>(?i)</code> <b>case-insensitive</b></p>
            </div>
          }
        />
      </ContextField>
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
      <RawReleaseTags />
    </div>
  );
}
