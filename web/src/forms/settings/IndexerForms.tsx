/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Select from "react-select";
import { XMarkIcon } from "@heroicons/react/24/solid";
import { Dialog, DialogPanel, DialogTitle, Transition, TransitionChild } from "@headlessui/react";

import { classNames, sleep } from "@utils";
import { extractCategoryTreeFromCaps, flattenCategoryIds, parseCapabilitiesPayload } from "@utils/caps";
import { DEBUG } from "@components/debug";
import { APIClient } from "@api/APIClient";
import { FeedKeys, IndexerKeys, ReleaseKeys } from "@api/query_keys";
import { IndexersSchemaQueryOptions, ProxiesQueryOptions } from "@api/queries";
import { SlideOver } from "@components/panels";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { TextFieldWide, PasswordFieldWide, SwitchGroupWide } from "@components/inputs/tanstack/text_wide";
import { SwitchButton } from "@components/inputs/tanstack/switch";
import { SelectFieldBasic, SelectFieldCreatable } from "@components/inputs/tanstack/select_wide";
import { FeedDownloadTypeOptions } from "@domain/constants";
import { DocsLink } from "@components/ExternalLink";
import * as common from "@components/inputs/tanstack/common";
import { useAppForm, ContextField, useFormContext, useStore, useFieldContext } from "@app/lib/form";
import { AddFormProps, UpdateFormProps } from "@forms/_shared";

// const isRequired = (message: string) => (value?: string | undefined) => (!!value ? undefined : message);

function validateFieldTanstack(s: IndexerSetting) {
  return ({ value }: { value: string }) => {
    if (s.required) {
      if (s.default !== "") {
        if (value && s.default === value) {
          return "Default value, please edit";
        }
      }
      return value ? undefined : "Required";
    }
  };
}

