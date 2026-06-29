/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Field } from "formik";
import type { FieldProps } from "formik";

import { WarningAlert } from "@components/alerts";
import { ExternalFilterWebhookMethodOptions } from "@domain/constants";
import { FilterHalfRow, FilterLayout, FilterSection } from "@screens/filters/sections/_components.tsx";
import { DownloadClientSelect, NumberField, Select, TextAreaAutoResize, TextField } from "@components/inputs";
import { useTranslation } from "react-i18next";

// WebhookHeadersField edits the action.webhook_headers string[] as a single
// "Key=Value;Key2=Value2" line, keeping the array model in sync.
const WebhookHeadersField = ({ idx }: { idx: number }) => {
  const { t } = useTranslation("filters");
  const name = `actions.${idx}.webhook_headers`;

  return (
    <div className="col-span-12 sm:col-span-6">
      <label htmlFor={name} className="flex ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide">
        {t("actionComponents.webhook.headers")}
      </label>
      <Field name={name}>
        {({ field, form }: FieldProps) => (
          <input
            id={name}
            type="text"
            value={Array.isArray(field.value) ? field.value.join(";") : (field.value ?? "")}
            onChange={(e) => {
              const value = e.target.value;
              form.setFieldValue(field.name, value ? value.split(";") : []);
            }}
            placeholder={t("actionComponents.webhook.headersPlaceholder")}
            className="mt-1 block border w-full dark:text-gray-100 rounded-md border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 bg-gray-100 dark:bg-gray-815"
            data-1p-ignore
          />
        )}
      </Field>
    </div>
  );
};


export const SABnzbd = ({ idx, action, clients }: ClientActionProps) => {
  const { t } = useTranslation("filters");

  return (
  <FilterSection
    title={t("actionComponents.instance.title")}
    subtitle={t("actionComponents.instance.subtitle")}
  >
    <FilterLayout>
      <FilterHalfRow>
        <DownloadClientSelect
          name={`actions.${idx}.client_id`}
          action={action}
          clients={clients}
        />
      </FilterHalfRow>
      <FilterHalfRow>
        <TextField
          name={`actions.${idx}.category`}
          label={t("actionComponents.common.category")}
          columns={6}
          placeholder={t("actionComponents.common.categoryPlaceholder")}
          tooltip={<p>{t("actionComponents.common.categoryTooltip")}</p>}
        />
      </FilterHalfRow>
    </FilterLayout>
  </FilterSection>
  );
};

export const NZBGet = ({ idx, action, clients }: ClientActionProps) => {
  const { t } = useTranslation("filters");

  return (
  <FilterSection
    title={t("actionComponents.instance.title")}
    subtitle={t("actionComponents.instance.subtitle")}
  >
    <FilterLayout>
      <FilterHalfRow>
        <DownloadClientSelect
          name={`actions.${idx}.client_id`}
          action={action}
          clients={clients}
        />
      </FilterHalfRow>
      <FilterHalfRow>
        <TextField
          name={`actions.${idx}.category`}
          label={t("actionComponents.common.category")}
          columns={6}
          placeholder={t("actionComponents.common.categoryPlaceholder")}
          tooltip={<p>{t("actionComponents.common.categoryTooltip")}</p>}
        />
      </FilterHalfRow>
    </FilterLayout>
  </FilterSection>
  );
};

export const Test = () => {
  const { t } = useTranslation("filters");

  return (
  <WarningAlert
    alert={t("actionComponents.test.alert")}
    className="mt-2"
    colors="text-fuchsia-700 bg-fuchsia-100 dark:bg-fuchsia-200 dark:text-fuchsia-800"
    text={t("actionComponents.test.text")}
  />
  );
};

export const Exec = ({ idx }: ClientActionProps) => {
  const { t } = useTranslation("filters");

  return (
  <FilterSection
    title={t("actionComponents.exec.title")}
    subtitle={t("actionComponents.exec.subtitle")}
  >
    <FilterLayout>
      <TextField
        name={`actions.${idx}.exec_cmd`}
        label={t("actionComponents.exec.path")}
        placeholder={t("actionComponents.exec.pathPlaceholder")}
      />

      <TextAreaAutoResize
        name={`actions.${idx}.exec_args`}
        label={t("actionComponents.exec.arguments")}
        placeholder={t("actionComponents.exec.argumentsPlaceholder")}
      />

      <div className="col-span-12 sm:col-span-2">
        <NumberField
          name={`actions.${idx}.exec_expect_status`}
          label={t("actionComponents.exec.expectedExitStatus")}
          placeholder="0"
        />
      </div>
    </FilterLayout>

  </FilterSection>
  );
};

