import { Fragment } from "react";
import { Field, FieldProps } from "formik";
import { Listbox, Transition } from "@headlessui/react";
import { CheckIcon, ChevronUpDownIcon } from "@heroicons/react/24/solid";
import { MultiSelect as RMSC } from "react-multi-select-component";

import { classNames, COL_WIDTHS } from "../../utils";
import { SettingsContext } from "../../utils/Context";

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
}

export const MultiSelect = ({
  name,
  label,
  options,
  columns,
  creatable
}: MultiSelectProps) => {
  const settingsContext = SettingsContext.useValue();

  const handleNewField = (value: string) => ({
    value: value.toUpperCase(),
    label: value.toUpperCase(),
    key: value
  });

  return (
    <div
      className={classNames(
        columns ? `col-span-${columns}` : "col-span-12"
      )}
    >
      <label
        className="block mb-2 text-xs font-bold tracking-wide text-gray-700 uppercase dark:text-gray-200"
        htmlFor={label}
      >
        {label}
      </label>

      <Field name={name} type="select" multiple={true}>
        {({
          field,
          form: { setFieldValue }
        }: FieldProps) => (
          <RMSC
            {...field}
            options={[...[...options, ...field.value.map((i: MultiSelectOption) => ({ value: i.value ?? i, label: i.label ?? i }))].reduce((map, obj) => map.set(obj.value, obj), new Map()).values()]}
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
            className={settingsContext.darkTheme ? "dark" : ""}
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
  const settingsContext = SettingsContext.useValue();
  return (
    <div
      className={classNames(
        columns ? `col-span-${columns}` : "col-span-12"
      )}
    >
      <label
        className="block mb-2 text-xs font-bold tracking-wide text-gray-700 uppercase dark:text-gray-200"
        htmlFor={label}
      >
        {label}
      </label>

      <Field name={name} type="select" multiple={true}>
        {({
          field,
          form: { setFieldValue }
        }: FieldProps) => (
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
            className={settingsContext.darkTheme ? "dark" : ""}
          />
        )}
      </Field>
    </div>
  );
};

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
    <div className="col-span-6 sm:col-span-6">
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
              <>
                <Listbox.Label className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
                                    Client
                </Listbox.Label>
                <div className="mt-2 relative">
                  <Listbox.Button className="bg-white dark:bg-gray-800 relative w-full border border-gray-300 dark:border-gray-700 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 dark:text-gray-200 sm:text-sm">
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
                  </Listbox.Button>

                  <Transition
                    show={open}
                    as={Fragment}
                    leave="transition ease-in duration-100"
                    leaveFrom="opacity-100"
                    leaveTo="opacity-0"
                  >
                    <Listbox.Options
                      static
                      className="absolute z-10 mt-1 w-full bg-white dark:bg-gray-800 shadow-lg max-h-60 rounded-md py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm"
                    >
                      {clients
                        .filter((c) => c.type === action.type)
                        .map((client) => (
                          <Listbox.Option
                            key={client.id}
                            className={({ active }) => classNames(
                              active
                                ? "text-white dark:text-gray-100 bg-blue-600 dark:bg-gray-800"
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
                                      active ? "text-white dark:text-gray-100" : "text-blue-600 dark:text-gray-700",
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
                          </Listbox.Option>
                        ))}
                    </Listbox.Options>
                  </Transition>
                </div>
              </>
            )}
          </Listbox>
        )}
      </Field>
    </div>
  );
}

interface SelectFieldOption {
    label: string;
    value: string;
}

interface SelectFieldProps {
    name: string;
    label: string;
    optionDefaultText: string;
    options: SelectFieldOption[];
}

export const Select = ({
  name,
  label,
  optionDefaultText,
  options
}: SelectFieldProps) => {
  return (
    <div className="col-span-6">
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
              <>
                <Listbox.Label className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
                  {label}
                </Listbox.Label>
                <div className="mt-2 relative">
                  <Listbox.Button className="bg-white dark:bg-gray-800 relative w-full border border-gray-300 dark:border-gray-700 rounded-md shadow-sm pl-3 pr-10 py-2.5 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 dark:text-gray-200 sm:text-sm">
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
                  </Listbox.Button>

                  <Transition
                    show={open}
                    as={Fragment}
                    leave="transition ease-in duration-100"
                    leaveFrom="opacity-100"
                    leaveTo="opacity-0"
                  >
                    <Listbox.Options
                      static
                      className="absolute z-10 mt-1 w-full bg-white dark:bg-gray-800 shadow-lg max-h-60 rounded-md py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm"
                    >
                      {options.map((opt) => (
                        <Listbox.Option
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
                        </Listbox.Option>
                      ))}
                    </Listbox.Options>
                  </Transition>
                </div>
              </>
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

                  <Listbox.Label className="block text-sm font-medium text-gray-900 dark:text-white">
                    {label}
                  </Listbox.Label>
                  <div className="w-full">
                    <Listbox.Button className="bg-white dark:bg-gray-800 relative w-full border border-gray-300 dark:border-gray-700 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 dark:text-gray-200 sm:text-sm">
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
                    </Listbox.Button>

                    <Transition
                      show={open}
                      as={Fragment}
                      leave="transition ease-in duration-100"
                      leaveFrom="opacity-100"
                      leaveTo="opacity-0"
                    >
                      <Listbox.Options
                        static
                        className="absolute z-10 mt-1 w-full bg-white dark:bg-gray-800 shadow-lg max-h-60 rounded-md py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm"
                      >
                        {options.map((opt) => (
                          <Listbox.Option
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
                          </Listbox.Option>
                        ))}
                      </Listbox.Options>
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
