/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useId, useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Menu, MenuButton, MenuItem, MenuItems, Popover, PopoverButton, PopoverPanel } from "@headlessui/react";
import { ChevronDownIcon, MagnifyingGlassIcon, XMarkIcon } from "@heroicons/react/24/solid";
import { useTranslation } from "react-i18next";
import type { AnyFieldMeta } from "@tanstack/react-form";

import { APIClient } from "@api/APIClient";
import { ApiKeys } from "@api/query_keys";
import { FormDebug } from "@components/debug";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { AddFormProps, UpdateFormProps } from "@forms/_shared";
import { useAppForm, fieldErrors } from "@hooks/form";
import type { FormFieldErrors } from "@hooks/form";
import { ErrorField } from "@components/inputs/common";
import { SlideOverShell, SlideOverTitle } from "@components/panels";
import { API_KEY_FULL_ACCESS, API_KEY_RESOURCES } from "@domain/constants";
import { classNames } from "@utils";

interface InitialValues {
  name: string;
  scopes: string[];
}

export function APIKeyAddForm({ isOpen, toggle }: AddFormProps) {
  return (
    <SlideOverShell isOpen={isOpen} toggle={toggle}>
      <APIKeyFormPanel toggle={toggle} />
    </SlideOverShell>
  );
}

export function APIKeyUpdateForm({ isOpen, toggle, data }: UpdateFormProps<APIKey>) {
  return (
    <SlideOverShell isOpen={isOpen} toggle={toggle}>
      <APIKeyFormPanel toggle={toggle} apikey={data} />
    </SlideOverShell>
  );
}

interface APIKeyFormPanelProps {
  toggle: () => void;
  apikey?: APIKey;
}

