/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { XMarkIcon } from "@heroicons/react/24/solid";
import { ExclamationTriangleIcon } from "@heroicons/react/24/outline";
import { useTranslation } from "react-i18next";

import { APIClient } from "@api/APIClient";
import { AlertKeys, IrcKeys } from "@api/query_keys";
import { fieldHasError, useFormContext } from "@hooks/form";
import type { FormFieldErrors } from "@hooks/form";
import { NumberFieldWide, PasswordFieldWide, SwitchButton, SwitchGroupWide, TextFieldWide } from "@components/inputs";
import { SlideOver } from "@components/panels";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { classNames } from "@utils";
import { ProxiesQueryOptions } from "@api/queries";
import { AddFormProps, UpdateFormProps } from "@forms/_shared";
import { SelectField } from "@forms/settings/IrcForms";

interface RoomsFieldArrayProps {
  channels: IrcChannel[];
}

const RoomsFieldArray = ({ channels }: RoomsFieldArrayProps) => {
  const { t } = useTranslation("settings");
  const form = useFormContext();

  return (
    <div className="px-4">
      <div className="flex flex-col space-y-2">
        {channels && channels.length > 0 ? (
          channels.map((channel, index) => (
            <div key={channel.id || `new-${index}`} className="flex justify-between border dark:border-gray-700 dark:bg-gray-815 p-2 rounded-md">
              <div className="flex gap-2 items-center w-full pr-2">
                <div className="shrink-0" title={t("forms.irc.channelEnabled")}>
                  <SwitchButton name={`channels[${index}].enabled`} defaultValue={true} />
                </div>
                <form.Field name={`channels[${index}].name`}>
                  {(field) => (
                    <input
                      name={field.name}
                      type="text"
                      placeholder="#blu-announce or !roomId:matrix.server"
                      value={field.state.value ?? ""}
                      onChange={(e) => field.handleChange(e.target.value)}
                      onBlur={field.handleBlur}
                      className={classNames(
                        fieldHasError(field.state.meta)
                          ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                          : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
                        "block w-full shadow-xs sm:text-sm rounded-md border py-2.5 bg-gray-100 dark:bg-gray-850 dark:text-gray-100"
                      )}
                    />
                  )}
                </form.Field>
              </div>

              <button
                type="button"
                className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer"
                onClick={() => form.removeFieldValue("channels", index)}
              >
                <span className="sr-only">{t("forms.irc.remove")}</span>
                <XMarkIcon className="h-6 w-6" aria-hidden="true" />
              </button>
            </div>
          ))
        ) : (
          <span className="text-center text-sm text-grey-darker dark:text-white">
            No rooms configured!
          </span>
        )}
        <button
          type="button"
          className="border dark:border-gray-600 dark:bg-gray-700 my-4 px-4 py-2 text-sm text-gray-700 dark:text-white hover:bg-gray-50 dark:hover:bg-gray-600 rounded-sm self-center text-center cursor-pointer"
          onClick={() => form.pushFieldValue("channels", { name: "", password: "", enabled: true })}
        >
          Add Matrix Room
        </button>
      </div>
    </div>
  );
};

interface MatrixNetworkFormValues {
  id?: number;
  name: string;
  enabled: boolean;
  server: string;
  port: number;
  tls: boolean;
  tls_skip_verify: boolean;
  pass: string;
  nick: string;
  auth: IrcAuth;
  invite_command?: string;
  use_bouncer?: boolean;
  bouncer_addr?: string;
  bot_mode?: boolean;
  channels: IrcChannel[];
  use_proxy?: boolean;
  proxy_id?: number;
}

const validateMatrixNetwork = (values: MatrixNetworkFormValues, requiredMessage: string) => {
  const errors: FormFieldErrors = {};
  if (!values.name) {
    errors.name = requiredMessage;
  }
  if (!values.server) {
    errors.server = requiredMessage;
  }
  if (!values.port) {
    errors.port = requiredMessage;
  }
  if (!values.nick) {
    errors.nick = requiredMessage;
  }
  return errors;
};

