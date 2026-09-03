/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment, JSX, useRef } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useSelector } from "@tanstack/react-form";
import Select from "react-select";
import { useTranslation } from "react-i18next";
import {
  Listbox,
  ListboxButton,
  ListboxOption,
  ListboxOptions,
  Transition
} from "@headlessui/react";
import { CheckIcon, ChevronUpDownIcon, XMarkIcon } from "@heroicons/react/24/solid";

import { APIClient } from "@api/APIClient";
import { ListKeys } from "@api/query_keys";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import * as common from "@components/inputs/common";
import {
  MultiSelectOption,
  PasswordFieldWide,
  SwitchGroupWide,
  TextFieldWide
} from "@components/inputs";
import {
  ListsMDBListOptions,
  ListsMetacriticOptions,
  ListsTraktOptions,
  ListsAniListOptions,
  ListTypeOptions,
  OptionBasicTyped
} from "@domain/constants";
import { DEBUG } from "@components/debug";
import {
  DownloadersArrTagsQueryOptions,
  DownloadersQueryOptions,
  FiltersGetAllQueryOptions
} from "@api/queries";
import { classNames, sleep } from "@utils";
import {
  ListFilterMultiSelectField,
  SelectFieldBasic,
  SelectFieldCreatable
} from "@components/inputs/select_wide";
import { DocsTooltip } from "@components/tooltips/DocsTooltip";
import { MultiSelect as RMSC } from "react-multi-select-component";
import { useToggle } from "@hooks/hooks.ts";
import { errorMessages, fieldErrors, fieldHasError, useAppForm, useFormContext, useFormValues } from "@hooks/form";
import type { FormFieldErrors } from "@hooks/form";
import { DeleteModal } from "@components/modals";
import { SlideOverShell, SlideOverTitle } from "@components/panels";
import {DocsLink} from "@components/ExternalLink.tsx";

interface ListAddFormValues {
  name: string;
  enabled: boolean;
}

interface AddFormProps {
  isOpen: boolean;
  toggle: () => void;
}

export function ListAddForm({ isOpen, toggle }: AddFormProps) {
  return (
    <SlideOverShell isOpen={isOpen} toggle={toggle}>
      <ListAddFormPanel toggle={toggle} />
    </SlideOverShell>
  );
}

interface ListAddFormPanelProps {
  toggle: () => void;
}

