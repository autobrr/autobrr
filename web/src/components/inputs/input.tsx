import { Field, FieldProps } from "formik";
import { classNames } from "../../utils";
import { EyeIcon, EyeSlashIcon, CheckCircleIcon, XCircleIcon } from "@heroicons/react/24/solid";
import { useToggle } from "../../hooks/hooks";
import { CustomTooltip } from "../tooltips/CustomTooltip";

type COL_WIDTHS = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;

interface TextFieldProps {
  name: string;
  defaultValue?: string;
  label?: string;
  placeholder?: string;
  columns?: COL_WIDTHS;
  autoComplete?: string;
  hidden?: boolean;
  disabled?: boolean;
  tooltip?: JSX.Element;
}

export const TextField = ({
  name,
  defaultValue,
  label,
  placeholder,
  columns,
  autoComplete,
  hidden,
  tooltip,
  disabled
}: TextFieldProps) => (
  <div
    className={classNames(
      hidden ? "hidden" : "",
      columns ? `col-span-${columns}` : "col-span-12"
    )}
  >
    {label && (
      <label htmlFor={name} className="flex float-left mb-2 text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
        <div className="flex">
          {label}
          {tooltip && (
            <CustomTooltip anchorId={name}>{tooltip}</CustomTooltip>
          )}
        </div>
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
  tooltip?: JSX.Element;
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
  const golangRegex = /^((\\\*|\\\?|\\[^\s\\])+|\(\?i\))(\|((\\\*|\\\?|\\[^\s\\])+|\(\?i\)))*$/;

  const validRegex = (pattern: string) => {
    try {
      new RegExp(golangRegex.source + pattern);
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

  return (
    <div
      className={classNames(
        hidden ? "hidden" : "",
        columns ? `col-span-${columns}` : "col-span-12"
      )}
    >
      {label && (
        <label
          htmlFor={name}
          className="flex float-left mb-2 text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide"
        >
          <div className="flex">
            {label}
            {tooltip && <CustomTooltip anchorId={name}>{tooltip}</CustomTooltip>}
          </div>
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
                meta.touched && meta.error
                  ? "focus:ring-red-500 focus:border-red-500 border-red-500"
                  : "focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700",
                disabled
                  ? "bg-gray-100 dark:bg-gray-700 cursor-not-allowed"
                  : "dark:bg-gray-800",
                "mt-2 block w-full dark:text-gray-100 rounded-md"
              )}
              disabled={disabled}
              placeholder={placeholder}
            />
            {useRegex && (
              <div className="relative">
                <div className="flex float-right items-center">
                  {meta.touched && !meta.error ? (
                    <CheckCircleIcon className="dark:bg-gray-800 bg-white h-8 w-8 mb-2.5 pl-1 text-green-500 right-2 absolute transform -translate-y-1/2" aria-hidden="true" style={{ overflow: "hidden" }} />
                  ) : (
                    <XCircleIcon className="dark:bg-gray-800 bg-white h-8 w-8 mb-2.5 pl-1 text-red-500 right-2 absolute transform -translate-y-1/2" aria-hidden="true" style={{ overflow: "hidden" }} />
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
                      ? "focus:ring-red-500 focus:border-red-500 border-red-500"
                      : "focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700",
                    "mt-2 block w-full dark:bg-gray-800 dark:text-gray-100 rounded-md"
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
  tooltip?: JSX.Element;
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
  required
}: NumberFieldProps) => (
  <div className="col-span-12 sm:col-span-6">
    <label
      htmlFor={name}
      className="flex float-left mb-2 text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide"
    >
      <div className="flex">
        {label}
        {tooltip && <CustomTooltip anchorId={name}>{tooltip}</CustomTooltip>}
      </div>
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
            onChange={event => {
              // safeguard and validation if user removes the number
              // it will then set 0 by default. Formik can't handle this properly
              if (event.target.value == "") {
                form.setFieldValue(field.name, 0);
                return;
              }
              form.setFieldValue(field.name, parseInt(event.target.value)); // Convert the input value to an integer using parseInt() to ensure that the backend can properly parse the numberfield as an integer.
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
