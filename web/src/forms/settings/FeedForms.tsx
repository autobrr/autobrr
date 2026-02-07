/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useFormikContext } from "formik";

import { APIClient } from "@api/APIClient";
import { FeedKeys } from "@api/query_keys";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { SlideOver } from "@components/panels";
import { NumberFieldWide, PasswordFieldWide, SwitchGroupWide, TextFieldWide } from "@components/inputs";
import { SelectFieldBasic } from "@components/inputs/select_wide";
import { sleep } from "@utils";
import { ImplementationBadges } from "@screens/settings/Indexer";
import { FeedDownloadTypeOptions } from "@domain/constants";
import { UpdateFormProps } from "@forms/_shared";
import { extractCategoryTreeFromCaps, flattenCategoryIds, parseCapabilitiesPayload } from "@utils/caps";

interface InitialValues {
  id: number;
  indexer: IndexerMinimal;
  enabled: boolean;
  type: FeedType;
  name: string;
  url: string;
  api_key: string;
  cookie: string;
  tls_skip_verify: boolean;
  interval: number;
  timeout: number;
  max_age: number;
  categories: number[];
  capabilities: FeedCaps | null;
  settings: FeedSettings;
}

export function FeedUpdateForm({ isOpen, toggle, data}: UpdateFormProps<Feed>) {
  const feed = data;
  const [isTesting, setIsTesting] = useState(false);
  const [isTestSuccessful, setIsSuccessfulTest] = useState(false);
  const [isTestError, setIsErrorTest] = useState(false);

  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (feed: Feed) => APIClient.feeds.update(feed),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: FeedKeys.lists() });
      
      toast.custom((t) => <Toast type="success" body={`${feed.name} was updated successfully`} t={t} />);
      toggle();
    }
  });

  const onSubmit = (formData: unknown) => mutation.mutate(formData as Feed);

  const deleteMutation = useMutation({
    mutationFn: (feedID: number) => APIClient.feeds.delete(feedID),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: FeedKeys.lists() });

      toast.custom((t) => <Toast type="success" body={`${feed.name} was deleted.`} t={t} />);
    }
  });

  const deleteAction = () => deleteMutation.mutate(feed.id);

  const testFeedMutation = useMutation({
    mutationFn: (feed: Feed) => APIClient.feeds.test(feed),
    onMutate: () => {
      setIsTesting(true);
      setIsErrorTest(false);
      setIsSuccessfulTest(false);
    },
    onSuccess: () => {
      toast.custom((t) => <Toast type="success" body={`${feed.name} test OK!`} t={t} />);

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
    onError: () => {
      setIsTesting(false);
      setIsErrorTest(true);
      sleep(2500).then(() => {
        setIsErrorTest(false);
      });
    }
  });

  const testFeed = (data: unknown) => testFeedMutation.mutate(data as Feed);

  const initialValues: InitialValues = {
    id: feed.id,
    indexer: feed.indexer,
    enabled: feed.enabled,
    type: feed.type,
    name: feed.name,
    url: feed.url,
    api_key: feed.api_key,
    cookie: feed.cookie || "",
    tls_skip_verify: feed.tls_skip_verify ?? false,
    interval: feed.interval,
    timeout: feed.timeout,
    max_age: feed.max_age,
    categories: feed.categories || [],
    capabilities: feed.capabilities || null,
    settings: feed.settings
  };

  return (
    <SlideOver<InitialValues>
      type="UPDATE"
      title="Feed"
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      deleteAction={deleteAction}
      initialValues={initialValues}
      testFn={testFeed}
      isTesting={isTesting}
      isTestSuccessful={isTestSuccessful}
      isTestError={isTestError}
    >
      {(values) => (
        <div>
          <TextFieldWide name="name" label="Name" required={true} />

          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            <div
              className="py-4 flex items-center justify-between space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:py-4">
              <div>
                <label
                  htmlFor="type"
                  className="block text-sm font-medium text-gray-900 dark:text-white"
                >
                  Type
                </label>
              </div>
              <div className="flex justify-end sm:col-span-2">
                {ImplementationBadges[feed.type.toLowerCase()]}
              </div>
            </div>

            <div className="py-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
              <SwitchGroupWide name="enabled" label="Enabled" />
            </div>
          </div>
          {values.type === "TORZNAB" && <FormFieldsTorznab feedID={feed.id} />}
          {values.type === "NEWZNAB" && <FormFieldsNewznab feedID={feed.id} />}
          {values.type === "RSS" && <FormFieldsRSS />}
        </div>
      )}
    </SlideOver>
  );
}

function WarningLabel() {
  return (
    <div className="px-4 py-1">
      <span className="w-full block px-2 py-2 bg-red-300 dark:bg-red-400 text-red-900 dark:text-red-900 text-sm rounded-sm">
        <span className="font-semibold">
          Warning: Indexers might ban you for too low interval!
        </span>
        <span className="ml-1">
          Read the indexer rules.
        </span>
      </span>
    </div>
  );
}

