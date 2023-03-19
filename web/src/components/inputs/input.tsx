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
  onChange?: (event: React.ChangeEvent<HTMLInputElement>) => void;
  isValidRegex?: (inputValue: string) => boolean;
  useRegex?: boolean;
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
  onChange,
  isValidRegex,
  useRegex,
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
    <Field name={name}>
      {({
        field,
        meta
      }: FieldProps) => {
        const isValid = isValidRegex ? isValidRegex(field.value) : undefined;
        return (
          <div className="relative">
            <input
              {...field}
              name={name}
              type="text"
              defaultValue={defaultValue}
              autoComplete={autoComplete}
              onChange={(event) => {
                field.onChange(event);
                if (onChange) {
                  onChange(event);
                }
              }}            
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
            {isValid !== undefined && useRegex && (
              <div className="relative">
                <div className="flex float-right items-center">
                  {isValid ? (
                    <CheckCircleIcon className="dark:bg-gray-800 bg-white h-6 w-6 mb-4 text-green-500 right-2 absolute transform -translate-y-2/4 z-10" aria-hidden="true" style={{ overflow: "hidden" }} />
                  ) : (
                    <XCircleIcon className="dark:bg-gray-800 bg-white h-6 w-6 mb-4 text-red-500 right-2 absolute transform -translate-y-2/4 z-10" aria-hidden="true" style={{ overflow: "hidden" }} />
                  )}
                </div>
              </div>
            )}

            {meta.touched && meta.error && (
              <p className="error text-sm text-red-600 mt-1">* {meta.error}</p>
            )}
          </div>
        );}
      }
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
    <label htmlFor={name} className="flex float-left mb-2 text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
      <div className="flex">
        {label}
        {tooltip && (
          <CustomTooltip anchorId={name}>{tooltip}</CustomTooltip>
        )}
      </div>
    </label>

    <Field name={name} type="number">
      {({
        field,
        meta
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