function ListAddFormPanel({ toggle }: ListAddFormPanelProps) {
  const { t } = useTranslation("settings");
  const queryClient = useQueryClient();

  const { data: clients } = useQuery(DownloadersQueryOptions());

  const filterQuery = useQuery(FiltersGetAllQueryOptions());

  const createMutation = useMutation({
    mutationFn: (list: List) => APIClient.lists.store(list),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ListKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.list.added")} t={toastInstance}/>);
      toggle();
    },
    onError: () => {
      toast.custom((toastInstance) => <Toast type="error" body={t("forms.list.addFailed")} t={toastInstance}/>);
    }
  });

  const onSubmit = (formData: unknown) => createMutation.mutate(formData as List);

  const validate = (values: ListAddFormValues) => {
    const errors: FormFieldErrors = {};
    if (!values.name)
      errors.name = t("forms.list.required");

    return errors;
  };

  const initialValues = {
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
    include_alternate_titles: false,
    include_year: false,
    skip_clean_sanitize: false,
  };

  const form = useAppForm({
    defaultValues: initialValues,
    validators: {
      onChange: ({ value }) => fieldErrors(validate(value))
    },
    onSubmit: ({ value }) => onSubmit(value)
  });

  const values = useSelector(form.store, (state) => state.values);

  return (
    <form.AppForm>
      <form
        className="h-full min-h-0 flex flex-col bg-white dark:bg-gray-800"
        onSubmit={(e) => {
          e.preventDefault();
          e.stopPropagation();
          form.handleSubmit();
        }}
      >
        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
            <div className="flex items-start justify-between space-x-3">
              <div className="space-y-1">
                <SlideOverTitle>
                  {t("forms.list.addTitle")}
                </SlideOverTitle>
                <p className="text-sm text-gray-500 dark:text-gray-200">
                  {t("forms.list.description")}
                </p>
              </div>
              <div className="h-7 flex items-center">
                <button
                  type="button"
                  className="cursor-pointer bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500"
                  onClick={toggle}
                >
                  <span className="sr-only">{t("forms.list.closePanel")}</span>
                  <XMarkIcon className="h-6 w-6" aria-hidden="true"/>
                </button>
              </div>
            </div>
          </div>

          <div className="flex flex-col space-y-4 py-6 sm:py-0 sm:space-y-0">
            <TextFieldWide
              name="name"
              label={t("forms.list.name")}
              required={true}
            />

            <div className="flex items-center justify-between space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
              <div>
                <label htmlFor="type" className="block text-sm font-medium text-gray-900 dark:text-white"
                >
                  {t("forms.list.type")}
                </label>
              </div>
              <div className="sm:col-span-2">
                <form.Field name="type">
                  {(field) => (
                    <Select
                      name={field.name}
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
                      placeholder={t("forms.list.chooseType")}
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
                      value={ListTypeOptions.find((o) => o.value === field.state.value) ?? null}
                      onChange={(newValue: unknown) => {
                        const option = newValue as { value: string };
                        const nextType = option?.value ?? "";
                        if (nextType !== field.state.value) {
                          form.setFieldValue("client_id", 0, { dontUpdateMeta: true });
                          form.setFieldValue("url", "", { dontUpdateMeta: true });
                        }
                        field.handleChange(nextType);
                      }}
                      onBlur={field.handleBlur}
                      options={ListTypeOptions}
                    />
                  )}
                </form.Field>
              </div>
            </div>

            <SwitchGroupWide name="enabled" label={t("forms.list.enabled")}/>
          </div>

          <ListTypeForm listType={values.type as ListType} clients={clients ?? []}/>

          <div className="flex flex-col space-y-4 py-6 sm:py-0 sm:space-y-0">
            <div className="border-t border-gray-200 dark:border-gray-700 py-4">
              <div className="px-4">
                <h2 className="text-lg font-medium text-gray-900 dark:text-white">
                  {t("forms.list.filters")}
                </h2>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  {t("forms.list.filtersDescription")}
                </p>
              </div>

              <ListFilterMultiSelectField
                name="filters"
                label={t("forms.list.filters")}
                required={true}
                options={filterQuery.data?.map(f => ({ value: f.id, label: f.name })) ?? []}
              />

            </div>
          </div>
          <DEBUG values={values}/>
        </div>

        <div className="shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-4 sm:px-6">
          <div className="space-x-3 flex justify-end">
            <button
              type="button"
              className="cursor-pointer bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
              onClick={toggle}
            >
              {t("forms.list.cancel")}
            </button>
            <SubmitButton isPending={createMutation.isPending} isError={createMutation.isError} isSuccess={createMutation.isSuccess} />
          </div>
        </div>
      </form>
    </form.AppForm>
  );
}

interface UpdateFormProps<T> {
  isOpen: boolean;
  toggle: () => void;
  data: T;
}

export function ListUpdateForm({ isOpen, toggle, data }: UpdateFormProps<List>) {
  return (
    <SlideOverShell isOpen={isOpen} toggle={toggle}>
      <ListUpdateFormPanel toggle={toggle} data={data} />
    </SlideOverShell>
  );
}

interface ListUpdateFormPanelProps {
  toggle: () => void;
  data: List;
}

