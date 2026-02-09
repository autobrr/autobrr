/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment, useRef, useState, ReactElement } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Dialog, DialogPanel, DialogTitle, Transition, TransitionChild } from "@headlessui/react";
import { XMarkIcon } from "@heroicons/react/24/solid";

import { classNames, sleep } from "@utils";
import { DEBUG } from "@components/debug";
import { APIClient } from "@api/APIClient";
import { DownloadClientKeys } from "@api/query_keys";
import { DownloadClientAuthType, DownloadClientTypeOptions, DownloadRuleConditionOptions } from "@domain/constants";
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
} from "@components/inputs/tanstack";
import { DocsLink, ExternalLink } from "@components/ExternalLink";
import { SelectFieldBasic } from "@components/inputs/tanstack/select_wide";
import { AddFormProps, UpdateFormProps } from "@forms/_shared";
import { useStore } from "@tanstack/react-store";
import { useAppForm, ContextField, useFormContext } from "@app/lib/form";

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
  const form = useFormContext() as any;
  const tls = useStore(form.store, (s: any) => s.values.tls);

  return (
    <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
      <ContextField name="host">
        <TextFieldWide
          required
          label="Host"
          help="Eg. client.domain.ltd, domain.ltd/client, domain.ltd:port"
          tooltip={
            <div>
              <p>See guides for how to connect to Deluge for various server types in our docs.</p>
              <br />
              <p>Dedicated servers:</p>
              <DocsLink href="https://autobrr.com/configuration/download-clients/dedicated#deluge" />
              <p>Shared seedbox providers:</p>
              <DocsLink href="https://autobrr.com/configuration/download-clients/shared-seedboxes#deluge" />
            </div>
          }
        />
      </ContextField>

      <ContextField name="port">
        <NumberFieldWide
          label="Port"
          help="Daemon port"
        />
      </ContextField>

      <ContextField name="tls">
        <SwitchGroupWide label="TLS" />
      </ContextField>

      {tls && (
        <ContextField name="tls_skip_verify">
          <SwitchGroupWide
            label="Skip TLS verification (insecure)"
          />
        </ContextField>
      )}

      <ContextField name="username">
        <TextFieldWide label="Username" />
      </ContextField>
      <ContextField name="password">
        <PasswordFieldWide label="Password" />
      </ContextField>
    </div>
  );
}

function FormFieldsArr() {
  const form = useFormContext() as any;
  const tls = useStore(form.store, (s: any) => s.values.tls);
  const settings = useStore(form.store, (s: any) => s.values.settings);

  return (
    <div className="flex flex-col space-y-4 px-1 mb-4 sm:py-0 sm:space-y-0">
      <ContextField name="host">
        <TextFieldWide
          required
          label="Host"
          help="Full url http(s)://domain.ltd and/or subdomain/subfolder"
          tooltip={
            <div>
              <p>See guides for how to connect to the *arr suite for various server types in our docs.</p>
              <br />
              <p>Dedicated servers:</p>
              <DocsLink href="https://autobrr.com/configuration/download-clients/dedicated/#sonarr" />
              <p>Shared seedbox providers:</p>
              <DocsLink href="https://autobrr.com/configuration/download-clients/shared-seedboxes#sonarr" />
            </div>
          }
        />
      </ContextField>

      <ContextField name="tls">
        <SwitchGroupWide label="TLS" />
      </ContextField>

      {tls && (
        <ContextField name="tls_skip_verify">
          <SwitchGroupWide
            label="Skip TLS verification (insecure)"
          />
        </ContextField>
      )}

      <ContextField name="settings.apikey">
        <PasswordFieldWide required label="API key" />
      </ContextField>

      <ContextField name="settings.basic.auth">
        <SwitchGroupWide label="Basic auth" />
      </ContextField>

      {settings.basic?.auth === true && (
        <>
          <ContextField name="settings.basic.username">
            <TextFieldWide label="Username" />
          </ContextField>
          <ContextField name="settings.basic.password">
            <PasswordFieldWide label="Password" />
          </ContextField>
        </>
      )}
    </div>
  );
}

