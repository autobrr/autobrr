/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment, useRef, useState, ReactElement } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Dialog, DialogPanel, DialogTitle, Transition, TransitionChild } from "@headlessui/react";
import { XMarkIcon } from "@heroicons/react/24/solid";
import { Form, Formik, useFormikContext } from "formik";
import { useTranslation } from "react-i18next";

import { classNames, sleep } from "@utils";
import { DEBUG } from "@components/debug";
import { APIClient } from "@api/APIClient";
import { DownloadClientKeys } from "@api/query_keys";
import { DownloadClientAuthType, DownloadRuleConditionOptions, getDownloadClientTypeOptions } from "@domain/constants";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { useToggle } from "@hooks/hooks";
import { DeleteModal } from "@components/modals";
import {
  NumberFieldWide,
  PasswordFieldWide,
  RadioFieldsetWide,
  SwitchGroupWide,
  TextFieldWide
} from "@components/inputs";
import { DocsLink, ExternalLink } from "@components/ExternalLink";
import { SelectFieldBasic } from "@components/inputs/select_wide";
import { AddFormProps, UpdateFormProps } from "@forms/_shared";

interface InitialValuesSettings {
  basic?: {
    auth: boolean;
    username: string;
    password: string;
  };
  auth?: {
    enabled: boolean;
    type: string;
    username: string;
    password: string;
  };
  rules?: {
    enabled?: boolean;
    ignore_slow_torrents?: boolean;
    ignore_slow_torrents_condition?: IgnoreTorrentsCondition;
    download_speed_threshold?: number;
    max_active_downloads?: number;
  };
  external_download_client_id?: number;
  external_download_client?: string;
}

interface InitialValues {
  name: string;
  type: DownloadClientType;
  enabled: boolean;
  host: string;
  port: number;
  tls: boolean;
  tls_skip_verify: boolean;
  username: string;
  password: string;
  settings: InitialValuesSettings;
}

function FormFieldsDeluge() {
  const { t } = useTranslation("settings");
  const {
    values: { tls }
  } = useFormikContext<InitialValues>();

  return (
    <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
      <TextFieldWide
        required
        name="host"
        label={t("forms.downloadClient.host")}
        help={t("forms.downloadClient.hostHelpDeluge")}
        tooltip={
          <div>
            <p>{t("forms.downloadClient.guidesDeluge")}</p>
            <br />
            <p>{t("forms.downloadClient.dedicatedServers")}</p>
            <DocsLink href="https://autobrr.com/configuration/download-clients/dedicated#deluge" />
            <p>{t("forms.downloadClient.sharedSeedboxProviders")}</p>
            <DocsLink href="https://autobrr.com/configuration/download-clients/shared-seedboxes#deluge" />
          </div>
        }
      />

      <NumberFieldWide
        name="port"
        label={t("forms.downloadClient.port")}
        help={t("forms.downloadClient.daemonPort")}
      />

      <SwitchGroupWide name="tls" label={t("forms.downloadClient.tls")} />

      {tls && (
        <SwitchGroupWide
          name="tls_skip_verify"
          label={t("forms.downloadClient.skipTls")}
        />
      )}

      <TextFieldWide name="username" label={t("forms.downloadClient.username")} />
      <PasswordFieldWide name="password" label={t("forms.downloadClient.password")} />
    </div>
  );
}

