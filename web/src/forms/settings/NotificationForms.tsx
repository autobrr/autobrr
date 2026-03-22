/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Dialog, DialogPanel, DialogTitle, Transition, TransitionChild } from "@headlessui/react";
import { Fragment, useMemo } from "react";
import type { FieldProps } from "formik";
import { Field, Form, Formik, FormikErrors, FormikValues, useFormikContext } from "formik";
import { XMarkIcon } from "@heroicons/react/24/solid";
import Select from "react-select";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { APIClient } from "@api/APIClient";
import { NotificationKeys } from "@api/query_keys";
import { PushoverSoundsQueryOptions } from "@api/queries";
import { ExternalFilterWebhookMethodOptions, getEventOptions, getNotificationTypeOptions, PushoverSoundOptions, SelectOption } from "@domain/constants";
import { DEBUG } from "@components/debug";
import { SlideOver } from "@components/panels";
import { ExternalLink } from "@components/ExternalLink";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import * as common from "@components/inputs/common";
import { NumberFieldWide, PasswordFieldWide, SelectFieldWide, SwitchGroupWide, TextFieldWide } from "@components/inputs";
import { Checkbox } from "@components/Checkbox";
import { EmptySimple } from "@components/emptystates";

import { componentMapType } from "./DownloadClientForms";
import { AddFormProps, UpdateFormProps } from "@forms/_shared";

function FormFieldsDiscord() {
  const { t } = useTranslation("settings");
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.notification.settings")}
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.notification.settingsDescDiscordPrefix")}
          <ExternalLink
            href="https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks"
            className="font-medium text-blue-500 underline underline-offset-1 hover:text-blue-400"
          >
            {t("forms.notification.discordWebhookIntegration")}
          </ExternalLink>
          {t("forms.notification.settingsDescDiscordSuffix")}
        </p>
      </div>

      <PasswordFieldWide
        name="webhook"
        label={t("forms.notification.webhookUrl")}
        help={t("forms.notification.discordWebhookHelp")}
        placeholder={t("forms.notification.discordWebhookPlaceholder")}
      />
    </div>
  );
}

function FormFieldsNotifiarr() {
  const { t } = useTranslation("settings");
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.notification.settings")}
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.notification.settingsDescNotifiarr")}
        </p>
      </div>

      <PasswordFieldWide
        name="api_key"
        label={t("forms.notification.notifiarrApiKey")}
        help={t("forms.notification.notifiarrApiKeyHelp")}
      />
    </div>
  );
}

function FormFieldsLunaSea() {
  const { t } = useTranslation("settings");
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.notification.settings")}
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.notification.settingsDescLunasea1")}
        </p>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.notification.settingsDescLunasea2Prefix")}
          <ExternalLink
            href="https://docs.lunasea.app/lunasea/notifications"
            className="font-medium text-blue-500 underline underline-offset-1 hover:text-blue-400"
          >
            {t("forms.notification.lunaseaDocs")}
          </ExternalLink>
          {t("forms.notification.settingsDescLunasea2Suffix")}
        </p>
      </div>

      <PasswordFieldWide
        name="webhook"
        label={t("forms.notification.webhookUrl")}
        help={t("forms.notification.lunaseaWebhookHelp")}
        placeholder={t("forms.notification.lunaseaWebhookPlaceholder")}
      />
    </div>
  );
}

