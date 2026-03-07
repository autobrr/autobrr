/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { JSX } from "react";
import Select from "react-select";
import CreatableSelect from "react-select/creatable";
import { MultiSelect as RMSC } from "react-multi-select-component";

import { OptionBasicTyped } from "@domain/constants";
import * as common from "./common";
import { DocsTooltip } from "@components/tooltips/DocsTooltip";
import { MultiSelectOption } from "./select";
import { useFieldContext } from "@app/lib/form";

interface SelectFieldProps<T> {
  label: string;
  help?: string;
  placeholder?: string;
  required?: boolean;
  defaultValue?: OptionBasicTyped<T>;
  tooltip?: JSX.Element;
  options: OptionBasicTyped<T>[];
}

export function SelectFieldCreatable<T>({ label, help, placeholder, tooltip, options }: SelectFieldProps<T>) {
  const field = useFieldContext<string>();

  return (
    <div className="space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label
          className="block ml-px text-sm font-medium text-gray-900 dark:text-white sm:pt-2"
        >
          <div className="flex">
            {tooltip ? (
              <DocsTooltip label={label}>{tooltip}</DocsTooltip>
            ) : label}
          </div>
        </label>
      </div>
      <div className="sm:col-span-2">
        <CreatableSelect
          isClearable={true}
          isSearchable={true}
          components={{
            Input: common.SelectInput,
            Control: common.SelectControl,
            Menu: common.SelectMenu,
            Option: common.SelectOption,
            IndicatorSeparator: common.IndicatorSeparator,
            DropdownIndicator: common.DropdownIndicator
          }}
          placeholder={placeholder ?? "Choose an option"}
          styles={{
            singleValue: (base) => ({
              ...base,
              color: "unset"
            })
          }}
          theme={(theme) => ({
            ...theme,
            spacing: {
              ...theme.spacing,
              controlHeight: 30,
              baseUnit: 2
            }
          })}
          value={field.state.value ? { value: field.state.value, label: field.state.value } : field.state.value}
          onChange={(newValue: unknown) => {
            const option = newValue as { value: string };
            field.handleChange(option?.value ?? "");
          }}
          options={[...[...options, { value: field.state.value, label: field.state.value }].reduce((map, obj) => map.set(obj.value, obj), new Map()).values()]}
        />
        {help && (
          <p className="mt-2 text-sm text-gray-500">{help}</p>
        )}
      </div>
    </div>
  );
}

export function SelectField<T>({ label, help, placeholder, options }: SelectFieldProps<T>) {
  const field = useFieldContext<string>();

  return (
    <div className="space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label
          className="block ml-px text-sm font-medium text-gray-900 dark:text-white sm:pt-2"
        >
          {label}
        </label>
      </div>
      <div className="sm:col-span-2">
        <Select
          components={{
            Input: common.SelectInput,
            Control: common.SelectControl,
            Menu: common.SelectMenu,
            Option: common.SelectOption,
            IndicatorSeparator: common.IndicatorSeparator,
            DropdownIndicator: common.DropdownIndicator
          }}
          placeholder={placeholder ?? "Choose an option"}
          styles={{
            singleValue: (base) => ({
              ...base,
              color: "unset"
            })
          }}
          theme={(theme) => ({
            ...theme,
            spacing: {
              ...theme.spacing,
              controlHeight: 30,
              baseUnit: 2
            }
          })}
          value={field.state.value ? { value: field.state.value, label: field.state.value } : field.state.value}
          onChange={(newValue: unknown) => {
            const option = newValue as { value: string };
            field.handleChange(option?.value ?? "");
          }}
          options={[...[...options, { value: field.state.value, label: field.state.value }].reduce((map, obj) => map.set(obj.value, obj), new Map()).values()]}
        />
        {help && (
          <p className="mt-2 text-sm text-gray-500">{help}</p>
        )}
      </div>
    </div>
  );
}

export function SelectFieldBasic<T>({ label, help, placeholder, required, tooltip, defaultValue, options }: SelectFieldProps<T>) {
  const field = useFieldContext<T>();

  return (
    <div className="space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label
          className="block ml-px text-sm font-medium text-gray-900 dark:text-white sm:pt-2"
        >
          <div className="flex">
            {tooltip ? (
              <DocsTooltip label={label}>{tooltip}</DocsTooltip>
            ) : label}
          </div>
        </label>
      </div>
      <div className="sm:col-span-2">
        <Select
          required={required}
          components={{
            Input: common.SelectInput,
            Control: common.SelectControl,
            Menu: common.SelectMenu,
            Option: common.SelectOption,
            IndicatorSeparator: common.IndicatorSeparator,
            DropdownIndicator: common.DropdownIndicator
          }}
          placeholder={placeholder ?? "Choose an option"}
          styles={{
            singleValue: (base) => ({
              ...base,
              color: "unset"
            })
          }}
          theme={(theme) => ({
            ...theme,
            spacing: {
              ...theme.spacing,
              controlHeight: 30,
              baseUnit: 2
            }
          })}
          defaultValue={defaultValue}
          value={field.state.value && options.find(o => o.value == field.state.value)}
          onChange={(newValue: unknown) => {
            const option = newValue as { value: T };
            field.handleChange(option?.value ?? ("" as T));
          }}
          options={options}
        />
        {help && (
          <p className="mt-2 text-sm text-gray-500">{help}</p>
        )}
      </div>
    </div>
  );
}

export interface MultiSelectFieldProps {
  label: string;
  help?: string;
  placeholder?: string;
  required?: boolean;
  tooltip?: JSX.Element;
  options: OptionBasicTyped<number>[];
}

interface ListFilterMultiSelectOption {
  id: number;
  name: string;
}

export function ListFilterMultiSelectField({ label, help, tooltip, options, required }: MultiSelectFieldProps) {
  const field = useFieldContext<ListFilterMultiSelectOption[]>();

  return (
    <div className="flex items-center space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label
          className="block ml-px text-sm font-medium text-gray-900 dark:text-white"
        >
          <div className="flex">
            {tooltip ? (
              <DocsTooltip label={label}>{tooltip}</DocsTooltip>
            ) : label}
            <common.RequiredField required={required} />
          </div>
        </label>
      </div>
      <div className="sm:col-span-2">
        <RMSC
          options={options}
          labelledBy={label}
          value={field.state.value && field.state.value.map((item: ListFilterMultiSelectOption) => ({
            value: item.id,
            label: item.name
          }))}
          onChange={(values: MultiSelectOption[]) => {
            const item = values && values.map((i) => ({ id: i.value as number, name: i.label }));
            field.handleChange(item);
          }}
        />
        {help && (
          <p className="mt-2 text-sm text-gray-500">{help}</p>
        )}
      </div>
    </div>
  );
}
