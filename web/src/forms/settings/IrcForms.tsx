/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "react-hot-toast";
import { XMarkIcon } from "@heroicons/react/24/solid";
import type { FieldProps } from "formik";
import type { FieldArrayRenderProps } from "formik";
import { Field, FieldArray, FormikErrors, FormikValues } from "formik";
import { ExclamationTriangleIcon } from "@heroicons/react/24/outline";
import Select from "react-select";
import { DialogTitle } from "@headlessui/react";

import { IrcAuthMechanismTypeOptions, OptionBasicTyped } from "@domain/constants";
import { APIClient } from "@api/APIClient";
import { IrcKeys } from "@api/query_keys";
import { NumberFieldWide, PasswordFieldWide, SwitchButton, SwitchGroupWide, TextFieldWide } from "@components/inputs";
import { SlideOver } from "@components/panels";
import Toast from "@components/notifications/Toast";
import * as common from "@components/inputs/common";
import { classNames } from "@utils";
import { ProxiesQueryOptions } from "@api/queries";
import { AddFormProps, UpdateFormProps } from "@forms/_shared";

interface ChannelsFieldArrayProps {
  channels: IrcChannel[];
}

const ChannelsFieldArray = ({ channels }: ChannelsFieldArrayProps) => (
  <div className="px-4">
    <FieldArray name="channels">
      {({ remove, push }: FieldArrayRenderProps) => (
        <div className="flex flex-col space-y-2">
          {channels && channels.length > 0 ? (
            channels.map((channel: IrcChannel, index) => {
              const isDisabled = channel.name === "#ptp-announce-dev";
              return (
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
                            "block w-full shadow-sm sm:text-sm rounded-md border py-2.5",
                            isDisabled ? "disabled dark:bg-gray-700 dark:text-gray-400 cursor-not-allowed" : "bg-gray-100 dark:bg-gray-850 dark:text-gray-100"
                          )}
                          disabled={isDisabled}
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
                          placeholder="Channel password"
                          className={classNames(
                            meta.touched && meta.error
                              ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                              : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
                            "block w-full shadow-sm sm:text-sm rounded-md border py-2.5",
                            isDisabled ? "disabled dark:bg-gray-700 dark:text-white cursor-not-allowed" : "bg-gray-100 dark:bg-gray-850 dark:text-gray-100"
                          )}
                          disabled={isDisabled}
                        />
                      )}
                    </Field>
                  </div>

                  <button
                    type="button"
                    className={classNames(
                      "bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-500",
                      isDisabled ? "hidden" : ""
                    )}
                    onClick={() => remove(index)}
                    disabled={isDisabled}
                  >
                    <span className="sr-only">Remove</span>
                    <XMarkIcon className="h-6 w-6" aria-hidden="true" />
                  </button>
                </div>
              );
            })
          ) : (
            <span className="text-center text-sm text-grey-darker dark:text-white">
              No channels!
            </span>
          )}
          <button
            type="button"
            className="border dark:border-gray-600 dark:bg-gray-700 my-4 px-4 py-2 text-sm text-gray-700 dark:text-white hover:bg-gray-50 dark:hover:bg-gray-600 rounded self-center text-center"
            onClick={() => push({ name: "", password: "" })}
          >
            Add Channel
          </button>
        </div>
      )}
    </FieldArray>
  </div>
);
interface IrcNetworkAddFormValues {
    name: string;
    enabled: boolean;
    server : string;
    port: number;
    tls: boolean;
    pass: string;
    nick: string;
    auth: IrcAuth;
    channels: IrcChannel[];
}

export function IrcNetworkAddForm({ isOpen, toggle }: AddFormProps) {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (network: IrcNetwork) => APIClient.irc.createNetwork(network),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: IrcKeys.lists() });

      toast.custom((t) => <Toast type="success" body="IRC Network added. Please allow up to 30 seconds for the network to come online." t={t} />);
      toggle();
    },
    onError: () => {
      toast.custom((t) => <Toast type="error" body="IRC Network could not be added" t={t} />);
    }
  });

  const onSubmit = (data: unknown) => mutation.mutate(data as IrcNetwork);

  const initialValues: IrcNetworkAddFormValues = {
    name: "",
    enabled: true,
    server: "",
    port: 6667,
    tls: false,
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
      title="Network"
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      initialValues={initialValues}
      validate={validateNetwork}
    >
      {(values) => (
        <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
          <div className="flex justify-center dark:bg-red-300 text-sm font-bold text-center p-4 py-8 dark:text-red-800"><span className="flex"><ExclamationTriangleIcon className="mr-2 h-6 w-6" /> ADD NETWORKS VIA INDEXERS! ONLY USE THIS IF YOU DELETED NETWORKS</span></div>

          <TextFieldWide
            name="name"
            label="Name"
            placeholder="Name"
            required={true}
          />

          <SwitchGroupWide name="enabled" label="Enabled" />
          <TextFieldWide
            name="server"
            label="Server"
            placeholder="Address: Eg irc.server.net"
            required={true}
          />
          <NumberFieldWide
            name="port"
            label="Port"
            placeholder="Eg 6667"
            required={true}
          />
          <SwitchGroupWide name="tls" label="TLS" />
          <PasswordFieldWide
            name="pass"
            label="Password"
            help="Network password"
          />
          <TextFieldWide
            name="nick"
            label="Nick"
            placeholder="bot nick"
            required={true}
          />
          <TextFieldWide
            name="auth.account"
            label="Auth Account"
            placeholder="Auth Account"
            required={true}
          />
          <PasswordFieldWide
            name="auth.password"
            label="Auth Password"
          />
          <PasswordFieldWide name="invite_command" label="Invite command" />

          <div className="border-t border-gray-200 dark:border-gray-700 py-5">
            <div className="px-4 space-y-1 mb-8">
              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">Channels</DialogTitle>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Channels to join.
              </p>
            </div>

            <ChannelsFieldArray channels={values.channels} />
          </div>
        </div>
      )}
    </SlideOver>
  );
}

