/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment, useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Select from "react-select";
import { Field, FieldProps, Form, Formik, FormikErrors, FormikValues, useFormikContext } from "formik";
import {
  Dialog,
  DialogPanel,
  DialogTitle,
  Listbox,
  ListboxButton, ListboxOption, ListboxOptions,
  Transition,
  TransitionChild
} from "@headlessui/react";
import { CheckIcon, ChevronUpDownIcon, XMarkIcon } from "@heroicons/react/24/solid";

import { APIClient } from "@api/APIClient";
import { ListKeys } from "@api/query_keys";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import * as common from "@components/inputs/common";
import {
  PasswordFieldWide,
  SwitchGroupWide,
  TextFieldWide
} from "@components/inputs";
import { ListTypeOptions, SelectOption } from "@domain/constants";
import { DEBUG } from "@components/debug";
import { DownloadClientsQueryOptions, FiltersGetAllQueryOptions } from "@api/queries";
import { classNames } from "@utils";
import { ListIndexerMultiSelectField } from "@components/inputs/select_wide.tsx";

interface ListAddFormValues {
  name: string;
  enabled: boolean;
}

interface AddFormProps {
  isOpen: boolean;
  toggle: () => void;
}

export function ListAddForm({ isOpen, toggle }: AddFormProps) {
  const queryClient = useQueryClient();

  const { data: clients } = useQuery(DownloadClientsQueryOptions());

  const filterQuery = useQuery(FiltersGetAllQueryOptions());

  const createMutation = useMutation({
    mutationFn: (list: List) => APIClient.lists.store(list),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ListKeys.lists() });

      toast.custom((t) => <Toast type="success" body="List added!" t={t}/>);
      toggle();
    },
    onError: () => {
      toast.custom((t) => <Toast type="error" body="List could not be added" t={t}/>);
    }
  });

  const onSubmit = (formData: unknown) => createMutation.mutate(formData as List);

  // const testMutation = useMutation({
  //   mutationFn: (n: ServiceNotification) => APIClient.notifications.test(n),
  //   onError: (err) => {
  //     console.error(err);
  //   }
  // });
  //
  // const testNotification = (data: unknown) => testMutation.mutate(data as ServiceNotification);

  const validate = (values: ListAddFormValues) => {
    const errors = {} as FormikErrors<FormikValues>;
    if (!values.name)
      errors.name = "Required";

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
              <div className="w-screen max-w-2xl dark:border-gray-700 border-l">
                <Formik
                  enableReinitialize={true}
                  initialValues={{
                    enabled: true,
                    type: "",
                    name: "",
                    client_id: 0,
                    url: "",
                    headers: [],
                    api_key: "",
                    filters: [],
                    match_release: false,
                    tags_included: [],
                    tags_excluded: [],
                    include_unmonitored: false,
                    exclude_alternate_titles: false,
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
                                Add List
                              </DialogTitle>
                              <p className="text-sm text-gray-500 dark:text-gray-200">
                                Auto update filters from lists and arrs.
                              </p>
                            </div>
                            <div className="h-7 flex items-center">
                              <button
                                type="button"
                                className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                                onClick={toggle}
                              >
                                <span className="sr-only">Close panel</span>
                                <XMarkIcon className="h-6 w-6" aria-hidden="true"/>
                              </button>
                            </div>
                          </div>
                        </div>

                        <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
                          <TextFieldWide
                            name="name"
                            label="Name"
                            required={true}
                          />

                          <div
                            className="flex items-center justify-between space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
                            <div>
                              <label
                                htmlFor="type"
                                className="block text-sm font-medium text-gray-900 dark:text-white"
                              >
                                Type
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
                                    options={ListTypeOptions}
                                  />
                                )}
                              </Field>
                            </div>
                          </div>

                          <SwitchGroupWide name="enabled" label="Enabled"/>
                        </div>
                        <ListTypeForm listType={values.type} clients={clients ?? []}/>

                        <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
                          <div className="border-t border-gray-200 dark:border-gray-700 py-4">
                            <div className="px-4 space-y-1">
                              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
                                Filters
                              </DialogTitle>
                              <p className="text-sm text-gray-500 dark:text-gray-400">
                                Select filters to update for this list.
                              </p>
                            </div>

                            <ListIndexerMultiSelectField name="filters" label="Filters" options={filterQuery.data?.map(f => ({ value: f.id, label: f.name })) ?? []} />

                            {/*<div className="mt-6 px-4 space-y-1">*/}
                            {/*  <ul>*/}
                            {/*    <li*/}
                            {/*      className="px-4 py-2 w-full flex justify-between border rounded-md dark:border-gray-700">*/}
                            {/*      <div>*/}
                            {/*        <span className="dark:text-white">Test</span>*/}
                            {/*      </div>*/}
                            {/*      x*/}
                            {/*    </li>*/}
                            {/*  </ul>*/}
                            {/*</div>*/}

                          </div>
                        </div>
                      </div>

                      <div className="flex-shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-4 sm:px-6">
                        <div className="space-x-3 flex justify-end">
                          <button
                            type="button"
                            className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                            // onClick={() => testNotification(values)}
                          >
                            Test
                          </button>
                          <button
                            type="button"
                            className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                            onClick={toggle}
                          >
                            Cancel
                          </button>
                          <button
                            type="submit"
                            className="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                          >
                            Save
                          </button>
                        </div>
                      </div>

                      <DEBUG values={values}/>
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

export function ListUpdateForm() {
  return (
    <div></div>
  );
}

interface ListTypeFormProps {
  listType: string;
  clients: DownloadClient[];
}

const ListTypeForm = (props: ListTypeFormProps) => {
  const { setFieldValue } = useFormikContext();
  const [prevActionType, setPrevActionType] = useState<string | null>(null);

  const { listType } = props;

  useEffect(() => {
    // if (prevActionType !== null && prevActionType !== list.type && ListTypeOptions.map(l => l.value).includes(list.type)) {
    if (prevActionType !== null && prevActionType !== listType && ListTypeOptions.map(l => l.value).includes(listType as ListType)) {
      // Reset the client_id field value
      setFieldValue(`client_id`, 0);
    }

    setPrevActionType(listType);
  }, [listType, prevActionType, setFieldValue]);

  switch (props.listType) {
    case "RADARR":
      return <ListTypeArr {...props} />;
    case "SONARR":
      return <ListTypeArr {...props} />;
    case "LIDARR":
      return <ListTypeArr {...props} />;
    case "READARR":
      return <ListTypeArr {...props} />;
    case "WHISPARR":
      return <ListTypeArr {...props} />;
    case "TRAKT":
      return <ListTypeTrakt {...props} />;
    case "STEAM":
      return <ListTypeTrakt {...props} />;
    case "METACRITIC":
      return <ListTypeTrakt {...props} />;
    case "MDBLIST":
      return <ListTypeTrakt {...props} />;
    case "PLAINTEXT":
      return <ListTypeTrakt {...props} />;
    default:
      return (
        <div></div>
      );
  }
}

const CheckBoxes = (props: ListTypeFormProps) => {
  switch (props.listType) {
    case "RADARR":
      return (
        <fieldset className="space-y-5">
          <legend className="sr-only">Settings</legend>
          <div className="relative flex items-start">
            <div className="flex items-center h-5">
              <Field
                id={`match_release`}
                aria-describedby={`match_release-description`}
                name="match_release"
                type="checkbox"
                // value={}
                className="focus:ring-blue-500 h-4 w-4 text-blue-600 border-gray-300 rounded"
              />
            </div>
            <div className="ml-3 text-sm">
              <label htmlFor={`match_release`}
                     className="font-medium text-gray-900 dark:text-gray-100">
                {/*{e.label}*/}
                Match Release
              </label>
              {/*{e.description && (*/}
              {/*  <p className="text-gray-500">{e.description}</p>*/}
              {/*)}*/}
            </div>
          </div>

          <div className="relative flex items-start">
            <div className="flex items-center h-5">
              <Field
                id={`include_unmonitored`}
                aria-describedby={`include_unmonitored-description`}
                name="include_unmonitored"
                type="checkbox"
                // value={}
                className="focus:ring-blue-500 h-4 w-4 text-blue-600 border-gray-300 rounded"
              />
            </div>
            <div className="ml-3 text-sm">
              <label htmlFor={`include_unmonitored`}
                     className="font-medium text-gray-900 dark:text-gray-100">
                {/*{e.label}*/}
                Include Unmonitored
              </label>
              {/*{e.description && (*/}
              {/*  <p className="text-gray-500">{e.description}</p>*/}
              {/*)}*/}
            </div>
          </div>
        </fieldset>
      )
    case "SONARR":
      return (
        <fieldset className="space-y-5">
          <legend className="sr-only">Settings</legend>
          <div className="relative flex items-start">
            <div className="flex items-center h-5">
              <Field
                id={`match_release`}
                aria-describedby={`match_release-description`}
                name="match_release"
                type="checkbox"
                // value={}
                className="focus:ring-blue-500 h-4 w-4 text-blue-600 border-gray-300 rounded"
              />
            </div>
            <div className="ml-3 text-sm">
              <label htmlFor={`match_release`}
                     className="font-medium text-gray-900 dark:text-gray-100">
                {/*{e.label}*/}
                Match Release
              </label>
              {/*{e.description && (*/}
              {/*  <p className="text-gray-500">{e.description}</p>*/}
              {/*)}*/}
            </div>
          </div>

          <div className="relative flex items-start">
            <div className="flex items-center h-5">
              <Field
                id={`include_unmonitored`}
                aria-describedby={`include_unmonitored-description`}
                name="include_unmonitored"
                type="checkbox"
                // value={}
                className="focus:ring-blue-500 h-4 w-4 text-blue-600 border-gray-300 rounded"
              />
            </div>
            <div className="ml-3 text-sm">
              <label htmlFor={`include_unmonitored`}
                     className="font-medium text-gray-900 dark:text-gray-100">
                {/*{e.label}*/}
                Include Unmonitored
              </label>
              {/*{e.description && (*/}
              {/*  <p className="text-gray-500">{e.description}</p>*/}
              {/*)}*/}
            </div>
          </div>

          <div className="relative flex items-start">
            <div className="flex items-center h-5">
              <Field
                id={`exclude_alternate_titles`}
                aria-describedby={`exclude_alternate_titles-description`}
                name="exclude_alternate_titles"
                type="checkbox"
                // value={}
                className="focus:ring-blue-500 h-4 w-4 text-blue-600 border-gray-300 rounded"
              />
            </div>
            <div className="ml-3 text-sm">
              <label htmlFor={`exclude_alternate_titles`}
                     className="font-medium text-gray-900 dark:text-gray-100">
                {/*{e.label}*/}
                Exclude Alternate Titles
              </label>
              {/*{e.description && (*/}
              {/*  <p className="text-gray-500">{e.description}</p>*/}
              {/*)}*/}
            </div>
          </div>
        </fieldset>
      )
  }
  // return (
  //   <fieldset className="space-y-5">
  //     <legend className="sr-only">Settings</legend>
  //     {/*{EventOptions.map((e, idx) => (*/}
  //     {/*  <div key={idx} className="relative flex items-start">*/}
  //     {/*    <div className="flex items-center h-5">*/}
  //     {/*      <Field*/}
  //     {/*        id={`events-${e.value}`}*/}
  //     {/*        aria-describedby={`events-${e.value}-description`}*/}
  //     {/*        name="events"*/}
  //     {/*        type="checkbox"*/}
  //     {/*        value={e.value}*/}
  //     {/*        className="focus:ring-blue-500 h-4 w-4 text-blue-600 border-gray-300 rounded"*/}
  //     {/*      />*/}
  //     {/*    </div>*/}
  //     {/*    <div className="ml-3 text-sm">*/}
  //     {/*      <label htmlFor={`events-${e.value}`}*/}
  //     {/*             className="font-medium text-gray-900 dark:text-gray-100">*/}
  //     {/*        {e.label}*/}
  //     {/*      </label>*/}
  //     {/*      {e.description && (*/}
  //     {/*        <p className="text-gray-500">{e.description}</p>*/}
  //     {/*      )}*/}
  //     {/*    </div>*/}
  //     {/*  </div>*/}
  //     {/*))}*/}
  //   </fieldset>
  // )
}

function ListTypeArr({ listType, clients }: ListTypeFormProps) {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4 space-y-1">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          Source
        </DialogTitle>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Select arr instance.
          {/*{" "}*/}
          {/*<ExternalLink*/}
          {/*  href="https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks"*/}
          {/*  className="font-medium text-blue-500 underline underline-offset-1 hover:text-blue-400"*/}
          {/*>*/}
          {/*  webhook integration*/}
          {/*</ExternalLink>*/}
          {/*{" in your server."}*/}
        </p>
      </div>

      <DownloadClientSelectCustom
        name={`client_id`}
        clients={clients}
        clientType={listType}
      />

      <div className="px-4 space-y-1">
        <CheckBoxes listType={listType} clients={[]}/>
      </div>

      {/*<PasswordFieldWide*/}
      {/*  name="webhook"*/}
      {/*  label="Webhook URL"*/}
      {/*  help="Discord channel webhook url"*/}
      {/*  placeholder="https://discordapp.com/api/webhooks/xx/xx"*/}
      {/*/>*/}
    </div>
  )
}

