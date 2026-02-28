/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect } from "react";
import { Field, FieldProps, useFormikContext } from "formik";
import { EyeIcon, EyeSlashIcon, CheckCircleIcon, XCircleIcon } from "@heroicons/react/24/solid";
import TextareaAutosize from "react-textarea-autosize";

import { useToggle } from "@hooks/hooks";
import { DocsTooltip } from "@components/tooltips/DocsTooltip";
import { classNames } from "@utils";

type COL_WIDTHS = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;

interface TextFieldProps {
  name: string;
  defaultValue?: string;
  label?: string;
  required?: boolean;
  placeholder?: string;
  columns?: COL_WIDTHS;
  autoComplete?: string;
  hidden?: boolean;
  disabled?: boolean;
  tooltip?: React.JSX.Element;
}


export const TextField = ({
  name,
  defaultValue,
  label,
  required,
  placeholder,
  columns,
  autoComplete,
  hidden,
  tooltip,
  disabled
}: TextFieldProps) => (
  <div
    className={classNames(
      "col-span-12",
      hidden ? "hidden" : "",
      columns ? `sm:col-span-${columns}` : ""
    )}
  >
    {label && (
      <label htmlFor={name} className="flex ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide">
        {tooltip ? (
          <DocsTooltip label={label}>{tooltip}</DocsTooltip>
        ) : label}
        {required ? (
          <span className="ml-1 text-red-500">*</span>
        ) : null}
      </label>
    )}
    <Field name={name} defaultValue={defaultValue}>
      {({
        field,
        meta
      }: FieldProps) => (
        <>
          <input
            {...field}
            name={name}
            type="text"
            defaultValue={defaultValue}
            autoComplete={autoComplete}
            className={classNames(
              meta.touched && meta.error
                ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
              disabled
                ? "bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400 cursor-not-allowed"
                : "bg-gray-100 dark:bg-gray-815 dark:text-gray-100",
              "mt-1 block border w-full dark:text-gray-100 rounded-md"
            )}
            disabled={disabled}
            placeholder={placeholder}
            data-1p-ignore
          />

          {meta.touched && meta.error && (
            <p className="error text-sm text-red-600 mt-1">* {meta.error}</p>
          )}
        </>
      )}
    </Field>
  </div>
);

interface RegexFieldProps {
  name: string;
  defaultValue?: string;
  label?: string;
  placeholder?: string;
  columns?: COL_WIDTHS;
  autoComplete?: string;
  useRegex?: boolean;
  hidden?: boolean;
  disabled?: boolean;
  tooltip?: React.JSX.Element;
}

