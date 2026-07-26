/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { WarningAlert } from "@components/alerts";
import { FilterHalfRow, FilterLayout, FilterSection } from "@screens/filters/sections/_components.tsx";
import { DownloadClientSelect, NumberField, TextAreaAutoResize, TextField } from "@components/inputs";
import { useTranslation } from "react-i18next";


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
    </FilterLayout>
    <TextAreaAutoResize
      name={`actions.${idx}.webhook_data`}
      label={t("actionComponents.webhook.payload")}
      placeholder={t("actionComponents.webhook.payloadPlaceholder")}
    />
  </FilterSection>
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
