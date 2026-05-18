/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { XMarkIcon } from "@heroicons/react/24/solid";
import type { FieldProps } from "formik";
import type { FieldArrayRenderProps } from "formik";
import { Field, FieldArray, FormikErrors, FormikValues } from "formik";
import { ExclamationTriangleIcon } from "@heroicons/react/24/outline";
import Select from "react-select";
import { DialogTitle } from "@headlessui/react";
import { useTranslation } from "react-i18next";

import { IrcAuthMechanismTypeOptions, OptionBasicTyped } from "@domain/constants";
import { APIClient } from "@api/APIClient";
import { IrcKeys } from "@api/query_keys";
import { NumberFieldWide, PasswordFieldWide, SwitchButton, SwitchGroupWide, TextFieldWide } from "@components/inputs";
import { SlideOver } from "@components/panels";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import * as common from "@components/inputs/common";
import { classNames } from "@utils";
import { ProxiesQueryOptions } from "@api/queries";
import { AddFormProps, UpdateFormProps } from "@forms/_shared";

interface ChannelsFieldArrayProps {
  channels: IrcChannel[];
}

const ChannelsFieldArray = ({ channels }: ChannelsFieldArrayProps) => {
  const { t } = useTranslation("settings");

  return (
  <div className="px-4">
    <FieldArray name="channels">
      {({ remove, push }: FieldArrayRenderProps) => (
        <div className="flex flex-col space-y-2">
          {channels && channels.length > 0 ? (
              channels.map((_, index) => (
                <div key={index} className="flex justify-between border dark:border-gray-700 dark:bg-gray-815 p-2 rounded-md">
                  <div className="flex gap-2">
                    <Field name={`channels.${index}.name`}>
                      {({ field, meta }: FieldProps) => (
                        <input
                          {...field}
                          type="text"
                          value={field.value ?? ""}
                          onChange={field.onChange}
                          className={classNames(
                            meta.touched && meta.error
                              ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                              : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
                            "block w-full shadow-xs sm:text-sm rounded-md border py-2.5 bg-gray-100 dark:bg-gray-850 dark:text-gray-100"
                          )}
                        />
                      )}
                    </Field>

                    <Field name={`channels.${index}.password`}>
                      {({ field, meta }: FieldProps) => (
                        <input
                          {...field}
                          type="text"
                          value={field.value ?? ""}
                          onChange={field.onChange}
                          placeholder={t("forms.irc.channelPassword")}
                          className={classNames(
                            meta.touched && meta.error
                              ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                              : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
                            "block w-full shadow-xs sm:text-sm rounded-md border py-2.5 bg-gray-100 dark:bg-gray-850 dark:text-gray-100"
                          )}
                        />
                      )}
                    </Field>
                  </div>

                  <button
                    type="button"
                    className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                    onClick={() => remove(index)}
                  >
                    <span className="sr-only">{t("forms.irc.remove")}</span>
                    <XMarkIcon className="h-6 w-6" aria-hidden="true" />
                  </button>
                </div>
              ))
          ) : (
            <span className="text-center text-sm text-grey-darker dark:text-white">
              {t("forms.irc.noChannels")}
            </span>
          )}
          <button
            type="button"
            className="border dark:border-gray-600 dark:bg-gray-700 my-4 px-4 py-2 text-sm text-gray-700 dark:text-white hover:bg-gray-50 dark:hover:bg-gray-600 rounded-sm self-center text-center"
            onClick={() => push({ name: "", password: "" })}
          >
            {t("forms.irc.addChannel")}
          </button>
        </div>
      )}
    </FieldArray>
  </div>
  );
};
interface IrcNetworkAddFormValues {
    name: string;
    enabled: boolean;
    server : string;
    port: number;
    tls: boolean;
    tls_skip_verify: boolean;
    pass: string;
    nick: string;
    auth: IrcAuth;
    channels: IrcChannel[];
}