function FormFieldsQbit() {
  const form = useFormContext() as any;
  const port = useStore(form.store, (s: any) => s.values.port);
  const tls = useStore(form.store, (s: any) => s.values.tls);
  const settings = useStore(form.store, (s: any) => s.values.settings);

  return (
    <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
      <ContextField name="host">
        <TextFieldWide
          required
          label="Host"
          help="Eg. http(s)://client.domain.ltd, http(s)://domain.ltd/qbittorrent, http://domain.ltd:port"
          tooltip={
            <div>
              <p>See guides for how to connect to qBittorrent for various server types in our docs.</p>
              <br />
              <p>Dedicated servers:</p>
              <DocsLink href="https://autobrr.com/configuration/download-clients/dedicated#qbittorrent" />
              <p>Shared seedbox providers:</p>
              <DocsLink href="https://autobrr.com/configuration/download-clients/shared-seedboxes#qbittorrent" />
            </div>
          }
        />
      </ContextField>

      {port > 0 && (
        <ContextField name="port">
          <NumberFieldWide
            label="Port"
            help="WebUI port for qBittorrent"
          />
        </ContextField>
      )}

      <ContextField name="tls">
        <SwitchGroupWide label="TLS" />
      </ContextField>

      {tls && (
        <ContextField name="tls_skip_verify">
          <SwitchGroupWide
            label="Skip TLS verification (insecure)"
          />
        </ContextField>
      )}

      <ContextField name="username">
        <TextFieldWide label="Username" />
      </ContextField>
      <ContextField name="password">
        <PasswordFieldWide label="Password" />
      </ContextField>

      <ContextField name="settings.basic.auth">
        <SwitchGroupWide label="Basic auth" />
      </ContextField>

      {settings.basic?.auth === true && (
        <>
          <ContextField name="settings.basic.username">
            <TextFieldWide label="Username" />
          </ContextField>
          <ContextField name="settings.basic.password">
            <PasswordFieldWide label="Password" />
          </ContextField>
        </>
      )}
    </div>
  );
}

function FormFieldsPorla() {
  const form = useFormContext() as any;
  const tls = useStore(form.store, (s: any) => s.values.tls);
  const settings = useStore(form.store, (s: any) => s.values.settings);

  return (
    <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
      <ContextField name="host">
        <TextFieldWide
          required
          label="Host"
          help="Eg. http(s)://client.domain.ltd, http(s)://domain.ltd/porla, http://domain.ltd:port"
        />
      </ContextField>

      <ContextField name="tls">
        <SwitchGroupWide label="TLS" />
      </ContextField>

      <ContextField name="settings.apikey">
        <PasswordFieldWide required label="Auth token" />
      </ContextField>

      {tls && (
        <ContextField name="tls_skip_verify">
          <SwitchGroupWide
            label="Skip TLS verification (insecure)"
          />
        </ContextField>
      )}

      <ContextField name="settings.basic.auth">
        <SwitchGroupWide label="Basic auth" />
      </ContextField>

      {settings.basic?.auth === true && (
        <>
          <ContextField name="settings.basic.username">
            <TextFieldWide label="Username" />
          </ContextField>
          <ContextField name="settings.basic.password">
            <PasswordFieldWide label="Password" />
          </ContextField>
        </>
      )}
    </div>
  );
}

function FormFieldsRTorrent() {
  const form = useFormContext() as any;
  const tls = useStore(form.store, (s: any) => s.values.tls);
  const settings = useStore(form.store, (s: any) => s.values.settings);

  return (
    <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
      <ContextField name="host">
        <TextFieldWide
          required
          label="Host"
          help="Eg. http(s)://client.domain.ltd/RPC2, http(s)://domain.ltd/client, http(s)://domain.ltd/RPC2"
          tooltip={
            <div>
              <p>See guides for how to connect to rTorrent for various server types in our docs.</p>
              <br />
              <p>Dedicated servers:</p>
              <DocsLink href="https://autobrr.com/configuration/download-clients/dedicated#rtorrent--rutorrent" />
              <p>Shared seedbox providers:</p>
              <DocsLink href="https://autobrr.com/configuration/download-clients/shared-seedboxes#rtorrent" />
            </div>
          }
        />
      </ContextField>

      <ContextField name="tls">
        <SwitchGroupWide label="TLS" />
      </ContextField>

      {tls && (
        <ContextField name="tls_skip_verify">
          <SwitchGroupWide
            label="Skip TLS verification (insecure)"
          />
        </ContextField>
      )}

      <ContextField name="settings.auth.enabled">
        <SwitchGroupWide label="Auth" />
      </ContextField>

      {settings.auth?.enabled && (
        <>
          <ContextField name="settings.auth.type">
            <SelectFieldBasic
              label="Auth type"
              placeholder="Select auth type"
              options={DownloadClientAuthType}
              tooltip={<p>This should in most cases be Basic Auth, but some providers use Digest Auth.</p>}
            />
          </ContextField>
          <ContextField name="settings.auth.username">
            <TextFieldWide label="Username" />
          </ContextField>
          <ContextField name="settings.auth.password">
            <PasswordFieldWide label="Password" />
          </ContextField>
        </>
      )}
    </div>
  );
}

