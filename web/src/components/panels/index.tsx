/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment, useRef, ReactNode, ReactElement } from "react";
import { XMarkIcon } from "@heroicons/react/24/solid";
import { Dialog, DialogPanel, DialogTitle, Transition, TransitionChild } from "@headlessui/react";
import { Form, Formik } from "formik";
import type { FormikValues, FormikProps } from "formik";

import { DEBUG } from "@components/debug";
import { useToggle } from "@hooks/hooks";
import { DeleteModal } from "@components/modals";
import { classNames } from "@utils";

interface SlideOverProps<DataType> {
  title: string;
  initialValues: DataType;
  validate?: (values: DataType) => void;
  onSubmit: (values?: DataType) => void;
  isOpen: boolean;
  toggle: () => void;
  children?: (values: DataType) => ReactNode;
  deleteAction?: () => void;
  type: "CREATE" | "UPDATE";
  testFn?: (data: unknown) => void;
  isTesting?: boolean;
  isTestSuccessful?: boolean;
  isTestError?: boolean;
  extraButtons?: (values: DataType) => ReactNode;
}

function SlideOver<DataType extends FormikValues>({
  title,
  initialValues,
  validate,
  onSubmit,
  deleteAction,
  isOpen,
  toggle,
  type,
  children,
  testFn,
  isTesting,
  isTestSuccessful,
  isTestError,
  extraButtons
}: SlideOverProps<DataType>): ReactElement {
  const cancelModalButtonRef = useRef<HTMLInputElement | null>(null);
  const formRef = useRef<FormikProps<DataType>>(null);

  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);

  return (
    <Transition show={isOpen} as={Fragment}>
      <Dialog as="div" static className="fixed inset-0 overflow-hidden" open={isOpen} onClose={toggle}>
        {deleteAction && (
          <DeleteModal
            isOpen={deleteModalIsOpen}
            isLoading={isTesting || false}
            toggle={toggleDeleteModal}
            buttonRef={cancelModalButtonRef}
            deleteAction={deleteAction}
            title={`Remove ${title}`}
            text={`Are you sure you want to remove this ${title}? This action cannot be undone.`}
          />
        )}

        <div className="absolute inset-0 overflow-hidden">
          <DialogPanel
            className="fixed inset-y-0 right-0 max-w-full flex"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
            }}
          >
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
                  initialValues={initialValues}
                  onSubmit={onSubmit}
                  validate={validate}
                  innerRef={formRef}
                >
                  {({ handleSubmit, values }) => (
                    <Form
                      className="h-full flex flex-col bg-white dark:bg-gray-800 shadow-xl overflow-y-auto"
                      onSubmit={(e) => {
                        e.preventDefault();
                        handleSubmit(e);
                      }}
                    >

                      <div className="flex-1">
                        <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
                          <div className="flex items-start justify-between space-x-3">
                            <div className="space-y-1">
                              <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">{type === "CREATE" ? "Create" : "Update"} {title}</DialogTitle>
                              <p className="text-sm text-gray-500 dark:text-gray-400">
                                {type === "CREATE" ? "Create" : "Update"} {title}.
                              </p>
                            </div>
                            <div className="h-7 flex items-center">
                              <button
                                type="button"
                                className="bg-white dark:bg-gray-900 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                                onClick={toggle}
                              >
                                <span className="sr-only">Close panel</span>
                                <XMarkIcon className="h-6 w-6" aria-hidden="true" />
                              </button>
                            </div>
                          </div>
                        </div>

                        {!!values && children !== undefined ? (
                          children(values)
                        ) : null}
                      </div>

                      <div className="shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-5 sm:px-6">
                        <div className={classNames(type === "CREATE" ? "justify-end" : "justify-between", "space-x-3 flex")}>
                          {type === "UPDATE" && (
                            <button
                              type="button"
                              className="inline-flex items-center justify-center px-4 py-2 border border-transparent font-medium rounded-md text-red-700 dark:text-white bg-red-100 dark:bg-red-700 hover:bg-red-200 dark:hover:bg-red-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:text-sm"
                              onClick={toggleDeleteModal}
                            >
                              Remove
                            </button>
                          )}
                          <div className="flex">
                            {!!values && extraButtons !== undefined && (
                              extraButtons(values)
                            )}

                            {testFn && (
                              <button
                                type="button"
                                className={classNames(
                                  isTestSuccessful
                                    ? "text-green-500 border-green-500 bg-green-50"
                                    : isTestError
                                      ? "text-red-500 border-red-500 bg-red-50"
                                      : "border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 focus:border-rose-700 active:bg-rose-700",
                                  isTesting ? "cursor-not-allowed" : "",
                                  "mr-2 inline-flex items-center px-4 py-2 border font-medium rounded-md shadow-xs text-sm transition ease-in-out duration-150 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                                )}
                                disabled={isTesting}
                                onClick={(e) => {
                                  e.preventDefault();
                                  testFn(values);
                                }}
                              >
                                {isTesting ? (
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
                                ) : isTestSuccessful ? (
                                  "OK!"
                                ) : isTestError ? (
                                  "ERROR"
                                ) : (
                                  "Test"
                                )}
                              </button>
                            )}

                            <button
                              type="button"
                              className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                              onClick={(e) => {
                                e.preventDefault();
                                toggle();
                              }}
                            >
                              Cancel
                            </button>
                            <button
                              type="button"
                              className="ml-4 inline-flex justify-center py-2 px-4 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                              onClick={(e) => {
                                e.preventDefault();
                                formRef.current?.submitForm();
                              }}
                            >
                              {type === "CREATE" ? "Create" : "Save"}
                            </button>
                          </div>
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

export { SlideOver };
