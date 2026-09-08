/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useSelector } from "@tanstack/react-form";
import Select from "react-select";
import { XMarkIcon } from "@heroicons/react/24/solid";
import { useTranslation } from "react-i18next";

import { classNames, sleep } from "@utils";
import { extractCategoryTreeFromCaps, flattenCategoryIds, parseCapabilitiesPayload } from "@utils/caps";
import { useAppForm, useFormContext, useFormValue } from "@hooks/form";
import { FormDebug } from "@components/debug";
import { APIClient } from "@api/APIClient";
import { FeedKeys, IndexerKeys, ReleaseKeys } from "@api/query_keys";
import { IndexersSchemaQueryOptions, ProxiesQueryOptions } from "@api/queries";
import { SlideOver, SlideOverShell, SlideOverTitle } from "@components/panels";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { PasswordFieldWide, SwitchButton, SwitchGroupWide, TextFieldWide } from "@components/inputs";
import { SelectFieldBasic, SelectFieldCreatable } from "@components/inputs/select_wide";
import { FeedDownloadTypeOptions } from "@domain/constants";
import { DocsLink } from "@components/ExternalLink";
import * as common from "@components/inputs/common";
import { selectComponents, selectStyles, selectTheme } from "@components/inputs/select_props";
import { SelectField } from "@forms/settings/IrcForms";
import { AddFormProps, UpdateFormProps } from "@forms/_shared";

// const isRequired = (message: string) => (value?: string | undefined) => (!!value ? undefined : message);

function validateField(s: IndexerSetting, t: (key: string) => string) {
  return (value?: string | undefined) => {
    if (s.required) {
      if (s.default !== "") {
        if (value && s.default === value) {
          return t("forms.indexer.defaultValueValidation");
        }
      }
      return value ? undefined : t("forms.indexer.required");
    }
  };
}