const validateNetwork = (values: FormikValues) => {
  const errors = {} as FormikErrors<FormikValues>;

  if (!values.name) {
    errors.name = "Required";
  }

  if (!values.server) {
    errors.server = "Required";
  }

  if (!values.port) {
    errors.port = "Required";
  }

  if (!values.nick) {
    errors.nick = "Required";
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
  const queryClient = useQueryClient();

  const proxies = useQuery(ProxiesQueryOptions());

  const updateMutation = useMutation({
    mutationFn: (network: IrcNetwork) => APIClient.irc.updateNetwork(network),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: IrcKeys.lists() });

      toast.custom((t) => <Toast type="success" body={`${network.name} was updated successfully`} t={t} />);

      toggle();
    }
  });

  const onSubmit = (data: unknown) => updateMutation.mutate(data as IrcNetwork);

  const deleteMutation = useMutation({
    mutationFn: (id: number) => APIClient.irc.deleteNetwork(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: IrcKeys.lists() });

      toast.custom((t) => <Toast type="success" body={`${network.name} was deleted.`} t={t} />);

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
      title="Network"
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      deleteAction={deleteAction}
      initialValues={initialValues}
      validate={validateNetwork}
    >
      {(values) => (
        <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
          <TextFieldWide
            name="name"
            label="Name"
            placeholder="Name"
            required={true}
          />

          <SwitchGroupWide name="enabled" label="Enabled"/>
          <TextFieldWide
            name="server"
            label="Server"
            placeholder="Address: Eg irc.server.net"
            required={true}
          />
          <NumberFieldWide
            name="port"
            label="Port"
            placeholder="Eg 6667"
            required={true}
          />

          <SwitchGroupWide name="tls" label="TLS"/>

          <PasswordFieldWide
            name="pass"
            label="Password"
            help="Network password, not commonly used."
          />

          <TextFieldWide
            name="nick"
            label="Nick"
            placeholder="nick"
            required={true}
          />

          <SwitchGroupWide name="use_bouncer" label="Bouncer (BNC)"/>
          {values.use_bouncer && (
            <TextFieldWide
              name="bouncer_addr"
              label="Bouncer address"
              help="Address: Eg bouncer.server.net:6697"
            />
          )}

          <SwitchGroupWide name="bot_mode" label="IRCv3 Bot Mode"/>

          <div className="border-t border-gray-200 dark:border-gray-700 py-4">
            <div className="flex justify-between px-4">
              <div className="space-y-1">
                <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
                  Proxy
                </DialogTitle>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Set a proxy to be used for connecting to the irc server.
                </p>
              </div>
              <SwitchButton name="use_proxy"/>
            </div>

            {values.use_proxy === true && (
              <div className="py-4 pt-6">
                <SelectField<number>
                  name="proxy_id"
                  label="Select proxy"
                  placeholder="Select a proxy"
                  options={proxies.data ? proxies.data.map((p) => ({ label: p.name, value: p.id })) : []}
                />
              </div>
            )}
          </div>

          <div className="border-t border-gray-200 dark:border-gray-700 py-5">
            <div className="px-4 space-y-1 mb-8">
              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">Identification</DialogTitle>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Identify with SASL or NickServ. Most networks support SASL but some don't.
              </p>
            </div>

            <SelectField<IrcAuthMechanism>
              name="auth.mechanism"
              label="Mechanism"
              options={IrcAuthMechanismTypeOptions}
            />

            <TextFieldWide
              name="auth.account"
              label="Account"
              placeholder="Auth Account"
              help="NickServ / SASL account. For grouped nicks try the main."
            />

            <PasswordFieldWide
              name="auth.password"
              label="Password"
              help="NickServ / SASL password."
            />
          </div>

          <PasswordFieldWide name="invite_command" label="Invite command"/>

          <div className="border-t border-gray-200 dark:border-gray-700 py-5">
            <div className="px-4 space-y-1 mb-8">
              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">Channels</DialogTitle>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Channels are added when you setup IRC indexers. Do not edit unless you know what you are doing.
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
              placeholder={placeholder ?? "Choose a type"}
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
