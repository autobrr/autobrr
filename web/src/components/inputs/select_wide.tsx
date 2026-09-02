/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { JSX } from "react";
import Select from "react-select";
import CreatableSelect from "react-select/creatable";
import { useTranslation } from "react-i18next";

import { OptionBasicTyped } from "@domain/constants";
import { useFormContext } from "@hooks/form";
import * as common from "@components/inputs/common";
import { DocsTooltip } from "@components/tooltips/DocsTooltip";
import { MultiSelect as RMSC } from "react-multi-select-component";
import { MultiSelectOption } from "@components/inputs/select.tsx";

interface SelectFieldProps<T> {
  name: string;
  label: string;
  help?: string;
  placeholder?: string;
  required?: boolean;
  defaultValue?: OptionBasicTyped<T>;
  tooltip?: JSX.Element;
  options: OptionBasicTyped<T>[];
}

export function SelectFieldCreatable<T>({ name, label, help, placeholder, tooltip, options }: SelectFieldProps<T>) {
  const { t } = useTranslation("common");
  const form = useFormContext();

  return (
    <div className="space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label
          htmlFor={name}
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
        <form.Field name={name}>
          {(field) => (
            <CreatableSelect
              id={name}
              name={name}
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
              placeholder={placeholder ?? t("forms.chooseOption")}
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
              onBlur={field.handleBlur}
              options={[...[...options, { value: field.state.value, label: field.state.value }].reduce((map, obj) => map.set(obj.value, obj), new Map()).values()]}
            />
          )}
        </form.Field>
        {help && (
          <p className="mt-2 text-sm text-gray-500" id={`${name}-description`}>{help}</p>
        )}
      </div>
    </div>
  );
}

export function SelectField<T>({ name, label, help, placeholder, options }: SelectFieldProps<T>) {
  const { t } = useTranslation("common");
  const form = useFormContext();

  return (
    <div className="space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label
          htmlFor={name}
          className="block ml-px text-sm font-medium text-gray-900 dark:text-white sm:pt-2"
        >
          {label}
        </label>
      </div>
      <div className="sm:col-span-2">
        <form.Field name={name}>
          {(field) => (
            <Select
              id={name}
              name={name}
              components={{
                Input: common.SelectInput,
                Control: common.SelectControl,
                Menu: common.SelectMenu,
                Option: common.SelectOption,
                IndicatorSeparator: common.IndicatorSeparator,
                DropdownIndicator: common.DropdownIndicator
              }}
              placeholder={placeholder ?? t("forms.chooseOption")}
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
              onBlur={field.handleBlur}
              options={[...[...options, { value: field.state.value, label: field.state.value }].reduce((map, obj) => map.set(obj.value, obj), new Map()).values()]}
            />
          )}
        </form.Field>
        {help && (
          <p className="mt-2 text-sm text-gray-500" id={`${name}-description`}>{help}</p>
        )}
      </div>
    </div>
  );
}

export function SelectFieldBasic<T>({ name, label, help, placeholder, required, tooltip, defaultValue, options }: SelectFieldProps<T>) {
  const { t } = useTranslation("common");
  const form = useFormContext();

  return (
    <div className="space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label
          htmlFor={name}
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
        <form.Field name={name}>
          {(field) => (
            <Select
              id={name}
              name={name}
              required={required}
              components={{
                Input: common.SelectInput,
                Control: common.SelectControl,
                Menu: common.SelectMenu,
                Option: common.SelectOption,
                IndicatorSeparator: common.IndicatorSeparator,
                DropdownIndicator: common.DropdownIndicator
              }}
              placeholder={placeholder ?? t("forms.chooseOption")}
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
                const option = newValue as { value: string };
                field.handleChange(option?.value ?? "");
              }}
              onBlur={field.handleBlur}
              options={options}
            />
          )}
        </form.Field>
        {help && (
          <p className="mt-2 text-sm text-gray-500" id={`${name}-description`}>{help}</p>
        )}
      </div>
    </div>
  );
}

export interface MultiSelectFieldProps {
  name: string;
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

export function ListFilterMultiSelectField({ name, label, help, tooltip, options, required }: MultiSelectFieldProps) {
  const form = useFormContext();

  return (
    <div className="flex items-center space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label
          htmlFor={name}
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
        <form.Field name={name}>
          {(field) => (
            <RMSC
              options={options}
              labelledBy={name}
              value={field.state.value && field.state.value.map((item: ListFilterMultiSelectOption) => ({
                value: item.id,
                label: item.name
              }))}
              onChange={(values: MultiSelectOption[]) => {
                const item = values && values.map((i) => ({ id: i.value, name: i.label }));
                field.handleChange(item);
              }}
            />
          )}
        </form.Field>
        {help && (
          <p className="mt-2 text-sm text-gray-500" id={`${name}-description`}>{help}</p>
        )}
      </div>
    </div>
  );
}
