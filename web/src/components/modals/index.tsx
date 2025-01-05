/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */


import { FC, Fragment, MutableRefObject, useState } from "react";
import { Dialog, DialogPanel, DialogTitle, Transition, TransitionChild } from "@headlessui/react";
import { ExclamationTriangleIcon } from "@heroicons/react/24/solid";

import { RingResizeSpinner } from "@components/Icons";

interface ModalUpperProps {
  title: string;
  text: string;
}

interface ModalLowerProps {
  isOpen: boolean;
  isLoading: boolean;
  toggle: () => void;
  deleteAction?: () => void;
  forceRunAction?: () => void;
}

interface DeleteModalProps extends ModalUpperProps, ModalLowerProps {
  buttonRef: MutableRefObject<HTMLElement | null> | undefined;
}

interface ForceRunModalProps {
  isOpen: boolean;
  isLoading: boolean;
  toggle: () => void;
  buttonRef: MutableRefObject<HTMLElement | null> | undefined;
  forceRunAction: () => void;
  title: string;
  text: string;
}

const ModalUpper = ({ title, text }: ModalUpperProps) => (
  <div className="bg-white dark:bg-gray-800 px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
    <div className="sm:flex sm:items-start">
      <ExclamationTriangleIcon className="h-16 w-16 text-red-500 dark:text-red-500" aria-hidden="true" />
      <div className="mt-3 text-left sm:mt-0 sm:ml-4 sm:pr-8 max-w-full">
        <DialogTitle as="h3" className="text-lg leading-6 font-medium text-gray-900 dark:text-white break-words">
          {title}
        </DialogTitle>
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
      <RingResizeSpinner className="text-blue-500 size-6" />
    ) : (
      <>
        <button
          type="button"
          className="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 bg-red-600 text-base font-medium text-white hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:ml-3 sm:w-auto sm:text-sm"
          onClick={(e) => {
            e.preventDefault();
            if (isOpen) {
              deleteAction?.();
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
  <Transition show={props.isOpen} as={Fragment}>
    <Dialog
      as="div"
      static
      className="fixed z-10 inset-0 overflow-y-auto bg-gray-700/60 dark:bg-black/60 transition-opacity"
      initialFocus={props.buttonRef}
      open={props.isOpen}
      onClose={props.toggle}
    >
      <div className="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
        <span className="hidden sm:inline-block sm:align-middle sm:h-screen" aria-hidden="true">
          &#8203;
        </span>
        <TransitionChild
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
          enterTo="opacity-100 translate-y-0 sm:scale-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100 translate-y-0 sm:scale-100"
          leaveTo="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
        >
          <DialogPanel className="inline-block align-bottom rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
            <ModalUpper {...props} />
            <ModalLower {...props} />
          </DialogPanel>
        </TransitionChild>
      </div>
    </Dialog>
  </Transition>
);

export const ForceRunModal: FC<ForceRunModalProps> = (props: ForceRunModalProps) => {
  const [inputValue, setInputValue] = useState("");
  const isInputCorrect = inputValue.trim().toLowerCase() === "i understand";

  // A function to reset the input and handle any necessary cleanup
  const resetAndClose = () => {
    setInputValue("");
    props.toggle();
  };

  // The handleClose function will be passed to the onClose prop of the Dialog
  const handleClose = () => {
    setTimeout(() => {
      resetAndClose();
    }, 200);
  };

  const handleForceRun = (e: React.SyntheticEvent) => {
    e.preventDefault();
    if (props.isOpen && isInputCorrect) {
      props.forceRunAction();
      props.toggle();
      // Delay the reset of the input until after the transition finishes
      setTimeout(() => {
        setInputValue("");
      }, 400);
    }
  };
  
  // When the 'Cancel' button is clicked
  const handleCancel = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.preventDefault();
    resetAndClose();
  };

  return (
    <Transition show={props.isOpen} as={Fragment}>
      <Dialog
        as="div"
        static
        className="fixed z-10 inset-0 overflow-y-auto"
        open={props.isOpen}
        onClose={handleClose}
      >
        <div className="grid place-items-center min-h-screen">
          <TransitionChild
            as={Fragment}
            enter="ease-out duration-300"
            enterFrom="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
            enterTo="opacity-100 translate-y-0 sm:scale-100"
            leave="ease-in duration-200"
            leaveFrom="opacity-100 translate-y-0 sm:scale-100"
            leaveTo="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
          >
            <DialogPanel className="inline-block align-bottom border border-transparent dark:border-gray-700 rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
              <ModalUpper title={props.title} text={props.text} />
              
              <div className="bg-gray-50 dark:bg-gray-800 px-4 py-3 sm:px-6 flex justify-center">
                <input
                  type="text"
                  data-autofocus
                  className="w-96 shadow-sm sm:text-sm rounded-md border py-2.5 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-400 dark:border-gray-700 bg-gray-100 dark:bg-gray-900 dark:text-gray-100"
                  placeholder="Type 'I understand' to enable the button"
                  value={inputValue}
                  onChange={(e) => setInputValue(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      handleForceRun(e);
                    }
                  }}
                />
              </div>

              <div className="bg-gray-50 dark:bg-gray-800 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
                {props.isLoading ? (
                  <RingResizeSpinner className="text-blue-500 size-6" />
                ) : (
                  <>
                    <button
                      type="button"
                      disabled={!isInputCorrect}
                      className={`w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 ${
                        isInputCorrect ? "bg-red-600 text-white hover:bg-red-700" : "bg-gray-300"
                      } text-base font-medium focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 sm:ml-3 sm:w-auto sm:text-sm`}
                      onClick={handleForceRun}
                    >
                      Force Run
                    </button>
                    <button
                      type="button"
                      className="mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 dark:border-gray-600 shadow-sm px-4 py-2 bg-white dark:bg-gray-700 text-base font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 sm:mt-0 sm:ml-3 sm:w-auto sm:text-sm"
                      onClick={handleCancel}
                    >
                      Cancel
                    </button>
                  </>
                )}
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </Dialog>
    </Transition>
  );
};
