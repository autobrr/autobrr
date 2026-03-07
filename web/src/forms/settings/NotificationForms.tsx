/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Dialog, DialogPanel, DialogTitle, Transition, TransitionChild } from "@headlessui/react";
import { Fragment, useMemo } from "react";
import { XMarkIcon } from "@heroicons/react/24/solid";
import Select from "react-select";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";

import { useAppForm, ContextField, useFormContext, useFieldContext, useStore } from "@app/lib/form";
import { APIClient } from "@api/APIClient";
import { NotificationKeys } from "@api/query_keys";
import { PushoverSoundsQueryOptions } from "@api/queries";
import { EventOptions, ExternalFilterWebhookMethodOptions, NotificationTypeOptions, PushoverSoundOptions, SelectOption } from "@domain/constants";
import { DEBUG } from "@components/debug";
import { SlideOver } from "@components/panels";
import { ExternalLink } from "@components/ExternalLink";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import * as common from "@components/inputs/tanstack/common";
import { NumberFieldWide, PasswordFieldWide, SelectFieldWide, SwitchGroupWide, TextFieldWide } from "@components/inputs/tanstack";
import { Checkbox } from "@components/Checkbox";
import { EmptySimple } from "@components/emptystates";

import { componentMapType } from "./DownloadClientForms";
import { AddFormProps, UpdateFormProps } from "@forms/_shared";

function FormFieldsDiscord() {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          Settings
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {"Create a "}
          <ExternalLink
            href="https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks"
            className="font-medium text-blue-500 underline underline-offset-1 hover:text-blue-400"
          >
            webhook integration
          </ExternalLink>
          {" in your server."}
        </p>
      </div>

      <ContextField name="webhook">
        <PasswordFieldWide
          label="Webhook URL"
          help="Discord channel webhook url"
          placeholder="https://discordapp.com/api/webhooks/xx/xx"
        />
      </ContextField>
    </div>
  );
}

function FormFieldsNotifiarr() {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          Settings
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Enable the autobrr integration and optionally create a new API Key.
        </p>
      </div>

      <ContextField name="api_key">
        <PasswordFieldWide
          label="API Key"
          help="Notifiarr API Key"
        />
      </ContextField>
    </div>
  );
}

function FormFieldsLunaSea() {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          Settings
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          LunaSea offers notifications across all devices linked to your account (User-Based) or to a single device without an account, using a unique webhook per device (Device-Based).
        </p>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {"Read the "}
          <ExternalLink
            href="https://docs.lunasea.app/lunasea/notifications"
            className="font-medium text-blue-500 underline underline-offset-1 hover:text-blue-400"
          >
            LunaSea docs
          </ExternalLink>
          {"."}
        </p>
      </div>

      <ContextField name="webhook">
        <PasswordFieldWide
          label="Webhook URL"
          help="LunaSea Webhook URL"
          placeholder="https://notify.lunasea.app/v1/custom/user/TOKEN"
        />
      </ContextField>
    </div>
  );
}

function FormFieldsTelegram() {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          Settings
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {"Read how to "}
          <ExternalLink
            href="https://core.telegram.org/bots#3-how-do-i-create-a-bot"
            className="font-medium text-blue-500 underline underline-offset-1 hover:text-blue-400"
          >
            create a bot
          </ExternalLink>
          {"."}
        </p>
      </div>

      <ContextField name="token">
        <PasswordFieldWide
          label="Bot token"
          help="Bot token"
        />
      </ContextField>
      <ContextField name="channel">
        <PasswordFieldWide
          label="Chat ID"
          help="Chat ID"
        />
      </ContextField>
      <ContextField name="topic">
        <PasswordFieldWide
          label="Message Thread ID"
          help="Message Thread (topic) of a Supergroup"
        />
      </ContextField>
      <ContextField name="host">
        <TextFieldWide
          label="Telegram Api Proxy"
          help="Reverse proxy domain for api.telegram.org, only needs to be specified if the network you are using has blocked the Telegram API."
          placeholder="http(s)://ip:port"
        />
      </ContextField>
      <ContextField name="username">
        <TextFieldWide
          label="Sender"
          help="Custom sender name to show at the top of a notification"
          placeholder="autobrr"
        />
      </ContextField>
    </div>
  );
}

interface SoundOption {
  label: string;
  value: string;
}

