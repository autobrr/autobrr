/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { EyeIcon, EyeSlashIcon, CheckCircleIcon, XCircleIcon } from "@heroicons/react/24/solid";
import TextareaAutosize from "react-textarea-autosize";
import { useTranslation } from "react-i18next";

import { useToggle } from "@hooks/hooks";
import { useFormContext, fieldHasError } from "@hooks/form";
import { DocsTooltip } from "@components/tooltips/DocsTooltip";
import { classNames } from "@utils";
import { ErrorField } from "./common";

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
}: TextFieldProps) => {
  const form = useFormContext();

  return (
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
      <form.Field name={name}>
        {(field) => (
          <>
            <input
              id={name}
              name={name}
              type="text"
              value={field.state.value ?? defaultValue ?? ""}
              onChange={(e) => field.handleChange(e.target.value)}
              onBlur={field.handleBlur}
              autoComplete={autoComplete}
              className={classNames(
                fieldHasError(field.state.meta)
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

            <ErrorField meta={field.state.meta} classNames="error text-sm text-red-600 mt-1" />
          </>
        )}
      </form.Field>
    </div>
  );
};

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

// The backend compiles patterns with Go's RE2, which lacks these PCRE features.
const validRegex = (pattern: string) => {
  // lookahead and lookbehind assertions
  if (/\(\?<=|\(\?<!|\(\?=|\(\?!/.test(pattern)) {
    return false;
  }

  // atomic groups
  if (/\(\?>/.test(pattern)) {
    return false;
  }

  // recursive patterns
  if (/\(\?(R|0)\)/.test(pattern)) {
    return false;
  }

  // possessive quantifiers
  if (/[*+?]{1}\+|\{[0-9]+,[0-9]*\}\+/.test(pattern)) {
    return false;
  }

  // control verbs
  if (/\\g</.test(pattern)) {
    return false;
  }

  // conditionals
  if (/\(\?\((\?[=!][^)]*)\)[^)]*\|?[^)]*\)/.test(pattern)) {
    return false;
  }

  // named backreferences
  if (/\\k</.test(pattern)) {
    return false;
  }

  try {
    new RegExp(pattern);
    return true;
  } catch {
    return false;
  }
};

const RegexStatusIcon = ({ valid }: { valid: boolean }) => (
  <div className="relative">
    <div className="flex float-right items-center">
      {valid ? (
        <CheckCircleIcon className="h-8 w-8 mb-2.5 pl-1 text-green-500 right-2 absolute transform -translate-y-1/2" aria-hidden="true" style={{ overflow: "hidden" }} />
      ) : (
        <XCircleIcon className="h-8 w-8 mb-2.5 pl-1 text-red-500 right-2 absolute transform -translate-y-1/2" aria-hidden="true" style={{ overflow: "hidden" }} />
      )}
    </div>
  </div>
);

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
  const { t } = useTranslation("common");
  const form = useFormContext();

  const validateRegexp = (value: string) => {
    if (!useRegex || validRegex(value)) {
      return undefined;
    }

    return t("input.invalidRegex");
  };

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
      <form.Field
        name={name}
        validators={{ onChange: ({ value }) => validateRegexp(value ?? "") }}
      >
        {(field) => {
          const valid = validRegex(field.state.value ?? "");

          return (
            <div className="relative">
              <input
                id={name}
                name={name}
                type="text"
                value={field.state.value ?? defaultValue ?? ""}
                onChange={(e) => field.handleChange(e.target.value)}
                onBlur={field.handleBlur}
                autoComplete={autoComplete}
                className={classNames(
                  useRegex && !valid
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
              {useRegex && <RegexStatusIcon valid={valid} />}
            </div>
          );
        }}
      </form.Field>

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
  const { t } = useTranslation("common");
  const form = useFormContext();

  const validateRegexp = (value: string) => {
    if (!useRegex || validRegex(value)) {
      return undefined;
    }

    return t("input.invalidRegex");
  };

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
      <form.Field
        name={name}
        validators={{ onChange: ({ value }) => validateRegexp(value ?? "") }}
      >
        {(field) => {
          const valid = validRegex(field.state.value ?? "");

          return (
            <div className="relative">
              <TextareaAutosize
                id={name}
                name={name}
                maxRows={10}
                value={field.state.value ?? defaultValue ?? ""}
                onChange={(e) => field.handleChange(e.target.value)}
                onBlur={field.handleBlur}
                autoComplete={autoComplete}
                className={classNames(
                  useRegex && !valid
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
              {useRegex && <RegexStatusIcon valid={valid} />}
            </div>
          );
        }}
      </form.Field>

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
}: TextAreaProps) => {
  const form = useFormContext();

  return (
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
      <form.Field name={name}>
        {(field) => (
          <div>
            <textarea
              id={name}
              name={name}
              rows={rows}
              value={field.state.value ?? defaultValue ?? ""}
              onChange={(e) => field.handleChange(e.target.value)}
              onBlur={field.handleBlur}
              autoComplete={autoComplete}
              className={classNames(
                fieldHasError(field.state.meta)
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

            <ErrorField meta={field.state.meta} classNames="error text-sm text-red-600 mt-1" />
          </div>
        )}
      </form.Field>
    </div>
  );
};

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
}: TextAreaAutoResizeProps) => {
  const form = useFormContext();

  return (
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
      <form.Field name={name}>
        {(field) => (
          <div>
            <TextareaAutosize
              id={name}
              name={name}
              rows={rows}
              maxRows={10}
              value={field.state.value ?? defaultValue ?? ""}
              onChange={(e) => field.handleChange(e.target.value)}
              onBlur={field.handleBlur}
              autoComplete={autoComplete}
              className={classNames(
                fieldHasError(field.state.meta)
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

            <ErrorField meta={field.state.meta} classNames="error text-sm text-red-600 mt-1" />
          </div>
        )}
      </form.Field>
    </div>
  );
};


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
  const form = useFormContext();

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
        <form.Field name={name}>
          {(field) => (
            <>
              <div className="sm:col-span-2 relative">
                <input
                  id={name}
                  name={name}
                  type={isVisible ? "text" : "password"}
                  value={field.state.value ?? defaultValue ?? ""}
                  onChange={(e) => field.handleChange(e.target.value)}
                  onBlur={field.handleBlur}
                  autoComplete={autoComplete}
                  className={classNames(
                    fieldHasError(field.state.meta)
                      ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                      : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
                    "mt-1 block w-full rounded-md bg-gray-100 dark:bg-gray-815 dark:text-gray-100"
                  )}
                  placeholder={placeholder}
                />

                <div className="absolute inset-y-0 right-0 px-3 flex items-center cursor-pointer" onClick={toggleVisibility}>
                  {!isVisible ? <EyeIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" />
                    : <EyeSlashIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" />}
                </div>
              </div>
              {help && (
                <p className="mt-2 text-sm text-gray-500" id="email-description">{help}</p>
              )}

              <ErrorField meta={field.state.meta} classNames="error text-sm text-red-600 mt-1" />
            </>
          )}
        </form.Field>
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
  tooltip,
  disabled,
  required,
  isDecimal,
  className = ""
}: NumberFieldProps) => {
  const form = useFormContext();

  return (
    <div className={classNames(className, "col-span-12 sm:col-span-6")}>
      <label
        htmlFor={name}
        className="flex ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide"
      >
        {tooltip ? (
          <DocsTooltip label={label}>{tooltip}</DocsTooltip>
        ) : label}
      </label>

      <form.Field name={name}>
        {(field) => (
          <div className="sm:col-span-2">
            <input
              type="number"
              id={name}
              name={name}
              value={field.state.value ?? ""}
              onBlur={field.handleBlur}
              step={step}
              min={min}
              max={max}
              inputMode={isDecimal ? "decimal" : "numeric"}
              required={required}
              className={classNames(
                fieldHasError(field.state.meta)
                  ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                  : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
                "mt-1 block w-full border rounded-md",
                disabled
                  ? "bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400 cursor-not-allowed"
                  : "bg-gray-100 dark:bg-gray-815 dark:text-gray-100"
              )}
              placeholder={placeholder}
              disabled={disabled}
              onChange={(event) => {
                // An emptied input would otherwise store NaN
                if (event.target.value == "") {
                  field.handleChange(0);
                  return;
                }
                if (isDecimal) {
                  field.handleChange(parseFloat(event.target.value));
                } else {
                  field.handleChange(parseInt(event.target.value));
                }
              }}
              onWheel={(event) => {
                if (event.currentTarget === document.activeElement) {
                  event.currentTarget.blur();
                  setTimeout(() => event.currentTarget.focus(), 0);
                }
              }}
            />
            <ErrorField meta={field.state.meta} classNames="error" />
          </div>
        )}
      </form.Field>
    </div>
  );
};
