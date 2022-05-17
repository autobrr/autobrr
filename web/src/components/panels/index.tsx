import React, { Fragment, useRef } from "react";
import { XIcon } from "@heroicons/react/solid";
import { Dialog, Transition } from "@headlessui/react";
import { Form, Formik } from "formik";
import DEBUG from "../debug";
import { useToggle } from "../../hooks/hooks";
import { DeleteModal } from "../modals";
import { classNames } from "../../utils";

interface SlideOverProps<DataType> {
    title: string;
    initialValues: DataType;
    validate?: (values: DataType) => void;
    onSubmit: (values?: DataType) => void;
    isOpen: boolean;
    toggle: () => void;
    children?: (values: DataType) => React.ReactNode;
    deleteAction?: () => void;
    type: "CREATE" | "UPDATE";
}

function SlideOver<DataType>({
  title,
  initialValues,
  validate,
  onSubmit,
  deleteAction,
  isOpen,
  toggle,
  type,
  children
}: SlideOverProps<DataType>): React.ReactElement {
  const cancelModalButtonRef = useRef<HTMLInputElement | null>(null);
  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);

  return (
    <Transition.Root show={isOpen} as={Fragment}>
      <Dialog as="div" static className="fixed inset-0 overflow-hidden" open={isOpen} onClose={toggle}>
        {deleteAction && (
          <DeleteModal
            isOpen={deleteModalIsOpen}
            toggle={toggleDeleteModal}
            buttonRef={cancelModalButtonRef}
            deleteAction={deleteAction}
            title={`Remove ${title}`}
            text={`Are you sure you want to remove this ${title}? This action cannot be undone.`}
          />
        )}

        <div className="absolute inset-0 overflow-hidden">
          <Dialog.Overlay className="absolute inset-0" />

          <div className="fixed inset-y-0 right-0 pl-10 max-w-full flex sm:pl-16">
            <Transition.Child
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
                >
                  {({ handleSubmit, values }) => ( 
                    <Form className="h-full flex flex-col bg-white dark:bg-gray-800 shadow-xl overflow-y-scroll"
                      onSubmit={handleSubmit}>

                      <div className="flex-1">
                        <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
                          <div className="flex items-start justify-between space-x-3">
                            <div className="space-y-1">
                              <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">{type === "CREATE" ? "Create" : "Update"} {title}</Dialog.Title>
                              <p className="text-sm text-gray-500 dark:text-gray-400">
                                {type === "CREATE" ? "Create" : "Update"} {title}.
                              </p>
                            </div>
                            <div className="h-7 flex items-center">
                              <button
                                type="button"
                                className="bg-white dark:bg-gray-900 rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                                onClick={toggle}
                              >
                                <span className="sr-only">Close panel</span>
                                <XIcon className="h-6 w-6" aria-hidden="true" />
                              </button>
                            </div>
                          </div>
                        </div>

                        {!!values && children !== undefined ? (
                          children(values)
                        ) : null}
                      </div>

                      <div className="flex-shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-5 sm:px-6">
                        <div className={classNames(type === "CREATE" ? "justify-end" : "justify-between", "space-x-3 flex")}>
                          {type === "UPDATE" && (
                            <button
                              type="button"
                              className="inline-flex items-center justify-center px-4 py-2 border border-transparent font-medium rounded-md text-red-700 dark:text-white bg-red-100 dark:bg-red-700 hover:bg-red-200 dark:hover:bg-red-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:text-sm"
                              onClick={toggleDeleteModal}
                            >
                                                                Remove
                            </button>
                          )}
                          <div>

                            <button
                              type="button"
                              className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                              onClick={toggle}
                            >
                                                                Cancel
                            </button>
                            <button
                              type="submit"
                              className="ml-4 inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 dark:bg-blue-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
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

            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition.Root>
  );
}

export { SlideOver };