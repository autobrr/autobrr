import { Field, FieldProps } from "formik";
import { classNames } from "../../utils";
import { InformationCircleIcon, EyeIcon, EyeSlashIcon } from "@heroicons/react/24/solid";
import { useToggle } from "../../hooks/hooks";
import { log } from "util";

type COL_WIDTHS = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;

interface TextFieldProps {
    name: string;
    defaultValue?: string;
    label?: string;
    placeholder?: string;
    id?: string;
    columns?: COL_WIDTHS;
    autoComplete?: string;
    hidden?: boolean;
    disabled?: boolean;
}

export const TextField = ({
  name,
  defaultValue,
  label,
  id, // this is for tooltips to identify their anchorpoint
  placeholder,
  columns,
  autoComplete,
  hidden,
  disabled
}: TextFieldProps) => (
  <div
    className={classNames(
      hidden ? "hidden" : "",
      columns ? `col-span-${columns}` : "col-span-12"
    )}
  >
    {label && (
      <label htmlFor={name} className="float-left mb-2 block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide" id={id}>
        {label}
      </label>

    )}
    <Field name={name}>
      {({
        field,
        meta
      }: FieldProps) => (
        <div>
          <input
            {...field}
            name={name}
            type="text"
            defaultValue={defaultValue}
            autoComplete={autoComplete}
            className={classNames(
              meta.touched && meta.error ? "focus:ring-red-500 focus:border-red-500 border-red-500" : "focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700",
              disabled ? "bg-gray-100 dark:bg-gray-700 cursor-not-allowed" : "dark:bg-gray-800",
              "mt-2 block w-full dark:text-gray-100 rounded-md"
            )}
            disabled={disabled}
            placeholder={placeholder}
          />

          {meta.touched && meta.error && (
            <p className="error text-sm text-red-600 mt-1">* {meta.error}</p>
          )}
        </div>
      )}
    </Field>
  </div>
);

interface TextFieldIconProps {
  name: string;
  defaultValue?: string;
  label?: string;
  placeholder?: string;
  id?: string;
  columns?: COL_WIDTHS;
  autoComplete?: string;
  hidden?: boolean;
  disabled?: boolean;
}