const IrcSettingFields = (ind: IndexerDefinition, indexer: string) => {
  const { t } = useTranslation("settings");
  if (!indexer.length) {
    return null;
  }

  return (
    <>
      {ind && ind.implementation == "irc" && ind.irc && ind.irc.settings && (
        <div className="border-t border-gray-200 dark:border-gray-700 py-5">
          <div className="px-4">
            <h2 className="text-lg font-medium text-gray-900 dark:text-white">{t("forms.indexer.settingsIrcTitle")}</h2>
            <p className="text-sm text-gray-500 dark:text-gray-200">
              {t("forms.indexer.settingsIrcDesc")}
            </p>
          </div>

          {ind.irc.settings.map((f: IndexerSetting, idx: number) => {
            switch (f.type) {
            case "text": {
              return (
                <TextFieldWide
                  key={idx}
                  name={`irc.${f.name}`}
                  label={f.label}
                  required={f.required}
                  help={f.help}
                  autoComplete="off"
                  validate={validateField(f, t)}
                  tooltip={
                    <div>
                      <p>{t("forms.indexer.ircGuideTooltip")}</p>
                      <DocsLink href="https://autobrr.com/configuration/irc" />
                    </div>
                  }
                />
              );
            }
            case "secret": {
              if (f.name === "invite_command") {
                return <PasswordFieldWide defaultVisible name={`irc.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} defaultValue={f.default} validate={validateField(f, t)} />;
              }
              return <PasswordFieldWide name={`irc.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} defaultValue={f.default} validate={validateField(f, t)} />;
            }
          }
            return null;
          })}
        </div>
      )}
    </>
  );

};

const TorznabFeedSettingFields = (ind: IndexerDefinition, indexer: string) => {
  const { t } = useTranslation("settings");
  if (indexer !== "") {
    return (
      <Fragment>
        {ind && ind.implementation == "torznab" && ind.feed && ind.feed.settings && (
          <div className="">
            <div className="pt-4 px-4">
              <h2 className="text-lg font-medium text-gray-900 dark:text-white">{t("forms.indexer.torznabTitle")}</h2>
              <p className="text-sm text-gray-500 dark:text-gray-200">
                {t("forms.indexer.torznabDesc")}
              </p>
            </div>

            <TextFieldWide name="name" label={t("forms.indexer.name")} defaultValue="" required={true} />

            <TextFieldWide
              name="feed.url"
              label={t("forms.indexer.url")}
              required={true}
              help={t("forms.indexer.torznabUrlHelp")}
              tooltip={
                <div>
                  <p>{t("forms.indexer.urlFormatTitle")}</p>
                  <br/>
                  <ul>
                    <li>{t("forms.indexer.prowlarr")}: <code className="text-blue-400">http(s)://url.tld/indexerID/api</code></li>
                    <li>{t("forms.indexer.jackett")}: <code className="text-blue-400">http(s)://url.tld/jackett/api/v2.0/indexers/indexerName/results/torznab/</code></li>
                  </ul>
                </div>
              }
            />

            <PasswordFieldWide name="feed.api_key" label={t("forms.indexer.apiKey")} help={t("forms.indexer.apiKey")} required={true} />

            <SelectFieldBasic
              name="feed.settings.download_type"
              label={t("forms.indexer.downloadType")}
              options={FeedDownloadTypeOptions}
              tooltip={<span>{t("forms.indexer.downloadTypeTooltip2")}</span>}
              help={t("forms.indexer.downloadTypeHelp")}
            />

            <FeedCategoriesDraftSection feedType="TORZNAB" />
          </div>
        )}
      </Fragment>
    );
  }
};

const NewznabFeedSettingFields = (ind: IndexerDefinition, indexer: string) => {
  const { t } = useTranslation("settings");
  if (indexer !== "") {
    return (
      <Fragment>
        {ind && ind.implementation == "newznab" && ind.feed && ind.feed.settings && (
          <div className="">
            <div className="pt-4 px-4">
              <h2 className="text-lg font-medium text-gray-900 dark:text-white">{t("forms.indexer.newznabTitle")}</h2>
              <p className="text-sm text-gray-500 dark:text-gray-200">
                {t("forms.indexer.newznabDesc")}
              </p>
            </div>

            <TextFieldWide name="name" label={t("forms.indexer.name")} defaultValue="" required={true} />

            <TextFieldWide
              name="feed.newznab_url"
              label={t("forms.indexer.url")}
              required={true}
              help={t("forms.indexer.newznabUrlHelp")}
              tooltip={
                <div>
                  <p>{t("forms.indexer.urlFormatTitle")}</p>
                  <br/>
                  <ul>
                    <li>{t("forms.indexer.prowlarr")}: <code className="text-blue-400">http(s)://url.tld/indexerID/api</code></li>
                    <li>{t("forms.indexer.jackett")}: <code className="text-blue-400">http(s)://url.tld/jackett/api/v2.0/indexers/indexerName/results/newznab/</code></li>
                  </ul>
                </div>
              }
            />

            <PasswordFieldWide name="feed.api_key" label={t("forms.indexer.apiKey")} help={t("forms.indexer.apiKey")} required={true} />

            <FeedCategoriesDraftSection feedType="NEWZNAB" />
          </div>
        )}
      </Fragment>
    );
  }
};

const RSSFeedSettingFields = (ind: IndexerDefinition, indexer: string) => {
  const { t } = useTranslation("settings");
  if (indexer !== "") {
    return (
      <Fragment>
        {ind && ind.implementation == "rss" && ind.feed && ind.feed.settings && (
          <div className="">
            <div className="pt-4 px-4">
              <h2 className="text-lg font-medium text-gray-900 dark:text-white">{t("forms.indexer.rssTitle")}</h2>
              <p className="text-sm text-gray-500 dark:text-gray-200">
                {t("forms.indexer.rssDesc")}
              </p>
            </div>

            <TextFieldWide name="name" label={t("forms.indexer.name")} defaultValue="" />

            {ind.feed.settings.map((f: IndexerSetting, idx: number) => {
              switch (f.type) {
              case "text": {
                return <TextFieldWide name={`feed.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} autoComplete="off" validate={validateField(f, t)} />;
              }
              case "secret": {
                return <PasswordFieldWide name={`feed.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} defaultValue={f.default} validate={validateField(f, t)} />;
              }
              }
              return null;
            })}

            <SelectFieldBasic
              name="feed.settings.download_type"
              label={t("forms.indexer.downloadType")}
              options={FeedDownloadTypeOptions}
              tooltip={<span>{t("forms.indexer.downloadTypeTooltip2")}</span>}
              help={t("forms.indexer.downloadTypeHelp")}
            />
          </div>
        )}
      </Fragment>
    );
  }
};

function FeedCategoriesDraftSection({ feedType }: { feedType: FeedType }) {
  const { t } = useTranslation("settings");
  const values = useFormValue((v: Record<string, unknown>) => ({ feed: v.feed }));
  const form = useFormContext();
  const feedValues = (values.feed ?? {}) as Record<string, unknown>;
  const capabilities = feedValues.capabilities ?? null;
  const categoriesValue = Array.isArray(feedValues.categories) ? (feedValues.categories as number[]) : [];
  const capsPayload = useMemo(() => parseCapabilitiesPayload(capabilities), [capabilities]);
  const categoriesTree = useMemo(() => extractCategoryTreeFromCaps(capsPayload), [capsPayload]);
  const url = feedType === "TORZNAB"
    ? String(feedValues.url ?? "")
    : String(feedValues.newznab_url ?? feedValues.url ?? "");
  const apiKey = typeof feedValues.api_key === "string" ? feedValues.api_key : "";
  const hasCaps = Boolean(capabilities);
  const canFetch = url.length > 0;

  const fetchCapsMutation = useMutation({
    mutationFn: () => APIClient.feeds.fetchCapsDraft({
      type: feedType,
      url,
      api_key: apiKey,
      timeout: 60
    }),
    onSuccess: (caps) => {
      const nextCategories = flattenCategoryIds(extractCategoryTreeFromCaps(caps));
      const filteredSelection = categoriesValue.filter((id) => nextCategories.includes(id));

      form.setFieldValue("feed.capabilities", caps ?? null);
      form.setFieldValue("feed.categories", filteredSelection);
    },
    onError: (error: unknown) => {
      const message = error instanceof Error ? error.message : t("forms.indexer.fetchFailed");
      toast.custom((toastInstance) => <Toast type="error" body={message} t={toastInstance} />);
    }
  });

  const toggleCategory = (id: number) => {
    if (categoriesValue.includes(id)) {
      form.setFieldValue(
        "feed.categories",
        categoriesValue.filter((category) => category !== id)
      );
      return;
    }

    form.setFieldValue("feed.categories", [...categoriesValue, id]);
  };

  const toggleParentCategory = (id: number, childIds: number[]) => {
    if (categoriesValue.includes(id)) {
      form.setFieldValue(
        "feed.categories",
        categoriesValue.filter((category) => category !== id)
      );
      return;
    }

    form.setFieldValue(
      "feed.categories",
      [...categoriesValue.filter((category) => !childIds.includes(category)), id]
    );
  };

  return (
    <div className="mt-6 border-t border-gray-200 dark:border-gray-700">
      <div className="pt-4 px-4 flex items-center justify-between">
        <div>
          <div className="text-lg font-medium text-gray-900 dark:text-white">{t("forms.indexer.categoriesTitle")}</div>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {t("forms.indexer.categoriesDescription")}
          </p>
        </div>
        <button
          type="button"
          className="inline-flex items-center rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-1.5 cursor-pointer text-sm font-medium text-gray-700 dark:text-gray-200 shadow-xs hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-blue-500"
          onClick={() => fetchCapsMutation.mutate()}
          disabled={!canFetch || fetchCapsMutation.isPending}
          title={!canFetch ? t("forms.indexer.fetchNeedsUrl") : undefined}
        >
          {fetchCapsMutation.isPending ? t("forms.indexer.fetching") : hasCaps ? t("forms.indexer.refetch") : t("forms.indexer.fetch")}
        </button>
      </div>

      {categoriesTree.length ? (
        <div className="px-4 pt-4 pb-2 space-y-3 max-h-max overflow-y-auto">
          {categoriesTree.map((category) => {
            const childIds = category.subcategories.map((sub) => sub.id);
            const isParentSelected = categoriesValue.includes(category.id);

            return (
              <div key={category.id} className="space-y-2">
                <label
                  className="flex items-center justify-between gap-3 cursor-pointer text-sm text-gray-700 dark:text-gray-200"
                  onClick={(event) => event.stopPropagation()}
                >
                  <span className="flex items-center gap-3">
                    <input
                      type="checkbox"
                      checked={categoriesValue.includes(category.id)}
                      onChange={() => toggleParentCategory(category.id, childIds)}
                      onClick={(event) => event.stopPropagation()}
                      className="h-5 w-5 rounded border-gray-300 text-blue-600 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-800 cursor-pointer"
                    />
                    <span className="font-medium truncate">{category.name}</span>
                  </span>
                  <span className="text-xs text-gray-400 dark:text-gray-500">{category.id}</span>
                </label>

                {category.subcategories.map((subCategory) => (
                  <label
                    key={subCategory.id}
                    className="flex items-center justify-between gap-3 pl-6 cursor-pointer text-sm text-gray-700 dark:text-gray-200"
                    onClick={(event) => event.stopPropagation()}
                  >
                    <span className="flex items-center gap-3">
                      <input
                        type="checkbox"
                        checked={categoriesValue.includes(subCategory.id)}
                        onChange={() => toggleCategory(subCategory.id)}
                        onClick={(event) => event.stopPropagation()}
                        disabled={isParentSelected}
                        className="h-5 w-5 rounded border-gray-300 text-blue-600 focus:ring-blue-500 disabled:cursor-not-allowed disabled:opacity-60 dark:border-gray-600 dark:bg-gray-800"
                      />
                      <span className="truncate">{subCategory.name}</span>
                    </span>
                    <span className="text-xs text-gray-400 dark:text-gray-500">{subCategory.id}</span>
                  </label>
                ))}
              </div>
            );
          })}
        </div>
      ) : (
        <div className="px-4 pt-3 pb-2 text-sm text-gray-500 dark:text-gray-400">
          {hasCaps ? t("forms.indexer.noCategories") : t("forms.indexer.fetchToSelect")}
        </div>
      )}
    </div>
  );
}

const SettingFields = (ind: IndexerDefinition, indexer: string) => {
  const { t } = useTranslation("settings");
  if (indexer !== "") {
    return (
      <div key="opt">
        {ind && ind.settings && ind.settings.map((f, idx: number) => {
          switch (f.type) {
          case "text": {
            return (
              <TextFieldWide name={`settings.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} autoComplete="off" validate={validateField(f, t)} />
            );
          }
          case "secret": {
            return (
              <PasswordFieldWide
                name={`settings.${f.name}`}
                label={f.label}
                required={f.required}
                key={idx}
                help={f.help}
                validate={validateField(f, t)}
                tooltip={
                  <div>
                    <p>{t("forms.indexer.secretFieldTooltip")}</p>
                    <br />
                    <DocsLink href="https://autobrr.com/faqs#common-action-rejections" />
                  </div>
                }
              />
            );
          }
          }
          return null;
        })}
        <div hidden={true}>
          <TextFieldWide name="name" label={t("forms.indexer.name")} defaultValue={ind?.name} />
        </div>
      </div>
    );
  }
};

type SelectValue = {
  label: string;
  value: string;
};

const buildIndexerIRCAuth = (
  definitionAuth: IndexerIRCAuth | undefined,
  formAuth: Partial<IrcAuth> | undefined
): IrcAuth => {
  const mechanism = definitionAuth?.mechanism ?? "SASL_PLAIN";
  const account = formAuth?.account ?? "";
  const password = formAuth?.password ?? "";

  if (mechanism === "NICKSERV" && password !== "") {
    return {
      mechanism,
      ...(account !== "" && { account }),
      password
    };
  }

  if (mechanism === "SASL_PLAIN" && account !== "" && password !== "") {
    return { mechanism, account, password };
  }

  return { mechanism: "NONE" };
};

interface IndexerAddIrcValues {
  nick?: string;
  pass?: string;
  invite_command?: string;
  auth?: Partial<IrcAuth>;
  channels?: { password: string };
}

interface IndexerAddFeedValues {
  url?: string;
  newznab_url?: string;
  api_key?: string;
  categories: number[];
  capabilities: FeedCaps | null;
  settings: FeedSettings;
}

interface IndexerAddInitialValues {
  enabled: boolean;
  identifier: string;
  implementation: string;
  name: string;
  base_url?: string;
  url?: string;
  use_proxy?: boolean;
  proxy_id?: number;
  irc: IndexerAddIrcValues;
  settings: Record<string, string>;
  feed: IndexerAddFeedValues;
}

export function IndexerAddForm({ isOpen, toggle }: AddFormProps) {
  return (
    <SlideOverShell isOpen={isOpen} toggle={toggle}>
      <IndexerAddFormPanel toggle={toggle} />
    </SlideOverShell>
  );
}

interface IndexerAddFormPanelProps {
  toggle: () => void;
}

function IndexerAddFormPanel({ toggle }: IndexerAddFormPanelProps) {
  const { t } = useTranslation("settings");
  const [indexer, setIndexer] = useState<IndexerDefinition>({} as IndexerDefinition);

  const queryClient = useQueryClient();
  const { data } = useQuery(IndexersSchemaQueryOptions(true));

  const mutation = useMutation({
    mutationFn: (indexer: IndexerAddInitialValues) => APIClient.indexers.create(indexer as unknown as Indexer),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: IndexerKeys.lists() });
      queryClient.invalidateQueries({ queryKey: IndexerKeys.options() });
      queryClient.invalidateQueries({ queryKey: ReleaseKeys.indexers() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.indexer.added")} t={toastInstance} />);
      sleep(1500);
      toggle();
    },
    onError: () => {
      toast.custom((toastInstance) => <Toast type="error" body={t("forms.indexer.addFailed")} t={toastInstance} />);
    }
  });

  const ircMutation = useMutation({
    mutationFn: (network: IrcNetworkCreate) => APIClient.irc.createNetwork(network)
  });

  const feedMutation = useMutation({
    mutationFn: (feed: FeedCreate) => APIClient.feeds.create(feed),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: FeedKeys.lists() });
    }
  });

  const onSubmit = (formData: IndexerAddInitialValues) => {
    const ind = data && data.find(i => i.identifier === formData.identifier);
    if (!ind) {
      console.error("could not find indexer: ", formData.identifier, " in ", data, " - ", formData);
      return;
    }

    switch (formData.implementation) {
      case "torznab": {
        const createFeed: FeedCreate = {
          name: formData.name,
          enabled: false,
          type: "TORZNAB",
          url: formData.feed.url ?? "",
          api_key: formData.feed.api_key,
          interval: 30,
          timeout: 60,
          indexer_id: 0,
          categories: formData.feed.categories ?? [],
          capabilities: formData.feed.capabilities ?? null,
          settings: formData.feed.settings
        };

        mutation.mutate(formData, {
          onSuccess: (indexer) => {
            // @eslint-ignore
            createFeed.indexer_id = indexer.id;

            feedMutation.mutate(createFeed);
          }
        });
        return;
      }

      case "newznab": {
        formData.url = formData.feed.url;

        const createFeed: FeedCreate = {
          name: formData.name,
          enabled: false,
          type: "NEWZNAB",
          url: formData.feed.newznab_url ?? "",
          api_key: formData.feed.api_key,
          interval: 30,
          timeout: 60,
          indexer_id: 0,
          categories: formData.feed.categories ?? [],
          capabilities: formData.feed.capabilities ?? null,
          settings: formData.feed.settings
        };

        mutation.mutate(formData, {
          onSuccess: (indexer) => {
            // @eslint-ignore
            createFeed.indexer_id = indexer.id;

            feedMutation.mutate(createFeed);
          }
        });
        return;
      }

      case "rss": {
        const createFeed: FeedCreate = {
          name: formData.name,
          enabled: false,
          type: "RSS",
          url: formData.feed.url ?? "",
          interval: 30,
          timeout: 60,
          indexer_id: 0,
          settings: formData.feed.settings
        };

        mutation.mutate(formData, {
          onSuccess: (indexer) => {
            // @eslint-ignore
            createFeed.indexer_id = indexer.id;

            feedMutation.mutate(createFeed);
          }
        });
        return;
      }

      case "irc": {
        const channels: IrcChannel[] = [];
        if (ind.irc?.channels.length) {
          let channelPass = "";
          if (formData.irc && formData.irc.channels && formData.irc?.channels?.password !== "") {
            channelPass = formData.irc.channels.password;
          }

          ind.irc.channels.forEach(element => {
            channels.push({
              id: 0,
              enabled: true,
              name: element.name,
              password: channelPass,
              detached: false,
              monitoring: false
            });
          });
        }

        const network: IrcNetworkCreate = {
          name: ind.irc.network,
          pass: formData.irc.pass || "",
          enabled: false,
          connected: false,
          server: ind.irc.server,
          port: ind.irc.port,
          tls: ind.irc.tls,
          tls_skip_verify: false,
          nick: formData.irc.nick ?? "",
          auth: buildIndexerIRCAuth(ind.irc.auth, formData.irc.auth),
          invite_command: formData.irc.invite_command ?? "",
          channels: channels
        };

        mutation.mutate(formData, {
          onSuccess: () => {
            ircMutation.mutate(network);
          }
        });
        return;
      }
      default: {
        console.error("unknown implementation: ", formData.implementation);
      }
    }
  };

  const initialValues: IndexerAddInitialValues = {
    enabled: true,
    identifier: "",
    implementation: "irc",
    name: "",
    use_proxy: false,
    irc: {},
    settings: {},
    feed: {
      categories: [],
      capabilities: null,
      settings: {} as FeedSettings
    }
  };

  const form = useAppForm({
    defaultValues: initialValues,
    onSubmit: ({ value }) => onSubmit(value)
  });

  const { identifier, use_proxy } = useSelector(form.store, (state) => ({ identifier: state.values.identifier, use_proxy: state.values.use_proxy }));

  return (
    <form.AppForm>
      <form
        className="h-full min-h-0 flex flex-col bg-white dark:bg-gray-800"
        onSubmit={(e) => {
          e.preventDefault();
          e.stopPropagation();
          form.handleSubmit();
        }}
      >
        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
            <div className="flex items-start justify-between space-x-3">
              <div className="space-y-1">
                <SlideOverTitle>
                  {t("forms.indexer.addTitle")}
                </SlideOverTitle>
                <p className="text-sm text-gray-500 dark:text-gray-200">
                  {t("forms.indexer.addDescription")}
                </p>
              </div>
              <div className="h-7 flex items-center">
                <button
                  type="button"
                  className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500 cursor-pointer"
                  onClick={toggle}
                >
                  <span className="sr-only">{t("forms.indexer.closePanel")}</span>
                  <XMarkIcon className="h-6 w-6" aria-hidden="true" />
                </button>
              </div>
            </div>
          </div>

          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            <div className="p-4 sm:py-6 flex items-center justify-between sm:grid sm:grid-cols-3 sm:gap-4">
              <div>
                <label
                  htmlFor="identifier"
                  className="block text-sm font-medium text-gray-900 dark:text-white"
                >
                  {t("forms.indexer.indexer")}
                </label>
              </div>
              <div className="sm:col-span-2">
                <IndexerIdentifierField data={data} setIndexer={setIndexer} />

              </div>
            </div>

            <SwitchGroupWide name="enabled" label={t("forms.indexer.enabled")} />

            {indexer.implementation == "irc" && (
              <SelectFieldCreatable
                name="base_url"
                label={t("forms.indexer.baseUrl")}
                help={t("forms.indexer.baseUrlHelp")}
                options={indexer.urls.map(u => ({ value: u, label: u, key: u }))}
              />
            )}

            {SettingFields(indexer, identifier)}

          </div>

          {IrcSettingFields(indexer, identifier)}
          {TorznabFeedSettingFields(indexer, identifier)}
          {NewznabFeedSettingFields(indexer, identifier)}
          {RSSFeedSettingFields(indexer, identifier)}

          {identifier !== "" && (
            <ProxyFields useProxy={use_proxy} />
          )}

          <FormDebug />
        </div>

        <div className="shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-5 sm:px-6">
          <div className="space-x-3 flex justify-end">
            <button
              type="button"
              className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer"
              onClick={toggle}
            >
              {t("forms.indexer.cancel")}
            </button>
            <button
              type="submit"
              className="inline-flex justify-center py-2 px-4 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer"
            >
              {t("forms.indexer.save")}
            </button>
          </div>
        </div>
      </form>
    </form.AppForm>
  );
}

interface IndexerIdentifierFieldProps {
  data?: IndexerDefinition[];
  setIndexer: (indexer: IndexerDefinition) => void;
}

function IndexerIdentifierField({ data, setIndexer }: IndexerIdentifierFieldProps) {
  const { t } = useTranslation("settings");
  const form = useFormContext();

  const options = useMemo(
    () => data?.toSorted((a, b) => a.name.localeCompare(b.name)).map((v) => ({
      label: v.name,
      value: v.identifier
    })),
    [data]
  );

  return (
    <form.Field name="identifier">
      {(field) => (
        <Select
          name={field.name}
          onBlur={field.handleBlur}
          isClearable={true}
          isSearchable={true}
          components={selectComponents}
          placeholder={t("forms.indexer.chooseIndexer")}
          styles={selectStyles}
          theme={selectTheme}
          onChange={(option: unknown) => {
            form.reset();

            if (option != null) {
              const opt = option as SelectValue;
              form.setFieldValue("name", opt.label ?? "");
              form.setFieldValue(field.name, opt.value ?? "");

              const ind = data && data.find(i => i.identifier === opt.value);
              if (ind) {
                setIndexer(ind);
                form.setFieldValue("implementation", ind.implementation);

                if (ind.irc && ind.irc.settings) {
                  form.setFieldValue("base_url", ind.urls[0]);
                  ind.irc.settings.forEach((s) => {
                    form.setFieldValue(`irc.${s.name}`, s.default ?? "");
                  });
                }
              }
            }
          }}
          options={options}
        />
      )}
    </form.Field>
  );
}

interface TestApiButtonProps {
  values: IndexerUpdateInitialValues;
  show: boolean;
}

function TestApiButton({ values, show }: TestApiButtonProps) {
  const { t } = useTranslation("settings");
  const [isTesting, setIsTesting] = useState(false);
  const [isSuccessfulTest, setIsSuccessfulTest] = useState(false);
  const [isErrorTest, setIsErrorTest] = useState(false);

  const testApiMutation = useMutation({
    mutationFn: (req: IndexerTestApiReq) => APIClient.indexers.testApi(req),
    onMutate: () => {
      setIsTesting(true);
      setIsErrorTest(false);
      setIsSuccessfulTest(false);
    },
    onSuccess: () => {
      toast.custom((toastInstance) => <Toast type="success" body={t("forms.indexer.testApiSuccess")} t={toastInstance} />);

      sleep(1000)
        .then(() => {
          setIsTesting(false);
          setIsSuccessfulTest(true);
        })
        .then(() => {
          sleep(2500).then(() => {
            setIsSuccessfulTest(false);
          });
        });
    },
    onError: (error: Error) => {
      toast.custom((t) => <Toast type="error" body={error.message} t={t} />);

      setIsTesting(false);
      setIsErrorTest(true);
      sleep(2500).then(() => {
        setIsErrorTest(false);
      });
    }
  });

  const testApi = () => {
    if (!show) {
      return;
    }

    const req: IndexerTestApiReq = {
      id: values.id,
      api_key: values.settings.api_key ?? ""
    };

    if (values.settings.api_user) {
      req.api_user = values.settings.api_user;
    }

    testApiMutation.mutate(req);
  };

  if (!show) {
    return null;
  }

  return (
    <button
      type="button"
      className={classNames(
        isSuccessfulTest
          ? "text-green-500 border-green-500 bg-green-50"
          : isErrorTest
            ? "text-red-500 border-red-500 bg-red-50"
            : "border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 hover:bg-gray-50 focus:border-rose-700 active:bg-rose-700",
        isTesting ? "cursor-not-allowed" : "",
        "mr-2 float-left items-center px-4 py-2 border font-medium rounded-md shadow-xs text-sm transition ease-in-out duration-150 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
      )}
      disabled={isTesting}
      onClick={testApi}
    >
      {isTesting ? (
        <svg
          className="animate-spin h-5 w-5 text-green-500"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          ></circle>
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          ></path>
        </svg>
      ) : isSuccessfulTest ? (
        t("forms.indexer.ok")
      ) : isErrorTest ? (
        t("forms.indexer.error")
      ) : (
        t("forms.indexer.testApi")
      )}
    </button>
  );
}

interface IndexerUpdateInitialValues {
  id: number;
  name: string;
  enabled: boolean;
  identifier: string;
  identifier_external: string;
  implementation: string;
  base_url: string;
  use_proxy?: boolean;
  proxy_id?: number;
  settings: {
    api_key?: string;
    api_user?: string;
    authkey?: string;
    torrent_pass?: string;
  }
}

export function IndexerUpdateForm({ isOpen, toggle, data: indexer }: UpdateFormProps<IndexerDefinition>) {
  const { t } = useTranslation("settings");
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (indexer: Indexer) => APIClient.indexers.update(indexer),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: IndexerKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.indexer.updated", { name: indexer.name })} t={toastInstance} />);
      sleep(1500);

      toggle();
    }
  });

  const onSubmit = (data: unknown) => {
    // TODO clear data depending on type
    mutation.mutate(data as Indexer);
  };

  const deleteMutation = useMutation({
    mutationFn: (id: number) => APIClient.indexers.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: IndexerKeys.lists() });
      queryClient.invalidateQueries({ queryKey: IndexerKeys.options() });
      queryClient.invalidateQueries({ queryKey: ReleaseKeys.indexers() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.indexer.deleted", { name: indexer.name })} t={toastInstance} />);

      toggle();
    }
  });

  const deleteAction = () => deleteMutation.mutate(indexer.id ?? 0);

  const renderSettingFields = (settings: IndexerSetting[]) => {
    if (settings === undefined) {
      return null;
    }

    return (
      <div key="opt">
        {settings.map((f: IndexerSetting, idx: number) => {
          switch (f.type) {
          case "text": {
            return (
              <TextFieldWide name={`settings.${f.name}`} label={f.label} key={idx} help={f.help} />
            );
          }
          case "secret": {
            return (
              <PasswordFieldWide
                key={idx}
                name={`settings.${f.name}`}
                label={f.label}
                help={f.help}
                tooltip={
                  <div>
                    <p>{t("forms.indexer.secretFieldTooltip")}</p>
                    <br />
                    <DocsLink href="https://autobrr.com/faqs#common-action-rejections" />
                  </div>
                }
              />
            );
          }
          }
          return null;
        })}
      </div>
    );
  };

  const initialValues: IndexerUpdateInitialValues = {
    id: indexer.id,
    name: indexer.name,
    enabled: indexer.enabled || false,
    identifier: indexer.identifier,
    identifier_external: indexer.identifier_external,
    implementation: indexer.implementation,
    base_url: indexer.base_url,
    use_proxy: indexer.use_proxy,
    proxy_id: indexer.proxy_id,
    settings: indexer.settings?.reduce(
      (o: Record<string, string>, obj: IndexerSetting) => ({
        ...o,
        [obj.name]: obj.value
      } as Record<string, string>),
      {} as Record<string, string>
    )
  };

  return (
    <SlideOver
      type="UPDATE"
      title={t("forms.indexer.title")}
      isOpen={isOpen}
      toggle={toggle}
      deleteAction={deleteAction}
      onSubmit={onSubmit}
      initialValues={initialValues}
      extraButtons={(values) => <TestApiButton values={values} show={indexer.implementation === "irc" && indexer.supports.includes("api")} />}
    >
      {(values) => (
        <div className="py-2 space-y-6 sm:py-0 sm:space-y-0 divide-y divide-gray-200 dark:divide-gray-700">
          <div className="p-4 sm:grid sm:grid-cols-3 sm:gap-4">
            <label
              htmlFor="name"
              className="block text-sm font-medium text-gray-900 dark:text-white sm:mt-px sm:pt-2"
            >
              {t("forms.indexer.name")}
            </label>
            <IndexerNameField />
          </div>

          <TextFieldWide
            name="identifier_external"
            label={t("forms.indexer.externalIdentifier")}
            help={t("forms.indexer.externalIdentifierHelp", { name: indexer.name })}
            tooltip={
              <div>
                <p>{t("forms.indexer.externalIdentifierTooltip1")}</p>
                <br/>
                <p>{t("forms.indexer.externalIdentifierTooltip2", { name: indexer.name })}</p>
                <br/>
                <DocsLink href="https://autobrr.com/configuration/indexers#setup"/>
              </div>
            }
          />
          <SwitchGroupWide name="enabled" label={t("forms.indexer.enabled")}/>

          {indexer.implementation == "irc" && (
            <SelectFieldCreatable
              name="base_url"
              label={t("forms.indexer.baseUrl")}
              help={t("forms.indexer.baseUrlHelp")}
              options={indexer.urls.map(u => ({ value: u, label: u, key: u }))}
            />
          )}

          {renderSettingFields(indexer.settings)}

          <ProxyFields useProxy={values.use_proxy} />

          {(indexer.implementation === "torznab" || indexer.implementation === "newznab" || indexer.implementation === "rss") && (
            <div className="py-4 pt-6">
              <FeedSettingsBanner />
            </div>
          )}

        </div>
      )}
    </SlideOver>
  );
}

interface ProxyFieldsProps {
  useProxy?: boolean;
}

function ProxyFields({ useProxy }: ProxyFieldsProps) {
  const { t } = useTranslation("settings");
  const proxies = useQuery(ProxiesQueryOptions());

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="flex justify-between px-4">
        <div className="space-y-1">
          <h2 className="text-lg font-medium text-gray-900 dark:text-white">
            {t("forms.indexer.proxy")}
          </h2>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {t("forms.indexer.proxyDesc")}
          </p>
        </div>
        <SwitchButton name="use_proxy" />
      </div>

      {useProxy === true && (
        <div className="py-4 pt-6">
          <SelectField<number>
            name="proxy_id"
            label={t("forms.indexer.selectProxy")}
            placeholder={t("forms.indexer.selectProxyPlaceholder")}
            options={proxies.data ? proxies.data.map((p) => ({ label: p.name, value: p.id })) : []}
          />
        </div>
      )}
    </div>
  );
}

function IndexerNameField() {
  const form = useFormContext();

  return (
    <form.Field name="name">
      {(field) => (
        <div className="sm:col-span-2">
          <input
            type="text"
            name={field.name}
            value={field.state.value ?? ""}
            onChange={(e) => field.handleChange(e.target.value)}
            onBlur={field.handleBlur}
            className="block w-full shadow-xs sm:text-sm focus:ring-blue-500 focus:border-blue-500 border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-gray-100 rounded-md"
          />
          <common.ErrorField meta={field.state.meta} />
        </div>
      )}
    </form.Field>
  );
}

function FeedSettingsBanner() {
  const { t } = useTranslation("settings");
  return (
    <div className="px-4">
      <span className="w-full block px-2 py-2 bg-green-300 dark:bg-green-400 text-green-900 dark:text-green-900 text-sm rounded-sm">
        <span className="font-semibold">
          {t("forms.indexer.editFeeds")}
        </span>
      </span>
    </div>
  );
}
