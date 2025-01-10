/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useFormikContext } from "formik";

import { APIClient } from "@api/APIClient";
import { FeedKeys } from "@api/query_keys";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { SlideOver } from "@components/panels";
import { NumberFieldWide, PasswordFieldWide, SwitchGroupWide, TextFieldWide } from "@components/inputs";
import { SelectFieldBasic } from "@components/inputs/select_wide";
import { componentMapType } from "./DownloadClientForms";
import { sleep } from "@utils";
import { ImplementationBadges } from "@screens/settings/Indexer";
import { FeedDownloadTypeOptions } from "@domain/constants";
import { UpdateFormProps } from "@forms/_shared";

interface InitialValues {
  id: number;
  indexer: IndexerMinimal;
  enabled: boolean;
  type: FeedType;
  name: string;
  url: string;
  api_key: string;
  cookie: string;
  interval: number;
  timeout: number;
  max_age: number;
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
    interval: feed.interval,
    timeout: feed.timeout,
    max_age: feed.max_age,
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

          <div className="space-y-4 divide-y divide-gray-200 dark:divide-gray-700">
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
          {componentMap[values.type]}
        </div>
      )}
    </SlideOver>
  );
}

function WarningLabel() {
  return (
    <div className="px-4 py-1">
      <span className="w-full block px-2 py-2 bg-red-300 dark:bg-red-400 text-red-900 dark:text-red-900 text-sm rounded">
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

function FormFieldsTorznab() {
  const {
    values: { interval }
  } = useFormikContext<InitialValues>();

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-5">
      <TextFieldWide
        name="url"
        label="URL"
        help="Torznab url"
      />

      <SelectFieldBasic name="settings.download_type" label="Download type" options={FeedDownloadTypeOptions} />

      <PasswordFieldWide name="api_key" label="API key" />

      {interval < 15 && <WarningLabel />}
      <NumberFieldWide name="interval" label="Refresh interval" help="Minutes. Recommended 15-30. Too low and risk ban."/>

      <NumberFieldWide name="timeout" label="Refresh timeout" help="Seconds to wait before cancelling refresh."/>
      <NumberFieldWide name="max_age" label="Max age" help="Enter the maximum age of feed content in seconds. It is recommended to set this to '0' to disable the age filter, ensuring all items in the feed are processed."/>
    </div>
  );
}

function FormFieldsNewznab() {
  const {
    values: { interval }
  } = useFormikContext<InitialValues>();

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-5">
      <TextFieldWide
        name="url"
        label="URL"
        help="Newznab url"
      />

      <PasswordFieldWide name="api_key" label="API key" />

      {interval < 15 && <WarningLabel />}
      <NumberFieldWide name="interval" label="Refresh interval" help="Minutes. Recommended 15-30. Too low and risk ban."/>

      <NumberFieldWide name="timeout" label="Refresh timeout" help="Seconds to wait before cancelling refresh."/>
      <NumberFieldWide name="max_age" label="Max age" help="Enter the maximum age of feed content in seconds. It is recommended to set this to '0' to disable the age filter, ensuring all items in the feed are processed."/>
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

      {interval < 15 && <WarningLabel />}
      <NumberFieldWide name="interval" label="Refresh interval" help="Minutes. Recommended 15-30. Too low and risk ban."/>
      <NumberFieldWide name="timeout" label="Refresh timeout" help="Seconds to wait before cancelling refresh."/>
      <NumberFieldWide name="max_age" label="Max age" help="Enter the maximum age of feed content in seconds. It is recommended to set this to '0' to disable the age filter, ensuring all items in the feed are processed."/>

      <PasswordFieldWide name="cookie" label="Cookie" help="Not commonly used" />
    </div>
  );
}

const componentMap: componentMapType = {
  TORZNAB: <FormFieldsTorznab />,
  NEWZNAB: <FormFieldsNewznab />,
  RSS: <FormFieldsRSS />
};