function FormFieldsArr() {
  const { t } = useTranslation("settings");
  const {
    values: { tls, settings }
  } = useFormikContext<InitialValues>();

  return (
    <div className="flex flex-col space-y-4 px-1 mb-4 sm:py-0 sm:space-y-0">
      <TextFieldWide
        required
        name="host"
        label={t("forms.downloadClient.host")}
        help={t("forms.downloadClient.hostHelpArr")}
        tooltip={
          <div>
            <p>{t("forms.downloadClient.guidesArr")}</p>
            <br />
            <p>{t("forms.downloadClient.dedicatedServers")}</p>
            <DocsLink href="https://autobrr.com/configuration/download-clients/dedicated/#sonarr" />
            <p>{t("forms.downloadClient.sharedSeedboxProviders")}</p>
            <DocsLink href="https://autobrr.com/configuration/download-clients/shared-seedboxes#sonarr" />
          </div>
        }
      />

      <SwitchGroupWide name="tls" label={t("forms.downloadClient.tls")} />

      {tls && (
        <SwitchGroupWide
          name="tls_skip_verify"
          label={t("forms.downloadClient.skipTls")}
        />
      )}

      <PasswordFieldWide required name="settings.apikey" label={t("forms.downloadClient.apiKey")} />

      <SwitchGroupWide name="settings.basic.auth" label={t("forms.downloadClient.basicAuth")} />

      {settings.basic?.auth === true && (
        <>
          <TextFieldWide name="settings.basic.username" label={t("forms.downloadClient.username")} />
          <PasswordFieldWide name="settings.basic.password" label={t("forms.downloadClient.password")} />
        </>
      )}
    </div>
  );
}

function FormFieldsQbit() {
  const { t } = useTranslation("settings");
  const {
    values: { port, tls, settings }
  } = useFormikContext<InitialValues>();

  return (
    <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
      <TextFieldWide
        required
        name="host"
        label={t("forms.downloadClient.host")}
        help={t("forms.downloadClient.hostHelpQbit")}
        tooltip={
          <div>
            <p>{t("forms.downloadClient.guidesQbit")}</p>
            <br />
            <p>{t("forms.downloadClient.dedicatedServers")}</p>
            <DocsLink href="https://autobrr.com/configuration/download-clients/dedicated#qbittorrent" />
            <p>{t("forms.downloadClient.sharedSeedboxProviders")}</p>
            <DocsLink href="https://autobrr.com/configuration/download-clients/shared-seedboxes#qbittorrent" />
          </div>
        }
      />

      {port > 0 && (
        <NumberFieldWide
          name="port"
          label={t("forms.downloadClient.port")}
          help={t("forms.downloadClient.webUiPortQbit")}
        />
      )}

      <SwitchGroupWide name="tls" label={t("forms.downloadClient.tls")} />

      {tls && (
        <SwitchGroupWide
          name="tls_skip_verify"
          label={t("forms.downloadClient.skipTls")}
        />
      )}

      <TextFieldWide name="username" label={t("forms.downloadClient.username")} />
      <PasswordFieldWide name="password" label={t("forms.downloadClient.password")} />

      <SwitchGroupWide name="settings.basic.auth" label={t("forms.downloadClient.basicAuth")} />

      {settings.basic?.auth === true && (
        <>
          <TextFieldWide name="settings.basic.username" label={t("forms.downloadClient.username")} />
          <PasswordFieldWide name="settings.basic.password" label={t("forms.downloadClient.password")} />
        </>
      )}
    </div>
  );
}

function FormFieldsPorla() {
  const { t } = useTranslation("settings");
  const {
    values: { tls, settings }
  } = useFormikContext<InitialValues>();

  return (
    <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
      <TextFieldWide
        required
        name="host"
        label={t("forms.downloadClient.host")}
        help={t("forms.downloadClient.hostHelpPorla")}
      />

      <SwitchGroupWide name="tls" label={t("forms.downloadClient.tls")} />

      <PasswordFieldWide required name="settings.apikey" label={t("forms.downloadClient.authToken")} />

      {tls && (
        <SwitchGroupWide
          name="tls_skip_verify"
          label={t("forms.downloadClient.skipTls")}
        />
      )}

      <SwitchGroupWide name="settings.basic.auth" label={t("forms.downloadClient.basicAuth")} />

      {settings.basic?.auth === true && (
        <>
          <TextFieldWide name="settings.basic.username" label={t("forms.downloadClient.username")} />
          <PasswordFieldWide name="settings.basic.password" label={t("forms.downloadClient.password")} />
        </>
      )}
    </div>
  );
}

