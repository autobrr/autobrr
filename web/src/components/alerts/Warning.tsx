/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { JSX } from "react";
import { classNames } from "@utils";

interface WarningAlertProps {
  text: string | JSX.Element;
  alert?: string;
  colors?: string;
  className?: string;
}

export const WarningAlert = ({ text, alert, colors, className }: WarningAlertProps) => (
  <div
    className={classNames(
      className ?? "",
      "col-span-12 flex items-center px-4 py-3 text-md font-medium rounded-lg",
      colors ?? "text-amber-800 bg-amber-100 border border-amber-700 dark:border-none dark:bg-amber-200 dark:text-amber-800"
    )}
    role="alert">
    <svg aria-hidden="true" className="flex-shrink-0 inline w-5 h-5 mr-3" fill="currentColor"
      viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
      <path fillRule="evenodd"
        d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
        clipRule="evenodd"></path>
    </svg>
    <span className="sr-only">Info</span>
    <div>
      <span className="font-extrabold">{alert ?? "Warning!"}</span>
      {" "}{text}
    </div>
  </div>
);
