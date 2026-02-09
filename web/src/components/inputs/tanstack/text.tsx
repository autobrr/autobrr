/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { EyeIcon, EyeSlashIcon, CheckCircleIcon, XCircleIcon } from "@heroicons/react/24/solid";
import TextareaAutosize from "react-textarea-autosize";

import { useFieldContext } from "@app/lib/form";
import { useToggle } from "@hooks/hooks";
import { DocsTooltip } from "@components/tooltips/DocsTooltip";
import { classNames } from "@utils";

type COL_WIDTHS = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;

interface TextFieldProps {
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
  const field = useFieldContext<string>();

  return (
    <div
      className={classNames(
        "col-span-12",
        hidden ? "hidden" : "",
        columns ? `sm:col-span-${columns}` : ""
      )}
    >
      {label && (
        <label htmlFor={field.name} className="flex ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide">
          {tooltip ? (
            <DocsTooltip label={label}>{tooltip}</DocsTooltip>
          ) : label}
          {required ? (
            <span className="ml-1 text-red-500">*</span>
          ) : null}
        </label>
      )}
      <input
        id={field.name}
        name={field.name}
        type="text"
        defaultValue={defaultValue}
        value={field.state.value ?? ""}
        onChange={(e) => field.handleChange(e.target.value)}
        onBlur={field.handleBlur}
        autoComplete={autoComplete}
        className={classNames(
          field.state.meta.isTouched && field.state.meta.errors.length > 0
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

      {field.state.meta.isTouched && field.state.meta.errors.length > 0 && (
        <p className="error text-sm text-red-600 mt-1">* {field.state.meta.errors[0]}</p>
      )}
    </div>
  );
};

interface RegexFieldProps {
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

export const RegexField = ({
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
  const field = useFieldContext<string>();

  const currentValue = field.state.value ?? "";
  const isRegexValid = validRegex(currentValue);
  const hasRegexError = useRegex && !isRegexValid;

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
          htmlFor={field.name}
          className="flex ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide"
        >
          {tooltip ? (
            <DocsTooltip label={label}>{tooltip}</DocsTooltip>
          ) : label}
        </label>
      )}
      <div className="relative">
        <input
          id={field.name}
          name={field.name}
          type="text"
          defaultValue={defaultValue}
          value={currentValue}
          onChange={(e) => field.handleChange(e.target.value)}
          onBlur={field.handleBlur}
          autoComplete={autoComplete}
          className={classNames(
            hasRegexError
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
              {isRegexValid ? (
                <CheckCircleIcon className="h-8 w-8 mb-2.5 pl-1 text-green-500 right-2 absolute transform -translate-y-1/2" aria-hidden="true" style={{ overflow: "hidden" }} />
              ) : (
                <XCircleIcon className="h-8 w-8 mb-2.5 pl-1 text-red-500 right-2 absolute transform -translate-y-1/2" aria-hidden="true" style={{ overflow: "hidden" }} />
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export const RegexTextAreaField = ({
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
  const field = useFieldContext<string>();

  const currentValue = field.state.value ?? "";
  const isRegexValid = validRegex(currentValue);
  const hasRegexError = useRegex && !isRegexValid;

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
          htmlFor={field.name}
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
      <div className="relative">
        <TextareaAutosize
          id={field.name}
          name={field.name}
          maxRows={10}
          defaultValue={defaultValue}
          value={currentValue}
          onChange={(e) => field.handleChange(e.target.value)}
          onBlur={field.handleBlur}
          autoComplete={autoComplete}
          className={classNames(
            hasRegexError
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
              {isRegexValid ? (
                <CheckCircleIcon className="h-8 w-8 mb-2.5 pl-1 text-green-500 right-2 absolute transform -translate-y-1/2" aria-hidden="true" style={{ overflow: "hidden" }} />
              ) : (
                <XCircleIcon className="h-8 w-8 mb-2.5 pl-1 text-red-500 right-2 absolute transform -translate-y-1/2" aria-hidden="true" style={{ overflow: "hidden" }} />
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

interface TextAreaProps {
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
  const field = useFieldContext<string>();

  return (
    <div
      className={classNames(
        "col-span-12",
        hidden ? "hidden" : "",
        columns ? `sm:col-span-${columns}` : ""
      )}
    >
      {label && (
        <label htmlFor={field.name} className="flex ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide">
          {tooltip ? (
            <DocsTooltip label={label}>{tooltip}</DocsTooltip>
          ) : label}
        </label>
      )}
      <div>
        <textarea
          id={field.name}
          name={field.name}
          rows={rows}
          defaultValue={defaultValue}
          value={field.state.value ?? ""}
          onChange={(e) => field.handleChange(e.target.value)}
          onBlur={field.handleBlur}
          autoComplete={autoComplete}
          className={classNames(
            field.state.meta.isTouched && field.state.meta.errors.length > 0
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

        {field.state.meta.isTouched && field.state.meta.errors.length > 0 && (
          <p className="error text-sm text-red-600 mt-1">* {field.state.meta.errors[0]}</p>
        )}
      </div>
    </div>
  );
};

interface TextAreaAutoResizeProps {
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
  const field = useFieldContext<string>();

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
        <label htmlFor={field.name} className="flex ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide">
          {tooltip ? (
            <DocsTooltip label={label}>{tooltip}</DocsTooltip>
          ) : label}
        </label>
      )}
      <div>
        <TextareaAutosize
          id={field.name}
          name={field.name}
          rows={rows}
          maxRows={10}
          defaultValue={defaultValue}
          value={field.state.value ?? ""}
          onChange={(e) => field.handleChange(e.target.value)}
          onBlur={field.handleBlur}
          autoComplete={autoComplete}
          className={classNames(
            field.state.meta.isTouched && field.state.meta.errors.length > 0
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

        {field.state.meta.isTouched && field.state.meta.errors.length > 0 && (
          <p className="error text-sm text-red-600 mt-1">* {field.state.meta.errors[0]}</p>
        )}
      </div>
    </div>
  );
};

interface PasswordFieldProps {
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
  label,
  placeholder,
  defaultValue,
  columns,
  autoComplete,
  help,
  tooltip,
  required
}: PasswordFieldProps) => {
  const field = useFieldContext<string>();
  const [isVisible, toggleVisibility] = useToggle(false);

  return (
    <div
      className={classNames(
        "col-span-12",
        columns ? `sm:col-span-${columns}` : ""
      )}
    >
      {label && (
        <label htmlFor={field.name} className="flex ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide">
          {tooltip ? (
            <DocsTooltip label={label}>{tooltip}</DocsTooltip>
          ) : (
            label
          )}
          {required && <span className="text-red-500">*</span>}
        </label>
      )}
      <div>
        <div className="sm:col-span-2 relative">
          <input
            id={field.name}
            name={field.name}
            type={isVisible ? "text" : "password"}
            defaultValue={defaultValue}
            value={field.state.value ?? ""}
            onChange={(e) => field.handleChange(e.target.value)}
            onBlur={field.handleBlur}
            autoComplete={autoComplete}
            className={classNames(
              field.state.meta.isTouched && field.state.meta.errors.length > 0
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

        {field.state.meta.isTouched && field.state.meta.errors.length > 0 && (
          <p className="error text-sm text-red-600 mt-1">* {field.state.meta.errors[0]}</p>
        )}
      </div>
    </div>
  );
};

interface NumberFieldProps {
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
  const field = useFieldContext<number>();

  return (
    <div className={classNames(className, "col-span-12 sm:col-span-6")}>
      <label
        htmlFor={field.name}
        className="flex ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide"
      >
        {tooltip ? (
          <DocsTooltip label={label}>{tooltip}</DocsTooltip>
        ) : label}
      </label>

      <div className="sm:col-span-2">
        <input
          id={field.name}
          name={field.name}
          type="number"
          value={field.state.value ?? 0}
          step={step}
          min={min}
          max={max}
          inputMode={isDecimal ? "decimal" : "numeric"}
          required={required}
          className={classNames(
            field.state.meta.isTouched && field.state.meta.errors.length > 0
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
            // safeguard and validation if user removes the number
            // it will then set 0 by default.
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
          onBlur={field.handleBlur}
          onWheel={(event) => {
            if (event.currentTarget === document.activeElement) {
              event.currentTarget.blur();
              setTimeout(() => event.currentTarget.focus(), 0);
            }
          }}
        />
        {field.state.meta.isTouched && field.state.meta.errors.length > 0 && (
          <div className="error">{field.state.meta.errors[0]}</div>
        )}
      </div>
    </div>
  );
};
