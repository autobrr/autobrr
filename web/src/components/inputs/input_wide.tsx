/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Field as FormikField } from "formik";
import Select from "react-select";
import { Field, Label, Description } from "@headlessui/react";
import type { FieldProps, FieldValidator } from "formik";

import { classNames } from "@utils";
import { useToggle } from "@hooks/hooks";
import { EyeIcon, EyeSlashIcon } from "@heroicons/react/24/solid";

import { SelectFieldProps } from "./select";

import { DocsTooltip } from "@components/tooltips/DocsTooltip";
import { Checkbox } from "@components/Checkbox";
import {
  DropdownIndicator,
  ErrorField, IndicatorSeparator,
  RequiredField,
  SelectControl,
  SelectInput,
  SelectMenu,
  SelectOption
} from "@components/inputs/common.tsx";

interface TextFieldWideProps {
  name: string;
  label?: string;
  help?: string;
  placeholder?: string;
  defaultValue?: string;
  required?: boolean;
  autoComplete?: string;
  hidden?: boolean;
  tooltip?: JSX.Element;
  validate?: FieldValidator;
}

export const TextFieldWide = ({
  name,
  label,
  help,
  placeholder,
  defaultValue,
  required,
  autoComplete,
  tooltip,
  hidden,
  validate
}: TextFieldWideProps) => (
  <div hidden={hidden} className="space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
    <div>
      <label htmlFor={name} className="flex ml-px text-sm font-medium text-gray-900 dark:text-white sm:mt-px sm:pt-2">
        <div className="flex">
          {tooltip ? (
            <DocsTooltip label={label}>{tooltip}</DocsTooltip>
          ) : label}
          <RequiredField required={required} />
        </div>
      </label>
    </div>
    <div className="sm:col-span-2">
      <FormikField
        name={name}
        value={defaultValue}
        required={required}
        validate={validate}
      >
        {({ field, meta }: FieldProps) => (
          <input
            {...field}
            id={name}
            type="text"
            value={field.value ? field.value : defaultValue ?? ""}
            onChange={field.onChange}
            className={classNames(
              meta.touched && meta.error
                ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
              "block w-full shadow-sm sm:text-sm rounded-md border py-2.5 bg-gray-100 dark:bg-gray-850 dark:text-gray-100"
            )}
            placeholder={placeholder}
            hidden={hidden}
            required={required}
            autoComplete={autoComplete}
            data-1p-ignore
          />
        )}
      </FormikField>
      {help && (
        <p className="mt-2 text-sm text-gray-500" id={`${name}-description`}>{help}</p>
      )}
      <ErrorField name={name} classNames="block text-red-500 mt-2" />
    </div>
  </div>
);

interface PasswordFieldWideProps {
  name: string;
  label?: string;
  placeholder?: string;
  defaultValue?: string;
  help?: string;
  required?: boolean;
  autoComplete?: string;
  defaultVisible?: boolean;
  tooltip?: JSX.Element;
  validate?: FieldValidator;
}

export const PasswordFieldWide = ({
  name,
  label,
  placeholder,
  defaultValue,
  help,
  required,
  autoComplete,
  defaultVisible,
  tooltip,
  validate
}: PasswordFieldWideProps) => {
  const [isVisible, toggleVisibility] = useToggle(defaultVisible);

  return (
    <div className="space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label htmlFor={name} className="flex ml-px text-sm font-medium text-gray-900 dark:text-white sm:mt-px sm:pt-2">
          <div className="flex">
            {tooltip ? (
              <DocsTooltip label={label}>{tooltip}</DocsTooltip>
            ) : label}
            <RequiredField required={required} />
          </div>
        </label>
      </div>
      <div className="sm:col-span-2">
        <FormikField
          name={name}
          defaultValue={defaultValue}
          validate={validate}
        >
          {({ field, meta }: FieldProps) => (
            <div className="relative">
              <input
                {...field}
                id={name}
                value={field.value ? field.value : defaultValue ?? ""}
                onChange={field.onChange}
                type={isVisible ? "text" : "password"}
                className={classNames(
                  meta.touched && meta.error
                    ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                    : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
                  "block w-full shadow-sm sm:text-sm rounded-md border py-2.5 bg-gray-100 dark:bg-gray-850 dark:text-gray-100 overflow-hidden pr-8"
                )}
                placeholder={placeholder}
                required={required}
                autoComplete={autoComplete}
                data-1p-ignore
              />
              <div className="absolute inset-y-0 right-0 px-3 flex items-center" onClick={toggleVisibility}>
                {!isVisible ? <EyeIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" /> : <EyeSlashIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" />}
              </div>
            </div>
          )}
        </FormikField>
        {help && (
          <p className="mt-2 text-sm text-gray-500" id={`${name}-description`}>{help}</p>
        )}
        <ErrorField name={name} classNames="block text-red-500 mt-2" />
      </div>
    </div>
  );
};

