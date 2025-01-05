/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Switch, Field, Label, Description } from "@headlessui/react";
import { classNames } from "@utils";

interface CheckboxProps {
  value: boolean;
  setValue: (newValue: boolean) => void;
  label?: string;
  description?: string;
  className?: string;
  disabled?: boolean;
}

export const Checkbox = ({
  label,
  description,
  value,
  className,
  setValue,
  disabled
}: CheckboxProps) => (
  <Field
    as="div"
    className={classNames(className ?? "py-2", "flex items-center justify-between")}
    onClick={(e) => {
      e.stopPropagation();
      e.nativeEvent.stopImmediatePropagation();
    }}
  >
    {(label || description) ? (
      <div className="flex flex-col mr-4">
        {label ? (
          <Label as="p" className="text-sm font-medium whitespace-nowrap text-gray-900 dark:text-white" passive>
            {label}
          </Label>
        ) : null}
        {description ? (
          <Description className="text-sm text-gray-500 dark:text-gray-400">
            {description}
          </Description>
        ) : null}
      </div>
    ) : null}
    <Switch
      checked={value}
      onChange={(newValue) => {
        !disabled && setValue(newValue);
      }}
      className={classNames(
        disabled
          ? "cursor-not-allowed bg-gray-450 dark:bg-gray-700 border-gray-375 dark:border-gray-800"
          : (
            value
              ? "cursor-pointer bg-blue-600 border-blue-525"
              : "cursor-pointer bg-gray-300 dark:bg-gray-700 border-gray-375 dark:border-gray-600"
          ),
        "border relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors"
      )}
    >
      <span
        className={classNames(
          value ? "translate-x-6" : "translate-x-[0.15rem]",
          disabled ? "bg-gray-650 dark:bg-gray-800" : "bg-white",
          "inline-flex items-center align-center h-4 w-4 transform rounded-full transition ring-0 shadow"
        )}
      >
        {value
          ? (
            <svg className={classNames(
              disabled ? "text-white dark:text-gray-300" : "text-blue-500", "w-4 h-4"
            )} fill="currentColor" viewBox="0 0 12 12"><path d="M3.707 5.293a1 1 0 00-1.414 1.414l1.414-1.414zM5 8l-.707.707a1 1 0 001.414 0L5 8zm4.707-3.293a1 1 0 00-1.414-1.414l1.414 1.414zm-7.414 2l2 2 1.414-1.414-2-2-1.414 1.414zm3.414 2l4-4-1.414-1.414-4 4 1.414 1.414z"></path></svg>
          )
          : (
            <svg className={classNames(
              disabled ? "text-white dark:text-gray-300" : "text-gray-600", "w-4 h-4"
            )} fill="none" viewBox="0 0 12 12"><path d="M4 8l2-2m0 0l2-2M6 6L4 4m2 2l2 2" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"></path></svg>
          )}
      </span>
    </Switch>
  </Field>
);
