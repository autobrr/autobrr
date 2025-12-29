/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment } from "react";
import { Field, FieldProps } from "formik";
import { Listbox, ListboxButton, Label, ListboxOption, ListboxOptions, Transition } from "@headlessui/react";
import { CheckIcon, ChevronUpDownIcon } from "@heroicons/react/24/solid";
import { MultiSelect as RMSC } from "react-multi-select-component";

import { classNames, COL_WIDTHS } from "@utils";
import { DocsTooltip } from "@components/tooltips/DocsTooltip";
import { SMColSpanClasses } from "./constants";

export interface MultiSelectOption {
  value: string | number;
  label: string;
  key?: string;
  disabled?: boolean;
}

interface MultiSelectProps {
  name: string;
  label?: string;
  options: MultiSelectOption[];
  columns?: COL_WIDTHS;
  creatable?: boolean;
  disabled?: boolean;
  tooltip?: React.JSX.Element;
}

export const MultiSelect = ({
  name,
  label,
  options,
  columns,
  creatable,
  tooltip,
  disabled
}: MultiSelectProps) => {
  const handleNewField = (value: string) => ({
    value: value.toUpperCase(),
    label: value.toUpperCase(),
    key: value
  });

  const smColClass = columns ? SMColSpanClasses[columns] : "";

  return (
    <div
      className={classNames(
        "col-span-12",
        smColClass
      )}
    >
      <label
        htmlFor={label} className="flex ml-px mb-1 text-xs font-bold tracking-wide text-gray-700 uppercase dark:text-gray-100">
        <div className="flex">
          {tooltip ? (
            <DocsTooltip label={label}>{tooltip}</DocsTooltip>
          ) : label}
        </div>
      </label>

      <Field name={name} type="select" multiple={true}>
        {({
          field,
          form: { setFieldValue }
        }: FieldProps) => (
          <RMSC
            {...field}
            options={options}
            disabled={disabled}
            labelledBy={name}
            isCreatable={creatable}
            onCreateOption={handleNewField}
            value={field.value && field.value.map((item: MultiSelectOption) => ({
              value: item.value ? item.value : item,
              label: item.label ? item.label : item
            }))}
            onChange={(values: Array<MultiSelectOption>) => {
              const am = values && values.map((i) => i.value);

              setFieldValue(field.name, am);
            }}
          />
        )}
      </Field>
    </div>
  );
};


interface IndexerMultiSelectOption {
  id: number;
  name: string;
}

export const IndexerMultiSelect = ({
  name,
  label,
  options,
  columns
}: MultiSelectProps) => {
  const smColClass = columns ? SMColSpanClasses[columns] : "";
  return (
    <div
      className={classNames(
        "col-span-12",
        smColClass
      )}
    >
      <label
        className="block ml-px mb-1 text-xs font-bold tracking-wide text-gray-700 uppercase dark:text-gray-200"
        htmlFor={label}
      >
        {label}
      </label>

      <Field name={name} type="select" multiple={true}>
        {({
            field,
            meta,
            form: { setFieldValue }
          }: FieldProps) => (
          <>
            <RMSC
              {...field}
              options={options}
              labelledBy={name}
              value={field.value && field.value.map((item: IndexerMultiSelectOption) => ({
                value: item.id, label: item.name
              }))}
              onChange={(values: MultiSelectOption[]) => {
                const item = values && values.map((i) => ({ id: i.value, name: i.label }));
                setFieldValue(field.name, item);
              }}
            />
            {meta.touched && meta.error && (
              <p className="error text-sm text-red-600 mt-1">* {meta.error}</p>
            )}
          </>
        )}
      </Field>
    </div>
  );
}

interface DownloadClientSelectProps {
  name: string;
  action: Action;
  clients: DownloadClient[];
}