interface NumberFieldWideProps {
  name: string;
  label?: string;
  help?: string;
  placeholder?: string;
  defaultValue?: number;
  required?: boolean;
  tooltip?: JSX.Element;
}

export const NumberFieldWide = ({
  name,
  label,
  placeholder,
  help,
  defaultValue,
  tooltip,
  required
}: NumberFieldWideProps) => (
  <div className="px-4 space-y-1 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:py-4">
    <div>
      <label
        htmlFor={name}
        className="block ml-px text-sm font-medium text-gray-900 dark:text-white sm:mt-px sm:pt-2"
      >
        <div className="flex">
          {tooltip ? (
            <DocsTooltip label={label}>{tooltip}</DocsTooltip>
          ) : label}
          <RequiredField required={required} />
        </div>
      </label>
    </div>
    <div className="sm:col-span-2">
      <FormikField
        name={name}
        defaultValue={defaultValue ?? 0}
      >
        {({ field, meta, form }: FieldProps) => (
          <input
            {...field}
            id={name}
            type="number"
            value={field.value ? field.value : defaultValue ?? 0}
            onChange={(e) => { form.setFieldValue(field.name, parseInt(e.target.value)); }}
            className={classNames(
              meta.touched && meta.error
                ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
              "block w-full shadow-sm sm:text-sm rounded-md border py-2.5 bg-gray-100 dark:bg-gray-850 dark:text-gray-100"
            )}
            onWheel={(event) => {
              if (event.currentTarget === document.activeElement) {
                event.currentTarget.blur();
                setTimeout(() => event.currentTarget.focus(), 0);
              }
            }}
            placeholder={placeholder}
          />
        )}
      </FormikField>
      {help && (
        <p className="mt-2 text-sm text-gray-500 dark:text-gray-500" id={`${name}-description`}>{help}</p>
      )}
      <ErrorField name={name} classNames="block text-red-500 mt-2" />
    </div>
  </div>
);

interface SwitchGroupWideProps {
  name: string;
  label: string;
  description?: string;
  defaultValue?: boolean;
  className?: string;
  tooltip?: JSX.Element;
}

export const SwitchGroupWide = ({
  name,
  label,
  description,
  tooltip,
  defaultValue
}: SwitchGroupWideProps) => (
  <ul className="px-4 divide-y divide-gray-200 dark:divide-gray-700">
    <Field as="li" className="py-4 flex items-center justify-between">
      <div className="flex flex-col">
        <Label as="div" passive className="text-sm font-medium text-gray-900 dark:text-white">
          <div className="flex">
            {tooltip ? (
              <DocsTooltip label={label}>{tooltip}</DocsTooltip>
            ) : label}
          </div>
        </Label>
        {description && (
          <Description className="text-sm text-gray-500 dark:text-gray-700">
            {description}
          </Description>
        )}
      </div>

      <FormikField
        name={name}
        defaultValue={defaultValue as boolean}
        type="checkbox"
      >
        {({
          field,
          form: { setFieldValue }
        }: FieldProps) => (
          <Checkbox
            {...field}
            value={!!field.checked}
            setValue={(value) => {
              setFieldValue(field?.name ?? "", value);
            }}
          />
        )}
      </FormikField>
    </Field>
  </ul>
);

export const SelectFieldWide = ({
  name,
  label,
  optionDefaultText,
  tooltip,
  options
}: SelectFieldProps) => (
  <div className="flex items-center justify-between space-y-1 px-4 py-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
    <div>
      <label
        htmlFor={name}
        className="flex ml-px text-sm font-medium text-gray-900 dark:text-white"
      >
        <div className="flex">
          {tooltip ? (
            <DocsTooltip label={label}>{tooltip}</DocsTooltip>
          ) : label}
        </div>
      </label>
    </div>
    <div className="sm:col-span-2">
      <FormikField name={name} type="select">
        {({
          field,
          form: { setFieldValue }
        }: FieldProps) => (
          <Select
            {...field}
            id={name}
            isClearable={true}
            isSearchable={true}
            components={{
              Input: SelectInput,
              Control: SelectControl,
              Menu: SelectMenu,
              Option: SelectOption,
              IndicatorSeparator: IndicatorSeparator,
              DropdownIndicator: DropdownIndicator
            }}
            placeholder={optionDefaultText}
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
            value={field?.value && field.value.value}
            onChange={(newValue: unknown) => {
              if (newValue) {
                setFieldValue(field.name, (newValue as { value: string }).value);
              }
              else {
                setFieldValue(field.name, "")
              }
            }}
            options={options}
          />
        )}
      </FormikField>
    </div>
  </div>
);
