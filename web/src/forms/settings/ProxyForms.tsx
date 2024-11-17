/*
 * Copyright (c) 2021-2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment } from "react";
import { Form, Formik, FormikValues } from "formik";
import { Dialog, DialogPanel, DialogTitle, Transition, TransitionChild } from "@headlessui/react";
import { XMarkIcon } from "@heroicons/react/24/solid";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "react-hot-toast";

import { AddFormProps } from "@forms/_shared";
import { DEBUG } from "@components/debug.tsx";
import { PasswordFieldWide, SwitchGroupWide, TextFieldWide } from "@components/inputs";
import { SelectFieldBasic } from "@components/inputs/select_wide";
import { ProxyTypeOptions } from "@domain/constants";
import { APIClient } from "@api/APIClient";
import { ProxyKeys } from "@api/query_keys";
import Toast from "@components/notifications/Toast";
import { SlideOver } from "@components/panels";

export function ProxyAddForm({ isOpen, toggle }: AddFormProps) {
  const queryClient = useQueryClient();

  const createMutation = useMutation({
    mutationFn: (req: ProxyCreate) => APIClient.proxy.store(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ProxyKeys.lists() });

      toast.custom((t) => <Toast type="success" body="Proxy added!" t={t} />);
      toggle();
    },
    onError: () => {
      toast.custom((t) => <Toast type="error" body="Proxy could not be added" t={t} />);
    }
  });

  const onSubmit = (formData: FormikValues) => {
    createMutation.mutate(formData as ProxyCreate);
  }

  const testMutation = useMutation({
    mutationFn: (data: Proxy) => APIClient.proxy.test(data),
    onError: (err) => {
      console.error(err);
    }
  });

  const testProxy = (data: unknown) => testMutation.mutate(data as Proxy);

  const initialValues: ProxyCreate = {
    enabled: true,
    name: "Proxy",
    type: "SOCKS5",
    addr: "socks5://ip:port",
    user: "",
    pass: "",
  }

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
                  initialValues={initialValues}
                  onSubmit={onSubmit}
                >
                  {({ values }) => (
                    <Form className="h-full flex flex-col bg-white dark:bg-gray-800 shadow-xl overflow-y-auto">
                      <div className="flex-1">
                        <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
                          <div className="flex items-start justify-between space-x-3">
                            <div className="space-y-1">
                              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
                                Add proxy
                              </DialogTitle>
                              <p className="text-sm text-gray-500 dark:text-gray-200">
                                Add proxy to be used with Indexers or IRC.
                              </p>
                            </div>
                            <div className="h-7 flex items-center">
                              <button
                                type="button"
                                className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                                onClick={toggle}
                              >
                                <span className="sr-only">Close panel</span>
                                <XMarkIcon className="h-6 w-6" aria-hidden="true" />
                              </button>
                            </div>
                          </div>
                        </div>

                        <div className="py-6 space-y-4 divide-y divide-gray-200 dark:divide-gray-700">
                          <SwitchGroupWide name="enabled" label="Enabled" />
                          <TextFieldWide name="name" label="Name" defaultValue="" required={true} />

                          <SelectFieldBasic
                            name="type"
                            label="Proxy type"
                            options={ProxyTypeOptions}
                            tooltip={<span>Proxy type. Commonly SOCKS5.</span>}
                            help="Usually SOCKS5"
                          />

                          <TextFieldWide name="addr" label="Addr" required={true} help="Addr: scheme://ip:port or scheme://domain" autoComplete="off"/>
                        </div>

                        <div>
                          <TextFieldWide name="user" label="User" help="auth: username" autoComplete="off" />
                          <PasswordFieldWide name="pass" label="Pass" help="auth: password" autoComplete="off"/>
                        </div>
                      </div>

                      <div
                        className="flex-shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-5 sm:px-6">
                        <div className="space-x-3 flex justify-end">
                          <button
                            type="button"
                            className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                            onClick={() => testProxy(values)}
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


interface UpdateFormProps<T> {
  isOpen: boolean;
  toggle: () => void;
  data: T;
}

export function ProxyUpdateForm({ isOpen, toggle, data }: UpdateFormProps<Proxy>) {
  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: (req: Proxy) => APIClient.proxy.update(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ProxyKeys.lists() });

      toast.custom((t) => <Toast type="success" body={`Proxy ${data.name} updated!`} t={t} />);
      toggle();
    },
    onError: () => {
      toast.custom((t) => <Toast type="error" body="Proxy could not be updated" t={t} />);
    }
  });

  const onSubmit = (formData: unknown) => {
    updateMutation.mutate(formData as Proxy);
  }

  const deleteMutation = useMutation({
    mutationFn: (proxyId: number) => APIClient.proxy.delete(proxyId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ProxyKeys.lists() });

      toast.custom((t) => <Toast type="success" body={`Proxy ${data.name} was deleted.`} t={t}/>);
    }
  });

  const deleteFn = () => deleteMutation.mutate(data.id);

  const testMutation = useMutation({
    mutationFn: (data: Proxy) => APIClient.proxy.test(data),
    onError: (err) => {
      console.error(err);
    }
  });

  const testProxy = (data: unknown) => testMutation.mutate(data as Proxy);

  const initialValues: Proxy = {
    id: data.id,
    enabled: data.enabled,
    name: data.name,
    type: data.type,
    addr: data.addr,
    user: data.user,
    pass: data.pass,
  }

  return (
    <SlideOver<Proxy>
      title="Proxy"
      initialValues={initialValues}
      onSubmit={onSubmit}
      deleteAction={deleteFn}
      testFn={testProxy}
      isOpen={isOpen}
      toggle={toggle}
      type="UPDATE"
    >
      {() => (
        <div>
          <div className="py-6 space-y-4 divide-y divide-gray-200 dark:divide-gray-700">

            <SwitchGroupWide name="enabled" label="Enabled"/>

            <TextFieldWide name="name" label="Name" defaultValue="" required={true}/>

            <SelectFieldBasic
              name="type"
              label="Proxy type"
              required={true}
              options={ProxyTypeOptions}
              tooltip={<span>Proxy type. Commonly SOCKS5.</span>}
              help="Usually SOCKS5"
            />

            <TextFieldWide name="addr" label="Addr" required={true} help="Addr: scheme://ip:port or scheme://domain" autoComplete="off"/>
          </div>

          <div>
            <TextFieldWide name="user" label="User" help="auth: username" autoComplete="off"/>
            <PasswordFieldWide name="pass" label="Pass" help="auth: password" autoComplete="off"/>
          </div>
        </div>
      )}
    </SlideOver>
  );
}