function ListUpdateFormPanel({ toggle, data }: ListUpdateFormPanelProps) {
  const { t } = useTranslation("settings");
  const cancelModalButtonRef = useRef<HTMLInputElement | null>(null);
  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);

  const queryClient = useQueryClient();

  const clientsQuery = useQuery(DownloadersQueryOptions());
  const filterQuery = useQuery(FiltersGetAllQueryOptions());

  const mutation = useMutation({
    mutationFn: (list: List) => APIClient.lists.update(list),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ListKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.list.updated", { name: data.name })} t={toastInstance}/>);

      sleep(1500);
      toggle();
    }
  });

  const onSubmit = (formData: unknown) => mutation.mutate(formData as List);

  const deleteMutation = useMutation({
    mutationFn: (listID: number) => APIClient.lists.delete(listID),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ListKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.list.deleted", { name: data.name })} t={toastInstance}/>);
    }
  });

  const deleteAction = () => deleteMutation.mutate(data.id);

  const initialValues = {
    id: data.id,
    enabled: data.enabled,
    type: data.type,
    name: data.name,
    client_id: data.client_id,
    url: data.url,
    headers: data.headers || [],
    api_key: data.api_key,
    filters: data.filters,
    match_release: data.match_release,
    tags_included: data.tags_included,
    tags_excluded: data.tags_excluded,
    include_unmonitored: data.include_unmonitored,
    include_alternate_titles: data.include_alternate_titles,
    include_year: data.include_year,
    skip_clean_sanitize: data.skip_clean_sanitize,
  };

  const form = useAppForm({
    defaultValues: initialValues,
    onSubmit: ({ value }) => onSubmit(value)
  });

  const values = useSelector(form.store, (state) => state.values);

  return (
    <>
      {deleteAction && (
        <DeleteModal
          isOpen={deleteModalIsOpen}
          isLoading={false}
          toggle={toggleDeleteModal}
          buttonRef={cancelModalButtonRef}
          deleteAction={deleteAction}
          title={t("forms.list.removeTitle", { name: data.name })}
          text={t("forms.list.removeText", { name: data.name })}
        />
      )}
      <form.AppForm>
        <form
          className="h-full min-h-0 flex flex-col bg-white dark:bg-gray-800"
          onSubmit={(e) => {
            e.preventDefault();
            e.stopPropagation();
            form.handleSubmit();
          }}
        >
          <div className="min-h-0 flex-1 overflow-y-auto">
            <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
              <div className="flex items-start justify-between space-x-3">
                <div className="space-y-1">
                  <SlideOverTitle>
                    {t("forms.list.updateTitle")}
                  </SlideOverTitle>
                  <p className="text-sm text-gray-500 dark:text-gray-200">
                    {t("forms.list.description")}
                  </p>
                </div>
                <div className="h-7 flex items-center">
                  <button
                    type="button"
                    className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500 cursor-pointer"
                    onClick={toggle}
                  >
                    <span className="sr-only">{t("forms.list.closePanel")}</span>
                    <XMarkIcon className="h-6 w-6" aria-hidden="true"/>
                  </button>
                </div>
              </div>
            </div>

            <div className="flex flex-col space-y-4 py-6 sm:py-0 sm:space-y-0">

              <TextFieldWide name="name" label={t("forms.list.name")} required={true}/>

              <TextFieldWide name="type" label={t("forms.list.type")} required={true} disabled={true} />

              <SwitchGroupWide name="enabled" label={t("forms.list.enabled")}/>

              <div className="space-y-2 divide-y divide-gray-200 dark:divide-gray-700">
                <ListTypeForm listType={values.type} clients={clientsQuery.data ?? []}/>
              </div>

              <div className="flex flex-col space-y-4 py-6 sm:py-0 sm:space-y-0">
                <div className="border-t border-gray-200 dark:border-gray-700 py-4">
                  <div className="px-4">
                    <h2 className="text-lg font-medium text-gray-900 dark:text-white">
                      {t("forms.list.filters")}
                    </h2>
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      {t("forms.list.filtersDescription")}
                    </p>
                  </div>

                  <ListFilterMultiSelectField
                    name="filters"
                    label={t("forms.list.filters")}
                    required={true}
                    options={filterQuery.data?.map(f => ({ value: f.id, label: f.name })) ?? []}
                  />

                </div>
              </div>

            </div>
            <DEBUG values={values}/>
          </div>

          <div className="shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-4">
            <div className="space-x-3 flex justify-between">
              <button
                type="button"
                className="cursor-pointer inline-flex items-center justify-center px-4 py-2 border border-transparent font-medium rounded-md text-red-700 dark:text-white bg-red-100 dark:bg-red-700 hover:bg-red-200 dark:hover:bg-red-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:text-sm"
                onClick={toggleDeleteModal}
              >
                {t("forms.list.remove")}
              </button>
              <div className="flex space-x-3">
              <button
                type="button"
                className="cursor-pointer bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                onClick={toggle}
              >
                {t("forms.list.cancel")}
              </button>
              <SubmitButton isPending={mutation.isPending} isError={mutation.isError} isSuccess={mutation.isSuccess} />
              </div>
            </div>
          </div>
        </form>
      </form.AppForm>
    </>
  );
}

