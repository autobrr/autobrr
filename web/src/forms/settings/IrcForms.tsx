/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { XMarkIcon } from "@heroicons/react/24/solid";
import { ExclamationTriangleIcon } from "@heroicons/react/24/outline";
import Select from "react-select";
import { DialogTitle } from "@headlessui/react";

import { IrcAuthMechanismTypeOptions, OptionBasicTyped } from "@domain/constants";
import { APIClient } from "@api/APIClient";
import { IrcKeys } from "@api/query_keys";
import { NumberFieldWide, PasswordFieldWide, SwitchGroupWide, SwitchButton, TextFieldWide } from "@components/inputs/tanstack";
import { SlideOver } from "@components/panels";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import * as common from "@components/inputs/tanstack/common";
import { classNames } from "@utils";
import { ProxiesQueryOptions } from "@api/queries";
import { AddFormProps, UpdateFormProps } from "@forms/_shared";
import { ContextField, useFormContext, useFieldContext, useStore } from "@app/lib/form";

const ChannelsFieldArray = () => {
  const form = useFormContext();
  const currentChannels = useStore(form.store, (s: any) => s.values.channels) as IrcChannel[];

  return (
    <div className="px-4">
      <div className="flex flex-col space-y-2">
        {currentChannels && currentChannels.length > 0 ? (
          currentChannels.map((channel, index) => (
            <div key={index} className="flex justify-between border dark:border-gray-700 dark:bg-gray-815 p-2 rounded-md">
              <div className="flex gap-2">
                <input
                  type="text"
                  value={channel.name ?? ""}
                  onChange={(e) => {
                    const newChannels = [...currentChannels];
                    newChannels[index] = { ...newChannels[index], name: e.target.value };
                    (form as any).setFieldValue("channels", newChannels);
                  }}
                  className={classNames(
                    "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
                    "block w-full shadow-xs sm:text-sm rounded-md border py-2.5 bg-gray-100 dark:bg-gray-850 dark:text-gray-100"
                  )}
                />

                <input
                  type="text"
                  value={channel.password ?? ""}
                  onChange={(e) => {
                    const newChannels = [...currentChannels];
                    newChannels[index] = { ...newChannels[index], password: e.target.value };
                    (form as any).setFieldValue("channels", newChannels);
                  }}
                  placeholder="Channel password"
                  className={classNames(
                    "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
                    "block w-full shadow-xs sm:text-sm rounded-md border py-2.5 bg-gray-100 dark:bg-gray-850 dark:text-gray-100"
                  )}
                />
              </div>

              <button
                type="button"
                className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                onClick={() => {
                  const newChannels = currentChannels.filter((_, i) => i !== index);
                  (form as any).setFieldValue("channels", newChannels);
                }}
              >
                <span className="sr-only">Remove</span>
                <XMarkIcon className="h-6 w-6" aria-hidden="true" />
              </button>
            </div>
          ))
        ) : (
          <span className="text-center text-sm text-grey-darker dark:text-white">
            No channels!
          </span>
        )}
        <button
          type="button"
          className="border dark:border-gray-600 dark:bg-gray-700 my-4 px-4 py-2 text-sm text-gray-700 dark:text-white hover:bg-gray-50 dark:hover:bg-gray-600 rounded-sm self-center text-center"
          onClick={() => {
            const newChannels = [...currentChannels, { name: "", password: "" }];
            (form as any).setFieldValue("channels", newChannels);
          }}
        >
          Add Channel
        </button>
      </div>
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
      title="Network"
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      initialValues={initialValues}
    >
      {(values) => (
        <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
          <div className="flex justify-center dark:bg-red-300 text-sm font-bold text-center p-4 py-8 dark:text-red-800"><span className="flex"><ExclamationTriangleIcon className="mr-2 h-6 w-6" /> ADD NETWORKS VIA INDEXERS! ONLY USE THIS IF YOU DELETED NETWORKS</span></div>

          <ContextField name="name">
            <TextFieldWide
              label="Name"
              placeholder="Name"
              required={true}
            />
          </ContextField>

          <ContextField name="enabled">
            <SwitchGroupWide label="Enabled" />
          </ContextField>
          <ContextField name="server">
            <TextFieldWide
              label="Server"
              placeholder="Address: Eg irc.server.net"
              required={true}
            />
          </ContextField>
          <ContextField name="port">
            <NumberFieldWide
              label="Port"
              placeholder="Eg 6667"
              required={true}
            />
          </ContextField>
          <ContextField name="tls">
            <SwitchGroupWide label="TLS" />
          </ContextField>
          {values.tls && (
            <ContextField name="tls_skip_verify">
              <SwitchGroupWide label="Skip TLS verification (insecure)"/>
            </ContextField>
          )}
          <ContextField name="pass">
            <PasswordFieldWide
              label="Password"
              help="Network password"
            />
          </ContextField>
          <ContextField name="nick">
            <TextFieldWide
              label="Nick"
              placeholder="bot nick"
              required={true}
            />
          </ContextField>
          <ContextField name="auth.account">
            <TextFieldWide
              label="Auth Account"
              placeholder="Auth Account"
              required={true}
            />
          </ContextField>
          <ContextField name="auth.password">
            <PasswordFieldWide
              label="Auth Password"
            />
          </ContextField>
          <ContextField name="invite_command">
            <PasswordFieldWide label="Invite command" />
          </ContextField>

          <div className="border-t border-gray-200 dark:border-gray-700 py-5">
            <div className="px-4 space-y-1 mb-8">
              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">Channels</DialogTitle>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Channels to join.
              </p>
            </div>

            <ChannelsFieldArray />
          </div>
        </div>
      )}
    </SlideOver>
  );
}

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
      title="Network"
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      deleteAction={deleteAction}
      initialValues={initialValues}
    >
      {(values) => (
        <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
          <ContextField name="name">
            <TextFieldWide
              label="Name"
              placeholder="Name"
              required={true}
            />
          </ContextField>

          <ContextField name="enabled">
            <SwitchGroupWide label="Enabled"/>
          </ContextField>
          <ContextField name="server">
            <TextFieldWide
              label="Server"
              placeholder="Address: Eg irc.server.net"
              required={true}
            />
          </ContextField>
          <ContextField name="port">
            <NumberFieldWide
              label="Port"
              placeholder="Eg 6667"
              required={true}
            />
          </ContextField>

          <ContextField name="tls">
            <SwitchGroupWide label="TLS"/>
          </ContextField>
          {values.tls && (
            <ContextField name="tls_skip_verify">
              <SwitchGroupWide label="Skip TLS verification (insecure)"/>
            </ContextField>
          )}

          <ContextField name="pass">
            <PasswordFieldWide
              label="Password"
              help="Network password, not commonly used."
            />
          </ContextField>

          <ContextField name="nick">
            <TextFieldWide
              label="Nick"
              placeholder="nick"
              required={true}
            />
          </ContextField>

          <ContextField name="use_bouncer">
            <SwitchGroupWide label="Bouncer (BNC)"/>
          </ContextField>
          {values.use_bouncer && (
            <ContextField name="bouncer_addr">
              <TextFieldWide
                label="Bouncer address"
                help="Address: Eg bouncer.server.net:6697"
              />
            </ContextField>
          )}

          <ContextField name="bot_mode">
            <SwitchGroupWide label="IRCv3 Bot Mode"/>
          </ContextField>

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
              <ContextField name="use_proxy">
                <SwitchButton />
              </ContextField>
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

            <ContextField name="auth.account">
              <TextFieldWide
                label="Account"
                placeholder="Auth Account"
                help="NickServ / SASL account. For grouped nicks try the main."
              />
            </ContextField>

            <ContextField name="auth.password">
              <PasswordFieldWide
                label="Password"
                help="NickServ / SASL password."
              />
            </ContextField>
          </div>

          <ContextField name="invite_command">
            <PasswordFieldWide label="Invite command"/>
          </ContextField>

          <div className="border-t border-gray-200 dark:border-gray-700 py-5">
            <div className="px-4 space-y-1 mb-8">
              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">Channels</DialogTitle>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Channels are added when you setup IRC indexers. Do not edit unless you know what you are doing.
              </p>
            </div>

            <ChannelsFieldArray />
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
        <ContextField name={name}>
          <SelectFieldInner<T>
            name={name}
            options={options}
            placeholder={placeholder}
          />
        </ContextField>
      </div>
    </div>
  );
}

function SelectFieldInner<T>({ name, options, placeholder }: { name: string; options: OptionBasicTyped<T>[]; placeholder?: string }) {
  const field = useFieldContext<T>();

  return (
    <Select
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
      value={field.state.value && options.find(o => o.value == field.state.value)}
      onChange={(newValue: unknown) => {
        if (newValue) {
          field.handleChange((newValue as { value: T }).value);
        }
        else {
          field.handleChange(0 as T);
        }
      }}
      options={options}
    />
  );
}
