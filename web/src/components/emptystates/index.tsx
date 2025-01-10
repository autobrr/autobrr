/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { PlusIcon } from "@heroicons/react/24/solid";

interface EmptySimpleProps {
    title: string;
    subtitle?: string;
    buttonText?: string;
    buttonAction?: () => void;
}

export const EmptySimple = ({
  title,
  subtitle,
  buttonText,
  buttonAction
}: EmptySimpleProps) => (
  <div className="text-center py-8">
    <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">{title}</h3>
    {subtitle ? (
      <p className="mt-1 text-sm text-gray-500 dark:text-gray-200">{subtitle}</p>
    ) : null}
    {buttonText && buttonAction ? (
      <div className="mt-6">
        <button
          type="button"
          onClick={buttonAction}
          className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
        >
          <PlusIcon className="-ml-1 mr-2 h-5 w-5" aria-hidden="true" />
          {buttonText}
        </button>
      </div>
    ) : null}
  </div>
);

interface EmptyListStateProps {
    text: string;
    buttonText?: string;
    buttonOnClick?: () => void;
}

export function EmptyListState({ text, buttonText, buttonOnClick }: EmptyListStateProps) {
  return (
    <div className="px-4 py-12 flex flex-col items-center">
      <p className="text-center text-gray-800 dark:text-white">{text}</p>
      {buttonText && buttonOnClick && (
        <button
          type="button"
          className="relative inline-flex items-center px-4 py-2 mt-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
          onClick={buttonOnClick}
        >
          {buttonText}
        </button>
      )}
    </div>
  );
}