export const RegexField = ({
  name,
  defaultValue,
  label,
  placeholder,
  columns,
  autoComplete,
  useRegex,
  hidden,
  tooltip,
  disabled
}: RegexFieldProps) => {
  const validRegex = (pattern: string) => {

    // Check for unsupported lookahead and lookbehind assertions
    if (/\(\?<=|\(\?<!|\(\?=|\(\?!/.test(pattern)) {
      return false;
    }

    // Check for unsupported atomic groups
    if (/\(\?>/.test(pattern)) {
      return false;
    }

    // Check for unsupported recursive patterns
    if (/\(\?(R|0)\)/.test(pattern)) {
      return false;
    }

    // Check for unsupported possessive quantifiers
    if (/[*+?]{1}\+|\{[0-9]+,[0-9]*\}\+/.test(pattern)) {
      return false;
    }

    // Check for unsupported control verbs
    if (/\\g</.test(pattern)) {
      return false;
    }

    // Check for unsupported conditionals
    if (/\(\?\((\?[=!][^)]*)\)[^)]*\|?[^)]*\)/.test(pattern)) {
      return false;
    }

    // Check for unsupported backreferences
    if (/\\k</.test(pattern)) {
      return false;
    }

    // Check if the pattern is a valid regex
    try {
      new RegExp(pattern);
      return true;
    } catch (e) {
      return false;
    }
  };


  const validateRegexp = (val: string) => {
    let error = "";

    if (!validRegex(val)) {
      error = "Invalid regex";
    }

    return error;
  };

  const { validateForm } = useFormikContext();
  useEffect(() => {
    if (useRegex) {
      validateForm();
    }
  }, [useRegex, validateForm]);

  return (
    <div
      className={classNames(
        "col-span-12",
        hidden ? "hidden" : "",
        columns ? `sm:col-span-${columns}` : ""
      )}
    >
      {label && (
        <label
          htmlFor={name}
          className="flex ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide"
        >
          {tooltip ? (
            <DocsTooltip label={label}>{tooltip}</DocsTooltip>
          ) : label}
        </label>
      )}
      <Field
        name={name}
        validate={useRegex && validateRegexp}
      >
        {({ field, meta }: FieldProps) => (
          <div className="relative">
            <input
              {...field}
              name={name}
              type="text"
              defaultValue={defaultValue}
              autoComplete={autoComplete}
              className={classNames(
                useRegex && meta.error
                  ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                  : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
                disabled
                  ? "bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400 cursor-not-allowed"
                  : "bg-gray-100 dark:bg-gray-815 dark:text-gray-100",
                useRegex
                  ? "pr-10"
                  : "",
                "mt-1 block w-full dark:text-gray-100 rounded-md"
              )}
              disabled={disabled}
              placeholder={placeholder}
            />
            {useRegex && (
              <div className="relative">
                <div className="flex float-right items-center">
                  {!meta.error ? (
                    <CheckCircleIcon className="h-8 w-8 mb-2.5 pl-1 text-green-500 right-2 absolute transform -translate-y-1/2" aria-hidden="true" style={{ overflow: "hidden" }} />
                  ) : (
                    <XCircleIcon className="h-8 w-8 mb-2.5 pl-1 text-red-500 right-2 absolute transform -translate-y-1/2" aria-hidden="true" style={{ overflow: "hidden" }} />
                  )}
                </div>
              </div>
            )}
          </div>
        )}
      </Field>

    </div>
  );
};

export const RegexTextAreaField = ({
  name,
  defaultValue,
  label,
  placeholder,
  columns,
  autoComplete = "off",
  useRegex,
  hidden,
  tooltip,
  disabled
}: RegexFieldProps) => {
  const validRegex = (pattern: string) => {

    // Check for unsupported lookahead and lookbehind assertions
    if (/\(\?<=|\(\?<!|\(\?=|\(\?!/.test(pattern)) {
      return false;
    }

    // Check for unsupported atomic groups
    if (/\(\?>/.test(pattern)) {
      return false;
    }

    // Check for unsupported recursive patterns
    if (/\(\?(R|0)\)/.test(pattern)) {
      return false;
    }

    // Check for unsupported possessive quantifiers
    if (/[*+?]{1}\+|\{[0-9]+,[0-9]*\}\+/.test(pattern)) {
      return false;
    }

    // Check for unsupported control verbs
    if (/\\g</.test(pattern)) {
      return false;
    }

    // Check for unsupported conditionals
    if (/\(\?\((\?[=!][^)]*)\)[^)]*\|?[^)]*\)/.test(pattern)) {
      return false;
    }

    // Check for unsupported backreferences
    if (/\\k</.test(pattern)) {
      return false;
    }

    // Check if the pattern is a valid regex
    try {
      new RegExp(pattern);
      return true;
    } catch (e) {
      return false;
    }
  };


  const validateRegexp = (val: string) => {
    let error = "";

    if (!validRegex(val)) {
      error = "Invalid regex";
    }

    return error;
  };

  const { validateForm } = useFormikContext();
  useEffect(() => {
    if (useRegex) {
      validateForm();
    }
  }, [useRegex, validateForm]);

  return (
    <div
      className={classNames(
        "col-span-12",
        hidden ? "hidden" : "",
        columns ? `sm:col-span-${columns}` : ""
      )}
    >
      {label && (
        <label
          htmlFor={name}
          className={classNames(
            tooltip ? "z-10" : "",
            "flex ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide"
          )}
        >
          {tooltip ? (
            <DocsTooltip label={label}>{tooltip}</DocsTooltip>
          ) : label}
        </label>
      )}
      <Field
        name={name}
        validate={useRegex && validateRegexp}
      >
        {({ field, meta }: FieldProps) => (
          <div className="relative">

            <TextareaAutosize
              {...field}
              id={name}
              maxRows={10}
              defaultValue={defaultValue}
              autoComplete={autoComplete}
              className={classNames(
                useRegex && meta.error
                  ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                  : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
                disabled
                  ? "bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400 cursor-not-allowed"
                  : "bg-gray-100 dark:bg-gray-815 dark:text-gray-100",
                useRegex
                  ? "pr-10"
                  : "",
                "mt-1 block w-full dark:text-gray-100 rounded-md"
              )}
              placeholder={placeholder}
              disabled={disabled}
            />

            {useRegex && (
              <div className="relative">
                <div className="flex float-right items-center">
                  {!meta.error ? (
                    <CheckCircleIcon className="h-8 w-8 mb-2.5 pl-1 text-green-500 right-2 absolute transform -translate-y-1/2" aria-hidden="true" style={{ overflow: "hidden" }} />
                  ) : (
                    <XCircleIcon className="h-8 w-8 mb-2.5 pl-1 text-red-500 right-2 absolute transform -translate-y-1/2" aria-hidden="true" style={{ overflow: "hidden" }} />
                  )}
                </div>
              </div>
            )}
          </div>
        )}
      </Field>

    </div>
  );
};

interface TextAreaProps {
  name: string;
  defaultValue?: string;
  label?: string;
  placeholder?: string;
  columns?: COL_WIDTHS;
  rows?: number;
  autoComplete?: string;
  hidden?: boolean;
  disabled?: boolean;
  tooltip?: React.JSX.Element;
}

export const TextArea = ({
  name,
  defaultValue,
  label,
  placeholder,
  columns,
  rows,
  autoComplete,
  hidden,
  tooltip,
  disabled
}: TextAreaProps) => (
  <div
    className={classNames(
      "col-span-12",
      hidden ? "hidden" : "",
      columns ? `sm:col-span-${columns}` : ""
    )}
  >
    {label && (
      <label htmlFor={name} className="flex ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide">
        {tooltip ? (
          <DocsTooltip label={label}>{tooltip}</DocsTooltip>
        ) : label}
      </label>
    )}
    <Field name={name}>
      {({
        field,
        meta
      }: FieldProps) => (
        <div>
          <textarea
            {...field}
            id={name}
            rows={rows}
            defaultValue={defaultValue}
            autoComplete={autoComplete}
            className={classNames(
              meta.touched && meta.error
                ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
              disabled
                ? "bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400 cursor-not-allowed"
                : "bg-gray-100 dark:bg-gray-815 dark:text-gray-100",
              "mt-1 block border w-full dark:text-gray-100 rounded-md"
            )}
            placeholder={placeholder}
            disabled={disabled}
          />

          {meta.touched && meta.error && (
            <p className="error text-sm text-red-600 mt-1">* {meta.error}</p>
          )}
        </div>
      )}
    </Field>
  </div>
);

interface TextAreaAutoResizeProps {
  name: string;
  defaultValue?: string;
  label?: string;
  placeholder?: string;
  columns?: COL_WIDTHS;
  rows?: number;
  autoComplete?: string;
  hidden?: boolean;
  disabled?: boolean;
  tooltip?: React.JSX.Element;
  className?: string;
}

export const TextAreaAutoResize = ({
  name,
  defaultValue,
  label,
  placeholder,
  columns,
  rows,
  autoComplete,
  hidden,
  tooltip,
  disabled,
  className = ""
}: TextAreaAutoResizeProps) => (
  <div
    className={classNames(
      className,
      "col-span-12",
      hidden ? "hidden" : "",
      columns ? `sm:col-span-${columns}` : ""
    )}
  >
    {label && (
      <label htmlFor={name} className="flex ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide">
        {tooltip ? (
          <DocsTooltip label={label}>{tooltip}</DocsTooltip>
        ) : label}
      </label>
    )}
    <Field name={name}>
      {({
        field,
        meta
      }: FieldProps) => (
        <div>
          <TextareaAutosize
            {...field}
            id={name}
            rows={rows}
            maxRows={10}
            defaultValue={defaultValue}
            autoComplete={autoComplete}
            className={classNames(
              meta.touched && meta.error
                ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
              disabled
                ? "bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400 cursor-not-allowed"
                : "bg-gray-100 dark:bg-gray-815 dark:text-gray-100",
              "mt-1 block w-full dark:text-gray-100 rounded-md"
            )}
            placeholder={placeholder}
            disabled={disabled}
          />

          {meta.touched && meta.error && (
            <p className="error text-sm text-red-600 mt-1">* {meta.error}</p>
          )}
        </div>
      )}
    </Field>
  </div>
);


interface PasswordFieldProps {
  name: string;
  label?: string;
  placeholder?: string;
  columns?: COL_WIDTHS;
  autoComplete?: string;
  defaultValue?: string;
  help?: string;
  required?: boolean;
  tooltip?: React.JSX.Element;
}

export const PasswordField = ({
  name,
  label,
  placeholder,
  defaultValue,
  columns,
  autoComplete,
  help,
  tooltip,
  required
}: PasswordFieldProps) => {
  const [isVisible, toggleVisibility] = useToggle(false);

  return (
    <div
      className={classNames(
        "col-span-12",
        columns ? `sm:col-span-${columns}` : ""
      )}
    >
      {label && (
        <label htmlFor={name} className="flex ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide">
          {tooltip ? (
            <DocsTooltip label={label}>{tooltip}</DocsTooltip>
          ) : (
            label
          )}
          {required && <span className="text-red-500">*</span>}
        </label>
      )}
      <div>
        <Field name={name} defaultValue={defaultValue}>
          {({
            field,
            meta
          }: FieldProps) => (
            <>
              <div className="sm:col-span-2 relative">
                <input
                  {...field}
                  id={name}
                  type={isVisible ? "text" : "password"}
                  autoComplete={autoComplete}
                  className={classNames(
                    meta.touched && meta.error
                      ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                      : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
                    "mt-1 block w-full rounded-md bg-gray-100 dark:bg-gray-815 dark:text-gray-100"
                  )}
                  placeholder={placeholder}
                />

                <div className="absolute inset-y-0 right-0 px-3 flex items-center" onClick={toggleVisibility}>
                  {!isVisible ? <EyeIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" />
                    : <EyeSlashIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" />}
                </div>
              </div>
              {help && (
                <p className="mt-2 text-sm text-gray-500" id="email-description">{help}</p>
              )}

              {meta.touched && meta.error && (
                <p className="error text-sm text-red-600 mt-1">* {meta.error}</p>
              )}
            </>
          )}
        </Field>
      </div>
    </div>
  );
};

interface NumberFieldProps {
  name: string;
  label?: string;
  placeholder?: string;
  step?: number;
  disabled?: boolean;
  required?: boolean;
  min?: number;
  max?: number;
  hidden?: boolean;
  columns?: number;
  tooltip?: React.JSX.Element;
  className?: string;
  isDecimal?: boolean;
}

export const NumberField = ({
  name,
  label,
  placeholder,
  step,
  min,
  max,
  hidden,
  columns = 6,
  tooltip,
  disabled,
  required,
  isDecimal,
  className = ""
}: NumberFieldProps) => (
    <div
    className={classNames(
      className,
      "col-span-12",
      hidden ? "hidden" : "",
      columns ? `sm:col-span-${columns}` : ""
    )}
  >
    <label
      htmlFor={name}
      className="flex ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide"
    >
      {tooltip ? (
        <DocsTooltip label={label}>{tooltip}</DocsTooltip>
      ) : label}
    </label>

    <Field name={name} type="number">
      {({ field, meta, form }: FieldProps<number>) => (
        <div className="sm:col-span-2">
          <input
            type="number"
            {...field}
            step={step}
            min={min}
            max={max}
            inputMode={isDecimal ? "decimal" : "numeric"}
            required={required}
            className={classNames(
              meta.touched && meta.error
                ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
              "mt-1 block w-full border rounded-md",
              disabled
                ? "bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400 cursor-not-allowed"
                : "bg-gray-100 dark:bg-gray-815 dark:text-gray-100"
            )}
            placeholder={placeholder}
            disabled={disabled}
            onChange={event => {
              // safeguard and validation if user removes the number
              // it will then set 0 by default. Formik can't handle this properly
              if (event.target.value == "") {
                form.setFieldValue(field.name, 0);
                return;
              }
              if (isDecimal) {
                form.setFieldValue(field.name, parseFloat(event.target.value));
              } else {
                form.setFieldValue(field.name, parseInt(event.target.value));
              }
            }}
            onWheel={(event) => {
              if (event.currentTarget === document.activeElement) {
                event.currentTarget.blur();
                setTimeout(() => event.currentTarget.focus(), 0);
              }
            }}
          />
          {meta.touched && meta.error && (
            <div className="error">{meta.error}</div>
          )}
        </div>
      )}
    </Field>
  </div>
);
