/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { classNames } from "@utils";

interface ButtonProps {
    className?: string;
    children: React.ReactNode;
    disabled?: boolean;
    onClick?: () => void;
}

export const TableButton = ({ children, className, disabled, onClick }: ButtonProps) => (
  <button
    type="button"
    className={classNames(
      className ?? "",
      "relative inline-flex items-center px-4 py-2 border border-gray-300 dark:border-gray-800 text-sm font-medium rounded-md text-gray-700 dark:text-gray-500 bg-white dark:bg-gray-800 hover:bg-gray-50"
    )}
    disabled={disabled}
    onClick={onClick}
  >
    {children}
  </button>
);

export const TablePageButton = ({ children, className, disabled, onClick }: ButtonProps) => (
  <button
    type="button"
    className={classNames(
      className ?? "",
      disabled
        ? "cursor-not-allowed text-gray-500 dark:text-gray-500 border-gray-300 dark:border-gray-700 dark:bg-gray-800"
        : "cursor-pointer text-gray-500 dark:text-gray-350 border-gray-300 dark:border-gray-700 dark:bg-gray-850 hover:bg-gray-100 dark:hover:bg-gray-700",
      "inline-flex items-center p-1.5 border text-sm font-medium transition"
    )}
    disabled={disabled}
    onClick={onClick}
  >
    {children}
  </button>
);