function FormFieldsTelegram() {
  const { t } = useTranslation("settings");
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.notification.settings")}
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.notification.settingsDescTelegramPrefix")}
          <ExternalLink
            href="https://core.telegram.org/bots#3-how-do-i-create-a-bot"
            className="font-medium text-blue-500 underline underline-offset-1 hover:text-blue-400"
          >
            {t("forms.notification.createBot")}
          </ExternalLink>
          {t("forms.notification.settingsDescTelegramSuffix")}
        </p>
      </div>

      <PasswordFieldWide
        name="token"
        label={t("forms.notification.botToken")}
        help={t("forms.notification.botTokenHelp")}
      />
      <PasswordFieldWide
        name="channel"
        label={t("forms.notification.chatId")}
        help={t("forms.notification.chatIdHelp")}
      />
      <PasswordFieldWide
        name="topic"
        label={t("forms.notification.messageThreadId")}
        help={t("forms.notification.messageThreadIdHelp")}
      />
      <TextFieldWide
        name="host"
        label={t("forms.notification.telegramProxy")}
        help={t("forms.notification.telegramProxyHelp")}
        placeholder={t("forms.notification.telegramProxyPlaceholder")}
      />
      <TextFieldWide
        name="username"
        label={t("forms.notification.sender")}
        help={t("forms.notification.senderHelp")}
        placeholder={t("forms.notification.senderPlaceholder")}
      />
    </div>
  );
}

interface SoundOption {
  label: string;
  value: string;
}

type NotificationEventOption = ReturnType<typeof getEventOptions>[number];

function FormFieldsPushover() {
  const { t } = useTranslation("settings");
  return (
    <div>

    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.notification.settings")}
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.notification.settingsDescPushoverPrefix")}
          <ExternalLink
            href="https://support.pushover.net/i175-how-do-i-get-an-api-or-application-token"
            className="font-medium text-blue-500 underline underline-offset-1 hover:text-blue-400"
          >
            {t("forms.notification.pushoverApplication")}
          </ExternalLink>
          {t("forms.notification.settingsDescPushoverSuffix")}
        </p>
      </div>

      <PasswordFieldWide
        name="api_key"
        label={t("forms.notification.apiToken")}
        help={t("forms.notification.apiTokenHelp")}
      />
      <PasswordFieldWide
        name="token"
        label={t("forms.notification.userKey")}
        help={t("forms.notification.userKeyHelp")}
      />
      <NumberFieldWide
        name="priority"
        label={t("forms.notification.priority")}
        help={t("forms.notification.pushoverPriorityHelp")}
        required={true}
      />
    </div>
      <div className="pb-2">
        <div className="flex justify-between items-center p-4">

        <div className="">
          <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
            {t("forms.notification.eventSounds")}
          </DialogTitle>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {t("forms.notification.eventSoundsDesc")}
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
  const { t } = useTranslation("settings");
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.notification.settings")}
        </DialogTitle>
      </div>

      <TextFieldWide
        name="host"
        label={t("forms.notification.gotifyUrl")}
        help={t("forms.notification.gotifyUrlHelp")}
        placeholder={t("forms.notification.gotifyUrlPlaceholder")}
        required={true}
      />
      <PasswordFieldWide
        name="token"
        label={t("forms.notification.applicationToken")}
        help={t("forms.notification.applicationTokenHelp")}
        required={true}
      />
    </div>
  );
}

function FormFieldsNtfy() {
  const { t } = useTranslation("settings");
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.notification.settings")}
        </DialogTitle>
      </div>

      <TextFieldWide
        name="host"
        label={t("forms.notification.ntfyUrl")}
        help={t("forms.notification.ntfyUrlHelp")}
        placeholder={t("forms.notification.ntfyUrlPlaceholder")}
        required={true}
      />

      <TextFieldWide
        name="username"
        label={t("forms.notification.username")}
        help={t("forms.notification.usernameHelp")}
      />

      <PasswordFieldWide
        name="password"
        label={t("forms.notification.password")}
        help={t("forms.notification.passwordHelp")}
      />

      <PasswordFieldWide
        name="token"
        label={t("forms.notification.accessToken")}
        help={t("forms.notification.accessTokenHelp")}
      />

      <NumberFieldWide
        name="priority"
        label={t("forms.notification.priority")}
        help={t("forms.notification.ntfyPriorityHelp")}
      />
    </div>
  );
}