interface SubmitButtonProps {
  isPending?: boolean;
  isError?: boolean;
  isSuccess?: boolean;
}

const SubmitButton = (props: SubmitButtonProps) => {
  const { t } = useTranslation("settings");
  return (
    <button
      type="submit"
      className={classNames(
        // isTestSuccessful
        //   ? "text-green-500 border-green-500 bg-green-50"
        //   : isError
        //     ? "text-red-500 border-red-500 bg-red-50"
        //     : "border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 focus:border-rose-700 active:bg-rose-700",
        props.isPending ? "cursor-not-allowed" : "cursor-pointer",
        "mr-2 inline-flex items-center px-4 py-2 border font-medium rounded-md shadow-xs text-sm transition ease-in-out duration-150 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 focus:border-blue-700 active:bg-blue-700"
      )}
    >
      {props.isPending ? (
        <>
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

          <span className="pl-2">{t("forms.list.saving")}</span>
        </>
      ) : (
        <span>{t("forms.list.save")}</span>
      )}
    </button>
  );
}

interface ListTypeFormProps {
  listID?: number;
  listType: ListType;
  clients: Downloader[];
}

const ListTypeForm = (props: ListTypeFormProps) => {
  const { listType } = props;

  switch (listType) {
    case "RADARR":
      return <ListTypeArr {...props} />;
    case "SONARR":
      return <ListTypeArr {...props} />;
    case "LIDARR":
      return <ListTypeArr {...props} />;
    case "READARR":
      return <ListTypeArr {...props} />;
    case "WHISPARR":
    case "WHISPARR_V3":
      return <ListTypeArr {...props} />;
    case "SPORTARR":
      return <ListTypeArr {...props} />;
    case "TRAKT":
      return <ListTypeTrakt />;
    case "STEAM":
      return <ListTypeSteam />;
    case "METACRITIC":
      return <ListTypeMetacritic />;
    case "MDBLIST":
      return <ListTypeMDBList />;
    case "PLAINTEXT":
      return <ListTypePlainText />;
    case "ANILIST":
        return <ListTypeAniList />;
    default:
      return null;
  }
}