function FormFieldsTransmission() {
  const form = useFormContext() as any;
  const tls = useStore(form.store, (s: any) => s.values.tls);

  return (
    <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
      <ContextField name="host">
        <TextFieldWide
          required
          label="Host"
          help="Eg. client.domain.ltd, domain.ltd/client, domain.ltd"
          tooltip={
            <div>
              <p>See guides for how to connect to Transmission for various server types in our docs.</p>
              <br />
              <p>Dedicated servers:</p>
              <DocsLink href="https://autobrr.com/configuration/download-clients/dedicated#transmission" />
              <p>Shared seedbox providers:</p>
              <DocsLink href="https://autobrr.com/configuration/download-clients/shared-seedboxes#transmisison" />
            </div>
          }
        />
      </ContextField>

      <ContextField name="port">
        <NumberFieldWide label="Port" help="Port for Transmission" />
      </ContextField>

      <ContextField name="tls">
        <SwitchGroupWide label="TLS" />
      </ContextField>

      {tls && (
        <ContextField name="tls_skip_verify">
          <SwitchGroupWide
            label="Skip TLS verification (insecure)"
          />
        </ContextField>
      )}

      <ContextField name="username">
        <TextFieldWide label="Username" />
      </ContextField>
      <ContextField name="password">
        <PasswordFieldWide label="Password" />
      </ContextField>
    </div>
  );
}

function FormFieldsSabnzbd() {
  const form = useFormContext() as any;
  const port = useStore(form.store, (s: any) => s.values.port);
  const tls = useStore(form.store, (s: any) => s.values.tls);
  const settings = useStore(form.store, (s: any) => s.values.settings);

  return (
    <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
      <ContextField name="host">
        <TextFieldWide
          label="Host"
          help="Eg. http://ip:port or https://url.com/sabnzbd"
          tooltip={
            <div>
              <p>See our guides on how to connect to qBittorrent for various server types in our docs.</p>
              <br />
              <p>Dedicated servers:</p>
              <ExternalLink href="https://autobrr.com/configuration/download-clients/dedicated#qbittorrent" />
              <p>Shared seedbox providers:</p>
              <ExternalLink href="https://autobrr.com/configuration/download-clients/shared-seedboxes#qbittorrent" />
            </div>
          }
        />
      </ContextField>

      {port > 0 && (
        <ContextField name="port">
          <NumberFieldWide
            label="Port"
            help="port for SABnzbd"
          />
        </ContextField>
      )}

      <ContextField name="tls">
        <SwitchGroupWide label="TLS" />
      </ContextField>

      {tls && (
        <ContextField name="tls_skip_verify">
          <SwitchGroupWide
            label="Skip TLS verification (insecure)"
          />
        </ContextField>
      )}

      {/*<TextFieldWide name="username" label="Username" />*/}
      {/*<PasswordFieldWide name="password" label="Password" />*/}

      <ContextField name="settings.apikey">
        <PasswordFieldWide label="API key" />
      </ContextField>

      <ContextField name="settings.basic.auth">
        <SwitchGroupWide label="Basic auth" />
      </ContextField>

      {settings.basic?.auth === true && (
        <>
          <ContextField name="settings.basic.username">
            <TextFieldWide label="Username" />
          </ContextField>
          <ContextField name="settings.basic.password">
            <PasswordFieldWide label="Password" />
          </ContextField>
        </>
      )}
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
  SABNZBD: <FormFieldsSabnzbd />
};

function FormFieldsRulesBasic() {
  const form = useFormContext() as any;
  const settings = useStore(form.store, (s: any) => s.values.settings);

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-5">

      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">Rules</DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Manage max downloads.
        </p>
      </div>

      <ContextField name="settings.rules.enabled">
        <SwitchGroupWide label="Enabled" />
      </ContextField>

      {settings && settings.rules?.enabled === true && (
        <ContextField name="settings.rules.max_active_downloads">
          <NumberFieldWide
            label="Max active downloads"
            tooltip={
              <span>
                <p>Limit the amount of active downloads (0 is unlimited), to give the maximum amount of bandwidth and disk for the downloads.</p>
                <DocsLink href="https://autobrr.com/configuration/download-clients/dedicated#deluge-rules" />
                <br /><br />
                <p>See recommendations for various server types here:</p>
                <DocsLink href='https://autobrr.com/filters/examples#build-buffer' />
              </span>
            }
          />
        </ContextField>
      )}
    </div>
  );
}