export const TextFieldIcon = ({
  name,
  defaultValue,
  label,
  id, // this is for tooltips to identify their anchorpoint
  placeholder,
  columns,
  autoComplete,
  hidden,
  disabled
}: TextFieldIconProps) => (
  <div
    className={classNames(
      hidden ? "hidden" : "",
      columns ? `col-span-${columns}` : "col-span-12"
    )}
  >
    {label && (
      <label htmlFor={name} className="float-left mb-2 block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide" id={id}>
        {label}
        <svg className="float-right ml-1 -mt-1 h-5 w-5 text-gray-500" width="800px" height="800px" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
          <path fill="#333" d="M512 64C264.6 64 64 264.6 64 512s200.6 448 448 448 448-200.6 448-448S759.4 64 512 64zm0 820c-205.4 0-372-166.6-372-372s166.6-372 372-372 372 166.6 372 372-166.6 372-372 372z"/>
          <path fill="#E6E6E6" d="M512 140c-205.4 0-372 166.6-372 372s166.6 372 372 372 372-166.6 372-372-166.6-372-372-372zm32 588c0 4.4-3.6 8-8 8h-48c-4.4 0-8-3.6-8-8V456c0-4.4 3.6-8 8-8h48c4.4 0 8 3.6 8 8v272zm-32-344a48.01 48.01 0 0 1 0-96 48.01 48.01 0 0 1 0 96z"/>
          <path fill="#333" d="M464 336a48 48 0 1 0 96 0 48 48 0 1 0-96 0zm72 112h-48c-4.4 0-8 3.6-8 8v272c0 4.4 3.6 8 8 8h48c4.4 0 8-3.6 8-8V456c0-4.4-3.6-8-8-8z"/>
        </svg>
      </label>

    )}
    <Field name={name}>
      {({
        field,
        meta
      }: FieldProps) => (
        <div>
          <input
            {...field}
            name={name}
            type="text"
            defaultValue={defaultValue}
            autoComplete={autoComplete}
            className={classNames(
              meta.touched && meta.error ? "focus:ring-red-500 focus:border-red-500 border-red-500" : "focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700",
              disabled ? "bg-gray-100 dark:bg-gray-700 cursor-not-allowed" : "dark:bg-gray-800",
              "mt-2 block w-full dark:text-gray-100 rounded-md"
            )}
            disabled={disabled}
            placeholder={placeholder}
          />

          {meta.touched && meta.error && (
            <p className="error text-sm text-red-600 mt-1">* {meta.error}</p>
          )}
        </div>
      )}
    </Field>
  </div>
);


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
  disabled
}: TextAreaProps) => (
  <div
    className={classNames(
      hidden ? "hidden" : "",
      columns ? `col-span-${columns}` : "col-span-12"
    )}
  >
    {label && (
      <label htmlFor={name} className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
        {label}
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
              meta.touched && meta.error ? "focus:ring-red-500 focus:border-red-500 border-red-500" : "focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700",
              disabled ? "bg-gray-100 dark:bg-gray-700 cursor-not-allowed" : "dark:bg-gray-800",
              "mt-2 block w-full dark:text-gray-100 rounded-md"
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
}

export const PasswordField = ({
  name,
  label,
  placeholder,
  defaultValue,
  columns,
  autoComplete,
  help,
  required
}: PasswordFieldProps) => {
  const [isVisible, toggleVisibility] = useToggle(false);

  return (
    <div
      className={classNames(
        columns ? `col-span-${columns}` : "col-span-12"
      )}
    >
      {label && (
        <label htmlFor={name} className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
          {label} {required && <span className="text-gray-500">*</span>}
        </label>
      )}
      <Field name={name} defaultValue={defaultValue}>
        {({
          field,
          meta
        }: FieldProps) => (
          <div className="sm:col-span-2 relative">
            <input
              {...field}
              id={name}
              type={isVisible ? "text" : "password"}
              autoComplete={autoComplete}
              className={classNames(meta.touched && meta.error ? "focus:ring-red-500 focus:border-red-500 border-red-500" : "focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700", "mt-2 block w-full dark:bg-gray-800 dark:text-gray-100 rounded-md")}
              placeholder={placeholder}
            />

            <div className="absolute inset-y-0 right-0 px-3 flex items-center" onClick={toggleVisibility}>
              {!isVisible ? <EyeIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" /> : <EyeSlashIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" />}
            </div>

            {help && (
              <p className="mt-2 text-sm text-gray-500" id="email-description">{help}</p>
            )}

            {meta.touched && meta.error && (
              <p className="error text-sm text-red-600 mt-1">* {meta.error}</p>
            )}
          </div>
        )}
      </Field>
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
}

export const NumberField = ({
  name,
  label,
  placeholder,
  step,
  min,
  max,
  disabled,
  required
}: NumberFieldProps) => (
  <div className="col-span-12 sm:col-span-6">
    <label htmlFor={name} className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
      {label}
    </label>

    <Field name={name} type="number">
      {({
        field,
        meta,
        form
      }: FieldProps) => (
        <div className="sm:col-span-2">
          <input
            type="number"
            {...field}
            step={step}
            min={min}
            max={max}
            inputMode="numeric"
            required={required}
            className={classNames(
              meta.touched && meta.error
                ? "focus:ring-red-500 focus:border-red-500 border-red-500"
                : "focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300",
              "mt-2 block w-full border border-gray-300 dark:border-gray-700 dark:text-gray-100 rounded-md",
              disabled ? "bg-gray-100 dark:bg-gray-700 cursor-not-allowed" : "dark:bg-gray-800"
            )}
            placeholder={placeholder}
            disabled={disabled}
          />
          {meta.touched && meta.error && (
            <div className="error">{meta.error}</div>
          )}
        </div>

      )}
    </Field>
  </div>
);
