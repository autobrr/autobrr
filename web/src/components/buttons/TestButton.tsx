/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useTranslation } from "react-i18next";

import type { TestStatus } from "@hooks/useTestAction";
import { classNames } from "@utils";

interface TestButtonProps {
  status: TestStatus;
  onClick: () => void;
  label?: string;
}

export function TestButton({ status, onClick, label }: TestButtonProps) {
  const { t } = useTranslation("settings");
  const isTesting = status === "testing";

  return (
    <button
      type="button"
      className={classNames(
        status === "passed"
          ? "text-green-500 border-green-500 bg-green-50"
          : status === "failed"
            ? "text-red-500 border-red-500 bg-red-50"
            : "border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 focus:border-rose-700 active:bg-rose-700",
        isTesting ? "cursor-not-allowed" : "cursor-pointer",
        "mr-2 inline-flex items-center px-4 py-2 border font-medium rounded-md shadow-xs text-sm transition ease-in-out duration-150 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
      )}
      disabled={isTesting}
      aria-busy={isTesting}
      onClick={onClick}
    >
      {isTesting ? (
        <svg
          className="animate-spin h-5 w-5 text-green-500"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          role="img"
          aria-label={t("panel.testing")}
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
      ) : status === "passed" ? (
        t("panel.ok")
      ) : status === "failed" ? (
        t("panel.error")
      ) : (
        label ?? t("panel.test")
      )}
    </button>
  );
}