function FormFieldsRTorrent() {
  const { t } = useTranslation("settings");
  const {
    values: { tls, settings }
  } = useFormikContext<InitialValues>();

  return (
    <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
      <TextFieldWide
        required
        name="host"
        label={t("forms.downloadClient.host")}
        help={t("forms.downloadClient.hostHelpRTorrent")}
        tooltip={
          <div>
            <p>{t("forms.downloadClient.guidesRTorrent")}</p>
            <br />
            <p>{t("forms.downloadClient.dedicatedServers")}</p>
            <DocsLink href="https://autobrr.com/configuration/download-clients/dedicated#rtorrent--rutorrent" />
            <p>{t("forms.downloadClient.sharedSeedboxProviders")}</p>
            <DocsLink href="https://autobrr.com/configuration/download-clients/shared-seedboxes#rtorrent" />
          </div>
        }
      />

      <SwitchGroupWide name="tls" label={t("forms.downloadClient.tls")} />

      {tls && (
        <SwitchGroupWide
          name="tls_skip_verify"
          label={t("forms.downloadClient.skipTls")}
        />
      )}

      <SwitchGroupWide name="settings.auth.enabled" label={t("forms.downloadClient.auth")} />

      {settings.auth?.enabled && (
        <>
          <SelectFieldBasic
            name="settings.auth.type"
            label={t("forms.downloadClient.authType")}
            placeholder={t("forms.downloadClient.selectAuthType")}
            options={DownloadClientAuthType}
            tooltip={<p>{t("forms.downloadClient.authTypeTooltip")}</p>}
          />
          <TextFieldWide name="settings.auth.username" label={t("forms.downloadClient.username")} />
          <PasswordFieldWide name="settings.auth.password" label={t("forms.downloadClient.password")} />
        </>
      )}
    </div>
  );
}

function FormFieldsTransmission() {
  const { t } = useTranslation("settings");
  const {
    values: { port, tls }
  } = useFormikContext<InitialValues>();

  return (
    <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
      <TextFieldWide
        required
        name="host"
        label={t("forms.downloadClient.host")}
        help={t("forms.downloadClient.hostHelpTransmission")}
        tooltip={
          <div>
            <p>{t("forms.downloadClient.guidesTransmission")}</p>
            <br />
            <p>{t("forms.downloadClient.dedicatedServers")}</p>
            <DocsLink href="https://autobrr.com/configuration/download-clients/dedicated#transmission" />
            <p>{t("forms.downloadClient.sharedSeedboxProviders")}</p>
            <DocsLink href="https://autobrr.com/configuration/download-clients/shared-seedboxes#transmission" />
          </div>
        }
      />

      {port > 0 && (
        <NumberFieldWide
          name="port"
          label={t("forms.downloadClient.port")}
          help={t("forms.downloadClient.transmissionPort")}
        />
      )}

      <SwitchGroupWide name="tls" label={t("forms.downloadClient.tls")} />

      {tls && (
        <SwitchGroupWide
          name="tls_skip_verify"
          label={t("forms.downloadClient.skipTls")}
        />
      )}

      <TextFieldWide name="username" label={t("forms.downloadClient.username")} />
      <PasswordFieldWide name="password" label={t("forms.downloadClient.password")} />
    </div>
  );
}