export function DownloadClientSelect({
  name,
  action,
  clients
}: DownloadClientSelectProps) {
  return (
    <div className="col-span-12 sm:col-span-6">
      <Field name={name} type="select">
        {({
          field,
          meta,
          form: { setFieldValue }
        }: FieldProps) => (
          <Listbox
            value={field.value}
            onChange={(value) => setFieldValue(field?.name, value)}
          >
            {({ open }) => (
              <>
                <Label className="block text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide">
                  Client
                </Label>
                <div className="mt-1 relative">
                  <ListboxButton className="block w-full shadow-xs sm:text-sm rounded-md border py-2 pl-3 pr-10 text-left focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-gray-100">
                    <span className="block truncate">
                      {field.value
                        ? clients.find((c) => c.id === field.value)?.name
                        : "Choose a client"}
                    </span>
                    <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
                      <ChevronUpDownIcon
                        className="h-5 w-5 text-gray-400 dark:text-gray-300"
                        aria-hidden="true" />
                    </span>
                  </ListboxButton>

                  <Transition
                    show={open}
                    as={Fragment}
                    leave="transition ease-in duration-100"
                    leaveFrom="opacity-100"
                    leaveTo="opacity-0"
                  >
                    <ListboxOptions
                      static
                      className="absolute z-10 mt-1 w-full border border-gray-400 dark:border-gray-700 bg-white dark:bg-gray-900 shadow-lg max-h-60 rounded-md py-1 text-base overflow-auto focus:outline-hidden sm:text-sm"
                    >
                      {clients
                        .filter((c) => c.type === action.type)
                        .map((client) => (
                          <ListboxOption
                            key={client.id}
                            className={({ active }) => classNames(
                              active
                                ? "text-white dark:text-gray-100 bg-blue-600 dark:bg-gray-950"
                                : "text-gray-900 dark:text-gray-300",
                              "cursor-default select-none relative py-2 pl-3 pr-9"
                            )}
                            value={client.id}
                          >
                            {({ selected, active }) => (
                              <>
                                <span
                                  className={classNames(
                                    selected ? "font-semibold" : "font-normal",
                                    "block truncate"
                                  )}
                                >
                                  {client.name}
                                </span>

                                {selected ? (
                                  <span
                                    className={classNames(
                                      active ? "text-white dark:text-gray-100" : "text-blue-600 dark:text-blue-500",
                                      "absolute inset-y-0 right-0 flex items-center pr-4"
                                    )}
                                  >
                                    <CheckIcon
                                      className="h-5 w-5"
                                      aria-hidden="true" />
                                  </span>
                                ) : null}
                              </>
                            )}
                          </ListboxOption>
                        ))}
                    </ListboxOptions>
                  </Transition>
                  {meta.touched && meta.error && (
                    <p className="error text-sm text-red-600 mt-1">* {meta.error}</p>
                  )}
                </div>
              </>
            )}
          </Listbox>
        )}
      </Field>
    </div>
  );
}

export interface SelectFieldOption {
  label: string;
  value: string | number | null;
}

export interface SelectFieldProps {
  name: string;
  label: string;
  optionDefaultText: string;
  options: SelectFieldOption[];
  columns?: COL_WIDTHS;
  tooltip?: React.JSX.Element;
  className?: string;
}

