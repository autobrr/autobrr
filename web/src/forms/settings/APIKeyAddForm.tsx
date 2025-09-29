/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { XMarkIcon } from "@heroicons/react/24/solid";
import { Dialog, DialogPanel, DialogTitle, Transition, TransitionChild } from "@headlessui/react";
import type { FieldProps } from "formik";
import { Field, Form, Formik, FormikErrors, FormikValues } from "formik";

import { APIClient } from "@api/APIClient";
import { ApiKeys } from "@api/query_keys";
import { DEBUG } from "@components/debug";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { AddFormProps } from "@forms/_shared";

export function APIKeyAddForm({ isOpen, toggle }: AddFormProps) {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (apikey: APIKey) => APIClient.apikeys.create(apikey),
    onSuccess: (_, key) => {
      queryClient.invalidateQueries({ queryKey: ApiKeys.lists() });

      toast.custom((t) => <Toast type="success" body={`API key ${key.name} was added`} t={t}/>);

      toggle();
    }
  });

  const handleSubmit = (data: unknown) => mutation.mutate(data as APIKey);
  const validate = (values: FormikValues) => {
    const errors = {} as FormikErrors<FormikValues>;
    if (!values.name) {
      errors.name = "Required";
    }
    return errors;
  };

  return (
    <Transition show={isOpen} as={Fragment}>
      <Dialog as="div" static className="fixed inset-0 overflow-hidden" open={isOpen} onClose={toggle}>
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
              <div className="w-screen max-w-2xl ">
                <Formik
                  initialValues={{
                    name: "",
                    scopes: []
                  }}
                  onSubmit={handleSubmit}
                  validate={validate}
                >
                  {({ values }) => (
                    <Form className="h-full flex flex-col bg-white dark:bg-gray-800 shadow-xl overflow-y-auto">
                      <div className="flex-1">
                        <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
                          <div className="flex items-start justify-between space-x-3">
                            <div className="space-y-1">
                              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">Create API key</DialogTitle>
                              <p className="text-sm text-gray-500 dark:text-gray-400">
                                Add new API key.
                              </p>
                            </div>
                            <div className="h-7 flex items-center">
                              <button
                                type="button"
                                className="light:bg-white rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                                onClick={toggle}
                              >
                                <span className="sr-only">Close panel</span>
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
                                Name
                              </label>
                            </div>
                            <Field name="name">
                              {({
                                field,
                                meta
                              }: FieldProps) => (
                                <div className="sm:col-span-2">
                                  <input
                                    {...field}
                                    id="name"
                                    type="text"
                                    data-1p-ignore
                                    autoComplete="off"
                                    className="block w-full shadow-xs sm:text-sm focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 rounded-md border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-gray-100"
                                  />
                                  {meta.touched && meta.error && <span className="block mt-2 text-red-500">{meta.error}</span>}
                                </div>
                              )}
                            </Field>
                          </div>
                        </div>
                      </div>

                      <div
                        className="shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-5 sm:px-6">
                        <div className="space-x-3 flex justify-end">
                          <button
                            type="button"
                            className="bg-white dark:bg-gray-800 py-2 px-4 border border-gray-300 dark:border-gray-700 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                            onClick={toggle}
                          >
                            Cancel
                          </button>
                          <button
                            type="submit"
                            className="inline-flex justify-center py-2 px-4 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                          >
                            Create
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