function FormFieldsSabnzbd() {
  const { t } = useTranslation("settings");
  const {
    values: { port, tls, settings }
  } = useFormikContext<InitialValues>();

  return (
    <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
      <TextFieldWide
        name="host"
        label={t("forms.downloadClient.host")}
        help={t("forms.downloadClient.hostHelpSabnzbd")}
        tooltip={
          <div>
            <p>{t("forms.downloadClient.guidesSabnzbd")}</p>
            <br />
            <p>{t("forms.downloadClient.dedicatedServers")}</p>
            <ExternalLink href="https://autobrr.com/configuration/download-clients/dedicated#sabnzbd" />
            <p>{t("forms.downloadClient.sharedSeedboxProviders")}</p>
            <ExternalLink href="https://autobrr.com/configuration/download-clients/shared-seedboxes#sabnzbd" />
          </div>
        }
      />

      {port > 0 && (
        <NumberFieldWide
          name="port"
          label={t("forms.downloadClient.port")}
          help={t("forms.downloadClient.sabnzbdPort")}
        />
      )}

      <SwitchGroupWide name="tls" label={t("forms.downloadClient.tls")} />

      {tls && (
        <SwitchGroupWide
          name="tls_skip_verify"
          label={t("forms.downloadClient.skipTls")}
        />
      )}

      {/*<TextFieldWide name="username" label="Username" />*/}
      {/*<PasswordFieldWide name="password" label="Password" />*/}

      <PasswordFieldWide name="settings.apikey" label={t("forms.downloadClient.apiKey")} />

      <SwitchGroupWide name="settings.basic.auth" label={t("forms.downloadClient.basicAuth")} />

      {settings.basic?.auth === true && (
        <>
          <TextFieldWide name="settings.basic.username" label={t("forms.downloadClient.username")} />
          <PasswordFieldWide name="settings.basic.password" label={t("forms.downloadClient.password")} />
        </>
      )}
    </div>
  );
}

function FormFieldsNzbget() {
  const { t } = useTranslation("settings");
  return (
    <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
      <TextFieldWide
        name="host"
        label={t("forms.downloadClient.host")}
        help={t("forms.downloadClient.hostHelpNzbget")}
      />

      <TextFieldWide name="username" label={t("forms.downloadClient.username")} />
      <PasswordFieldWide name="password" label={t("forms.downloadClient.password")} />
    </div>
  );
}

export interface componentMapType {
  [key: string]: ReactElement;
}

export const componentMap: componentMapType = {
  DELUGE_V1: <FormFieldsDeluge />,
  DELUGE_V2: <FormFieldsDeluge />,
  QBITTORRENT: <FormFieldsQbit />,
  RTORRENT: <FormFieldsRTorrent />,
  TRANSMISSION: <FormFieldsTransmission />,
  PORLA: <FormFieldsPorla />,
  RADARR: <FormFieldsArr />,
  SONARR: <FormFieldsArr />,
  LIDARR: <FormFieldsArr />,
  WHISPARR: <FormFieldsArr />,
  READARR: <FormFieldsArr />,
  SABNZBD: <FormFieldsSabnzbd />,
  NZBGET: <FormFieldsNzbget />
};

function FormFieldsRulesBasic() {
  const { t } = useTranslation("settings");
  const {
    values: { settings }
  } = useFormikContext<InitialValues>();

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-5">

      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">{t("forms.downloadClient.rulesTitle")}</DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.downloadClient.rulesDescriptionBasic")}
        </p>
      </div>

      <SwitchGroupWide name="settings.rules.enabled" label={t("forms.downloadClient.enabled")} />

      {settings && settings.rules?.enabled === true && (
        <NumberFieldWide
          name="settings.rules.max_active_downloads"
          label={t("forms.downloadClient.maxActiveDownloads")}
          tooltip={
            <span>
              <p>{t("forms.downloadClient.maxActiveDownloadsTooltip")}</p>
              <DocsLink href="https://autobrr.com/configuration/download-clients/dedicated#deluge-rules" />
              <br /><br />
              <p>{t("forms.downloadClient.seeRecommendations")}</p>
              <DocsLink href='https://autobrr.com/filters/examples#build-buffer' />
            </span>
          }
        />
      )}
    </div>
  );
}

function FormFieldsRulesArr() {
  const { t } = useTranslation("settings");
  // const {
  //   values: { settings }
  // } = useFormikContext<InitialValues>();

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-5 px-2">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.downloadClient.downloadClientTitle")}
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.downloadClient.downloadClientDescription")}
        </p>
      </div>

      <TextFieldWide name="settings.external_download_client" label={t("forms.downloadClient.clientName")} tooltip={<div><p>{t("forms.downloadClient.clientNameTooltip")}</p></div>} />

      <NumberFieldWide name="settings.external_download_client_id" label={t("forms.downloadClient.clientIdDeprecated")} tooltip={<div><p>{t("forms.downloadClient.clientIdDeprecatedTooltip")}</p></div>} />
    </div>
  );
}

