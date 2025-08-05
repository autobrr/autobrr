/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useToggle } from "@hooks/hooks";
import { CheckIcon, DocumentDuplicateIcon, EyeIcon, EyeSlashIcon } from "@heroicons/react/24/outline";
import { useState } from "react";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";

interface KeyFieldProps {
  value: string;
}

export const KeyField = ({ value }: KeyFieldProps) => {
  const [isVisible, toggleVisibility] = useToggle(false);
  const [isCopied, setIsCopied] = useState(false);

  async function copyTextToClipboard(text: string) {
    if ("clipboard" in navigator) {
      return await navigator.clipboard.writeText(text);
    } else {
      return document.execCommand("copy", true, text);
    }
  }

  // onClick handler function for the copy button
  const handleCopyClick = () => {
    // Asynchronously call copyTextToClipboard
    copyTextToClipboard(value)
      .then(() => {
        // If successful, update the isCopied state value
        setIsCopied(true);

        toast.custom(t => (
          <Toast
            type="success"
            body="API key copied to clipboard!"
            t={t}
          />
        ));

        setTimeout(() => {
          setIsCopied(false);
        }, 1500);
      })
      .catch((err) => {
        console.error(err);

        toast.custom(t => (
          <Toast
            type="error"
            body="Failed to copy API key."
            t={t}
          />
        ));
      });
  };

  return (
    <div className="sm:col-span-2 w-full">
      <div className="flex rounded-md shadow-xs">
        <div className="relative flex items-stretch grow focus-within:z-10">
          <input
            id="keyfield"
            type={isVisible ? "text" : "password"}
            value={value}
            readOnly={true}
            className="focus:outline-hidden dark:focus:border-blue-500 focus:border-blue-500 dark:focus:ring-blue-500 block w-full rounded-none rounded-l-md sm:text-sm border border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-gray-100"
          />
        </div>
        <button
          type="button"
          className="-ml-px relative inline-flex items-center space-x-2 px-4 py-2 border border-gray-300 dark:border-gray-700 hover:bg-gray-100  text-sm font-medium text-gray-700 bg-gray-50 dark:bg-gray-800 dark:hover:bg-gray-700  focus:outline-hidden"
          onClick={toggleVisibility}
          title="show"
        >
          {!isVisible ? <EyeIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" /> : <EyeSlashIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" />}
        </button>
        <button
          type="button"
          className="-ml-px relative inline-flex items-center space-x-2 px-4 py-2 border border-gray-300 dark:border-gray-700 hover:bg-gray-100  text-sm font-medium rounded-r-md text-gray-700 dark:text-gray-100 bg-gray-50 dark:bg-gray-800 dark:hover:bg-gray-700 focus:outline-hidden"
          onClick={handleCopyClick}
          title="Copy to clipboard"
        >
          {isCopied
            ? <CheckIcon
              className="text-blue-500 w-5 h-5"
              aria-hidden="true"
            />
            : <DocumentDuplicateIcon
              className="text-blue-500 w-5 h-5"
              aria-hidden="true"
            />
          }
        </button>
      </div>
    </div>
  );
};