export function MatrixNetworkAddForm({ isOpen, toggle }: AddFormProps) {
  const { t } = useTranslation("settings");
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (network: IrcNetwork) => APIClient.irc.createNetwork(network),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: IrcKeys.lists() });
      toast.custom((toastInstance) => <Toast type="success" body="Matrix homeserver added." t={toastInstance} />);
      toggle();
    },
    onError: () => {
      toast.custom((toastInstance) => <Toast type="error" body="Matrix homeserver could not be added" t={toastInstance} />);
    }
  });

  const onSubmit = (data: MatrixNetworkFormValues) => mutation.mutate(data as IrcNetwork);

  const initialValues: MatrixNetworkFormValues = {
    name: "Watchtower-Matrix",
    enabled: true,
    server: "matrix.watchtower.pm",
    port: 443,
    tls: true,
    tls_skip_verify: false,
    pass: "",
    nick: "",
    auth: {
      mechanism: "NONE"
    },
    channels: [
      { id: 0, name: "#blu-announce", password: "", enabled: true, detached: false, monitoring: false }
    ]
  };

  return (
    <SlideOver
      type="CREATE"
      title="Matrix Homeserver"
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      initialValues={initialValues}
      validate={(values) => validateMatrixNetwork(values, t("forms.irc.required"))}
    >
      {(values) => (
        <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
          <div className="flex justify-center dark:bg-purple-300 bg-purple-100 text-xs font-semibold text-center p-3 mx-4 rounded-md dark:text-purple-900 text-purple-900">
            <span className="flex items-center">
              <ExclamationTriangleIcon className="mr-2 h-5 w-5 shrink-0" />
              Matrix homeservers are automatically created when you add a Matrix indexer (e.g. Blutopia).
            </span>
          </div>

          <TextFieldWide
            name="name"
            label={t("forms.irc.name")}
            placeholder="Watchtower-Matrix"
            required={true}
          />

          <SwitchGroupWide name="enabled" label={t("forms.irc.enabled")} />

          <TextFieldWide
            name="server"
            label="Homeserver"
            placeholder="matrix.watchtower.pm"
            required={true}
          />
          <NumberFieldWide
            name="port"
            label={t("forms.irc.port")}
            placeholder="443"
            required={true}
          />
          <SwitchGroupWide name="tls" label={t("forms.irc.tls")} />
          {values.tls && (
            <SwitchGroupWide name="tls_skip_verify" label={t("forms.irc.skipTls")} />
          )}

          <TextFieldWide
            name="nick"
            label="Matrix Username"
            placeholder="Your Matrix username"
            required={true}
          />

          <PasswordFieldWide
            name="pass"
            label="Access Token"
            help="Matrix Access Token (e.g. mat_... or syt_...) from Element Settings -> Help & About -> Access Token."
          />

          <div className="border-t border-gray-200 dark:border-gray-700 py-5">
            <div className="px-4 space-y-1 mb-6">
              <h2 className="text-lg font-medium text-gray-900 dark:text-white">Matrix Rooms</h2>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Matrix room aliases (e.g. #blu-announce) or room IDs (!roomId:server) to monitor for announces.
              </p>
            </div>

            <RoomsFieldArray channels={values.channels} />
          </div>
        </div>
      )}
    </SlideOver>
  );
}