function FormFieldsRulesQbit() {
  const { t } = useTranslation("settings");
  const {
    values: { settings }
  } = useFormikContext<InitialValues>();

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-5 px-2">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.downloadClient.rulesTitle")}
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.downloadClient.rulesDescriptionAdvanced")}
        </p>
      </div>

      <SwitchGroupWide name="settings.rules.enabled" label={t("forms.downloadClient.enabled")} />

      {settings.rules?.enabled === true && (
        <>
          <NumberFieldWide
            name="settings.rules.max_active_downloads"
            label={t("forms.downloadClient.maxActiveDownloads")}
            tooltip={
              <>
                <p>{t("forms.downloadClient.maxActiveDownloadsTooltip")}</p>
                <DocsLink href="https://autobrr.com/configuration/download-clients/dedicated#qbittorrent-rules" />
                <br /><br />
                <p>{t("forms.downloadClient.seeRecommendations")}</p>
                <DocsLink href="https://autobrr.com/filters/examples#build-buffer" />
              </>
            }
          />

          <SwitchGroupWide
            name="settings.rules.ignore_slow_torrents"
            label={t("forms.downloadClient.ignoreSlowTorrents")}
          />

          {settings.rules?.ignore_slow_torrents === true && (
            <>
              <SelectFieldBasic
                name="settings.rules.ignore_slow_torrents_condition"
                label={t("forms.downloadClient.ignoreCondition")}
                placeholder={t("forms.downloadClient.selectIgnoreCondition")}
                options={DownloadRuleConditionOptions}
                tooltip={<p>{t("forms.downloadClient.ignoreConditionTooltip")}</p>}
              />
              <NumberFieldWide
                name="settings.rules.download_speed_threshold"
                label={t("forms.downloadClient.downloadSpeedThreshold")}
                placeholder={t("forms.downloadClient.speedThresholdPlaceholder")}
                help={t("forms.downloadClient.downloadSpeedThresholdHelp")}
              />
              <NumberFieldWide
                name="settings.rules.upload_speed_threshold"
                label={t("forms.downloadClient.uploadSpeedThreshold")}
                placeholder={t("forms.downloadClient.speedThresholdPlaceholder")}
                help={t("forms.downloadClient.uploadSpeedThresholdHelp")}
              />
            </>
          )}
        </>
      )}
    </div>
  );
}

function FormFieldsRulesTransmission() {
  const { t } = useTranslation("settings");
  const {
    values: { settings }
  } = useFormikContext<InitialValues>();

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-5 px-2">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.downloadClient.rulesTitle")}
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.downloadClient.rulesDescriptionAdvanced")}
        </p>
      </div>

      <SwitchGroupWide name="settings.rules.enabled" label={t("forms.downloadClient.enabled")} />

      {settings.rules?.enabled === true && (
        <>
          <NumberFieldWide
            name="settings.rules.max_active_downloads"
            label={t("forms.downloadClient.maxActiveDownloads")}
            tooltip={
              <>
                <p>{t("forms.downloadClient.maxActiveDownloadsTooltip")}</p>
                <DocsLink href="https://autobrr.com/configuration/download-clients/dedicated#transmission-rules" />
                <br /><br />
                <p>{t("forms.downloadClient.seeRecommendations")}</p>
                <DocsLink href="https://autobrr.com/filters/examples#build-buffer" />
              </>
            }
          />
        </>
      )}
    </div>
  );
}

export const rulesComponentMap: componentMapType = {
  DELUGE_V1: <FormFieldsRulesBasic />,
  DELUGE_V2: <FormFieldsRulesBasic />,
  QBITTORRENT: <FormFieldsRulesQbit />,
  PORLA: <FormFieldsRulesBasic />,
  TRANSMISSION: <FormFieldsRulesTransmission />,
  RADARR: <FormFieldsRulesArr />,
  SONARR: <FormFieldsRulesArr />,
  LIDARR: <FormFieldsRulesArr />,
  WHISPARR: <FormFieldsRulesArr />,
  READARR: <FormFieldsRulesArr />,
};