export const Select = ({
  name,
  label,
  tooltip,
  optionDefaultText,
  options,
  columns = 6,
  className
}: SelectFieldProps) => {
  const smColClass = SMColSpanClasses[columns] || "sm:col-span-6";
  return (
    <div
      className={classNames(
        className ?? "col-span-12",
        smColClass,
      )}
    >
      <Field name={name} type="select">
        {({
          field,
          form: { setFieldValue }
        }: FieldProps) => (
          <Listbox
            // ?? null is required here otherwise React throws:
            // "console.js:213 A component is changing from uncontrolled to controlled.
            // This may be caused by the value changing from undefined to a defined value, which should not happen."
            value={field.value ?? null}
            onChange={(value) => setFieldValue(field.name, value)}
          >
            {({ open }) => (
              <div>
                <Label className="flex text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide">
                  {tooltip ? (
                    <DocsTooltip label={label}>{tooltip}</DocsTooltip>
                  ) : label}
                </Label>
                <div className="mt-1 relative">
                  <ListboxButton className="block w-full relative shadow-xs sm:text-sm text-left rounded-md border pl-3 pr-10 py-2.5 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-gray-100">
                    <span className="block truncate">
                      {field.value
                        ? options.find((c) => c.value === field.value)?.label
                        : optionDefaultText
                      }
                    </span>
                    <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
                      <ChevronUpDownIcon
                        className="h-5 w-5 text-gray-400 dark:text-gray-300"
                        aria-hidden="true"
                      />
                    </span>
                  </ListboxButton>

                  <Transition
                    show={open}
                    as={Fragment}
                    leave="transition ease-in duration-100"
                    leaveFrom="opacity-100"
                    leaveTo="opacity-0"
                  >
                    <ListboxOptions
                      static
                      className="absolute z-10 mt-1 w-full shadow-lg max-h-60 rounded-md py-1 text-base overflow-auto border border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-gray-100 focus:outline-hidden sm:text-sm"
                    >
                      {options.map((opt) => (
                        <ListboxOption
                          key={opt.value}
                          className={({ active: hovered, selected }) =>
                            classNames(
                              selected
                                ? "font-bold text-black dark:text-white bg-gray-300 dark:bg-gray-950"
                                : (
                                  hovered
                                    ? "text-black dark:text-gray-100 font-normal"
                                    : "text-gray-700 dark:text-gray-300 font-normal"
                                ),
                              hovered ? "bg-gray-200 dark:bg-gray-800" : "",
                              "transition-colors cursor-default select-none relative py-2 pl-3 pr-9"
                            )
                          }
                          value={opt.value}
                        >
                          {({ selected }) => (
                            <>
                              <span className="block truncate">
                                {opt.label}
                              </span>
                              <span
                                className={classNames(
                                  selected ? "visible" : "invisible",
                                  "absolute inset-y-0 right-0 flex items-center pr-4"
                                )}
                              >
                                <CheckIcon className="h-5 w-5 text-blue-600 dark:text-blue-500" aria-hidden="true" />
                              </span>
                            </>
                          )}
                        </ListboxOption>
                      ))}
                    </ListboxOptions>
                  </Transition>
                </div>
              </div>
            )}
          </Listbox>
        )}
      </Field>
    </div>
  );
};

export const SelectWide = ({
  name,
  label,
  optionDefaultText,
  options
}: SelectFieldProps) => {
  return (
    <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">

      <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:py-4">
        <Field name={name} type="select">
          {({
            field,
            form: { setFieldValue }
          }: FieldProps) => (
            <Listbox
              value={field.value}
              onChange={(value) => setFieldValue(field?.name, value)}
            >
              {({ open }) => (
                <div className="py-4 flex items-center justify-between">

                  <Label className="block text-sm font-medium text-gray-900 dark:text-white">
                    {label}
                  </Label>
                  <div className="w-full">
                    <ListboxButton className="bg-white dark:bg-gray-800 relative w-full border border-gray-300 dark:border-gray-700 rounded-md shadow-xs pl-3 pr-10 py-2 text-left cursor-default focus:outline-hidden focus:ring-1 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 dark:text-gray-200 sm:text-sm">
                      <span className="block truncate">
                        {field.value
                          ? options.find((c) => c.value === field.value)?.label
                          : optionDefaultText
                        }
                      </span>
                      <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
                        <ChevronUpDownIcon
                          className="h-5 w-5 text-gray-400 dark:text-gray-300"
                          aria-hidden="true"
                        />
                      </span>
                    </ListboxButton>

                    <Transition
                      show={open}
                      as={Fragment}
                      leave="transition ease-in duration-100"
                      leaveFrom="opacity-100"
                      leaveTo="opacity-0"
                    >
                      <ListboxOptions
                        static
                        className="absolute z-10 mt-1 w-full bg-white dark:bg-gray-800 shadow-lg max-h-60 rounded-md py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-hidden sm:text-sm"
                      >
                        {options.map((opt) => (
                          <ListboxOption
                            key={opt.value}
                            className={({ active }) =>
                              classNames(
                                active
                                  ? "text-white dark:text-gray-100 bg-blue-600 dark:bg-gray-800"
                                  : "text-gray-900 dark:text-gray-300",
                                "cursor-default select-none relative py-2 pl-3 pr-9"
                              )
                            }
                            value={opt.value}
                          >
                            {({ selected, active }) => (
                              <>
                                <span
                                  className={classNames(
                                    selected ? "font-semibold" : "font-normal",
                                    "block truncate"
                                  )}
                                >
                                  {opt.label}
                                </span>

                                {selected ? (
                                  <span
                                    className={classNames(
                                      active ? "text-white dark:text-gray-100" : "text-blue-600 dark:text-gray-700",
                                      "absolute inset-y-0 right-0 flex items-center pr-4"
                                    )}
                                  >
                                    <CheckIcon
                                      className="h-5 w-5"
                                      aria-hidden="true"
                                    />
                                  </span>
                                ) : null}
                              </>
                            )}
                          </ListboxOption>
                        ))}
                      </ListboxOptions>
                    </Transition>
                  </div>
                </div>
              )}
            </Listbox>
          )}
        </Field>
      </div>
    </div>
  );
};