function APIKeyFormPanel({ toggle, apikey }: APIKeyFormPanelProps) {
  const { t } = useTranslation("settings");
  const queryClient = useQueryClient();

  const createMutation = useMutation({
    mutationFn: (key: APIKey) => APIClient.apikeys.create(key),
    onSuccess: (_, key) => {
      queryClient.invalidateQueries({ queryKey: ApiKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.apiKey.created", { name: key.name })} t={toastInstance}/>);

      toggle();
    }
  });

  const updateMutation = useMutation({
    mutationFn: (key: APIKey) => APIClient.apikeys.update(key),
    onSuccess: (_, key) => {
      queryClient.invalidateQueries({ queryKey: ApiKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.apiKey.updated", { name: key.name })} t={toastInstance}/>);

      toggle();
    }
  });

  const onSubmit = (formData: InitialValues) => {
    if (apikey) {
      updateMutation.mutate({ ...apikey, ...formData });
      return;
    }

    createMutation.mutate(formData as APIKey);
  };

  const validate = (values: InitialValues) => {
    const errors: FormFieldErrors = {};
    if (!values.name) {
      errors.name = t("forms.apiKey.required");
    }
    if (!values.scopes.length) {
      errors.scopes = t("forms.apiKey.permissionsRequired");
    }
    return errors;
  };

  const initialValues: InitialValues = {
    name: apikey?.name ?? "",
    scopes: apikey?.scopes ?? []
  };

  const form = useAppForm({
    defaultValues: initialValues,
    validators: {
      onChange: ({ value }) => fieldErrors(validate(value))
    },
    onSubmit: ({ value }) => onSubmit(value)
  });

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
                <SlideOverTitle>{apikey ? t("forms.apiKey.updateTitle") : t("forms.apiKey.createTitle")}</SlideOverTitle>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  {apikey ? t("forms.apiKey.updateDescription") : t("forms.apiKey.description")}
                </p>
              </div>
              <div className="h-7 flex items-center">
                <button
                  type="button"
                  className="light:bg-white rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer"
                  onClick={toggle}
                >
                  <span className="sr-only">{t("forms.apiKey.closePanel")}</span>
                  <XMarkIcon className="h-6 w-6" aria-hidden="true"/>
                </button>
              </div>
            </div>
          </div>

          <div
            className="py-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
            <div
              className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:py-4">
              <div>
                <label
                  htmlFor="name"
                  className="block text-sm font-medium text-gray-900 dark:text-white sm:mt-px sm:pt-2"
                >
                  {t("forms.apiKey.name")}
                </label>
              </div>
              <form.Field name="name">
                {(field) => (
                  <div className="sm:col-span-2">
                    <input
                      name={field.name}
                      value={field.state.value}
                      onChange={(e) => field.handleChange(e.target.value)}
                      onBlur={field.handleBlur}
                      id="name"
                      type="text"
                      data-1p-ignore
                      autoComplete="off"
                      className="block w-full shadow-xs sm:text-sm focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 rounded-md border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-gray-100"
                    />
                    <ErrorField meta={field.state.meta} classNames="block mt-2 text-red-500" />
                  </div>
                )}
              </form.Field>
            </div>
          </div>

          <form.Field name="scopes">
            {(field) => (
              <APIKeyPermissions
                scopes={field.state.value}
                setScopes={(scopes) => field.handleChange(scopes)}
                meta={field.state.meta}
              />
            )}
          </form.Field>

          <FormDebug />
        </div>

        <div className="shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-5 sm:px-6">
          <div className="space-x-3 flex justify-end">
            <button
              type="button"
              className="bg-white dark:bg-gray-800 py-2 px-4 border border-gray-300 dark:border-gray-700 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer"
              onClick={toggle}
            >
              {t("forms.apiKey.cancel")}
            </button>
            <button
              type="submit"
              disabled={createMutation.isPending || updateMutation.isPending}
              className="inline-flex justify-center py-2 px-4 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer disabled:cursor-not-allowed disabled:opacity-50"
            >
              {apikey ? t("forms.apiKey.save") : t("forms.apiKey.create")}
            </button>
          </div>
        </div>
      </form>
    </form.AppForm>
  );
}

interface APIKeyPermissionsProps {
  scopes: string[];
  setScopes: (scopes: string[]) => void;
  meta: AnyFieldMeta;
}

function APIKeyPermissions({ scopes, setScopes, meta }: APIKeyPermissionsProps) {
  const { t } = useTranslation("settings");

  const accessModeName = useId();
  const fullAccess = scopes.includes(API_KEY_FULL_ACCESS);

  const selected = useMemo(() => {
    const access = new Map(scopes.map((scope) => scope.split(":") as [string, APIKeyAccess]));

    return API_KEY_RESOURCES.flatMap((resource) => {
      const level = access.get(resource.name);
      return level ? [{ resource, access: level }] : [];
    });
  }, [scopes]);

  const toggleResource = (resource: APIKeyResource) => {
    if (selected.some((s) => s.resource.name === resource.name)) {
      removeResource(resource.name);
      return;
    }

    setScopes([...scopes, `${resource.name}:${resource.access[0]}`]);
  };

  const removeResource = (name: string) => {
    setScopes(scopes.filter((scope) => !scope.startsWith(`${name}:`)));
  };

  const setAccess = (name: string, access: APIKeyAccess) => {
    setScopes(scopes.map((scope) => (scope.startsWith(`${name}:`) ? `${name}:${access}` : scope)));
  };

  return (
    <div className="px-4 py-6 sm:py-4 border-t border-gray-200 dark:border-gray-700">
      <h2 className="text-lg font-medium text-gray-900 dark:text-white">{t("forms.apiKey.permissions")}</h2>
      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("forms.apiKey.permissionsDescription")}</p>

      <fieldset className="mt-4 space-y-3">
        <legend className="sr-only">{t("forms.apiKey.permissions")}</legend>
        <AccessModeOption
          name={accessModeName}
          checked={fullAccess}
          onChange={() => setScopes([API_KEY_FULL_ACCESS])}
          label={t("forms.apiKey.fullAccess")}
          description={t("forms.apiKey.fullAccessDescription")}
        />
        <AccessModeOption
          name={accessModeName}
          checked={!fullAccess}
          onChange={() => setScopes([])}
          label={t("forms.apiKey.customAccess")}
          description={t("forms.apiKey.customAccessDescription")}
        />
      </fieldset>

      {!fullAccess && (
        <div className="mt-4 rounded-md border border-gray-250 dark:border-gray-750">
          <div className="flex items-center justify-between px-4 py-2 border-b border-gray-250 dark:border-gray-750">
            <div className="flex items-center text-sm font-medium text-gray-900 dark:text-white">
              {t("forms.apiKey.permissions")}
              <span className="ml-2 inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400">
                {selected.length}
              </span>
            </div>
            <AddPermissionsPopover selected={selected.map((s) => s.resource.name)} toggleResource={toggleResource} />
          </div>

          {selected.length ? (
            <ul className="divide-y divide-gray-250 dark:divide-gray-750">
              {selected.map(({ resource, access }) => (
                <li key={resource.name} className="flex items-center justify-between gap-4 px-4 py-3">
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-gray-900 dark:text-white">
                      {t(`forms.apiKey.resources.${resource.name}.label`)}
                    </p>
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      {t(`forms.apiKey.resources.${resource.name}.description`)}
                    </p>
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    <AccessMenu resource={resource} access={access} setAccess={(value) => setAccess(resource.name, value)} />
                    <button
                      type="button"
                      className="rounded-md p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 focus:outline-hidden focus:ring-2 focus:ring-blue-500 cursor-pointer"
                      onClick={() => removeResource(resource.name)}
                      title={t("forms.apiKey.removePermission", { name: t(`forms.apiKey.resources.${resource.name}.label`) })}
                    >
                      <XMarkIcon className="h-4 w-4" aria-hidden="true" />
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          ) : (
            <p className="px-4 py-6 text-center text-sm text-gray-500 dark:text-gray-400">
              {t("forms.apiKey.noPermissions")}
            </p>
          )}
        </div>
      )}

      <ErrorField meta={meta} classNames="block mt-2 text-red-500" />
    </div>
  );
}

interface AccessModeOptionProps {
  name: string;
  checked: boolean;
  onChange: () => void;
  label: string;
  description: string;
}

const AccessModeOption = ({ name, checked, onChange, label, description }: AccessModeOptionProps) => (
  <label className="flex items-start gap-3 cursor-pointer">
    <input
      type="radio"
      name={name}
      checked={checked}
      onChange={onChange}
      className="mt-0.5 h-4 w-4 border-gray-300 dark:border-gray-600 text-blue-600 bg-white dark:bg-gray-700 checked:bg-blue-600 dark:checked:bg-blue-600 focus:ring-blue-500 cursor-pointer"
    />
    <span>
      <span className="block text-sm font-medium text-gray-900 dark:text-white">{label}</span>
      <span className="block text-sm text-gray-500 dark:text-gray-400">{description}</span>
    </span>
  </label>
);

interface AddPermissionsPopoverProps {
  selected: string[];
  toggleResource: (resource: APIKeyResource) => void;
}

function AddPermissionsPopover({ selected, toggleResource }: AddPermissionsPopoverProps) {
  const { t } = useTranslation("settings");
  const [query, setQuery] = useState("");

  const resources = API_KEY_RESOURCES.filter((resource) =>
    t(`forms.apiKey.resources.${resource.name}.label`).toLowerCase().includes(query.toLowerCase())
  );

  return (
    <Popover>
      <PopoverButton
        className="inline-flex items-center rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-200 shadow-xs hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-blue-500 cursor-pointer"
      >
        {t("forms.apiKey.addPermissions")}
      </PopoverButton>
      <PopoverPanel
        anchor={{ to: "bottom end", gap: "4px", padding: "8px" }}
        className="w-72 bg-white dark:bg-gray-825 rounded-md shadow-lg border border-gray-250 dark:border-gray-750 focus:outline-hidden z-10"
      >
        <div className="px-3 pt-3 pb-2">
          <p className="text-sm font-medium text-gray-900 dark:text-white">{t("forms.apiKey.selectPermissions")}</p>
          <div className="relative mt-2">
            <MagnifyingGlassIcon className="pointer-events-none absolute left-2 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" aria-hidden="true" />
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={t("forms.apiKey.searchPermissions")}
              autoFocus
              data-1p-ignore
              autoComplete="off"
              className="block w-full pl-8 shadow-xs sm:text-sm focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 rounded-md border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-gray-100"
            />
          </div>
        </div>
        <ul className="max-h-64 overflow-y-auto pb-2">
          {resources.map((resource) => (
            <li key={resource.name}>
              <label className="flex items-center gap-3 px-3 py-1.5 text-sm text-gray-900 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-775 cursor-pointer">
                <input
                  type="checkbox"
                  checked={selected.includes(resource.name)}
                  onChange={() => toggleResource(resource)}
                  className="h-4 w-4 rounded-sm border-gray-300 dark:border-gray-600 text-blue-600 bg-white dark:bg-gray-700 checked:bg-blue-600 dark:checked:bg-blue-600 focus:ring-blue-500 cursor-pointer"
                />
                {t(`forms.apiKey.resources.${resource.name}.label`)}
              </label>
            </li>
          ))}
          {!resources.length && (
            <li className="px-3 py-1.5 text-sm text-gray-500 dark:text-gray-400">{t("forms.apiKey.noPermissionsFound")}</li>
          )}
        </ul>
      </PopoverPanel>
    </Popover>
  );
}

interface AccessMenuProps {
  resource: APIKeyResource;
  access: APIKeyAccess;
  setAccess: (access: APIKeyAccess) => void;
}

function AccessMenu({ resource, access, setAccess }: AccessMenuProps) {
  const { t } = useTranslation("settings");

  if (resource.access.length === 1) {
    return (
      <span className="px-2.5 py-1 text-xs text-gray-500 dark:text-gray-400">
        {t("forms.apiKey.accessLabel")} <span className="font-medium text-gray-900 dark:text-gray-200">{t(`forms.apiKey.accessShort.${access}`)}</span>
      </span>
    );
  }

  return (
    <Menu>
      <MenuButton
        className="inline-flex items-center rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-2.5 py-1 text-xs text-gray-500 dark:text-gray-400 shadow-xs hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-blue-500 cursor-pointer"
      >
        {t("forms.apiKey.accessLabel")}&nbsp;<span className="font-medium text-gray-900 dark:text-gray-200">{t(`forms.apiKey.access.${access}`)}</span>
        <ChevronDownIcon className="ml-1 h-3 w-3" aria-hidden="true" />
      </MenuButton>
      <MenuItems
        anchor={{ to: "bottom end", padding: "8px" }}
        className="w-44 bg-white dark:bg-gray-825 divide-y divide-gray-200 dark:divide-gray-750 rounded-md shadow-lg border border-gray-250 dark:border-gray-750 focus:outline-hidden z-10"
      >
        {resource.access.map((value) => (
          <MenuItem key={value}>
            {({ focus }) => (
              <button
                type="button"
                className={classNames(
                  focus ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                  "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm cursor-pointer"
                )}
                onClick={() => setAccess(value)}
              >
                {t(`forms.apiKey.access.${value}`)}
              </button>
            )}
          </MenuItem>
        ))}
      </MenuItems>
    </Menu>
  );
}