function FormFieldsRulesArr() {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-5 px-2">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          Download Client
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Override download client to use. Can also be overridden per Filter Action.
        </p>
      </div>

      <ContextField name="settings.external_download_client">
        <TextFieldWide label="Client Name" tooltip={<div><p>Specify what client the arr should use by default. Can be overridden per filter action.</p></div>} />
      </ContextField>

      <ContextField name="settings.external_download_client_id">
        <NumberFieldWide label="Client ID DEPRECATED" tooltip={<div><p>DEPRECATED: Use Client name field instead.</p></div>} />
      </ContextField>
    </div>
  );
}

function FormFieldsRulesQbit() {
  const form = useFormContext() as any;
  const settings = useStore(form.store, (s: any) => s.values.settings);

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-5 px-2">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          Rules
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Manage max downloads etc.
        </p>
      </div>

      <ContextField name="settings.rules.enabled">
        <SwitchGroupWide label="Enabled" />
      </ContextField>

      {settings.rules?.enabled === true && (
        <>
          <ContextField name="settings.rules.max_active_downloads">
            <NumberFieldWide
              label="Max active downloads"
              tooltip={
                <>
                  <p>Limit the amount of active downloads (0 is unlimited), to give the maximum amount of bandwidth and disk for the downloads.</p>
                  <DocsLink href="https://autobrr.com/configuration/download-clients/dedicated#qbittorrent-rules" />
                  <br /><br />
                  <p>See recommendations for various server types here:</p>
                  <DocsLink href="https://autobrr.com/filters/examples#build-buffer" />
                </>
              }
            />
          </ContextField>

          <ContextField name="settings.rules.ignore_slow_torrents">
            <SwitchGroupWide
              label="Ignore slow torrents"
            />
          </ContextField>

          {settings.rules?.ignore_slow_torrents === true && (
            <>
              <ContextField name="settings.rules.ignore_slow_torrents_condition">
                <SelectFieldBasic
                  label="Ignore condition"
                  placeholder="Select ignore condition"
                  options={DownloadRuleConditionOptions}
                  tooltip={<p>Choose whether to respect or ignore the <code className="text-blue-400">Max active downloads</code> setting before checking speed thresholds.</p>}
                />
              </ContextField>
              <ContextField name="settings.rules.download_speed_threshold">
                <NumberFieldWide
                  label="Download speed threshold"
                  placeholder="in KB/s"
                  help="If download speed is below this when max active downloads is hit, download anyways. KB/s"
                />
              </ContextField>
              <ContextField name="settings.rules.upload_speed_threshold">
                <NumberFieldWide
                  label="Upload speed threshold"
                  placeholder="in KB/s"
                  help="If upload speed is below this when max active downloads is hit, download anyways. KB/s"
                />
              </ContextField>
            </>
          )}
        </>
      )}
    </div>
  );
}

function FormFieldsRulesTransmission() {
  const form = useFormContext() as any;
  const settings = useStore(form.store, (s: any) => s.values.settings);

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-5 px-2">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          Rules
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Manage max downloads etc.
        </p>
      </div>

      <ContextField name="settings.rules.enabled">
        <SwitchGroupWide label="Enabled" />
      </ContextField>

      {settings.rules?.enabled === true && (
        <>
          <ContextField name="settings.rules.max_active_downloads">
            <NumberFieldWide
              label="Max active downloads"
              tooltip={
                <>
                  <p>Limit the amount of active downloads (0 is unlimited), to give the maximum amount of bandwidth and disk for the downloads.</p>
                  <DocsLink href="https://autobrr.com/configuration/download-clients/dedicated#transmission-rules" />
                  <br /><br />
                  <p>See recommendations for various server types here:</p>
                  <DocsLink href="https://autobrr.com/filters/examples#build-buffer" />
                </>
              }
            />
          </ContextField>
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
            Remove
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
              "OK!"
            ) : isErrorTest ? (
              "ERROR"
            ) : (
              "Test"
            )}
          </button>

          <button
            type="button"
            className="mr-4 bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
            onClick={cancelFn}
          >
            Cancel
          </button>
          <button
            type="submit"
            className="inline-flex justify-center py-2 px-4 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
          >
            {type === "CREATE" ? "Create" : "Save"}
          </button>
        </div>
      </div>
    </div>
  );
}