export const AgeSelect = ({
  duration,
  setDuration,
  setParsedDuration,
  columns = 6
}: {
  duration: string;
  setDuration: (value: string) => void;
  setParsedDuration: (value: number) => void;
  columns?: number;
}) => {
  const options = [
    { value: '1', label: '1 hour' },
    { value: '12', label: '12 hours' },
    { value: '24', label: '1 day' },
    { value: '168', label: '1 week' },
    { value: '720', label: '1 month' },
    { value: '2160', label: '3 months' },
    { value: '4320', label: '6 months' },
    { value: '8760', label: '1 year' },
    { value: '0', label: 'Delete everything' }
  ];

  const smColClass = SMColSpanClasses[columns] || 'sm:col-span-6';

  return (
    <div className={`col-span-12 ${smColClass}`}>
      <Listbox value={duration} onChange={(value) => {
        const parsedValue = parseInt(value, 10);
        setParsedDuration(parsedValue);
        setDuration(value);
      }}>
        {({ open }) => (
          <div>
            <div className="mt-0 relative">
              <ListboxButton className="block w-full relative shadow-xs text-sm text-left rounded-md border pl-3 pr-10 py-2.5 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-gray-400">
                <span className="block truncate text-gray-500 dark:text-white">
                  {duration ? options.find(opt => opt.value === duration)?.label : 'Select...'}
                </span>
                <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
                  <ChevronUpDownIcon className="h-5 w-5 text-gray-700 dark:text-gray-500" aria-hidden="true" />
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
                  {options.map((option) => (
                    <ListboxOption
                      key={option.value}
                      className={({ active, selected }) =>
                        `relative cursor-default select-none py-2 pl-3 pr-9 ${selected ? "font-bold text-black dark:text-white bg-gray-300 dark:bg-gray-950" : active ? "text-black dark:text-gray-100 font-normal bg-gray-200 dark:bg-gray-800" : "text-gray-700 dark:text-gray-300 font-normal"
                        }`
                      }
                      value={option.value}
                    >
                      {({ selected }) => (
                        <>
                          <span className="block truncate">{option.label}</span>
                          {selected && (
                            <span className="absolute inset-y-0 right-0 flex items-center pr-4">
                              <CheckIcon className="h-5 w-5 text-blue-600 dark:text-blue-500" aria-hidden="true" />
                            </span>
                          )}
                        </>
                      )}
                    </ListboxOption>
                  ))}
                </ListboxOptions>
              </Transition>
            </div>
          </div>
        )}
      </Listbox>
    </div>
  );
};