const FilterOptionCheckBoxes = (props: ListTypeFormProps) => {
  const { t } = useTranslation("settings");
  switch (props.listType) {
    case "RADARR":
    case "SONARR":
      return (
        <fieldset>
          <legend className="sr-only">{t("forms.list.settingsLegend")}</legend>
          <SwitchGroupWide name="match_release" label={t("forms.list.matchRelease")} description={t("forms.list.matchReleaseDesc")} />
          <SwitchGroupWide name="include_unmonitored" label={t("forms.list.includeUnmonitored")} description={t("forms.list.includeUnmonitoredDesc")} />
          <SwitchGroupWide name="include_alternate_titles" label={t("forms.list.includeAlternateTitles")} description={t("forms.list.includeAlternateTitlesDesc")} />
        </fieldset>
      );
    case "LIDARR":
    case "WHISPARR":
    case "READARR":
    case "SPORTARR":
      return (
        <fieldset>
          <legend className="sr-only">{t("forms.list.settingsLegend")}</legend>
          <SwitchGroupWide name="include_unmonitored" label={t("forms.list.includeUnmonitored")} description={t("forms.list.includeUnmonitoredDesc")} />
        </fieldset>
      );
    case "WHISPARR_V3":
      return (
        <fieldset>
          <legend className="sr-only">{t("forms.list.settingsLegend")}</legend>
          <SwitchGroupWide name="include_unmonitored" label={t("forms.list.includeUnmonitored")} description={t("forms.list.includeUnmonitoredDesc")} />
          <SwitchGroupWide name="include_alternate_titles" label={t("forms.list.includeAlternateTitles")} description={t("forms.list.includeAlternateTitlesDesc")} />
        </fieldset>
      );
    case "PLAINTEXT":
      return (
        <fieldset>
          <legend className="sr-only">{t("forms.list.settingsLegend")}</legend>
          <SwitchGroupWide name="skip_clean_sanitize" label={t("forms.list.skipCleanSanitize")} description={t("forms.list.skipCleanSanitizeDesc")} />
        </fieldset>
      );
  }
}

function ListTypeArr({ listType, clients }: ListTypeFormProps) {
  const { t } = useTranslation("settings");
  const values = useFormValues<List>();

  const arrTagsQuery = useQuery(DownloadersArrTagsQueryOptions(values.client_id));

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <h2 className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.list.source")}
        </h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.list.arrSourceDescription")}
        </p>
      </div>

      <DownloaderSelectCustom
        name={`client_id`}
        clients={clients}
        clientType={listType}
      />

      {values.client_id > 0 && (values.type === "RADARR" || values.type === "SONARR" || values.type === "WHISPARR" || values.type === "WHISPARR_V3" || values.type === "SPORTARR") && (
        <>
          <ListArrTagsMultiSelectField name="tags_included" label={t("forms.list.tagsIncluded")} options={arrTagsQuery.data?.map(f => ({
            value: f.label,
            label: f.label
          })) ?? []}/>

          <ListArrTagsMultiSelectField name="tags_excluded" label={t("forms.list.tagsExcluded")} options={arrTagsQuery.data?.map(f => ({
            value: f.label,
            label: f.label
          })) ?? []}/>
        </>
      )}

      <div className="space-y-1">
        <FilterOptionCheckBoxes listType={listType} clients={[]}/>
      </div>
    </div>
  )
}

function ListTypeTrakt() {
  const { t } = useTranslation("settings");
  const values = useFormValues<List>();

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <h2 className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.list.sourceList")}
        </h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.list.traktSourceDescription")}
        </p>
      </div>

      <SelectFieldCreatable
        name="url"
        label={t("forms.list.listUrl")}
        help={t("forms.list.traktHelp")}
        options={ListsTraktOptions.map(u => ({ value: u.value, label: u.label, key: u.label }))}
      />

      {!values.url.startsWith("https://api.autobrr.com/") && (
        <PasswordFieldWide
          name="api_key"
          label={t("forms.list.traktApiKey")}
          help={t("forms.list.traktApiKeyHelp")}
        />
      )}

      <div className="space-y-1">
        <fieldset>
          <legend className="sr-only">{t("forms.list.settingsLegend")}</legend>
          <SwitchGroupWide name="match_release" label={t("forms.list.matchRelease")} description={t("forms.list.matchReleaseDesc")} />
        </fieldset>
      </div>
    </div>
  )
}

