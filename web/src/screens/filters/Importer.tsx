/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment, useRef, useState } from "react";
import { Dialog, DialogPanel, DialogTitle, Transition, TransitionChild } from "@headlessui/react";
import { useQueryClient } from "@tanstack/react-query";

import { APIClient } from "@api/APIClient";
import { FilterKeys } from "@api/query_keys";
import toast from "@components/hot-toast";
import Toast from "@components/notifications/Toast";

import { AutodlIrssiConfigParser } from "./_configParser";
import { ExclamationTriangleIcon } from "@heroicons/react/24/outline";

interface ImporterProps {
  isOpen: boolean;
  setIsOpen: (newState: boolean) => void;
}

interface ModalLowerProps extends ImporterProps {
  onImportClick: () => void;
}

const ModalUpper = ({ children }: { children: React.ReactNode; }) => (
  <div className="bg-white dark:bg-gray-800 px-4 pt-5 pb-4 sm:py-6 sm:px-4 sm:pb-4">
    <div className="mt-3 text-left sm:mt-0 max-w-full">
      <DialogTitle as="h3" className="mb-3 text-lg leading-6 font-medium text-gray-900 dark:text-white break-words">
        Import filter (in JSON or autodl-irssi format)
      </DialogTitle>
      {children}
    </div>
  </div>
);

const ModalLower = ({ isOpen, setIsOpen, onImportClick }: ModalLowerProps) => (
  <div className="bg-gray-50 dark:bg-gray-800 border-t border-gray-300 dark:border-gray-700 px-4 py-3 sm:px-4 sm:flex sm:flex-row-reverse">
    <button
      type="button"
      className="w-full inline-flex justify-center rounded-md border border-blue-500 shadow-xs px-4 py-2 bg-blue-700 text-base font-medium text-white hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 sm:ml-3 sm:w-auto sm:text-sm"
      onClick={(e) => {
        e.preventDefault();
        if (isOpen) {
          onImportClick();
          setIsOpen(false);
        }
      }}
    >
      Import
    </button>
    <button
      type="button"
      className="mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 dark:border-gray-600 shadow-xs px-4 py-2 bg-white dark:bg-gray-700 text-base font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 sm:mt-0 sm:ml-3 sm:w-auto sm:text-sm"
      onClick={(e) => {
        e.preventDefault();
        setIsOpen(false);
      }}
    >
      Cancel
    </button>
  </div>
);

const ImportJSON = async (inputFilterText: string) => {
  let newFilter = {} as Filter;
  try {
    const importedData = JSON.parse(inputFilterText);

    // Fetch existing filters from the API
    const existingFilters = await APIClient.filters.getAll();

    // Create a unique filter title by appending an incremental number if title is taken by another filter
    let nameCounter = 0;
    let uniqueFilterName = importedData.name;
    while (existingFilters.some((filter) => filter.name === uniqueFilterName)) {
      nameCounter++;
      uniqueFilterName = `${importedData.name}-${nameCounter}`;
    }

    // Create a new filter using the API
    newFilter = {
      resolutions: [],
      sources: [],
      codecs: [],
      containers: [],
      ...importedData.data,
      name: uniqueFilterName
    } as Filter;

    await APIClient.filters.create(newFilter);

    toast.custom((t) =>
      <Toast
        type="success"
        body={`Filter '${uniqueFilterName}' imported successfully!`}
        t={t}
      />
    );
  } catch (e) {
    console.error("Failure while importing JSON filter: ", e);
    console.error("  --> Filter: ", newFilter);

    toast.custom((t) =>
      <Toast
        type="error"
        body="Failed to import JSON data. Information logged to console."
        t={t}
      />
    );
  }
};

const ImportAutodlIrssi = async (inputText: string) => {
  const parser = new AutodlIrssiConfigParser();
  parser.Parse(inputText);

  if (!parser.releaseFilters) {
    toast.custom((t) =>
      <Toast
        type="warning"
        body="Cannot import given filter -- it is neither in JSON or autodl-irssi format."
        t={t}
      />
    );
    return;
  }

  let numSuccess = 0;
  for (const filter of parser.releaseFilters) {
    try {
      await APIClient.filters.create(filter.values as unknown as Filter);
      ++numSuccess;
    } catch (e) {
      console.error(`Failed to import autodl-irssi filter '${filter.name}': `, e);
      console.error("  --> Filter: ", filter);

      toast.custom((t) =>
        <Toast
          type="error"
          body={`Failed to import filter autodl-irssi filter '${filter.name}'. Information logged to console.`}
          t={t}
        />
      );
    }
  }

  if (numSuccess === parser.releaseFilters.length) {
    toast.custom((t) =>
      <Toast
        type="success"
        body={
          numSuccess === 1
            ? `Filter '${parser.releaseFilters[0].name}' imported successfully!`
            : `All ${numSuccess} filters imported successfully!`
        }
        t={t}
      />
    );
  } else {
    toast.custom((t) =>
      <Toast
        type="info"
        body={`${numSuccess}/${parser.releaseFilters.length} filters imported successfully. See console for details.`}
        t={t}
      />
    );
  }
};