function FormFieldsPushover() {
  return (
    <div>

    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          Settings
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {"Register a new "}
          <ExternalLink
            href="https://support.pushover.net/i175-how-do-i-get-an-api-or-application-token"
            className="font-medium text-blue-500 underline underline-offset-1 hover:text-blue-400"
          >
            application
          </ExternalLink>
          {" and add its API Token here."}
        </p>
      </div>

      <ContextField name="api_key">
        <PasswordFieldWide
          label="API Token"
          help="API Token"
        />
      </ContextField>
      <ContextField name="token">
        <PasswordFieldWide
          label="User Key"
          help="User Key"
        />
      </ContextField>
      <ContextField name="priority">
        <NumberFieldWide
          label="Priority"
          help="-2, -1, 0 (default), 1, or 2"
          required={true}
        />
      </ContextField>
    </div>
      <div className="pb-2">
        <div className="flex justify-between items-center p-4">

        <div className="">
          <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
            Events sounds
          </DialogTitle>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Set custom sounds per event
          </p>
        </div>
          {/*<button*/}
          {/*  // type="submit"*/}
          {/*  className="inline-flex justify-center py-2 px-4 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"*/}
          {/*>*/}
          {/*  Fetch*/}
          {/*</button>*/}
        </div>

        <EventSounds />
      </div>
    </div>
  );
}

function FormFieldsGotify() {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          Settings
        </DialogTitle>
      </div>

      <ContextField name="host">
        <TextFieldWide
          label="Gotify URL"
          help="Gotify URL (without /message)"
          placeholder="https://some.gotify.server.com"
          required={true}
        />
      </ContextField>
      <ContextField name="token">
        <PasswordFieldWide
          label="Application Token"
          help="Application Token"
          required={true}
        />
      </ContextField>
    </div>
  );
}

function FormFieldsNtfy() {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          Settings
        </DialogTitle>
      </div>

      <ContextField name="host">
        <TextFieldWide
          label="NTFY URL"
          help="NTFY URL"
          placeholder="https://ntfy.sh/mytopic"
          required={true}
        />
      </ContextField>

      <ContextField name="username">
        <TextFieldWide
          label="Username"
          help="Username"
        />
      </ContextField>

      <ContextField name="password">
        <PasswordFieldWide
          label="Password"
          help="Password"
        />
      </ContextField>

      <ContextField name="token">
        <PasswordFieldWide
          label="Access token"
          help="Access token. Use this or Usernmae+password"
        />
      </ContextField>

      <ContextField name="priority">
        <NumberFieldWide
          label="Priority"
          help="Max 5, 4, 3 (default), 2, 1 Min"
        />
      </ContextField>
    </div>
  );
}

function FormFieldsShoutrrr() {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          Settings
        </DialogTitle>
      </div>

      <ContextField name="host">
        <TextFieldWide
          label="URL"
          help="URL"
          tooltip={
            <div><p>See full documentation </p>
              <ExternalLink
                href="https://containrrr.dev/shoutrrr/services/overview/"
                className="font-medium text-blue-500 underline underline-offset-1 hover:text-blue-400"
              >
                Services
              </ExternalLink>
            </div>
          }
          placeholder="smtp://username:password@host:port/?from=fromAddress&to=recipient1"
          required={true}
        />
      </ContextField>
    </div>
  );
}

function FormFieldsGenericWebhook() {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          Settings
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Send a generic autobrr JSON payload to a user-defined webhook URL.
        </p>
      </div>

      <ContextField name="webhook">
        <PasswordFieldWide
          label="Webhook URL"
          help="Webhook URL"
          placeholder="https://example.com/webhook"
          required={true}
        />
      </ContextField>
      <ContextField name="method">
        <SelectFieldWide
          label="HTTP Method"
          optionDefaultText="POST (default)"
          options={ExternalFilterWebhookMethodOptions}
          tooltip={<p>HTTP method for the webhook request. Defaults to POST.</p>}
        />
      </ContextField>
      <ContextField name="headers">
        <TextFieldWide
          label="Custom Headers"
          help="Comma-separated KEY=value pairs (e.g., Authorization=Bearer token,X-Custom=value)"
          placeholder="Authorization=Bearer token,X-Custom-Header=value"
        />
      </ContextField>
    </div>
  );
}

const componentMap: componentMapType = {
  DISCORD: <FormFieldsDiscord />,
  NOTIFIARR: <FormFieldsNotifiarr />,
  TELEGRAM: <FormFieldsTelegram />,
  PUSHOVER: <FormFieldsPushover />,
  GOTIFY: <FormFieldsGotify />,
  NTFY: <FormFieldsNtfy />,
  SHOUTRRR: <FormFieldsShoutrrr />,
  LUNASEA: <FormFieldsLunaSea />,
  WEBHOOK: <FormFieldsGenericWebhook />
};

