/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { components } from "react-select";
import type {
  InputProps,
  ControlProps,
  MenuProps,
  OptionProps,
  IndicatorSeparatorProps,
  DropdownIndicatorProps
} from "react-select";

import { classNames } from "@utils";
import { useFieldContext } from "@app/lib/form";

interface ErrorFieldProps {
  classNames?: string;
}

export const ErrorField = ({ classNames }: ErrorFieldProps) => {
  const field = useFieldContext<unknown>();
  return field.state.meta.isTouched && field.state.meta.errors.length > 0
    ? <span className={classNames}>{field.state.meta.errors[0]}</span>
    : null;
};

interface RequiredFieldProps {
  required?: boolean
}

export const RequiredField = ({ required }: RequiredFieldProps) => (
  <>
    {required && <span className="ml-1 text-red-500">*</span>}
  </>
);

export const SelectInput = (props: InputProps) => (
  <components.Input
    {...props}
    inputClassName="outline-hidden border-none shadow-none focus:ring-transparent"
    className="text-gray-400! dark:text-gray-100!"
    children={props.children}
  />
);

export const SelectControl = (props: ControlProps) => (
  <components.Control
    {...props}
    className="p-1 block w-full bg-gray-100! dark:bg-gray-850! border border-gray-300 dark:border-gray-700! dark:hover:border-gray-600 rounded-md shadow-xs focus:outline-hidden focus:ring-blue-500 focus:border-blue-500 dark:text-gray-100 sm:text-sm"
    children={props.children}
  />
);

export const SelectMenu = (props: MenuProps) => (
  <components.Menu
    {...props}
    className="dark:bg-gray-800! border border-gray-300 dark:border-gray-700 dark:text-gray-400 rounded-md shadow-xs cursor-pointer"
    children={props.children}
  />
);

export const SelectOption = (props: OptionProps) => (
  <components.Option
    {...props}
    className={classNames(
      "transition dark:hover:bg-gray-900! dark:focus:bg-gray-900!",
      props.isSelected ? "dark:bg-gray-875! dark:text-gray-200" : "dark:bg-gray-800! dark:text-gray-400"
    )}
    children={props.children}
  />
);

export const IndicatorSeparator = (props: IndicatorSeparatorProps) => (
  <components.IndicatorSeparator
    {...props}
    className="bg-gray-400! dark:bg-gray-700!"
  />
);

export const DropdownIndicator = (props: DropdownIndicatorProps) => (
  <components.DropdownIndicator
    {...props}
    className="text-gray-400! dark:text-gray-300!"
  />
);
