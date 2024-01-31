import type { FormikValues } from "formik";

import { DocsLink } from "@components/ExternalLink";
import { WarningAlert } from "@components/alerts";

import * as Input from "@components/inputs";
import * as CONSTS from "@domain/constants";

import { CollapsibleSection } from "./_components";
import * as Components from "./_components";
import { classNames } from "@utils";

type ValueConsumer = {
  values: FormikValues;
};

const Releases = ({ values }: ValueConsumer) => (
  <CollapsibleSection
    defaultOpen={values.use_regex || values.match_releases || values.except_releases}
    title="Release Names"
    subtitle="Match only certain release names and/or ignore other release names."
  >
    <Components.Layout>
      <Components.HalfRow>
        <Input.SwitchGroup name="use_regex" label="Use Regex" className="pt-2" />
      </Components.HalfRow>
    </Components.Layout>

    <Components.Layout>
      <Components.HalfRow>
        <Input.RegexTextAreaField
          name="match_releases"
          label="Match releases"
          useRegex={values.use_regex}
          columns={6}
          placeholder="eg. *some?movie*,*some?show*s01*"
          tooltip={
            <div>
              <p>This field has full regex support (Golang flavour).</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
              <br />
              <br />
              <p>Remember to tick <b>Use Regex</b> if using more than <code>*</code> and <code>?</code>.</p>
            </div>
          }
        />
      </Components.HalfRow>

      <Components.HalfRow>
        <Input.RegexTextAreaField
          name="except_releases"
          label="Except releases"
          useRegex={values.use_regex}
          columns={6}
          placeholder="eg. *bad?movie*,*bad?show*s03*"
          tooltip={
            <div>
              <p>This field has full regex support (Golang flavour).</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
              <br />
              <br />
              <p>Remember to tick <b>Use Regex</b> below if using more than <code>*</code> and <code>?</code>.</p>
            </div>
          }
        />
      </Components.HalfRow>

    </Components.Layout>

    {values.match_releases ? (
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
    {values.except_releases ? (
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

const Groups = ({ values }: ValueConsumer) => (
  <CollapsibleSection
    defaultOpen={values.match_release_groups || values.except_release_groups}
    title="Groups"
    subtitle="Match only certain groups and/or ignore other groups."
  >
    <Input.TextAreaAutoResize
      name="match_release_groups"
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
    <Input.TextAreaAutoResize
      name="except_release_groups"
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
  </CollapsibleSection>
);

const Categories = ({ values }: ValueConsumer) => (
  <CollapsibleSection
    defaultOpen={values.match_categories || values.except_categories}
    title="Categories"
    subtitle="Match or exclude categories (if announced)"
  >
    <Input.TextAreaAutoResize
      name="match_categories"
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
    <Input.TextAreaAutoResize
      name="except_categories"
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
  </CollapsibleSection>
);

const Tags = ({ values }: ValueConsumer) => (
  <CollapsibleSection
    defaultOpen={values.tags || values.except_tags}
    title="Tags"
    subtitle="Match or exclude tags (if announced)"
  >
    <div className={classNames("sm:col-span-6", Components.LayoutClass, Components.TightGridGapClass)}>
      <Input.TextAreaAutoResize
        name="tags"
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
      <Input.Select
        name="tags_match_logic"
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
    </div>
    <div className={classNames("sm:col-span-6", Components.LayoutClass, Components.TightGridGapClass)}>
      <Input.TextAreaAutoResize
        name="except_tags"
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
      <Input.Select
        name="except_tags_match_logic"
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
    </div>
  </CollapsibleSection>
);

const Uploaders = ({ values }: ValueConsumer) => (
  <CollapsibleSection
    defaultOpen={values.uploaders || values.except_uploaders}
    title="Uploaders"
    subtitle="Match or ignore uploaders (if announced)"
  >
    <Input.TextAreaAutoResize
      name="match_uploaders"
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
    <Input.TextAreaAutoResize
      name="except_uploaders"
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
  </CollapsibleSection>
);

const Language = ({ values }: ValueConsumer) => (
  <CollapsibleSection
    defaultOpen={(values.match_language && values.match_language.length > 0) || (values.except_language && values.except_language.length > 0)}
    title="Language"
    subtitle="Match or ignore languages (if announced)"
  >
    <Input.MultiSelect
      name="match_language"
      options={CONSTS.LANGUAGE_OPTIONS}
      label="Match Language"
      columns={6}
    />
    <Input.MultiSelect
      name="except_language"
      options={CONSTS.LANGUAGE_OPTIONS}
      label="Except Language"
      columns={6}
    />
  </CollapsibleSection>
);

const Origins = ({ values }: ValueConsumer) => (
  <CollapsibleSection
    defaultOpen={(values.origins && values.origins.length > 0 || values.except_origins && values.except_origins.length > 0)}
    title="Origins"
    subtitle="Match Internals, Scene, P2P, etc. (if announced)"
  >
    <Input.MultiSelect
      name="origins"
      options={CONSTS.ORIGIN_OPTIONS}
      label="Match Origins"
      columns={6}
    />
    <Input.MultiSelect
      name="except_origins"
      options={CONSTS.ORIGIN_OPTIONS}
      label="Except Origins"
      columns={6}
    />
  </CollapsibleSection>
);

const Freeleech = ({ values }: ValueConsumer) => (
  <CollapsibleSection
    defaultOpen={values.freeleech || values.freeleech_percent}
    title="Freeleech"
    subtitle="Match based off freeleech (if announced)"
  >
    <Input.TextField
      name="freeleech_percent"
      label="Freeleech percent"
      disabled={values.freeleech}
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
    <Components.HalfRow>
      <Input.SwitchGroup
        name="freeleech"
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
    </Components.HalfRow>
  </CollapsibleSection>
);

const FeedSpecific = ({ values }: ValueConsumer) => (
  <CollapsibleSection
    defaultOpen={values.use_regex_description || values.match_description || values.except_description}
    title="RSS/Torznab/Newznab-specific"
    subtitle={
      <>These options are <span className="font-bold">only</span> for Feeds such as RSS, Torznab and Newznab</>
    }
  >
    <Components.Layout>
      <Input.SwitchGroup
        name="use_regex_description"
        label="Use Regex"
        className="col-span-12 sm:col-span-6"
      />
    </Components.Layout>

    <Input.RegexTextAreaField
      name="match_description"
      label="Match description"
      useRegex={values.use_regex_description}
      columns={6}
      placeholder="eg. *some?movie*,*some?show*s01*"
      tooltip={
        <div>
          <p>This field has full regex support (Golang flavour).</p>
          <DocsLink href="https://autobrr.com/filters#advanced" />
          <br />
          <br />
          <p>Remember to tick <b>Use Regex</b> below if using more than <code>*</code> and <code>?</code>.</p>
        </div>
      }
    />
    <Input.RegexTextAreaField
      name="except_description"
      label="Except description"
      useRegex={values.use_regex_description}
      columns={6}
      placeholder="eg. *bad?movie*,*bad?show*s03*"
      tooltip={
        <div>
          <p>This field has full regex support (Golang flavour).</p>
          <DocsLink href="https://autobrr.com/filters#advanced" />
          <br />
          <br />
          <p>Remember to tick <b>Use Regex</b> below if using more than <code>*</code> and <code>?</code>.</p>
        </div>
      }
    />
    <Input.NumberField
      name="min_seeders"
      label="Min Seeders"
      placeholder="Takes any number (0 is infinite)"
      tooltip={
        <div>
          <p>Number of min seeders as specified by the respective unit. Only for Torznab</p>
          <DocsLink href="https://autobrr.com/filters#rules" />
        </div>
      }
    />
    <Input.NumberField
      name="max_seeders"
      label="Max Seeders"
      placeholder="Takes any number (0 is infinite)"
      tooltip={
        <div>
          <p>Number of max seeders as specified by the respective unit. Only for Torznab</p>
          <DocsLink href="https://autobrr.com/filters#rules" />
        </div>
      }
    />
    <Input.NumberField
      name="min_leechers"
      label="Min Leechers"
      placeholder="Takes any number (0 is infinite)"
      tooltip={
        <div>
          <p>Number of min leechers as specified by the respective unit. Only for Torznab</p>
          <DocsLink href="https://autobrr.com/filters#rules" />
        </div>
      }
    />
    <Input.NumberField
      name="max_leechers"
      label="Max Leechers"
      placeholder="Takes any number (0 is infinite)"
      tooltip={
        <div>
          <p>Number of max leechers as specified by the respective unit. Only for Torznab</p>
          <DocsLink href="https://autobrr.com/filters#rules" />
        </div>
      }
    />
  </CollapsibleSection>
);

const RawReleaseTags = ({ values }: ValueConsumer) => (
  <CollapsibleSection
    defaultOpen={values.use_regex_release_tags || values.match_release_tags || values.except_release_tags}
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

    <Components.Layout>
      <Input.SwitchGroup
        name="use_regex_release_tags"
        label="Use Regex"
        className="col-span-12 sm:col-span-6"
      />
    </Components.Layout>

    <Input.RegexField
      name="match_release_tags"
      label="Match release tags"
      useRegex={values.use_regex_release_tags}
      columns={6}
      placeholder="eg. *mkv*,*foreign*"
    />
    <Input.RegexField
      name="except_release_tags"
      label="Except release tags"
      useRegex={values.use_regex_release_tags}
      columns={6}
      placeholder="eg. *mkv*,*foreign*"
    />
  </CollapsibleSection>
);

export const Advanced = ({ values }: { values: FormikValues; }) => (
  <div className="flex flex-col w-full gap-y-4 py-2 sm:-mx-1">
    <Releases values={values} />
    <Groups values={values} />
    <Categories values={values} />
    <Freeleech values={values} />
    <Tags values={values}/>
    <Uploaders values={values}/>
    <Language values={values}/>
    <Origins values={values} />
    <FeedSpecific values={values} />
    <RawReleaseTags values={values} />
  </div>
);