export function NotificationAddForm({ isOpen, toggle }: AddFormProps) {
  const queryClient = useQueryClient();

  const createMutation = useMutation({
    mutationFn: (notification: ServiceNotification) => APIClient.notifications.create(notification),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NotificationKeys.lists() });

      toast.custom((t) => <Toast type="success" body="Notification added!" t={t} />);
      toggle();
    },
    onError: () => {
      toast.custom((t) => <Toast type="error" body="Notification could not be added" t={t} />);
    }
  });

  const onSubmit = (formData: unknown) => createMutation.mutate(formData as ServiceNotification);

  const testMutation = useMutation({
    mutationFn: (n: ServiceNotification) => APIClient.notifications.test(n),
    onError: (err) => {
      console.error(err);
    }
  });

  const testNotification = (data: unknown) => testMutation.mutate(data as ServiceNotification);

  const form = useAppForm({
    defaultValues: {
      enabled: true,
      type: "",
      name: "",
      webhook: "",
      events: [] as string[],
      username: "",
      sound: "",
      event_sounds: {} as Record<string, string>
    },
    onSubmit: async ({ value }) => onSubmit(value),
    validators: {
      onSubmit: ({ value }) => {
        if (!value.name) {
          return "Name is required";
        }
        return undefined;
      }
    }
  });

  const typeValue = useStore(form.store, (s: any) => s.values.type);

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
              <div className="w-screen max-w-2xl">
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
                              Add Notifications
                            </DialogTitle>
                            <p className="text-sm text-gray-500 dark:text-gray-200">
                              Trigger notifications on different events.
                            </p>
                          </div>
                          <div className="h-7 flex items-center">
                            <button
                              type="button"
                              className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500"
                              onClick={toggle}
                            >
                              <span className="sr-only">Close panel</span>
                              <XMarkIcon className="h-6 w-6" aria-hidden="true" />
                            </button>
                          </div>
                        </div>
                      </div>

                      <div className="flex flex-col space-y-4 px-1 pt-6 sm:py-0 sm:space-y-0">
                        <ContextField name="name">
                          <TextFieldWide
                            label="Name"
                            required={true}
                          />
                        </ContextField>

                        <div className="flex items-center justify-between space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
                          <div>
                            <label
                              htmlFor="type"
                              className="block text-sm font-medium text-gray-900 dark:text-white"
                            >
                              Type
                            </label>
                          </div>
                          <div className="sm:col-span-2">
                            <form.AppField name="type">
                              {(field) => (
                                <Select
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
                                  placeholder="Choose a type"
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
                                  value={field.state.value ? NotificationTypeOptions.find(o => o.value === field.state.value) ?? null : null}
                                  onChange={(option: unknown) => {
                                    form.reset();

                                    const opt = option as SelectOption;
                                    field.handleChange(opt?.value ?? "");
                                  }}
                                  options={NotificationTypeOptions}
                                />
                              )}
                            </form.AppField>
                          </div>
                        </div>

                        <ContextField name="enabled">
                          <SwitchGroupWide label="Enabled" />
                        </ContextField>

                        <div className="border-t border-gray-200 dark:border-gray-700 py-4">
                          <div className="px-4">
                            <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
                              Global Events
                            </DialogTitle>
                            <p className="text-sm text-gray-500 dark:text-gray-400">
                              Select default events that trigger globally. These can be overridden on a per-filter basis. Leave all unchecked to use this service only for filter-specific notifications.
                            </p>
                          </div>

                          <div className="p-4 sm:grid sm:gap-4">
                            <EventCheckBoxes />
                          </div>
                        </div>
                      </div>
                      {componentMap[typeValue]}
                    </div>

                    <div className="shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-4 sm:px-6">
                      <div className="space-x-3 flex justify-end">
                        <button
                          type="button"
                          className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                          onClick={() => testNotification(form.state.values)}
                        >
                          Test
                        </button>
                        <button
                          type="button"
                          className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                          onClick={toggle}
                        >
                          Cancel
                        </button>
                        <button
                          type="submit"
                          className="inline-flex justify-center py-2 px-4 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                        >
                          Save
                        </button>
                      </div>
                    </div>

                    <DEBUG values={form.state.values} />
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