function ListTypeAniList() {
  const { t } = useTranslation("settings");
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4 space-y-1">
        <h2 className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.list.sourceList")}
        </h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.list.anilistSourceDescription")}
        </p>
      </div>

      <SelectFieldBasic
        name="url"
        label={t("forms.list.listUrl")}
        options={ListsAniListOptions.map(u => ({ value: u.value, label: u.label, key: u.label }))}
      />

      <div className="space-y-1">
        <fieldset>
          <legend className="sr-only">{t("forms.list.settingsLegend")}</legend>
          <SwitchGroupWide name="match_release" label={t("forms.list.matchRelease")} description={t("forms.list.matchReleaseDesc")} />
        </fieldset>
      </div>
    </div>
  )
}

function ListTypePlainText() {
  const { t } = useTranslation("settings");
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <h2 className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.list.sourceList")}
        </h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.list.plaintextSourceDescription")}
        </p>
      </div>

      <TextFieldWide
        name="url"
        label={t("forms.list.listUrl")}
        help={t("forms.list.plaintextUrlHelp")}
        placeholder={t("forms.list.plaintextUrlPlaceholder")}
        tooltip={
            <div>
                <p>{t("forms.list.plaintextTooltip1")}</p>
                <br />
                <p>{t("forms.list.plaintextTooltipRemote")}</p>
                <br />
                <p>{t("forms.list.plaintextTooltipLocal")}</p>
                <DocsLink href="https://autobrr.com/filters/lists" />
            </div>
        }
      />

      <div className="space-y-1">
        <fieldset>
          <legend className="sr-only">{t("forms.list.settingsLegend")}</legend>
          <SwitchGroupWide name="match_release" label={t("forms.list.matchRelease")} description={t("forms.list.matchReleaseDesc")} />
        </fieldset>
      </div>
      <div className="space-y-1">
        <fieldset>
          <legend className="sr-only">{t("forms.list.settingsLegend")}</legend>
          <SwitchGroupWide name="skip_clean_sanitize" label={t("forms.list.skipCleanSanitize")} description={t("forms.list.skipCleanSanitizeDesc")} />
        </fieldset>
      </div>
    </div>
  )
}

function ListTypeSteam() {
  const { t } = useTranslation("settings");
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <h2 className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.list.sourceList")}
        </h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.list.steamSourceDescription")}
        </p>
      </div>

      <TextFieldWide name="url" label={t("forms.list.url")} help={t("forms.list.steamUrlHelp")} placeholder={t("forms.list.steamUrlPlaceholder")}/>
    </div>
  )
}

function ListTypeMetacritic() {
  const { t } = useTranslation("settings");
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <h2 className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.list.sourceList")}
        </h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.list.metacriticSourceDescription")}
        </p>
      </div>

      <SelectFieldCreatable
        name="url"
        label={t("forms.list.listUrl")}
        help={t("forms.list.metacriticHelp")}
        options={ListsMetacriticOptions.map(u => ({ value: u.value, label: u.label, key: u.label }))}
      />

      <div className="space-y-1">
        <fieldset>
          <legend className="sr-only">{t("forms.list.settingsLegend")}</legend>
          <SwitchGroupWide name="match_release" label={t("forms.list.matchRelease")} description={t("forms.list.matchReleaseDesc")} />
        </fieldset>
      </div>
    </div>
  )
}

function ListTypeMDBList() {
    const { t } = useTranslation("settings");
    const form = useFormContext();

    return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4">
        <h2 className="text-lg font-medium text-gray-900 dark:text-white">
          {t("forms.list.sourceList")}
        </h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("forms.list.mdblistSourceDescription")}
        </p>
      </div>

      <SelectFieldCreatable
        name="url"
        label={t("forms.list.listUrl")}
        help={t("forms.list.mdblistHelp")}
        options={ListsMDBListOptions.map(u => ({ value: u.value, label: u.label, key: u.label }))}
      />

      <div className="space-y-1">
        <fieldset>
          <legend className="sr-only">{t("forms.list.settingsLegend")}</legend>
          <SwitchGroupWide name="match_release" label={t("forms.list.matchRelease")} description={t("forms.list.matchReleaseDesc")} />
          <SwitchGroupWide
            name="include_year"
            label={t("forms.list.includeYear")}
            description={t("forms.list.includeYearDesc")}
            onChange={(includeYear) => {
              if (includeYear) {
                form.setFieldValue("match_release", true);
              }
            }}
          />
        </fieldset>
      </div>
    </div>
  )
}

