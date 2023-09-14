/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { FC, Fragment, MutableRefObject } from "react";
import { Dialog, Transition } from "@headlessui/react";
import { ExclamationTriangleIcon } from "@heroicons/react/24/solid";

import { SectionLoader } from "@components/SectionLoader";

interface ModalUpperProps {
  title: string;
  text: string;
}

interface ModalLowerProps {
  isOpen: boolean;
  isLoading: boolean;
  toggle: () => void;
  deleteAction: () => void;
}

interface DeleteModalProps extends ModalUpperProps, ModalLowerProps {
  buttonRef: MutableRefObject<HTMLElement | null> | undefined;
}

const ModalUpper = ({ title, text }: ModalUpperProps) => (
  <div className="bg-white dark:bg-gray-800 px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
    <div className="sm:flex sm:items-start">
      <ExclamationTriangleIcon className="h-16 w-16 text-red-500 dark:text-red-500" aria-hidden="true" />
      <div className="mt-3 text-center sm:mt-0 sm:ml-4 sm:pr-8 sm:text-left max-w-full">
        <Dialog.Title as="h3" className="text-lg leading-6 font-medium text-gray-900 dark:text-white break-words">
          {title}
        </Dialog.Title>
        <div className="mt-2">
          <p className="text-sm text-gray-500 dark:text-gray-300">
            {text}
          </p>
        </div>
      </div>
    </div>
  </div>
);

const ModalLower = ({ isOpen, isLoading, toggle, deleteAction }: ModalLowerProps) => (
  <div className="bg-gray-50 dark:bg-gray-800 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
    {isLoading ? (
      <SectionLoader $size="small" />
    ) : (
      <>
        <button
          type="button"
          className="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 bg-red-600 text-base font-medium text-white hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:ml-3 sm:w-auto sm:text-sm"
          onClick={(e) => {
            e.preventDefault();
            if (isOpen) {
              deleteAction();
              toggle();
            }
          }}
        >
          Remove
        </button>
        <button
          type="button"
          className="mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 dark:border-gray-600 shadow-sm px-4 py-2 bg-white dark:bg-gray-700 text-base font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 sm:mt-0 sm:ml-3 sm:w-auto sm:text-sm"
          onClick={(e) => {
            e.preventDefault();
            toggle();
          }}
        >
          Cancel
        </button>
      </>
    )}
  </div>
);

export const DeleteModal: FC<DeleteModalProps> = (props: DeleteModalProps) => (
  <Transition.Root show={props.isOpen} as={Fragment}>
    <Dialog
      as="div"
      static
      className="fixed z-10 inset-0 overflow-y-auto"
      initialFocus={props.buttonRef}
      open={props.isOpen}
      onClose={props.toggle}
    >
      <div className="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <Dialog.Overlay className="fixed inset-0 bg-gray-700/60 dark:bg-black/60 transition-opacity" />
        </Transition.Child>

        <span className="hidden sm:inline-block sm:align-middle sm:h-screen" aria-hidden="true">
          &#8203;
        </span>
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
          enterTo="opacity-100 translate-y-0 sm:scale-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100 translate-y-0 sm:scale-100"
          leaveTo="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
        >
          <div className="inline-block align-bottom rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
            <ModalUpper {...props} />
            <ModalLower {...props} />
          </div>
        </Transition.Child>
      </div>
    </Dialog>
  </Transition.Root>
);
