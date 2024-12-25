/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { FC } from "react";
import { CheckCircleIcon, ExclamationCircleIcon, ExclamationTriangleIcon, InformationCircleIcon, XMarkIcon } from "@heroicons/react/24/solid";
import { toast, Toast as Tooast } from "@components/hot-toast";
import { classNames } from "@utils";

type Props = {
  type: "error" | "success" | "warning" | "info";
  body?: string
  t?: Tooast;
};

const Toast: FC<Props> = ({ type, body, t }) => (
  <div
    className={classNames(
      t?.visible ? "animate-enter" : "animate-leave",
      "max-w-sm w-full bg-white dark:bg-gray-800 whitespace-pre-wrap shadow-2xl rounded-lg pointer-events-auto border border-gray-250 dark:border-gray-775 overflow-hidden transition-all"
    )}
  >
    <div className="p-4">
      <div className="flex items-start">
        <div className="flex-shrink-0">
          {type === "success" && <CheckCircleIcon className="h-6 w-6 text-green-400" aria-hidden="true" />}
          {type === "error" && <ExclamationCircleIcon className="h-6 w-6 text-red-400" aria-hidden="true" />}
          {type === "warning" && <ExclamationTriangleIcon className="h-6 w-6 text-yellow-400" aria-hidden="true" />}
          {type === "info" && <InformationCircleIcon className="h-6 w-6 text-blue-400" aria-hidden="true" />}
        </div>
        <div className="ml-3 w-0 flex-1 pt-0.5">
          <p className="text-sm font-medium text-gray-900 dark:text-gray-200">
            {type === "success" && "Success"}
            {type === "error" && "Error"}
            {type === "warning" && "Warning"}
            {type === "info" && "Info"}
          </p>
          <span className="mt-1 text-sm text-gray-500 dark:text-gray-400">{body}</span>
        </div>
        <div className="ml-4 flex-shrink-0 flex">
          <button
            className="bg-white dark:bg-gray-700 rounded-md inline-flex text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
            onClick={() => {
              toast.dismiss(t?.id);
            }}
          >
            <span className="sr-only">Close</span>
            <XMarkIcon className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>
      </div>
    </div>
  </div>
);

export default Toast;