export function IrcNetworkAddForm({ isOpen, toggle }: AddFormProps) {
  const { t } = useTranslation("settings");
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (network: IrcNetwork) => APIClient.irc.createNetwork(network),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: IrcKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.irc.added")} t={toastInstance} />);
      toggle();
    },
    onError: () => {
      toast.custom((toastInstance) => <Toast type="error" body={t("forms.irc.addFailed")} t={toastInstance} />);
    }
  });

  const onSubmit = (data: unknown) => mutation.mutate(data as IrcNetwork);

  const initialValues: IrcNetworkAddFormValues = {
    name: "",
    enabled: true,
    server: "",
    port: 6667,
    tls: false,
    tls_skip_verify: false,
    pass: "",
    nick: "",
    auth: {
      mechanism: "SASL_PLAIN",
      account: ""
    },
    channels: []
  };

  return (
    <SlideOver
      type="CREATE"
      title={t("forms.irc.title")}
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      initialValues={initialValues}
      validate={(values) => {
        return validateNetwork(values, t("forms.irc.required"));
      }}
    >
      {(values) => (
        <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
          <div className="flex justify-center dark:bg-red-300 text-sm font-bold text-center p-4 py-8 dark:text-red-800"><span className="flex"><ExclamationTriangleIcon className="mr-2 h-6 w-6" /> {t("forms.irc.addWarning")}</span></div>

          <TextFieldWide
            name="name"
            label={t("forms.irc.name")}
            placeholder={t("forms.irc.name")}
            required={true}
          />

          <SwitchGroupWide name="enabled" label={t("forms.irc.enabled")} />
          <TextFieldWide
            name="server"
            label={t("forms.irc.server")}
            placeholder={t("forms.irc.serverPlaceholder")}
            required={true}
          />
          <NumberFieldWide
            name="port"
            label={t("forms.irc.port")}
            placeholder={t("forms.irc.portPlaceholder")}
            required={true}
          />
          <SwitchGroupWide name="tls" label={t("forms.irc.tls")} />
          {values.tls && (
            <SwitchGroupWide name="tls_skip_verify" label={t("forms.irc.skipTls")}/>
          )}
          <PasswordFieldWide
            name="pass"
            label={t("forms.irc.password")}
            help={t("forms.irc.passwordHelp")}
          />
          <TextFieldWide
            name="nick"
            label={t("forms.irc.nick")}
            placeholder={t("forms.irc.nickPlaceholderAdd")}
            required={true}
          />
          <TextFieldWide
            name="auth.account"
            label={t("forms.irc.authAccount")}
            placeholder={t("forms.irc.authAccountPlaceholder")}
            required={true}
          />
          <PasswordFieldWide
            name="auth.password"
            label={t("forms.irc.authPassword")}
          />
          <PasswordFieldWide name="invite_command" label={t("forms.irc.inviteCommand")} />

          <div className="border-t border-gray-200 dark:border-gray-700 py-5">
            <div className="px-4 space-y-1 mb-8">
              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">{t("forms.irc.channels")}</DialogTitle>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                {t("forms.irc.channelsDesc")}
              </p>
            </div>

            <ChannelsFieldArray channels={values.channels} />
          </div>
        </div>
      )}
    </SlideOver>
  );
}