function FormFieldsShoutrrr() {
  const { t } = useTranslation("settings");
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.notification.settings")}
        </DialogTitle>
      </div>

      <TextFieldWide
        name="host"
        label={t("forms.notification.url")}
        help={t("forms.notification.urlHelp")}
        tooltip={
          <div><p>{t("forms.notification.shoutrrrDocsPrefix")}</p>
            <ExternalLink
              href="https://containrrr.dev/shoutrrr/services/overview/"
              className="font-medium text-blue-500 underline underline-offset-1 hover:text-blue-400"
            >
              {t("forms.notification.shoutrrrServices")}
            </ExternalLink>
          </div>
        }
        placeholder={t("forms.notification.shoutrrrUrlPlaceholder")}
        required={true}
      />
    </div>
  );
}

function FormFieldsGenericWebhook() {
  const { t } = useTranslation("settings");
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.notification.settings")}
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.notification.settingsDescWebhook")}
        </p>
      </div>

      <PasswordFieldWide
        name="webhook"
        label={t("forms.notification.webhookUrl")}
        help={t("forms.notification.webhookUrl")}
        placeholder={t("forms.notification.webhookPlaceholder")}
        required={true}
      />
      <SelectFieldWide
        name="method"
        label={t("forms.notification.httpMethod")}
        optionDefaultText={t("forms.notification.httpMethodDefault")}
        options={ExternalFilterWebhookMethodOptions}
        tooltip={<p>{t("forms.notification.httpMethodTooltip")}</p>}
      />
      <TextFieldWide
        name="headers"
        label={t("forms.notification.customHeaders")}
        help={t("forms.notification.customHeadersHelp")}
        placeholder={t("forms.notification.customHeadersPlaceholder")}
      />
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

interface NotificationAddFormValues {
  name: string;
  enabled: boolean;
}

export function NotificationAddForm({ isOpen, toggle }: AddFormProps) {
  const { t } = useTranslation(["options", "settings"]);
  const notificationTypeOptions = getNotificationTypeOptions(t);
  const queryClient = useQueryClient();

  const createMutation = useMutation({
    mutationFn: (notification: ServiceNotification) => APIClient.notifications.create(notification),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NotificationKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("settings:forms.notification.added")} t={toastInstance} />);
      toggle();
    },
    onError: () => {
      toast.custom((toastInstance) => <Toast type="error" body={t("settings:forms.notification.addFailed")} t={toastInstance} />);
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

  const validate = (values: NotificationAddFormValues) => {
    const errors = {} as FormikErrors<FormikValues>;
    if (!values.name)
      errors.name = t("settings:forms.notification.required");

    return errors;
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
                <Formik
                  enableReinitialize={true}
                  initialValues={{
                    enabled: true,
                    type: "",
                    name: "",
                    webhook: "",
                    events: [],
                    username: "",
                    sound: "",
                    event_sounds: {}
                  }}
                  onSubmit={onSubmit}
                  validate={validate}
                >
                  {({ values }) => (
                    <Form className="h-full flex flex-col bg-white dark:bg-gray-800 shadow-xl overflow-y-auto">
                      <div className="flex-1">
                        <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
                          <div className="flex items-start justify-between space-x-3">
                            <div className="space-y-1">
                              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
                                {t("settings:forms.notification.addTitle")}
                              </DialogTitle>
                              <p className="text-sm text-gray-500 dark:text-gray-200">
                                {t("settings:forms.notification.addDescription")}
                              </p>
                            </div>
                            <div className="h-7 flex items-center">
                              <button
                                type="button"
                                className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500"
                                onClick={toggle}
                              >
                                <span className="sr-only">{t("settings:forms.notification.closePanel")}</span>
                                <XMarkIcon className="h-6 w-6" aria-hidden="true" />
                              </button>
                            </div>
                          </div>
                        </div>

                        <div className="flex flex-col space-y-4 px-1 pt-6 sm:py-0 sm:space-y-0">
                          <TextFieldWide
                            name="name"
                            label={t("settings:forms.notification.name")}
                            required={true}
                          />

                          <div className="flex items-center justify-between space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
                            <div>
                              <label
                                htmlFor="type"
                                className="block text-sm font-medium text-gray-900 dark:text-white"
                              >
                                {t("settings:forms.notification.type")}
                              </label>
                            </div>
                            <div className="sm:col-span-2">
                              <Field name="type" type="select">
                                {({
                                  field,
                                  form: { setFieldValue, resetForm }
                                }: FieldProps) => (
                                  <Select
                                    {...field}
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
                                    placeholder={t("settings:forms.notification.chooseType")}
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
                                    value={field?.value && field.value.value}
                                    onChange={(option: unknown) => {
                                      resetForm();

                                      const opt = option as SelectOption;
                                      // setFieldValue("name", option?.label ?? "")
                                      setFieldValue(
                                        field.name,
                                        opt.value ?? ""
                                      );
                                    }}
                                    options={notificationTypeOptions}
                                  />
                                )}
                              </Field>
                            </div>
                          </div>

                          <SwitchGroupWide name="enabled" label={t("settings:forms.notification.enabled")} />

                          <div className="border-t border-gray-200 dark:border-gray-700 py-4">
                            <div className="px-4">
                              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
                                {t("settings:forms.notification.globalEvents")}
                              </DialogTitle>
                              <p className="text-sm text-gray-500 dark:text-gray-400">
                                {t("settings:forms.notification.globalEventsDesc")}
                              </p>
                            </div>

                            <div className="p-4 sm:grid sm:gap-4">
                              <EventCheckBoxes />
                            </div>
                          </div>
                        </div>
                        {componentMap[values.type]}
                      </div>

                      <div className="shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-4 sm:px-6">
                        <div className="space-x-3 flex justify-end">
                          <button
                            type="button"
                            className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                            onClick={() => testNotification(values)}
                          >
                            {t("settings:forms.notification.test")}
                          </button>
                          <button
                            type="button"
                            className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                            onClick={toggle}
                          >
                            {t("settings:forms.notification.cancel")}
                          </button>
                          <button
                            type="submit"
                            className="inline-flex justify-center py-2 px-4 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                          >
                            {t("settings:forms.notification.save")}
                          </button>
                        </div>
                      </div>

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

const EventCheckBox = ({ event }: { event: NotificationEventOption; }) => (
  <Field name="events">
    {({ field, form }: FieldProps<string[]>) => (
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-sm">
            <span className="font-medium text-gray-900 dark:text-gray-100">{event.label}</span>
            {event.description && <p className="text-gray-500">{event.description}</p>}
          </span>
          <Checkbox
            value={field.value.includes(event.value)}
            setValue={(checked) =>
              form.setFieldValue('events',
                checked
                  ? [...field.value, event.value]
                  : field.value.filter(e => e !== event.value)
              )
            }
          />
        </div>
      </div>
    )}
  </Field>
);

const EventCheckBoxes = () => {
  const { t } = useTranslation(["options", "settings"]);
  const eventOptions = getEventOptions(t);

  return (
    <fieldset className="space-y-5">
      <legend className="sr-only">{t("settings:forms.notification.notificationsLegend")}</legend>
      {eventOptions.map((event, idx) => (
        <EventCheckBox
          key={idx}
          event={event}
        />
      ))}
    </fieldset>
  );
};

const EventSoundSelector = ({event, soundOptions}: {
  event: NotificationEventOption;
  soundOptions: SoundOption[];
}) => {
  const {values, setFieldValue} = useFormikContext<ServiceNotification>();
  const eventSounds = values.event_sounds || {};
  const currentSound = eventSounds[event.value] || "";

  return (
    <div className="space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <span className="text-sm">
        <span className="font-medium text-gray-900 dark:text-gray-100">{event.label}</span>
      </span>

      <div className="sm:col-span-2">
        <Field name={`event_sounds.${event.value}`} type="select">
          {({field: soundField}: FieldProps) => (
            <Select
              {...soundField}
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
              placeholder={t("settings:forms.notification.defaultTone")}
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
                setFieldValue("event_sounds", newEventSounds);
              }}
              options={soundOptions}
            />
          )}
        </Field>
      </div>
    </div>
  );
};

const EventSounds = () => {
  const { t } = useTranslation(["options", "settings"]);
  const eventOptions = getEventOptions(t);
  const { values } = useFormikContext<ServiceNotification>();
  const apiKey = values.api_key || "";

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
      { label: t("settings:forms.notification.defaultTone"), value: "" },
      ...PushoverSoundOptions,
      ...customSounds
    ];
  }, [soundsQuery.data]);

  return (
    <fieldset className="">
      <legend className="sr-only">{t("settings:forms.notification.notificationsLegend")}</legend>
      {eventOptions.map((event, idx) => (
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
  const { t } = useTranslation(["options", "settings"]);
  const notificationTypeOptions = getNotificationTypeOptions(t);
  const filterEventOptions: Record<NotificationFilterEvent, string> = {
    "PUSH_APPROVED": t("event.PUSH_APPROVED.label"),
    "PUSH_REJECTED": t("event.PUSH_REJECTED.label"),
    "PUSH_ERROR": t("event.PUSH_ERROR.label"),
    "RELEASE_NEW": t("event.RELEASE_NEW.label"),
  };
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (notification: ServiceNotification) => APIClient.notifications.update(notification),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NotificationKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("settings:forms.notification.updated", { name: notification.name })} t={toastInstance} />);
      toggle();
    }
  });

  const onSubmit = (formData: unknown) => mutation.mutate(formData as ServiceNotification);

  const deleteMutation = useMutation({
    mutationFn: (notificationID: number) => APIClient.notifications.delete(notificationID),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NotificationKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("settings:forms.notification.deleted", { name: notification.name })} t={toastInstance} />);
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
      title={t("settings:forms.notification.title")}
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      deleteAction={deleteAction}
      initialValues={initialValues}
      testFn={testNotification}
    >
      {(values) => (
        <div>
          <TextFieldWide name="name" label={t("settings:forms.notification.name")} required={true} />

          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            <div className="py-4 flex items-center justify-between space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:py-4">
              <div>
                <label
                  htmlFor="type"
                  className="block text-sm font-medium text-gray-900 dark:text-white"
                >
                  {t("settings:forms.notification.type")}
                </label>
              </div>
              <div className="sm:col-span-2">
                <Field name="type" type="select">
                  {({ field, form: { setFieldValue, resetForm } }: FieldProps) => (
                    <Select {...field}
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
                      placeholder={t("settings:forms.notification.chooseType")}
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
                      value={field?.value && notificationTypeOptions.find(o => o.value == field?.value)}
                      onChange={(option: unknown) => {
                        resetForm();
                        const opt = option as SelectOption;
                        setFieldValue(field.name, opt.value ?? "");
                      }}
                      options={notificationTypeOptions}
                    />
                  )}
                </Field>
              </div>
            </div>
            <SwitchGroupWide name="enabled" label={t("settings:forms.notification.enabled")} />
            <div className="pb-2">
              <div className="p-4">
                <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
                  {t("settings:forms.notification.globalEvents")}
                </DialogTitle>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  {t("settings:forms.notification.globalEventsDesc")}
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
                  {t("settings:forms.notification.perFilterEvents")}
                </DialogTitle>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  {t("settings:forms.notification.perFilterEventsDesc")}
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
                          ? f.events.map((e) => <span className="inline-flex items-center rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-600 dark:bg-gray-400/10 dark:text-gray-400">{filterEventOptions[e]}</span>)
                          : <span className="inline-flex items-center rounded-md bg-yellow-100 px-2 py-1 text-xs font-medium text-yellow-600 dark:bg-yellow-400/10 dark:text-yellow-400">{t("settings:forms.notification.muted")}</span>}
                      </div>
                    </div>
                  </Link>
                ))
                :
                <EmptySimple
                  title={t("settings:forms.notification.notUsedInFilters")}
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