export const WatchFolder = ({ idx }: ClientActionProps) => {
  const { t } = useTranslation("filters");

  return (
  <FilterSection
    title={t("actionComponents.watchFolder.title")}
    subtitle={t("actionComponents.watchFolder.subtitle")}
  >
    <FilterLayout>
      <TextAreaAutoResize
        name={`actions.${idx}.watch_folder`}
        label={t("actionComponents.watchFolder.directory")}
        placeholder={t("actionComponents.watchFolder.directoryPlaceholder")}
      />
    </FilterLayout>
  </FilterSection>
  );
};

export const WebHook = ({ idx }: ClientActionProps) => {
  const { t } = useTranslation("filters");

  return (
  <>
    <FilterSection
      title={t("actionComponents.webhook.title")}
      subtitle={t("actionComponents.webhook.subtitle")}
    >
      <FilterLayout>
        <TextField
          name={`actions.${idx}.webhook_host`}
          label={t("actionComponents.webhook.endpoint")}
          columns={6}
          placeholder={t("actionComponents.webhook.endpointPlaceholder")}
          tooltip={
            <p>{t("actionComponents.webhook.endpointTooltip")}</p>
          }
        />
        <Select
          name={`actions.${idx}.webhook_method`}
          label={t("actionComponents.webhook.httpMethod")}
          optionDefaultText={t("actionComponents.webhook.httpMethodDefault")}
          options={ExternalFilterWebhookMethodOptions}
          tooltip={<div><p>{t("actionComponents.webhook.httpMethodTooltip")}</p></div>}
        />
        <WebhookHeadersField idx={idx} />
        <NumberField
          name={`actions.${idx}.webhook_expect_status`}
          label={t("actionComponents.webhook.expectedStatus")}
          placeholder="200"
        />
      </FilterLayout>
    </FilterSection>

    <FilterSection
      title={t("actionComponents.webhook.retryTitle")}
      subtitle={t("actionComponents.webhook.retrySubtitle")}
    >
      <FilterLayout>
        <TextField
          name={`actions.${idx}.webhook_retry_status`}
          label={t("actionComponents.webhook.retryStatus")}
          placeholder={t("actionComponents.webhook.retryStatusPlaceholder")}
          columns={6}
        />
        <NumberField
          name={`actions.${idx}.webhook_retry_attempts`}
          label={t("actionComponents.webhook.retryAttempts")}
          placeholder="10"
        />
        <NumberField
          name={`actions.${idx}.webhook_retry_delay_seconds`}
          label={t("actionComponents.webhook.retryDelaySeconds")}
          placeholder="1"
        />
      </FilterLayout>
    </FilterSection>

    <FilterSection
      title={t("actionComponents.webhook.payloadTitle")}
      subtitle={t("actionComponents.webhook.payloadSubtitle")}
    >
      <FilterLayout>
        <TextAreaAutoResize
          name={`actions.${idx}.webhook_data`}
          label={t("actionComponents.webhook.payload")}
          placeholder={t("actionComponents.webhook.payloadPlaceholder")}
        />
      </FilterLayout>
    </FilterSection>
  </>
  );
};

export const Arr = ({ idx, action, clients }: ClientActionProps) => {
  const { t } = useTranslation("filters");

  return (
  <FilterSection
    title={t("actionComponents.instance.title")}
    subtitle={t("actionComponents.instance.subtitle")}
  >
    <FilterLayout>
      <FilterHalfRow>
        <DownloadClientSelect
          name={`actions.${idx}.client_id`}
          action={action}
          clients={clients}
        />
      </FilterHalfRow>

      <FilterHalfRow>
        <div className="">
          <TextField
            name={`actions.${idx}.external_download_client`}
            label={t("actionComponents.arr.overrideClientName")}
            tooltip={
              <p>{t("actionComponents.arr.overrideClientNameTooltip")}</p>
            }
          />
          <NumberField
            name={`actions.${idx}.external_download_client_id`}
            label={t("actionComponents.arr.overrideClientId")}
            className="mt-4"
            tooltip={
              <p>{t("actionComponents.arr.overrideClientIdTooltip")}</p>
            }
          />
        </div>
      </FilterHalfRow>
    </FilterLayout>
  </FilterSection>
  );
};