export const Importer = ({
  isOpen,
  setIsOpen
}: ImporterProps) => {
  const textAreaRef = useRef<HTMLTextAreaElement>(null);

  const [inputFilterText, setInputFilterText] = useState("");
  const [parserWarnings, setParserWarnings] = useState<string[]>([]);

  const queryClient = useQueryClient();

  const isJSON = (inputText: string) => (
    inputText.indexOf("{") <= 3 && inputText.lastIndexOf("}") >= (inputText.length - 3 - 1)
  );

  const showAutodlImportWarnings = (inputText: string) => {
    inputText = inputText.trim();

    if (isJSON(inputText)) {
      // If it's JSON, don't do anything
      return setParserWarnings([]);
    } else {
      const parser = new AutodlIrssiConfigParser();
      parser.Parse(inputText);

      setParserWarnings(parser.GetWarnings());
    }
  };

  // This function handles the import of a filter from a JSON string
  const handleImportJson = async () => {
    try {
      const inputText = inputFilterText.trim();

      if (isJSON(inputText)) {
        console.log("Parsing import filter as JSON");
        await ImportJSON(inputText);
      } else {
        console.log("Parsing import filter in autodl-irssi format");
        await ImportAutodlIrssi(inputText);
      }
    } catch (error) {
      // This should never be called
      console.error("Critical error while importing filter: ", error);
    } finally {
      setIsOpen(false);
      // Invalidate filter cache, and trigger refresh request
      await queryClient.invalidateQueries({ queryKey: FilterKeys.lists() });
    }
  };

  return (
    <Transition show={isOpen} as={Fragment}>
      <Dialog
        as="div"
        static
        className="fixed z-10 inset-0 overflow-y-auto bg-gray-700/60 dark:bg-black/60 transition-opacity"
        initialFocus={textAreaRef}
        open={isOpen}
        onClose={() => setIsOpen(false)}
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
            <DialogPanel className="inline-block align-bottom border border-transparent dark:border-gray-700 rounded-lg text-left overflow-hidden shadow-xl transform transition sm:my-8 sm:align-middle w-full sm:max-w-6xl">
              <ModalUpper>
                <textarea
                  className="form-input resize-y block w-full shadow-xs sm:text-sm rounded-md border py-2.5 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-gray-100"
                  placeholder="Paste your filter data here (either autobrr JSON format or your entire autodl-irssi config)"
                  value={inputFilterText}
                  onChange={(event) => {
                    const inputText = event.target.value;
                    showAutodlImportWarnings(inputText);
                    setInputFilterText(inputText);
                  }}
                  style={{ minHeight: "30vh", maxHeight: "50vh" }}
                />
                {parserWarnings.length ? (
                  <>
                    <h4 className="flex flex-row items-center gap-1 text-base text-black dark:text-white mt-2 mb-1">
                      <ExclamationTriangleIcon
                        className="h-6 w-6 text-amber-500 dark:text-yellow-500"
                        aria-hidden="true"
                      />
                      Import Warnings
                    </h4>

                    <div className="overflow-y-auto pl-2 pr-1 py-1 rounded-lg min-w-full border border-gray-200 dark:border-gray-700 bg-gray-100 dark:bg-gray-900 text-gray-800 dark:text-gray-400">
                      {parserWarnings.map((line, idx) => (
                        <p key={`parser-warning-${idx}`}>{line}</p>
                      ))}
                    </div>
                  </>
                ) : null}
              </ModalUpper>
              <ModalLower isOpen={isOpen} setIsOpen={setIsOpen} onImportClick={handleImportJson} />
            </DialogPanel>
          </TransitionChild>
        </div>
      </Dialog>
    </Transition>
  );
};