const validateNetwork = (values: FormikValues, requiredMessage: string) => {
  const errors = {} as FormikErrors<FormikValues>;

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

interface IrcNetworkUpdateFormValues {
    id: number;
    name: string;
    enabled: boolean;
    server: string;
    port: number;
    tls: boolean;
    tls_skip_verify: boolean;
    pass: string;
    nick: string;
    auth?: IrcAuth;
    invite_command: string;
    use_bouncer: boolean;
    bouncer_addr: string;
    bot_mode: boolean;
    channels: Array<IrcChannel>;
    use_proxy: boolean;
    proxy_id: number;
}

export function IrcNetworkUpdateForm({
  isOpen,
  toggle,
  data: network
}: UpdateFormProps<IrcNetwork>) {
  const { t } = useTranslation("settings");
  const queryClient = useQueryClient();

  const proxies = useQuery(ProxiesQueryOptions());

  const updateMutation = useMutation({
    mutationFn: (network: IrcNetwork) => APIClient.irc.updateNetwork(network),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: IrcKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.irc.updated", { name: network.name })} t={toastInstance} />);

      toggle();
    }
  });

  const onSubmit = (data: unknown) => updateMutation.mutate(data as IrcNetwork);

  const deleteMutation = useMutation({
    mutationFn: (id: number) => APIClient.irc.deleteNetwork(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: IrcKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.irc.deleted", { name: network.name })} t={toastInstance} />);

      toggle();
    }
  });

  const deleteAction = () => deleteMutation.mutate(network.id);

  const initialValues: IrcNetworkUpdateFormValues = {
    id: network.id,
    name: network.name,
    enabled: network.enabled,
    server: network.server,
    port: network.port,
    tls: network.tls,
    tls_skip_verify: network.tls_skip_verify,
    nick: network.nick,
    pass: network.pass,
    auth: network.auth,
    invite_command: network.invite_command,
    use_bouncer: network.use_bouncer,
    bouncer_addr: network.bouncer_addr,
    bot_mode: network.bot_mode,
    channels: network.channels,
    use_proxy: network.use_proxy,
    proxy_id: network.proxy_id,
  };

  return (
    <SlideOver
      type="UPDATE"
      title={t("forms.irc.title")}
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      deleteAction={deleteAction}
      initialValues={initialValues}
      validate={(values) => {
        return validateNetwork(values, t("forms.irc.required"));
      }}
    >
      {(values) => (
        <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
          <TextFieldWide
            name="name"
            label={t("forms.irc.name")}
            placeholder={t("forms.irc.name")}
            required={true}
          />

          <SwitchGroupWide name="enabled" label={t("forms.irc.enabled")}/>
          <TextFieldWide
            name="server"
            label={t("forms.irc.server")}
            placeholder={t("forms.irc.serverPlaceholder")}
            required={true}
          />
          <NumberFieldWide
            name="port"
            label={t("forms.irc.port")}
            placeholder={t("forms.irc.portPlaceholder")}
            required={true}
          />

          <SwitchGroupWide name="tls" label={t("forms.irc.tls")}/>
          {values.tls && (
            <SwitchGroupWide name="tls_skip_verify" label={t("forms.irc.skipTls")}/>
          )}

          <PasswordFieldWide
            name="pass"
            label={t("forms.irc.password")}
            help={t("forms.irc.passwordUpdateHelp")}
          />

          <TextFieldWide
            name="nick"
            label={t("forms.irc.nick")}
            placeholder={t("forms.irc.nickPlaceholderUpdate")}
            required={true}
          />

          <SwitchGroupWide name="use_bouncer" label={t("forms.irc.bouncer")}/>
          {values.use_bouncer && (
            <TextFieldWide
              name="bouncer_addr"
              label={t("forms.irc.bouncerAddress")}
              help={t("forms.irc.bouncerAddressHelp")}
            />
          )}

          <SwitchGroupWide name="bot_mode" label={t("forms.irc.botMode")}/>

          <div className="border-t border-gray-200 dark:border-gray-700 py-4">
            <div className="flex justify-between px-4">
              <div className="space-y-1">
                <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
                  {t("forms.irc.proxy")}
                </DialogTitle>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  {t("forms.irc.proxyDesc")}
                </p>
              </div>
              <SwitchButton name="use_proxy"/>
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
            <div className="px-4 space-y-1 mb-8">
              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">{t("forms.irc.identification")}</DialogTitle>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                {t("forms.irc.identificationDesc")}
              </p>
            </div>

            <SelectField<IrcAuthMechanism>
              name="auth.mechanism"
              label={t("forms.irc.mechanism")}
              options={IrcAuthMechanismTypeOptions}
            />

            <TextFieldWide
              name="auth.account"
              label={t("forms.irc.account")}
              placeholder={t("forms.irc.authAccountPlaceholder")}
              help={t("forms.irc.accountHelp")}
            />

            <PasswordFieldWide
              name="auth.password"
              label={t("forms.irc.password")}
              help={t("forms.irc.passwordSaslHelp")}
            />
          </div>

          <PasswordFieldWide name="invite_command" label={t("forms.irc.inviteCommand")}/>

          <div className="border-t border-gray-200 dark:border-gray-700 py-5">
            <div className="px-4 space-y-1 mb-8">
              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">{t("forms.irc.channels")}</DialogTitle>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                {t("forms.irc.channelsUpdateDesc")}
              </p>
            </div>

            <ChannelsFieldArray channels={values.channels}/>
          </div>
        </div>
      )}
    </SlideOver>
  );
}

interface SelectFieldProps<T> {
  name: string;
  label: string;
  options: OptionBasicTyped<T>[]
  placeholder?: string;
}

export function SelectField<T>({ name, label, options, placeholder }: SelectFieldProps<T>) {
  const { t } = useTranslation("settings");
  return (
    <div className="flex items-center justify-between space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label
          htmlFor={name}
          className="block text-sm font-medium text-gray-900 dark:text-white"
        >
          {label}
        </label>
      </div>
      <div className="sm:col-span-2">
        <Field name={name} type="select">
          {({
              field,
              form: { setFieldValue }
            }: FieldProps) => (
            <Select
              {...field}
              id={name}
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
              placeholder={placeholder ?? t("forms.irc.chooseType")}
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
              value={field?.value && options.find(o => o.value == field?.value)}
              onChange={(newValue: unknown) => {
                if (newValue) {
                  setFieldValue(field.name, (newValue as { value: number }).value);
                }
                else {
                  setFieldValue(field.name, 0)
                }
              }}
              options={options}
            />
          )}
        </Field>
      </div>
    </div>
  );
}