export function DownloadClientAddForm({ isOpen, toggle }: AddFormProps) {
  const [isTesting, setIsTesting] = useState(false);
  const [isSuccessfulTest, setIsSuccessfulTest] = useState(false);
  const [isErrorTest, setIsErrorTest] = useState(false);

  const queryClient = useQueryClient();

  const addMutation = useMutation({
    mutationFn: (client: DownloadClient) => APIClient.download_clients.create(client),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: DownloadClientKeys.lists() });
      toast.custom((t) => <Toast type="success" body="Client was added" t={t} />);

      toggle();
    },
    onError: () => {
      toast.custom((t) => <Toast type="error" body="Client could not be added" t={t} />);
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
      console.log("not added");
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

  const form = useAppForm({
    defaultValues: initialValues,
    onSubmit: async ({ value }) => onSubmit(value)
  });

  const formValues = useStore(form.store, (s) => s.values);

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
                              Add client
                            </DialogTitle>
                            <p className="text-sm text-gray-500 dark:text-gray-400">
                              Add download client.
                            </p>
                          </div>
                          <div className="h-7 flex items-center">
                            <button
                              type="button"
                              className="bg-white dark:bg-gray-800 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                              onClick={toggle}
                            >
                              <span className="sr-only">Close panel</span>
                              <XMarkIcon
                                className="h-6 w-6"
                                aria-hidden="true"
                              />
                            </button>
                          </div>
                        </div>
                      </div>

                      <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
                        <ContextField name="name">
                          <TextFieldWide required label="Name" />
                        </ContextField>
                        <ContextField name="enabled">
                          <SwitchGroupWide label="Enabled" />
                        </ContextField>
                        <ContextField name="type">
                          <RadioFieldsetWide
                            legend="Type"
                            options={DownloadClientTypeOptions}
                          />
                        </ContextField>
                        <div>{componentMap[formValues.type]}</div>
                      </div>
                    </div>

                    {rulesComponentMap[formValues.type]}

                    <DownloadClientFormButtons
                      type="CREATE"
                      isTesting={isTesting}
                      isSuccessfulTest={isSuccessfulTest}
                      isErrorTest={isErrorTest}
                      cancelFn={toggle}
                      testFn={testClient}
                      values={formValues}
                    />

                    <DEBUG values={formValues} />
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

export function DownloadClientUpdateForm({ isOpen, toggle, data: client}: UpdateFormProps<DownloadClient>) {
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

      toast.custom((t) => <Toast type="success" body={`${client.name} was updated successfully`} t={t} />);
      toggle();
    }
  });

  const onSubmit = (data: unknown) => mutation.mutate(data as DownloadClient);

  const deleteMutation = useMutation({
    mutationFn: (clientID: number) => APIClient.download_clients.delete(clientID),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: DownloadClientKeys.lists() });
      queryClient.invalidateQueries({ queryKey: DownloadClientKeys.detail(client.id) });

      toast.custom((t) => <Toast type="success" body={`${client.name} was deleted.`} t={t} />);
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

  const form = useAppForm({
    defaultValues: initialValues,
    onSubmit: async ({ value }) => onSubmit(value)
  });

  const formValues = useStore(form.store, (s) => s.values);

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
          title="Remove download client"
          text="Are you sure you want to remove this download client? This action cannot be undone."
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
                              Edit client
                            </DialogTitle>
                            <p className="text-sm text-gray-500 dark:text-gray-400">
                              Edit download client settings.
                            </p>
                          </div>
                          <div className="h-7 flex items-center">
                            <button
                              type="button"
                              className="bg-white dark:bg-gray-800 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                              onClick={toggle}
                            >
                              <span className="sr-only">Close panel</span>
                              <XMarkIcon
                                className="h-6 w-6"
                                aria-hidden="true"
                              />
                            </button>
                          </div>
                        </div>
                      </div>

                      <div className="py-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y dark:divide-gray-700">
                        <ContextField name="name">
                          <TextFieldWide required label="Name" />
                        </ContextField>
                        <ContextField name="enabled">
                          <SwitchGroupWide label="Enabled" />
                        </ContextField>
                        <ContextField name="type">
                          <RadioFieldsetWide
                            legend="Type"
                            options={DownloadClientTypeOptions}
                          />
                        </ContextField>
                        <div>{componentMap[formValues.type]}</div>
                      </div>
                    </div>

                    {rulesComponentMap[formValues.type]}

                    <DownloadClientFormButtons
                      type="UPDATE"
                      toggleDeleteModal={toggleDeleteModal}
                      isTesting={isTesting}
                      isSuccessfulTest={isSuccessfulTest}
                      isErrorTest={isErrorTest}
                      cancelFn={toggle}
                      testFn={testClient}
                      values={formValues}
                    />

                    <DEBUG values={formValues} />
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