const EventCheckBox = ({ event }: { event: typeof EventOptions[number]; }) => {
  const form = useFormContext();
  const events = useStore(form.store, (s: any) => (s.values as any).events as string[]);

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-sm">
          <span className="font-medium text-gray-900 dark:text-gray-100">{event.label}</span>
          {event.description && <p className="text-gray-500">{event.description}</p>}
        </span>
        <Checkbox
          value={events.includes(event.value)}
          setValue={(checked) =>
            (form as any).setFieldValue("events",
              checked
                ? [...events, event.value]
                : events.filter((e: string) => e !== event.value)
            )
          }
        />
      </div>
    </div>
  );
};

const EventCheckBoxes = () => {
  return (
    <fieldset className="space-y-5">
      <legend className="sr-only">Notifications</legend>
      {EventOptions.map((event, idx) => (
        <EventCheckBox
          key={idx}
          event={event}
        />
      ))}
    </fieldset>
  );
};

const EventSoundSelector = ({event, soundOptions}: {
  event: typeof EventOptions[number];
  soundOptions: SoundOption[];
}) => {
  const form = useFormContext();
  const eventSounds = useStore(form.store, (s: any) => (s.values as any).event_sounds as Record<string, string> || {});
  const currentSound = eventSounds[event.value] || "";

  return (
    <div className="space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <span className="text-sm">
        <span className="font-medium text-gray-900 dark:text-gray-100">{event.label}</span>
      </span>

      <div className="sm:col-span-2">
        <Select
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
          placeholder="Default (user's default tone)"
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
          value={soundOptions.find(o => o.value === currentSound) || null}
          onChange={(option: unknown) => {
            const opt = option as SoundOption | null;
            const newEventSounds = {...eventSounds};
            if (opt?.value) {
              newEventSounds[event.value] = opt.value;
            } else {
              delete newEventSounds[event.value];
            }
            (form as any).setFieldValue("event_sounds", newEventSounds);
          }}
          options={soundOptions}
        />
      </div>
    </div>
  );
};

const EventSounds = () => {
  const form = useFormContext();
  const apiKey = useStore(form.store, (s: any) => (s.values as any).api_key as string || "");

  const canFetchCustomSounds = Boolean(apiKey && apiKey !== "<redacted>");

  const soundsQuery = useQuery({
    ...PushoverSoundsQueryOptions(apiKey),
    enabled: canFetchCustomSounds
  });

  const soundOptions: SoundOption[] = useMemo(() => {
    const builtInKeys = new Set(PushoverSoundOptions.map(s => s.value));

    const customSounds: SoundOption[] = soundsQuery.data
      ? Object.entries(soundsQuery.data)
          .filter(([key]) => !builtInKeys.has(key))
          .sort(([, a], [, b]) => a.localeCompare(b))
          .map(([key, value]) => ({ label: `${value} (custom)`, value: key }))
      : [];

    return [
      { label: "Default (user's default tone)", value: "" },
      ...PushoverSoundOptions,
      ...customSounds
    ];
  }, [soundsQuery.data]);

  return (
    <fieldset className="">
      <legend className="sr-only">Notifications</legend>
      {EventOptions.map((event, idx) => (
        <EventSoundSelector
          key={idx}
          event={event}
          soundOptions={soundOptions}
        />
      ))}
    </fieldset>
  );
};

interface InitialValues {
  id: number;
  enabled: boolean;
  type: NotificationType;
  name: string;
  webhook?: string;
  token?: string;
  api_key?: string;
  priority?: number;
  channel?: string;
  topic?: string;
  sound?: string;
  event_sounds?: Record<string, string>;
  host?: string;
  events: NotificationEvent[];
  username?: string;
  password?: string;
  used_by_filters?: NotificationFilter[];
}