export function MatrixNetworkUpdateForm({ isOpen, toggle, data: network }: UpdateFormProps<IrcNetwork>) {
  const { t } = useTranslation("settings");
  const queryClient = useQueryClient();
  const proxies = useQuery(ProxiesQueryOptions());

  const updateMutation = useMutation({
    mutationFn: (net: IrcNetwork) => APIClient.irc.updateNetwork(net),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: IrcKeys.lists() });
      queryClient.invalidateQueries({ queryKey: AlertKeys.all });
      toast.custom((toastInstance) => <Toast type="success" body={t("forms.irc.updated", { name: network.name })} t={toastInstance} />);
      toggle();
    }
  });

  const onSubmit = (data: MatrixNetworkFormValues) => updateMutation.mutate(data as IrcNetwork);

  const deleteMutation = useMutation({
    mutationFn: (id: number) => APIClient.irc.deleteNetwork(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: IrcKeys.lists() });
      queryClient.invalidateQueries({ queryKey: AlertKeys.all });
      toast.custom((toastInstance) => <Toast type="success" body={t("forms.irc.deleted", { name: network.name })} t={toastInstance} />);
      toggle();
    }
  });

  const deleteAction = () => deleteMutation.mutate(network.id);

  const initialValues: MatrixNetworkFormValues = {
    id: network.id,
    name: network.name,
    enabled: network.enabled,
    server: network.server,
    port: network.port,
    tls: network.tls,
    tls_skip_verify: network.tls_skip_verify,
    nick: network.nick,
    pass: network.pass,
    auth: network.auth || { mechanism: "NONE" },
    invite_command: network.invite_command,
    use_bouncer: network.use_bouncer,
    bouncer_addr: network.bouncer_addr,
    bot_mode: network.bot_mode,
    channels: network.channels,
    use_proxy: network.use_proxy,
    proxy_id: network.proxy_id
  };

  return (
    <SlideOver
      type="UPDATE"
      title="Matrix Homeserver"
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      deleteAction={deleteAction}
      initialValues={initialValues}
      validate={(values) => validateMatrixNetwork(values, t("forms.irc.required"))}
    >
      {(values) => (
        <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
          <TextFieldWide
            name="name"
            label={t("forms.irc.name")}
            placeholder={t("forms.irc.name")}
            required={true}
          />

          <SwitchGroupWide name="enabled" label={t("forms.irc.enabled")} />

          <TextFieldWide
            name="server"
            label="Homeserver"
            placeholder="matrix.watchtower.pm"
            required={true}
          />
          <NumberFieldWide
            name="port"
            label={t("forms.irc.port")}
            placeholder="443"
            required={true}
          />

          <SwitchGroupWide name="tls" label={t("forms.irc.tls")} />
          {values.tls && (
            <SwitchGroupWide name="tls_skip_verify" label={t("forms.irc.skipTls")} />
          )}

          <TextFieldWide
            name="nick"
            label="Matrix Username"
            placeholder="Your Matrix username"
            required={true}
          />

          <PasswordFieldWide
            name="pass"
            label="Access Token"
            help="Matrix Access Token (e.g. mat_... or syt_...) from Element Settings -> Help & About -> Access Token."
          />

          <div className="border-t border-gray-200 dark:border-gray-700 py-4">
            <div className="flex justify-between px-4">
              <div className="space-y-1">
                <h2 className="text-lg font-medium text-gray-900 dark:text-white">
                  {t("forms.irc.proxy")}
                </h2>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Set a proxy to be used for connecting to the Matrix homeserver.
                </p>
              </div>
              <SwitchButton name="use_proxy" />
            </div>

            {values.use_proxy === true && (
              <div className="py-4 pt-6">
                <SelectField<number>
                  name="proxy_id"
                  label={t("forms.irc.selectProxy")}
                  placeholder={t("forms.irc.selectProxyPlaceholder")}
                  options={proxies.data ? proxies.data.map((p) => ({ label: p.name, value: p.id })) : []}
                />
              </div>
            )}
          </div>

          <div className="border-t border-gray-200 dark:border-gray-700 py-5">
            <div className="px-4 space-y-1 mb-6">
              <h2 className="text-lg font-medium text-gray-900 dark:text-white">Matrix Rooms</h2>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Rooms are added automatically when you set up Matrix indexers.
              </p>
            </div>

            <RoomsFieldArray channels={values.channels} />
          </div>
        </div>
      )}
    </SlideOver>
  );
}