function ListTypeTrakt(props: ListTypeFormProps) {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4 space-y-1">
        <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
          Source list
        </DialogTitle>
        {/*<p className="text-sm text-gray-500 dark:text-gray-400">*/}
        {/*  {"Create a "}*/}
        {/*  <ExternalLink*/}
        {/*    href="https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks"*/}
        {/*    className="font-medium text-blue-500 underline underline-offset-1 hover:text-blue-400"*/}
        {/*  >*/}
        {/*    webhook integration*/}
        {/*  </ExternalLink>*/}
        {/*  {" in your server."}*/}
        {/*</p>*/}
      </div>

      <TextFieldWide name="url" label="URL" help={"Trakt List URL"} placeholder="https://trakt.tv"/>

      <PasswordFieldWide
        name="api_key"
        label="API Key"
        help="Trakt API Key"
        // placeholder="https://discordapp.com/api/webhooks/xx/xx"
      />

      <div className="px-4 space-y-1">
        <CheckBoxes listType={props.listType} clients={[]}/>
      </div>
    </div>
  )
}

interface DownloadClientSelectProps {
  name: string;
  clientType: string;
  clients: DownloadClient[];
}

function DownloadClientSelectCustom({ name, clientType, clients }: DownloadClientSelectProps) {
  return (
    <div className="flex items-center space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label
          htmlFor={name}
          className="block ml-px text-sm font-medium text-gray-900 dark:text-white"
        >
          <div className="flex">
            Select Client
          </div>
        </label>
      </div>
      <div className="sm:col-span-2">
        <Field name={name} type="select">
          {({
              field,
              meta,
              form: { setFieldValue }
            }: FieldProps) => (
            <Listbox
              value={field.value}
              onChange={(value) => setFieldValue(field?.name, value)}
            >
              {({ open }) => (
                <>
                  {/*<Label className="block text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide">*/}
                  {/*  Client*/}
                  {/*</Label>*/}
                  <div className="relative">
                    <ListboxButton
                      className="block w-full shadow-sm sm:text-sm rounded-md border py-2 pl-3 pr-10 text-left focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-gray-100">
                    <span className="block truncate">
                      {field.value
                        ? clients.find((c) => c.id === field.value)?.name
                        : "Choose a client"}
                    </span>
                      <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
                      <ChevronUpDownIcon
                        className="h-5 w-5 text-gray-400 dark:text-gray-300"
                        aria-hidden="true"/>
                    </span>
                    </ListboxButton>

                    <Transition
                      show={open}
                      as={Fragment}
                      leave="transition ease-in duration-100"
                      leaveFrom="opacity-100"
                      leaveTo="opacity-0"
                    >
                      <ListboxOptions
                        static
                        className="absolute z-10 mt-1 w-full border border-gray-400 dark:border-gray-700 bg-white dark:bg-gray-900 shadow-lg max-h-60 rounded-md py-1 text-base overflow-auto focus:outline-none sm:text-sm"
                      >
                        {clients
                          .filter((c) => c.type === clientType)
                          .map((client) => (
                            <ListboxOption
                              key={client.id}
                              className={({ focus }) => classNames(
                                focus
                                  ? "text-white dark:text-gray-100 bg-blue-600 dark:bg-gray-950"
                                  : "text-gray-900 dark:text-gray-300",
                                "cursor-default select-none relative py-2 pl-3 pr-9"
                              )}
                              value={client.id}
                            >
                              {({ selected, focus }) => (
                                <>
                                <span
                                  className={classNames(
                                    selected ? "font-semibold" : "font-normal",
                                    "block truncate"
                                  )}
                                >
                                  {client.name}
                                </span>

                                  {selected ? (
                                    <span
                                      className={classNames(
                                        focus ? "text-white dark:text-gray-100" : "text-blue-600 dark:text-blue-500",
                                        "absolute inset-y-0 right-0 flex items-center pr-4"
                                      )}
                                    >
                                    <CheckIcon
                                      className="h-5 w-5"
                                      aria-hidden="true"/>
                                  </span>
                                  ) : null}
                                </>
                              )}
                            </ListboxOption>
                          ))}
                      </ListboxOptions>
                    </Transition>
                    {meta.touched && meta.error && (
                      <p className="error text-sm text-red-600 mt-1">* {meta.error}</p>
                    )}
                  </div>
                </>
              )}
            </Listbox>
          )}
        </Field>
      </div>
    </div>
  );
}