function FormFieldsTorznab({ feedID }: { feedID: number }) {
  const {
    values: { interval }
  } = useFormikContext<InitialValues>();

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-5">
      <TextFieldWide
        name="url"
        label="URL"
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

      <SelectFieldBasic name="settings.download_type" label="Download type" options={FeedDownloadTypeOptions} />

      <PasswordFieldWide name="api_key" label="API key" />

      <SwitchGroupWide name="tls_skip_verify" label="Skip TLS verification (insecure)" />

      {interval < 15 && <WarningLabel />}
      <NumberFieldWide name="interval" label="Refresh interval" help="Minutes. Recommended 15-30. Too low and risk ban."/>

      <NumberFieldWide name="timeout" label="Refresh timeout" help="Seconds to wait before cancelling refresh."/>
      <NumberFieldWide name="max_age" label="Max age" help="Enter the maximum age of feed content in seconds. It is recommended to set this to '0' to disable the age filter, ensuring all items in the feed are processed."/>

      <FeedCategoriesSection feedID={feedID} />
    </div>
  );
}

function FormFieldsNewznab({ feedID }: { feedID: number }) {
  const {
    values: { interval }
  } = useFormikContext<InitialValues>();

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-5">
      <TextFieldWide
        name="url"
        label="URL"
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

      <PasswordFieldWide name="api_key" label="API key" />

      <SwitchGroupWide name="tls_skip_verify" label="Skip TLS verification (insecure)" />

      {interval < 15 && <WarningLabel />}
      <NumberFieldWide name="interval" label="Refresh interval" help="Minutes. Recommended 15-30. Too low and risk ban."/>

      <NumberFieldWide name="timeout" label="Refresh timeout" help="Seconds to wait before cancelling refresh."/>
      <NumberFieldWide name="max_age" label="Max age" help="Enter the maximum age of feed content in seconds. It is recommended to set this to '0' to disable the age filter, ensuring all items in the feed are processed."/>

      <FeedCategoriesSection feedID={feedID} />
    </div>
  );
}

function FormFieldsRSS() {
  const {
    values: { interval }
  } = useFormikContext<InitialValues>();

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-5">
      <TextFieldWide
        name="url"
        label="URL"
        help="RSS url"
      />

      <SelectFieldBasic name="settings.download_type" label="Download type" options={FeedDownloadTypeOptions} />

      <SwitchGroupWide name="tls_skip_verify" label="Skip TLS verification (insecure)" />

      {interval < 15 && <WarningLabel />}
      <NumberFieldWide name="interval" label="Refresh interval" help="Minutes. Recommended 15-30. Too low and risk ban."/>
      <NumberFieldWide name="timeout" label="Refresh timeout" help="Seconds to wait before cancelling refresh."/>
      <NumberFieldWide name="max_age" label="Max age" help="Enter the maximum age of feed content in seconds. It is recommended to set this to '0' to disable the age filter, ensuring all items in the feed are processed."/>

      <PasswordFieldWide name="cookie" label="Cookie" help="Not commonly used" />
    </div>
  );
}

function FeedCategoriesSection({ feedID }: { feedID: number }) {
  const { values, setFieldValue } = useFormikContext<InitialValues>();
  const capsPayload = useMemo(() => parseCapabilitiesPayload(values.capabilities), [values.capabilities]);
  const categoriesTree = useMemo(() => extractCategoryTreeFromCaps(capsPayload), [capsPayload]);
  const hasCaps = Boolean(values.capabilities);

  const fetchCapsMutation = useMutation({
    mutationFn: () => APIClient.feeds.fetchCaps(feedID),
    onSuccess: (caps) => {
      const nextCategories = flattenCategoryIds(extractCategoryTreeFromCaps(caps));
      const selected = values.categories ?? [];

      setFieldValue("capabilities", caps ?? null);
      setFieldValue(
        "categories",
        selected.filter((id) => nextCategories.includes(id))
      );
    },
    onError: (error: unknown) => {
      const message = error instanceof Error ? error.message : "Failed to fetch categories";
      toast.custom((t) => <Toast type="error" body={message} t={t} />);
    }
  });

  const toggleCategory = (id: number) => {
    const selected = values.categories ?? [];
    if (selected.includes(id)) {
      setFieldValue(
        "categories",
        selected.filter((category) => category !== id)
      );
      return;
    }

    setFieldValue("categories", [...selected, id]);
  };

  const toggleParentCategory = (id: number, childIds: number[]) => {
    const selected = values.categories ?? [];
    if (selected.includes(id)) {
      setFieldValue(
        "categories",
        selected.filter((category) => category !== id)
      );
      return;
    }

    setFieldValue(
      "categories",
      [...selected.filter((category) => !childIds.includes(category)), id]
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
          disabled={fetchCapsMutation.isPending}
        >
          {fetchCapsMutation.isPending ? "Fetching" : hasCaps ? "Refetch" : "Fetch"}
        </button>
      </div>

      {categoriesTree.length ? (
        <div className="px-4 pt-4 pb-2 space-y-3 overflow-y-auto">
          {categoriesTree.map((category) => {
            const childIds = category.subcategories.map((sub) => sub.id);
            const isParentSelected = (values.categories ?? []).includes(category.id);

            return (
              <div key={category.id} className="space-y-2">
                <label
                  className="flex items-center justify-between gap-3 text-sm text-gray-700 dark:text-gray-200"
                  onClick={(event) => event.stopPropagation()}
                >
                  <span className="flex items-center gap-3">
                    <input
                      type="checkbox"
                      checked={(values.categories ?? []).includes(category.id)}
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
                    className="flex items-center justify-between gap-3 pl-6 text-sm text-gray-700 dark:text-gray-200"
                    onClick={(event) => event.stopPropagation()}
                  >
                    <span className="flex items-center gap-3">
                      <input
                        type="checkbox"
                        checked={(values.categories ?? []).includes(subCategory.id)}
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
