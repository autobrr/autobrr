/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useRef, RefObject } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { XMarkIcon } from "@heroicons/react/24/solid";
import { useTranslation } from "react-i18next";

import { APIClient } from "@api/APIClient";
import { FilterKeys } from "@api/query_keys";
import { FormDebug } from "@components/debug";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { AddFormProps } from "@forms/_shared";
import { useAppForm, fieldErrors } from "@hooks/form";
import type { FormFieldErrors } from "@hooks/form";
import { ErrorField } from "@components/inputs/common";
import { SlideOverShell, SlideOverTitle } from "@components/panels";

interface InitialValues {
  name: string;
  enabled: boolean;
  resolutions: string[];
  codecs: string[];
  sources: string[];
  containers: string[];
  origins: string[];
}

export function FilterAddForm({ isOpen, toggle }: AddFormProps) {
  const inputRef = useRef<HTMLInputElement | null>(null);

  return (
    <SlideOverShell isOpen={isOpen} toggle={toggle} initialFocus={inputRef} zIndexClass="z-20">
      <FilterAddFormPanel toggle={toggle} inputRef={inputRef} />
    </SlideOverShell>
  );
}

interface FilterAddFormPanelProps {
  toggle: () => void;
  inputRef: RefObject<HTMLInputElement | null>;
}

function FilterAddFormPanel({ toggle, inputRef }: FilterAddFormPanelProps) {
  const { t } = useTranslation("filters");
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const mutation = useMutation({
    mutationFn: (filter: Filter) => APIClient.filters.create(filter),
    onSuccess: (filter) => {
      queryClient.invalidateQueries({ queryKey: FilterKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("addForm.added", { name: filter.name })} t={toastInstance} />);

      if (filter.id) {
        navigate({ to: "/filters/$filterId", params: { filterId: filter.id }})
      }
    }
  });

  const handleSubmit = (data: InitialValues) => mutation.mutate(data as Filter);

  const validate = (values: InitialValues) => {
    const errors: FormFieldErrors = {};
    if (!values.name) {
      errors.name = t("addForm.required");
    }
    return errors;
  };

  const initialValues: InitialValues = {
    name: "",
    enabled: false,
    resolutions: [],
    codecs: [],
    sources: [],
    containers: [],
    origins: []
  };

  const form = useAppForm({
    defaultValues: initialValues,
    validators: {
      onChange: ({ value }) => fieldErrors(validate(value))
    },
    onSubmit: ({ value }) => handleSubmit(value)
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
                <SlideOverTitle>{t("addForm.title")}</SlideOverTitle>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  {t("addForm.subtitle")}
                </p>
              </div>
              <div className="h-7 flex items-center">
                <button
                  type="button"
                  className="light:bg-white rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer"
                  onClick={toggle}
                >
                  <span className="sr-only">{t("addForm.closePanel")}</span>
                  <XMarkIcon className="h-6 w-6" aria-hidden="true" />
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
                  {t("addForm.name")}
                  <span className="text-red-500"> *</span>
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
                      ref={inputRef}
                      className="block w-full shadow-xs sm:text-sm rounded-md border py-2.5 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-gray-100"
                    />

                    <ErrorField meta={field.state.meta} classNames="block mt-2 text-red-500" />

                  </div>
                )}
              </form.Field>
            </div>
          </div>

          <FormDebug />
        </div>

        <div className="shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-5 sm:px-6">
          <div className="space-x-3 flex justify-end">
            <button
              type="button"
              className="bg-white dark:bg-gray-800 py-2 px-4 border border-gray-300 dark:border-gray-700 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer"
              onClick={toggle}
            >
              {t("addForm.cancel")}
            </button>
            <button
              type="submit"
              className="inline-flex justify-center py-2 px-4 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer"
            >
              {t("addForm.create")}
            </button>
          </div>
        </div>
      </form>
    </form.AppForm>
  );
}