interface formButtonsProps {
  isSuccessfulTest: boolean;
  isErrorTest: boolean;
  isTesting: boolean;
  cancelFn: () => void;
  testFn: (data: unknown) => void;
  values: unknown;
  type: "CREATE" | "UPDATE";
  toggleDeleteModal?: () => void;
}

function DownloadClientFormButtons({
  type,
  isSuccessfulTest,
  isErrorTest,
  isTesting,
  cancelFn,
  testFn,
  values,
  toggleDeleteModal
}: formButtonsProps) {
  const { t } = useTranslation("settings");

  const test = () => {
    testFn(values);
  };

  return (
    <div className="shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-5 sm:px-6">
      <div className={classNames(type === "CREATE" ? "justify-end" : "justify-between", "space-x-3 flex")}>
        {type === "UPDATE" && (
          <button
            type="button"
            className="inline-flex items-center justify-center px-4 py-2 border border-transparent font-medium rounded-md text-red-700 dark:text-white bg-red-100 dark:bg-red-700 hover:bg-red-200 dark:hover:bg-red-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:text-sm"
            onClick={toggleDeleteModal}
          >
            {t("forms.downloadClient.remove")}
          </button>
        )}
        <div className="flex">
          <button
            type="button"
            className={classNames(
              isSuccessfulTest
                ? "text-green-500 border-green-500 bg-green-50"
                : isErrorTest
                  ? "text-red-500 border-red-500 bg-red-50"
                  : "border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-400 bg-white dark:bg-gray-700 hover:bg-gray-50 focus:border-rose-700 active:bg-rose-700",
              isTesting ? "cursor-not-allowed" : "",
              "mr-2 inline-flex items-center px-4 py-2 border font-medium rounded-md shadow-xs text-sm transition ease-in-out duration-150 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
            )}
            disabled={isTesting}
            // onClick={() => testClient(values)}
            onClick={test}
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
              t("forms.downloadClient.ok")
            ) : isErrorTest ? (
              t("forms.downloadClient.error")
            ) : (
              t("forms.downloadClient.test")
            )}
          </button>

          <button
            type="button"
            className="mr-4 bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
            onClick={cancelFn}
          >
            {t("forms.downloadClient.cancel")}
          </button>
          <button
            type="submit"
            className="inline-flex justify-center py-2 px-4 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
          >
            {type === "CREATE" ? t("forms.downloadClient.create") : t("forms.downloadClient.save")}
          </button>
        </div>
      </div>
    </div>
  );
}