interface DownloaderSelectProps {
  name: string;
  clientType: string;
  clients: Downloader[];
}

function DownloaderSelectCustom({ name, clientType, clients }: DownloaderSelectProps) {
  const { t } = useTranslation("settings");
  const form = useFormContext();
  return (
    <div className="flex items-center space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label
          htmlFor={name}
          className="block ml-px text-sm font-medium text-gray-900 dark:text-white"
        >
          <div className="flex">
            {t("forms.list.selectClient")}
          </div>
        </label>
      </div>
      <div className="sm:col-span-2">
        <form.Field name={name}>
          {(field) => (
            <Listbox
              value={field.state.value}
              onChange={(value) => field.handleChange(value)}
            >
              {({ open }) => (
                <>
                  {/*<Label className="block text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide">*/}
                  {/*  Client*/}
                  {/*</Label>*/}
                  <div className="relative">
                    <ListboxButton
                      className="block w-full shadow-xs sm:text-sm rounded-md border py-2 pl-3 pr-10 text-left focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-gray-100">
                    <span className="block truncate">
                      {field.state.value
                        ? clients.find((c) => c.id === field.state.value)?.name
                        : t("forms.list.chooseClient")}
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
                        className="absolute z-10 mt-1 w-full border border-gray-400 dark:border-gray-700 bg-white dark:bg-gray-900 shadow-lg max-h-60 rounded-md py-1 text-base overflow-auto focus:outline-hidden sm:text-sm"
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
                    {fieldHasError(field.state.meta) && (
                      <p className="error text-sm text-red-600 mt-1">* {errorMessages(field.state.meta.errors)[0]}</p>
                    )}
                  </div>
                </>
              )}
            </Listbox>
          )}
        </form.Field>
      </div>
    </div>
  );
}

export interface ListMultiSelectFieldProps {
  name: string;
  label: string;
  help?: string;
  placeholder?: string;
  required?: boolean;
  tooltip?: JSX.Element;
  options: OptionBasicTyped<number | string>[];
}

export function ListArrTagsMultiSelectField({ name, label, help, tooltip, options }: ListMultiSelectFieldProps) {
  const { t } = useTranslation("settings");
  const form = useFormContext();
  return (
    <div className="flex items-center space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label
          htmlFor={name}
          className="block ml-px text-sm font-medium text-gray-900 dark:text-white"
        >
          <div className="flex">
            {tooltip ? (
              <DocsTooltip label={label}>{tooltip}</DocsTooltip>
            ) : label}
          </div>
        </label>
      </div>
      <div className="sm:col-span-2">
        <form.Field name={name}>
          {(field) => (
            <>
              <RMSC
                options={options}
                overrideStrings={{
                  selectSomeItems: t("forms.list.selectSomeItems"),
                  allItemsAreSelected: t("forms.list.allItemsSelected"),
                  selectAll: t("forms.list.selectAll"),
                  search: t("forms.list.search"),
                  noOptions: t("forms.list.noOptions")
                }}
                // disabled={disabled}
                labelledBy={name}
                // isCreatable={creatable}
                // onCreateOption={handleNewField}
                value={field.state.value && field.state.value.map((item: MultiSelectOption) => ({
                  value: item.value ? item.value : item,
                  label: item.label ? item.label : item
                }))}
                onChange={(values: Array<MultiSelectOption>) => {
                  const am = values && values.map((i) => i.value);

                  field.handleChange(am);
                }}
              />
            </>
          )}
        </form.Field>
        {help && (
          <p className="mt-2 text-sm text-gray-500" id={`${name}-description`}>{help}</p>
        )}
      </div>
    </div>
  );
}