export function NotificationUpdateForm({ isOpen, toggle, data: notification }: UpdateFormProps<ServiceNotification>) {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (notification: ServiceNotification) => APIClient.notifications.update(notification),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NotificationKeys.lists() });

      toast.custom((t) => <Toast type="success" body={`${notification.name} was updated successfully`} t={t} />);
      toggle();
    }
  });

  const onSubmit = (formData: unknown) => mutation.mutate(formData as ServiceNotification);

  const deleteMutation = useMutation({
    mutationFn: (notificationID: number) => APIClient.notifications.delete(notificationID),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NotificationKeys.lists() });

      toast.custom((t) => <Toast type="success" body={`${notification.name} was deleted.`} t={t} />);
    }
  });

  const deleteAction = () => deleteMutation.mutate(notification.id);

  const testMutation = useMutation({
    mutationFn: (n: ServiceNotification) => APIClient.notifications.test(n),
    onError: (err) => {
      console.error(err);
    }
  });

  const testNotification = (data: unknown) => testMutation.mutate(data as ServiceNotification);

  const initialValues: InitialValues = {
    id: notification.id,
    enabled: notification.enabled,
    type: notification.type,
    name: notification.name,
    webhook: notification.webhook,
    token: notification.token,
    api_key: notification.api_key,
    priority: notification.priority,
    channel: notification.channel,
    topic: notification.topic,
    sound: notification.sound,
    event_sounds: notification.event_sounds || {},
    host: notification.host,
    events: notification.events || [],
    username: notification.username,
    password: notification.password,
    used_by_filters: notification.used_by_filters || [],
  };

  return (
    <SlideOver<InitialValues>
      type="UPDATE"
      title="Notification"
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      deleteAction={deleteAction}
      initialValues={initialValues}
      testFn={testNotification}
    >
      {(values) => (
        <div>
          <ContextField name="name">
            <TextFieldWide label="Name" required={true} />
          </ContextField>

          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            <div className="py-4 flex items-center justify-between space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:py-4">
              <div>
                <label
                  htmlFor="type"
                  className="block text-sm font-medium text-gray-900 dark:text-white"
                >
                  Type
                </label>
              </div>
              <div className="sm:col-span-2">
                <ContextField name="type">
                  <TypeSelectInSlideOver />
                </ContextField>
              </div>
            </div>
            <ContextField name="enabled">
              <SwitchGroupWide label="Enabled" />
            </ContextField>
            <div className="pb-2">
              <div className="p-4">
                <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
                  Global Events
                </DialogTitle>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Select default events that trigger globally. These can be overridden on a per-filter basis. Leave all unchecked to use this service only for filter-specific notifications.
                </p>
              </div>

              <div className="p-4 sm:grid sm:gap-4">
                <EventCheckBoxes />
              </div>
            </div>
          </div>

          <div className="pb-2">
            <div className="p-4">
              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
                Per filter Events
              </DialogTitle>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                This notification is used in these filters.
              </p>
            </div>

            <div className="p-4 sm:grid sm:gap-4">
              {values.used_by_filters && values.used_by_filters?.length > 0
                ? values.used_by_filters?.map(f => (
                  <Link key={f.filter_id} to="/filters/$filterId/notifications" params={{ filterId: f.filter_id }}>
                    <div key={f.filter_id} className="flex justify-between px-2 py-2 bg-gray-50 dark:bg-gray-750 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md">
                      <span className="font-medium text-gray-500 dark:text-gray-300">{f.filter_name}</span>
                      <div className="flex gap-2">
                        {f.events.length > 0
                          ? f.events.map((e) => <span className="inline-flex items-center rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-600 dark:bg-gray-400/10 dark:text-gray-400">{FilterEventOptions[e]}</span>)
                          : <span className="inline-flex items-center rounded-md bg-yellow-100 px-2 py-1 text-xs font-medium text-yellow-600 dark:bg-yellow-400/10 dark:text-yellow-400">Muted</span>}
                      </div>
                    </div>
                  </Link>
                ))
                :
                <EmptySimple
                  title="Not used in any filters"
                  subtitle=""
                  border={true}
                />

              }
            </div>
          </div>

          {componentMap[values.type]}

        </div>
      )}
    </SlideOver>
  );
}

/**
 * TypeSelectInSlideOver: extracted select for the "type" field in the update form.
 * Uses useFieldContext to get the field, and useFormContext to access form.reset().
 */
function TypeSelectInSlideOver() {
  const field = useFieldContext<string>();
  const form = useFormContext();

  return (
    <Select
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
      placeholder="Choose a type"
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
      value={field.state.value ? NotificationTypeOptions.find(o => o.value == field.state.value) ?? null : null}
      onChange={(option: unknown) => {
        form.reset();
        const opt = option as SelectOption;
        field.handleChange(opt?.value ?? "");
      }}
      options={NotificationTypeOptions}
    />
  );
}

const FilterEventOptions: Record<NotificationFilterEvent, string> = {
  "PUSH_APPROVED": "Push Approved",
  "PUSH_REJECTED": "Push Rejected",
  "PUSH_ERROR": "Push Error",
  "RELEASE_NEW": "New Release",
}
