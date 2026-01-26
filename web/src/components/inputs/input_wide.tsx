/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { JSX, useState, Fragment } from "react";
import { Field as FormikField } from "formik";
import Select from "react-select";
import { Field, Label, Description, Listbox, ListboxButton, ListboxOption, ListboxOptions, Transition } from "@headlessui/react";
import type { FieldProps, FieldValidator } from "formik";

import { classNames } from "@utils";
import { useToggle } from "@hooks/hooks";
import { EyeIcon, EyeSlashIcon, CheckIcon, ChevronUpDownIcon } from "@heroicons/react/24/solid";

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
  disabled?: boolean;
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
  disabled,
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
        disabled={disabled}
      >
        {({ field, meta }: FieldProps) => (
          <input
            {...field}
            id={name}
            type="text"
            value={field.value ? field.value : defaultValue ?? ""}
            onChange={field.onChange}
            disabled={disabled}
            className={classNames(
              meta.touched && meta.error
                ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
              "block w-full shadow-xs sm:text-sm rounded-md border py-2.5 dark:text-gray-100",
              disabled ? "bg-gray-200 dark:bg-gray-700" : "bg-gray-100 dark:bg-gray-850 "
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
                  "block w-full shadow-xs sm:text-sm rounded-md border py-2.5 bg-gray-100 dark:bg-gray-850 dark:text-gray-100 overflow-hidden pr-8"
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
              "block w-full shadow-xs sm:text-sm rounded-md border py-2.5 bg-gray-100 dark:bg-gray-850 dark:text-gray-100"
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
          <Description className="text-sm text-gray-500 dark:text-gray-500">
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

interface DurationFieldWideProps {
  // Standard props (matches NumberFieldWide, TextFieldWide pattern)
  name: string;                    // Required - Formik field name
  label?: string;                  // Optional - Display label
  help?: string;                   // Optional - Help text below field
  placeholder?: string;            // Optional - Placeholder for number input
  defaultValue?: number;           // Optional - Initial numeric value
  required?: boolean;              // Optional - Show required asterisk
  tooltip?: JSX.Element;           // Optional - Info tooltip

  // Duration-specific props
  units?: string[];                // Optional - Available units (default: ["hours", "days", "weeks", "months", "years"])
  defaultUnit?: string;            // Optional - Initial unit (default: "hours")
  storeAsHours?: boolean;          // Optional - Convert to hours before storing (default: true)
}

// Props for the inner duration field component
interface DurationFieldInnerProps extends Pick<FieldProps, 'meta' | 'form'> {
  name: string;
  placeholder: string;
  defaultValue: number;
  defaultUnit: string;
  units: string[];
  storeAsHours: boolean;
}

const UNIT_TO_HOURS: Record<string, number> = {
  "hours": 1,
  "days": 24,
  "weeks": 168,
  "months": 720,
  "years": 8760
};

const UNIT_LABELS: Record<string, string> = {
  "hours": "Hours",
  "days": "Days",
  "weeks": "Weeks",
  "months": "Months",
  "years": "Years",
  "minutes": "Minutes"
};

// Converts stored hours to the largest evenly-divisible time unit for display
const convertHoursToBestUnit = (hours: number, units: string[], defaultUnit: string): { value: number; unit: string } => {
  if (hours === 0) return { value: 0, unit: defaultUnit };

  // Try to find the largest unit that divides evenly
  if (hours % 8760 === 0 && units.includes("years")) return { value: hours / 8760, unit: "years" };
  if (hours % 720 === 0 && units.includes("months")) return { value: hours / 720, unit: "months" };
  if (hours % 168 === 0 && units.includes("weeks")) return { value: hours / 168, unit: "weeks" };
  if (hours % 24 === 0 && units.includes("days")) return { value: hours / 24, unit: "days" };
  return { value: hours, unit: "hours" };
};

// Inner component to allow React Hooks usage within FormikField render prop
const DurationFieldInner = ({
  name,
  placeholder,
  defaultValue,
  defaultUnit,
  units,
  storeAsHours,
  meta,
  form
}: DurationFieldInnerProps) => {
  // Initialize state with computed values from Formik field (for edit forms)
  const [selectedUnit, setSelectedUnit] = useState(() => {
    const fieldValue = form.values[name];
    if (fieldValue !== undefined && fieldValue !== null && fieldValue !== 0) {
      const { unit } = convertHoursToBestUnit(fieldValue, units, defaultUnit);
      return unit;
    }
    return defaultUnit;
  });

  const [displayValue, setDisplayValue] = useState(() => {
    const fieldValue = form.values[name];
    if (fieldValue !== undefined && fieldValue !== null && fieldValue !== 0) {
      const { value } = convertHoursToBestUnit(fieldValue, units, defaultUnit);
      return value;
    }
    return defaultValue;
  });

  // Calculate hours value for storage
  const calculateHours = (value: number, unit: string) => {
    return storeAsHours ? value * UNIT_TO_HOURS[unit] : value;
  };

  const handleValueChange = (newValue: number) => {
    setDisplayValue(newValue);
    const hoursValue = calculateHours(newValue, selectedUnit);
    form.setFieldValue(name, hoursValue);
  };

  const handleUnitChange = (newUnit: string) => {
    setSelectedUnit(newUnit);
    const hoursValue = calculateHours(displayValue, newUnit);
    form.setFieldValue(name, hoursValue);
  };

  return (
    <div className="grid grid-cols-12 gap-2">
      {/* Number Input - 9 columns (75%) */}
      <div className="col-span-9">
        <input
          type="number"
          id={name}
          placeholder={placeholder}
          value={displayValue}
          onChange={(e) => handleValueChange(parseInt(e.target.value) || 0)}
          className={classNames(
            meta.touched && meta.error
              ? "border-red-500 focus:ring-red-500 focus:border-red-500"
              : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
            "block w-full shadow-xs sm:text-sm rounded-md border py-2.5 bg-gray-100 dark:bg-gray-850 dark:text-gray-100"
          )}
          min={0}
        />
      </div>

      <div className="col-span-3">
        <Listbox value={selectedUnit} onChange={handleUnitChange}>
          {({ open }) => (
            <div className="relative">
              <ListboxButton className="block w-full shadow-xs text-sm rounded-md border pl-3 pr-8 py-2.5 text-left focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-white">
                <span className="block truncate">
                  {UNIT_LABELS[selectedUnit]}
                </span>
                <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
                  <ChevronUpDownIcon className="h-5 w-5 text-gray-400" aria-hidden="true" />
                </span>
              </ListboxButton>
              <Transition
                show={open}
                as={Fragment}
                leave="transition ease-in duration-100"
                leaveFrom="opacity-100"
                leaveTo="opacity-0"
              >
                <ListboxOptions className="absolute z-10 mt-1 w-full shadow-lg max-h-60 rounded-md py-1 overflow-auto border border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-white focus:outline-hidden text-sm">
                  {units.map((unit) => (
                    <ListboxOption
                      key={unit}
                      className={({ focus, selected }) =>
                        `relative cursor-default select-none py-2 pl-3 pr-9 ${
                          selected
                            ? "font-bold text-black dark:text-white bg-gray-300 dark:bg-gray-950"
                            : focus
                            ? "text-black dark:text-gray-100 font-normal bg-gray-200 dark:bg-gray-800"
                            : "text-gray-700 dark:text-gray-300 font-normal"
                        }`
                      }
                      value={unit}
                    >
                      {({ selected }) => (
                        <>
                          <span className={classNames(selected ? "font-semibold" : "font-normal", "block truncate")}>
                            {UNIT_LABELS[unit]}
                          </span>
                          {selected && (
                            <span className="absolute inset-y-0 right-0 flex items-center pr-4">
                              <CheckIcon className="h-5 w-5" aria-hidden="true" />
                            </span>
                          )}
                        </>
                      )}
                    </ListboxOption>
                  ))}
                </ListboxOptions>
              </Transition>
            </div>
          )}
        </Listbox>
      </div>
    </div>
  );
};

export const DurationFieldWide = ({
  name,
  label,
  placeholder = "0",
  help,
  defaultValue = 0,
  tooltip,
  required,
  units = ["hours", "days", "weeks", "months", "years"],
  defaultUnit = "hours",
  storeAsHours = true
}: DurationFieldWideProps) => {
  return (
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
          {({ meta, form }: FieldProps) => (
            <DurationFieldInner
              name={name}
              placeholder={placeholder}
              defaultValue={defaultValue}
              defaultUnit={defaultUnit}
              units={units}
              storeAsHours={storeAsHours}
              meta={meta}
              form={form}
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
};