export function DownloadClientAddForm({ isOpen, toggle }: AddFormProps) {
  const { t } = useTranslation(["options", "settings"]);
  const downloadClientTypeOptions = getDownloadClientTypeOptions(t);
  const [isTesting, setIsTesting] = useState(false);
  const [isSuccessfulTest, setIsSuccessfulTest] = useState(false);
  const [isErrorTest, setIsErrorTest] = useState(false);

  const queryClient = useQueryClient();

  const addMutation = useMutation({
    mutationFn: (client: DownloadClient) => APIClient.download_clients.create(client),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: DownloadClientKeys.lists() });
      toast.custom((toastInstance) => <Toast type="success" body={t("settings:forms.downloadClient.added")} t={toastInstance} />);

      toggle();
    },
    onError: () => {
      toast.custom((toastInstance) => <Toast type="error" body={t("settings:forms.downloadClient.addFailed")} t={toastInstance} />);
    }
  });

  const onSubmit = (data: unknown) => addMutation.mutate(data as DownloadClient);

  const testClientMutation = useMutation({
    mutationFn: (client: DownloadClient) => APIClient.download_clients.test(client),
    onMutate: () => {
      setIsTesting(true);
      setIsErrorTest(false);
      setIsSuccessfulTest(false);
    },
    onSuccess: () => {
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

  const testClient = (data: unknown) => testClientMutation.mutate(data as DownloadClient);

  const initialValues: InitialValues = {
    name: "",
    type: "QBITTORRENT",
    enabled: true,
    host: "",
    port: 0,
    tls: false,
    tls_skip_verify: false,
    username: "",
    password: "",
    settings: {}
  };

  return (
    <Transition show={isOpen} as={Fragment}>
      <Dialog
        as="div"
        static
        className="fixed inset-0 overflow-hidden"
        open={isOpen}
        onClose={toggle}
      >
        <div className="absolute inset-0 overflow-hidden">
          <DialogPanel className="fixed inset-y-0 right-0 max-w-full flex">
            <TransitionChild
              as={Fragment}
              enter="transform transition ease-in-out duration-500 sm:duration-700"
              enterFrom="translate-x-full"
              enterTo="translate-x-0"
              leave="transform transition ease-in-out duration-500 sm:duration-700"
              leaveFrom="translate-x-0"
              leaveTo="translate-x-full"
            >
              <div className="w-screen max-w-2xl ">
                <Formik
                  initialValues={initialValues}
                  onSubmit={onSubmit}
                >
                  {({ handleSubmit, values }) => (
                    <Form
                      className="h-full flex flex-col bg-white dark:bg-gray-800 shadow-xl overflow-y-auto"
                      onSubmit={handleSubmit}
                    >
                      <div className="flex-1">
                        <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
                          <div className="flex items-start justify-between space-x-3">
                            <div className="space-y-1">
                              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
                                {t("settings:forms.downloadClient.addTitle")}
                              </DialogTitle>
                              <p className="text-sm text-gray-500 dark:text-gray-400">
                                {t("settings:forms.downloadClient.addDescription")}
                              </p>
                            </div>
                            <div className="h-7 flex items-center">
                              <button
                                type="button"
                                className="bg-white dark:bg-gray-800 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                                onClick={toggle}
                              >
                                <span className="sr-only">{t("settings:forms.downloadClient.closePanel")}</span>
                                <XMarkIcon
                                  className="h-6 w-6"
                                  aria-hidden="true"
                                />
                              </button>
                            </div>
                          </div>
                        </div>

                        <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
                          <TextFieldWide required name="name" label={t("settings:forms.downloadClient.name")} />
                          <SwitchGroupWide name="enabled" label={t("settings:forms.downloadClient.enabled")} />
                          <RadioFieldsetWide
                            name="type"
                            legend={t("settings:forms.downloadClient.type")}
                            options={downloadClientTypeOptions}
                          />
                          <div>{componentMap[values.type]}</div>
                        </div>
                      </div>

                      {rulesComponentMap[values.type]}

                      <DownloadClientFormButtons
                        type="CREATE"
                        isTesting={isTesting}
                        isSuccessfulTest={isSuccessfulTest}
                        isErrorTest={isErrorTest}
                        cancelFn={toggle}
                        testFn={testClient}
                        values={values}
                      />

                      <DEBUG values={values} />
                    </Form>
                  )}
                </Formik>
              </div>
            </TransitionChild>
          </DialogPanel>
        </div>
      </Dialog>
    </Transition>
  );
}

export function DownloadClientUpdateForm({ isOpen, toggle, data: client }: UpdateFormProps<DownloadClient>) {
  const { t } = useTranslation(["options", "settings"]);
  const downloadClientTypeOptions = getDownloadClientTypeOptions(t);
  const [isTesting, setIsTesting] = useState(false);
  const [isSuccessfulTest, setIsSuccessfulTest] = useState(false);
  const [isErrorTest, setIsErrorTest] = useState(false);
  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);

  const cancelButtonRef = useRef(null);
  const cancelModalButtonRef = useRef(null);

  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (client: DownloadClient) => APIClient.download_clients.update(client),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: DownloadClientKeys.lists() });
      queryClient.invalidateQueries({ queryKey: DownloadClientKeys.detail(client.id) });

      toast.custom((toastInstance) => <Toast type="success" body={t("settings:forms.downloadClient.updated", { name: client.name })} t={toastInstance} />);
      toggle();
    }
  });

  const onSubmit = (data: unknown) => mutation.mutate(data as DownloadClient);

  const deleteMutation = useMutation({
    mutationFn: (clientID: number) => APIClient.download_clients.delete(clientID),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: DownloadClientKeys.lists() });
      queryClient.invalidateQueries({ queryKey: DownloadClientKeys.detail(client.id) });

      toast.custom((toastInstance) => <Toast type="success" body={t("settings:forms.downloadClient.deleted", { name: client.name })} t={toastInstance} />);
      toggleDeleteModal();
    }
  });

  const deleteAction = () => deleteMutation.mutate(client.id);


  const testClientMutation = useMutation({
    mutationFn: (client: DownloadClient) => APIClient.download_clients.test(client),
    onMutate: () => {
      setIsTesting(true);
      setIsErrorTest(false);
      setIsSuccessfulTest(false);
    },
    onSuccess: () => {
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

  const testClient = (data: unknown) => testClientMutation.mutate(data as DownloadClient);

  const initialValues = {
    id: client.id,
    name: client.name,
    type: client.type,
    enabled: client.enabled,
    host: client.host,
    port: client.port,
    tls: client.tls,
    tls_skip_verify: client.tls_skip_verify,
    username: client.username,
    password: client.password,
    settings: client.settings
  };

  return (
    <Transition show={isOpen} as={Fragment}>
      <Dialog
        as="div"
        static
        className="fixed inset-0 overflow-hidden"
        open={isOpen}
        onClose={toggle}
        initialFocus={cancelButtonRef}
      >
        <DeleteModal
          isOpen={deleteModalIsOpen}
          isLoading={deleteMutation.isPending}
          toggle={toggleDeleteModal}
          buttonRef={cancelModalButtonRef}
          deleteAction={deleteAction}
          title={t("settings:forms.downloadClient.removeTitle")}
          text={t("settings:forms.downloadClient.removeText")}
        />
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
              <div className="w-screen max-w-2xl ">
                <Formik
                  initialValues={initialValues}
                  onSubmit={onSubmit}
                >
                  {({ handleSubmit, values }) => {
                    return (
                      <Form
                        className="h-full flex flex-col bg-white dark:bg-gray-800 shadow-xl overflow-y-auto"
                        onSubmit={handleSubmit}
                      >
                        <div className="flex-1">
                          <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
                            <div className="flex items-start justify-between space-x-3">
                              <div className="space-y-1">
                                <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
                                  {t("settings:forms.downloadClient.editTitle")}
                                </DialogTitle>
                                <p className="text-sm text-gray-500 dark:text-gray-400">
                                  {t("settings:forms.downloadClient.editDescription")}
                                </p>
                              </div>
                              <div className="h-7 flex items-center">
                                <button
                                  type="button"
                                  className="bg-white dark:bg-gray-800 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                                  onClick={toggle}
                                >
                                  <span className="sr-only">{t("settings:forms.downloadClient.closePanel")}</span>
                                  <XMarkIcon
                                    className="h-6 w-6"
                                    aria-hidden="true"
                                  />
                                </button>
                              </div>
                            </div>
                          </div>

                          <div className="py-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y dark:divide-gray-700">
                            <TextFieldWide required name="name" label={t("settings:forms.downloadClient.name")} />
                            <SwitchGroupWide name="enabled" label={t("settings:forms.downloadClient.enabled")} />
                            <RadioFieldsetWide
                              name="type"
                              legend={t("settings:forms.downloadClient.type")}
                              options={downloadClientTypeOptions}
                            />
                            <div>{componentMap[values.type]}</div>
                          </div>
                        </div>

                        {rulesComponentMap[values.type]}

                        <DownloadClientFormButtons
                          type="UPDATE"
                          toggleDeleteModal={toggleDeleteModal}
                          isTesting={isTesting}
                          isSuccessfulTest={isSuccessfulTest}
                          isErrorTest={isErrorTest}
                          cancelFn={toggle}
                          testFn={testClient}
                          values={values}
                        />

                        <DEBUG values={values} />
                      </Form>
                    );
                  }}
                </Formik>
              </div>
            </TransitionChild>
          </DialogPanel>
        </div>
      </Dialog>
    </Transition>
  );
}