const IrcSettingFields = (ind: IndexerDefinition, indexer: string) => {
  if (!indexer.length) {
    return null;
  }

  return (
    <>
      {ind && ind.irc && ind.irc.settings && (
        <div className="border-t border-gray-200 dark:border-gray-700 py-5">
          <div className="px-4">
            <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">IRC</DialogTitle>
            <p className="text-sm text-gray-500 dark:text-gray-200">
              Networks and channels are configured automatically in the background.
            </p>
          </div>

          {ind.irc.settings.map((f: IndexerSetting, idx: number) => {
            switch (f.type) {
            case "text": {
              return (
                <ContextField
                  key={idx}
                  name={`irc.${f.name}`}
                  validators={{ onChange: validateFieldTanstack(f) }}
                >
                  <TextFieldWide
                    label={f.label}
                    required={f.required}
                    help={f.help}
                    autoComplete="off"
                    tooltip={
                      <div>
                        <p>Please read our IRC guide if you are unfamiliar with IRC.</p>
                        <DocsLink href="https://autobrr.com/configuration/irc" />
                      </div>
                    }
                  />
                </ContextField>
              );
            }
            case "secret": {
              if (f.name === "invite_command") {
                return (
                  <ContextField key={idx} name={`irc.${f.name}`} validators={{ onChange: validateFieldTanstack(f) }}>
                    <PasswordFieldWide defaultVisible label={f.label} required={f.required} help={f.help} defaultValue={f.default} />
                  </ContextField>
                );
              }
              return (
                <ContextField key={idx} name={`irc.${f.name}`} validators={{ onChange: validateFieldTanstack(f) }}>
                  <PasswordFieldWide label={f.label} required={f.required} help={f.help} defaultValue={f.default} />
                </ContextField>
              );
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
  if (indexer !== "") {
    return (
      <Fragment>
        {ind && ind.torznab && ind.torznab.settings && (
          <div className="">
            <div className="pt-4 px-4">
              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">Torznab</DialogTitle>
              <p className="text-sm text-gray-500 dark:text-gray-200">
                Torznab feed
              </p>
            </div>

            <ContextField name="name">
              <TextFieldWide label="Name" defaultValue="" required={true} />
            </ContextField>

            <ContextField name="feed.url">
              <TextFieldWide
                label="URL"
                required={true}
                help="Torznab url. Just URL without extra params."
                tooltip={
                  <div>
                    <p>Prowlarr and Jackett have different formats:</p>
                    <br/>
                    <ul>
                      <li>Prowlarr: <code className="text-blue-400">http(s)://url.tld/indexerID/api</code></li>
                      <li>Jackett: <code className="text-blue-400">http(s)://url.tld/jackett/api/v2.0/indexers/indexerName/results/torznab/</code></li>
                    </ul>
                  </div>
                }
              />
            </ContextField>

            <ContextField name="feed.api_key">
              <PasswordFieldWide label="API key" help="API key" required={true} />
            </ContextField>

            <ContextField name="feed.settings.download_type">
              <SelectFieldBasic
                label="Download type"
                options={FeedDownloadTypeOptions}
                tooltip={<span>Some feeds needs to force set as Magnet.</span>}
                help="Set to Torrent or Magnet depending on indexer."
              />
            </ContextField>

            <FeedCategoriesDraftSection feedType="TORZNAB" />
          </div>
        )}
      </Fragment>
    );
  }
};

const NewznabFeedSettingFields = (ind: IndexerDefinition, indexer: string) => {
  if (indexer !== "") {
    return (
      <Fragment>
        {ind && ind.newznab && ind.newznab.settings && (
          <div className="">
            <div className="pt-4 px-4">
              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">Newznab</DialogTitle>
              <p className="text-sm text-gray-500 dark:text-gray-200">
                Newznab feed
              </p>
            </div>

            <ContextField name="name">
              <TextFieldWide label="Name" defaultValue="" required={true} />
            </ContextField>

            <ContextField name="feed.newznab_url">
              <TextFieldWide
                label="URL"
                required={true}
                help="Newznab url. Just URL without extra params."
                tooltip={
                  <div>
                    <p>Prowlarr and Jackett have different formats:</p>
                    <br/>
                    <ul>
                      <li>Prowlarr: <code className="text-blue-400">http(s)://url.tld/indexerID/api</code></li>
                      <li>Jackett: <code className="text-blue-400">http(s)://url.tld/jackett/api/v2.0/indexers/indexerName/results/newznab/</code></li>
                    </ul>
                  </div>
                }
              />
            </ContextField>

            <ContextField name="feed.api_key">
              <PasswordFieldWide label="API key" help="API key" required={true} />
            </ContextField>

            {ind.newznab.settings.map((f: IndexerSetting, idx: number) => {
              switch (f.type) {
              case "text": {
                return (
                  <ContextField key={idx} name={`feed.${f.name}`} validators={{ onChange: validateFieldTanstack(f) }}>
                    <TextFieldWide label={f.label} required={f.required} help={f.help} autoComplete="off" />
                  </ContextField>
                );
              }
              case "secret": {
                return (
                  <ContextField key={idx} name={`feed.${f.name}`} validators={{ onChange: validateFieldTanstack(f) }}>
                    <PasswordFieldWide label={f.label} required={f.required} help={f.help} defaultValue={f.default} />
                  </ContextField>
                );
              }
              }
              return null;
            })}

            <FeedCategoriesDraftSection feedType="NEWZNAB" />
          </div>
        )}
      </Fragment>
    );
  }
};

const RSSFeedSettingFields = (ind: IndexerDefinition, indexer: string) => {
  if (indexer !== "") {
    return (
      <Fragment>
        {ind && ind.rss && ind.rss.settings && (
          <div className="">
            <div className="pt-4 px-4">
              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">RSS</DialogTitle>
              <p className="text-sm text-gray-500 dark:text-gray-200">
                RSS feed
              </p>
            </div>

            <ContextField name="name">
              <TextFieldWide label="Name" defaultValue="" />
            </ContextField>

            {ind.rss.settings.map((f: IndexerSetting, idx: number) => {
              switch (f.type) {
              case "text": {
                return (
                  <ContextField key={idx} name={`feed.${f.name}`} validators={{ onChange: validateFieldTanstack(f) }}>
                    <TextFieldWide label={f.label} required={f.required} help={f.help} autoComplete="off" />
                  </ContextField>
                );
              }
              case "secret": {
                return (
                  <ContextField key={idx} name={`feed.${f.name}`} validators={{ onChange: validateFieldTanstack(f) }}>
                    <PasswordFieldWide label={f.label} required={f.required} help={f.help} defaultValue={f.default} />
                  </ContextField>
                );
              }
              }
              return null;
            })}

            <ContextField name="feed.settings.download_type">
              <SelectFieldBasic
                label="Download type"
                options={FeedDownloadTypeOptions}
                tooltip={<span>Some feeds needs to force set as Magnet.</span>}
                help="Set to Torrent or Magnet depending on indexer."
              />
            </ContextField>
          </div>
        )}
      </Fragment>
    );
  }
};

function FeedCategoriesDraftSection({ feedType }: { feedType: FeedType }) {
  const form = useFormContext();
  const feedValues = useStore(form.store, (s: any) => s.values.feed ?? {}) as Record<string, unknown>;
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

      (form as any).setFieldValue("feed.capabilities", caps ?? null);
      (form as any).setFieldValue("feed.categories", filteredSelection);
    },
    onError: (error: unknown) => {
      const message = error instanceof Error ? error.message : "Failed to fetch categories";
      toast.custom((t) => <Toast type="error" body={message} t={t} />);
    }
  });

  const toggleCategory = (id: number) => {
    if (categoriesValue.includes(id)) {
      (form as any).setFieldValue(
        "feed.categories",
        categoriesValue.filter((category) => category !== id)
      );
      return;
    }

    (form as any).setFieldValue("feed.categories", [...categoriesValue, id]);
  };

  const toggleParentCategory = (id: number, childIds: number[]) => {
    if (categoriesValue.includes(id)) {
      (form as any).setFieldValue(
        "feed.categories",
        categoriesValue.filter((category) => category !== id)
      );
      return;
    }

    (form as any).setFieldValue(
      "feed.categories",
      [...categoriesValue.filter((category) => !childIds.includes(category)), id]
    );
  };

  return (
    <div className="mt-6 border-t border-gray-200 dark:border-gray-700">
      <div className="pt-4 px-4 flex items-center justify-between">
        <div>
          <div className="text-lg font-medium text-gray-900 dark:text-white">Categories</div>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Fetch available categories and select what to include.
          </p>
        </div>
        <button
          type="button"
          className="inline-flex items-center rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-1.5 cursor-pointer text-sm font-medium text-gray-700 dark:text-gray-200 shadow-xs hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-blue-500"
          onClick={() => fetchCapsMutation.mutate()}
          disabled={!canFetch || fetchCapsMutation.isPending}
          title={!canFetch ? "Enter a URL to fetch categories" : undefined}
        >
          {fetchCapsMutation.isPending ? "Fetching" : hasCaps ? "Refetch" : "Fetch"}
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
                      className="h-5 w-5 rounded border-gray-300 text-blue-600 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-800"
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
          {hasCaps ? "No categories found." : "Fetch categories to select."}
        </div>
      )}
    </div>
  );
}

const SettingFields = (ind: IndexerDefinition, indexer: string) => {
  if (indexer !== "") {
    return (
      <div key="opt">
        {ind && ind.settings && ind.settings.map((f, idx: number) => {
          switch (f.type) {
          case "text": {
            return (
              <ContextField key={idx} name={`settings.${f.name}`} validators={{ onChange: validateFieldTanstack(f) }}>
                <TextFieldWide label={f.label} required={f.required} help={f.help} autoComplete="off" />
              </ContextField>
            );
          }
          case "secret": {
            return (
              <ContextField key={idx} name={`settings.${f.name}`} validators={{ onChange: validateFieldTanstack(f) }}>
                <PasswordFieldWide
                  label={f.label}
                  required={f.required}
                  help={f.help}
                  tooltip={
                    <div>
                      <p>This field does not take a full URL. Only use alphanumeric strings like <code>uqcdi67cibkx3an8cmdm</code>.</p>
                      <br />
                      <DocsLink href="https://autobrr.com/faqs#common-action-rejections" />
                    </div>
                  }
                />
              </ContextField>
            );
          }
          }
          return null;
        })}
        <div hidden={true}>
          <ContextField name="name">
            <TextFieldWide label="Name" defaultValue={ind?.name} hidden={true} />
          </ContextField>
        </div>
      </div>
    );
  }
};

type SelectValue = {
  label: string;
  value: string;
};

export function IndexerAddForm({ isOpen, toggle }: AddFormProps) {
  const [indexer, setIndexer] = useState<IndexerDefinition>({} as IndexerDefinition);

  const queryClient = useQueryClient();
  const { data } = useQuery(IndexersSchemaQueryOptions(isOpen));

  const mutation = useMutation({
    mutationFn: (indexer: Indexer) => APIClient.indexers.create(indexer),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: IndexerKeys.lists() });
      queryClient.invalidateQueries({ queryKey: IndexerKeys.options() });
      queryClient.invalidateQueries({ queryKey: ReleaseKeys.indexers() });

      toast.custom((t) => <Toast type="success" body="Indexer was added" t={t} />);
      sleep(1500);
      toggle();
    },
    onError: () => {
      toast.custom((t) => <Toast type="error" body="Indexer could not be added" t={t} />);
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

  const onSubmit = (formData: Record<string, any>) => {
    const ind = data && data.find(i => i.identifier === formData.identifier);
    if (!ind)
      return;

    if (formData.implementation === "torznab") {
      const createFeed: FeedCreate = {
        name: formData.name,
        enabled: false,
        type: "TORZNAB",
        url: formData.feed.url,
        api_key: formData.feed.api_key,
        interval: 30,
        timeout: 60,
        indexer_id: 0,
        categories: formData.feed.categories ?? [],
        capabilities: formData.feed.capabilities ?? null,
        settings: formData.feed.settings
      };

      mutation.mutate(formData as Indexer, {
        onSuccess: (indexer) => {
          // @eslint-ignore
          createFeed.indexer_id = indexer.id;

          feedMutation.mutate(createFeed);
        }
      });
      return;

    } else if (formData.implementation === "newznab") {
      formData.url = formData.feed.url;

      const createFeed: FeedCreate = {
        name: formData.name,
        enabled: false,
        type: "NEWZNAB",
        url: formData.feed.newznab_url,
        api_key: formData.feed.api_key,
        interval: 30,
        timeout: 60,
        indexer_id: 0,
        categories: formData.feed.categories ?? [],
        capabilities: formData.feed.capabilities ?? null,
        settings: formData.feed.settings
      };

      mutation.mutate(formData as Indexer, {
        onSuccess: (indexer) => {
          // @eslint-ignore
          createFeed.indexer_id = indexer.id;

          feedMutation.mutate(createFeed);
        }
      });
      return;

    } else if (formData.implementation === "rss") {
      const createFeed: FeedCreate = {
        name: formData.name,
        enabled: false,
        type: "RSS",
        url: formData.feed.url,
        interval: 30,
        timeout: 60,
        indexer_id: 0,
        settings: formData.feed.settings
      };

      mutation.mutate(formData as Indexer, {
        onSuccess: (indexer) => {
          // @eslint-ignore
          createFeed.indexer_id = indexer.id;

          feedMutation.mutate(createFeed);
        }
      });
      return;

    } else if (formData.implementation === "irc") {
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
            name: element,
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
        nick: formData.irc.nick,
        auth: {
          mechanism: "NONE"
          // account: formData.irc.auth.account,
          // password: formData.irc.auth.password
        },
        invite_command: formData.irc.invite_command,
        channels: channels
      };

      if (formData.irc.auth) {
        if (formData.irc.auth.account !== "" && formData.irc.auth.password !== "") {
          network.auth.mechanism = "SASL_PLAIN";
          network.auth.account = formData.irc.auth.account;
          network.auth.password = formData.irc.auth.password;
        }
      }

      mutation.mutate(formData as Indexer, {
        onSuccess: () => {
          ircMutation.mutate(network);
        }
      });
    }
  };

  const form = useAppForm({
    defaultValues: {
      enabled: true,
      identifier: "",
      implementation: "irc",
      name: "",
      irc: {} as Record<string, any>,
      settings: {} as Record<string, any>,
      feed: {
        categories: [] as number[],
        capabilities: null as unknown,
        settings: {} as Record<string, any>
      } as Record<string, any>,
      base_url: ""
    },
    onSubmit: async ({ value }) => {
      onSubmit(value as Record<string, any>);
    }
  });

  const valuesIdentifier = useStore(form.store, (s: any) => s.values.identifier);
  const valuesAll = useStore(form.store, (s: any) => s.values);

  return (
    <Transition show={isOpen} as={Fragment}>
      <Dialog as="div" static className="fixed inset-0 overflow-hidden" open={isOpen} onClose={toggle}>
        <div className="absolute inset-0 overflow-hidden">
          <DialogPanel className="absolute inset-y-0 right-0 max-w-full flex">
            <TransitionChild
              as={Fragment}
              enter="transform transition ease-in-out duration-500 sm:duration-700"
              enterFrom="translate-x-full"
              enterTo="translate-x-0"
              leave="transform transition ease-in-out duration-500 sm:duration-700"
              leaveFrom="translate-x-0"
              leaveTo="translate-x-full"
            >
              <div className="w-screen max-w-2xl">
                <form.AppForm>
                  <form
                    className="h-full flex flex-col bg-white dark:bg-gray-800 shadow-xl overflow-y-auto"
                    onSubmit={(e) => {
                      e.preventDefault();
                      form.handleSubmit();
                    }}
                  >
                    <div className="flex-1">
                      <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
                        <div className="flex items-start justify-between space-x-3">
                          <div className="space-y-1">
                            <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
                              Add indexer
                            </DialogTitle>
                            <p className="text-sm text-gray-500 dark:text-gray-200">
                              Add indexer.
                            </p>
                          </div>
                          <div className="h-7 flex items-center">
                            <button
                              type="button"
                              className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500"
                              onClick={toggle}
                            >
                              <span className="sr-only">Close panel</span>
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
                              Indexer
                            </label>
                          </div>
                          <div className="sm:col-span-2">
                            <Select
                              isClearable={true}
                              isSearchable={true}
                              components={{
                                Input: common.SelectInput,
                                Control: common.SelectControl,
                                Menu: common.SelectMenu,
                                Option: common.SelectOption,
                                IndicatorSeparator: common.IndicatorSeparator,
                                DropdownIndicator: common.DropdownIndicator
                              }}
                              placeholder="Choose an indexer"
                              styles={{
                                singleValue: (base) => ({
                                  ...base,
                                  color: "unset"
                                })
                              }}
                              theme={(theme) => ({
                                ...theme,
                                spacing: {
                                  ...theme.spacing,
                                  controlHeight: 30,
                                  baseUnit: 2
                                }
                              })}
                              value={valuesIdentifier ? { label: valuesIdentifier, value: valuesIdentifier } : null}
                              onChange={(option: unknown) => {
                                form.reset();

                                if (option != null) {
                                  const opt = option as SelectValue;
                                  (form as any).setFieldValue("name", opt.label ?? "");
                                  (form as any).setFieldValue("identifier", opt.value ?? "");

                                  const ind = data && data.find(i => i.identifier === opt.value);
                                  if (ind) {
                                    setIndexer(ind);
                                    (form as any).setFieldValue("implementation", ind.implementation);

                                    if (ind.irc && ind.irc.settings) {
                                      (form as any).setFieldValue("base_url", ind.urls[0]);
                                      ind.irc.settings.forEach((s) => {
                                        (form as any).setFieldValue(`irc.${s.name}`, s.default ?? "");
                                      });
                                    }
                                  }
                                }
                              }}
                              options={data && data.sort((a, b) => a.name.localeCompare(b.name)).map(v => ({
                                label: v.name,
                                value: v.identifier
                              }))}
                            />
                          </div>
                        </div>

                        <ContextField name="enabled">
                          <SwitchGroupWide label="Enabled" />
                        </ContextField>

                        {indexer.implementation == "irc" && (
                          <ContextField name="base_url">
                            <SelectFieldCreatable
                              label="Base URL"
                              help="Override baseurl if it's blocked by your ISP."
                              options={indexer.urls.map(u => ({ value: u, label: u, key: u }))}
                            />
                          </ContextField>
                        )}

                        {SettingFields(indexer, valuesIdentifier)}

                      </div>

                      {IrcSettingFields(indexer, valuesIdentifier)}
                      {TorznabFeedSettingFields(indexer, valuesIdentifier)}
                      {NewznabFeedSettingFields(indexer, valuesIdentifier)}
                      {RSSFeedSettingFields(indexer, valuesIdentifier)}
                    </div>

                    <div
                      className="shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-5 sm:px-6">
                      <div className="space-x-3 flex justify-end">
                        <button
                          type="button"
                          className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                          onClick={toggle}
                        >
                          Cancel
                        </button>
                        <button
                          type="submit"
                          className="inline-flex justify-center py-2 px-4 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                        >
                          Save
                        </button>
                      </div>
                    </div>

                    <DEBUG values={valuesAll} />
                  </form>
                </form.AppForm>
              </div>
            </TransitionChild>
          </DialogPanel>
        </div>
      </Dialog>
    </Transition>
  );
}

interface TestApiButtonProps {
  values: Record<string, any>;
  show: boolean;
}

function TestApiButton({ values, show }: TestApiButtonProps) {
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
      toast.custom((t) => <Toast type="success" body="API test successful!" t={t} />);

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
      api_key: values.settings.api_key
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
        "OK!"
      ) : isErrorTest ? (
        "ERROR"
      ) : (
        "Test API"
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

function ProxySelectField({ options, placeholder }: { options: { label: string; value: number }[]; placeholder?: string }) {
  const field = useFieldContext<number>();

  return (
    <div className="flex items-center justify-between space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label
          htmlFor={field.name}
          className="block text-sm font-medium text-gray-900 dark:text-white"
        >
          Select proxy
        </label>
      </div>
      <div className="sm:col-span-2">
        <Select
          id={field.name}
          isClearable={true}
          isSearchable={true}
          components={{
            Input: common.SelectInput,
            Control: common.SelectControl,
            Menu: common.SelectMenu,
            Option: common.SelectOption,
            IndicatorSeparator: common.IndicatorSeparator,
            DropdownIndicator: common.DropdownIndicator
          }}
          placeholder={placeholder ?? "Select a proxy"}
          styles={{
            singleValue: (base) => ({
              ...base,
              color: "unset"
            })
          }}
          theme={(theme) => ({
            ...theme,
            spacing: {
              ...theme.spacing,
              controlHeight: 30,
              baseUnit: 2
            }
          })}
          value={field.state.value ? options.find(o => o.value == field.state.value) ?? null : null}
          onChange={(newValue: unknown) => {
            if (newValue) {
              field.handleChange((newValue as { value: number }).value);
            } else {
              field.handleChange(0);
            }
          }}
          options={options}
        />
      </div>
    </div>
  );
}

export function IndexerUpdateForm({ isOpen, toggle, data: indexer }: UpdateFormProps<IndexerDefinition>) {
  const queryClient = useQueryClient();

  const proxies = useQuery(ProxiesQueryOptions());

  const mutation = useMutation({
    mutationFn: (indexer: Indexer) => APIClient.indexers.update(indexer),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: IndexerKeys.lists() });

      toast.custom((t) => <Toast type="success" body={`${indexer.name} was updated successfully`} t={t} />);
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

      toast.custom((t) => <Toast type="success" body={`${indexer.name} was deleted.`} t={t} />);

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
              <ContextField key={idx} name={`settings.${f.name}`}>
                <TextFieldWide label={f.label} help={f.help} />
              </ContextField>
            );
          }
          case "secret": {
            return (
              <ContextField key={idx} name={`settings.${f.name}`}>
                <PasswordFieldWide
                  label={f.label}
                  help={f.help}
                  tooltip={
                    <div>
                      <p>This field does not take a full URL. Only use alphanumeric strings like <code>uqcdi67cibkx3an8cmdm</code>.</p>
                      <br />
                      <DocsLink href="https://autobrr.com/faqs#common-action-rejections" />
                    </div>
                  }
                />
              </ContextField>
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
      title="Indexer"
      isOpen={isOpen}
      toggle={toggle}
      deleteAction={deleteAction}
      onSubmit={onSubmit}
      initialValues={initialValues}
      extraButtons={(values) => <TestApiButton values={values as Record<string, any>} show={indexer.implementation === "irc" && indexer.supports.includes("api")} />}
    >
      {(values) => (
        <div className="py-2 space-y-6 sm:py-0 sm:space-y-0 divide-y divide-gray-200 dark:divide-gray-700">
          <div className="p-4 sm:grid sm:grid-cols-3 sm:gap-4">
            <label
              htmlFor="name"
              className="block text-sm font-medium text-gray-900 dark:text-white sm:mt-px sm:pt-2"
            >
              Name
            </label>
            <ContextField name="name">
              <NameFieldInline />
            </ContextField>
          </div>

          <ContextField name="identifier_external">
            <TextFieldWide
              label="External Identifier"
              help={`External Identifier for ARRs. If using Prowlarr set like: ${indexer.name} (Prowlarr)`}
              tooltip={
                <div>
                  <p>External Identifier for use with ARRs to get features like seed limits working.</p>
                  <br/>
                  <p>This needs to match the indexer name in your ARR. If using Prowlarr it will likely be
                    "{indexer.name} (Prowlarr)"</p>
                  <br/>
                  <DocsLink href="https://autobrr.com/configuration/indexers#setup"/>
                </div>
              }
            />
          </ContextField>
          <ContextField name="enabled">
            <SwitchGroupWide label="Enabled"/>
          </ContextField>

          {indexer.implementation == "irc" && (
            <ContextField name="base_url">
              <SelectFieldCreatable
                label="Base URL"
                help="Override baseurl if it's blocked by your ISP."
                options={indexer.urls.map(u => ({ value: u, label: u, key: u }))}
              />
            </ContextField>
          )}

          {renderSettingFields(indexer.settings)}

          <div className="border-t border-gray-200 dark:border-gray-700 py-4">
            <div className="flex justify-between px-4">
              <div className="space-y-1">
                <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
                  Proxy
                </DialogTitle>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Set a proxy to be used for downloads of .torrent files and feeds.
                </p>
              </div>
              <ContextField name="use_proxy">
                <SwitchButton />
              </ContextField>
            </div>

            {values.use_proxy === true && (
              <div className="py-4 pt-6">
                <ContextField name="proxy_id">
                  <ProxySelectField
                    placeholder="Select a proxy"
                    options={proxies.data ? proxies.data.map((p) => ({ label: p.name, value: p.id })) : []}
                  />
                </ContextField>
              </div>
            )}
          </div>
        </div>
      )}
    </SlideOver>
  );
}

function NameFieldInline() {
  const field = useFieldContext<string>();

  return (
    <div className="sm:col-span-2">
      <input
        type="text"
        value={field.state.value ?? ""}
        onChange={(e) => field.handleChange(e.target.value)}
        onBlur={field.handleBlur}
        className="block w-full shadow-xs sm:text-sm focus:ring-blue-500 focus:border-blue-500 border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-gray-100 rounded-md"
      />
      {field.state.meta.isTouched && field.state.meta.errors.length > 0 && (
        <span>{field.state.meta.errors.join(", ")}</span>
      )}
    </div>
  );
}
