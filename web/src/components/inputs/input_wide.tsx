import type { FieldProps, FieldValidator } from "formik";
import { Field } from "formik";
import { classNames } from "../../utils";
import { useToggle } from "../../hooks/hooks";
import { EyeIcon, EyeSlashIcon } from "@heroicons/react/24/solid";
import { Switch } from "@headlessui/react";
import { ErrorField } from "./common";

interface TextFieldWideProps {
    name: string;
    label?: string;
    help?: string;
    placeholder?: string;
    defaultValue?: string;
    required?: boolean;
    hidden?: boolean;
    validate?: FieldValidator;
}

export const TextFieldWide = ({
  name,
  label,
  help,
  placeholder,
  defaultValue,
  required,
  hidden,
  validate
}: TextFieldWideProps) => (
  <div hidden={hidden} className="space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
    <div>
      <label htmlFor={name} className="block text-sm font-medium text-gray-900 dark:text-white sm:mt-px sm:pt-2">
        {label} {required && <span className="text-gray-500">*</span>}
      </label>
    </div>
    <div className="sm:col-span-2">
      <Field
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
            className={classNames(meta.touched && meta.error ? "focus:ring-red-500 focus:border-red-500 border-red-500" : "focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700", "block w-full shadow-sm dark:bg-gray-800 sm:text-sm dark:text-white rounded-md")}
            placeholder={placeholder}
            hidden={hidden}
          />
        )}
      </Field>
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
    defaultVisible?: boolean;
    validate?: FieldValidator;
}

export const PasswordFieldWide = ({
  name,
  label,
  placeholder,
  defaultValue,
  help,
  required,
  defaultVisible,
  validate
}: PasswordFieldWideProps) => {
  const [isVisible, toggleVisibility] = useToggle(defaultVisible);

  return (
    <div className="space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label htmlFor={name} className="block text-sm font-medium text-gray-900 dark:text-white sm:mt-px sm:pt-2">
          {label} {required && <span className="text-gray-500">*</span>}
        </label>
      </div>
      <div className="sm:col-span-2">
        <Field
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
                className={classNames(meta.touched && meta.error ? "focus:ring-red-500 focus:border-red-500 border-red-500" : "focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700", "block w-full pr-10 dark:bg-gray-800 shadow-sm dark:text-gray-100 sm:text-sm rounded-md")}
                placeholder={placeholder}
              />
              <div className="absolute inset-y-0 right-0 px-3 flex items-center" onClick={toggleVisibility}>
                {!isVisible ? <EyeIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" /> : <EyeSlashIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" />}
              </div>
            </div>
          )}
        </Field>
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
}

export const NumberFieldWide = ({
  name,
  label,
  placeholder,
  help,
  defaultValue,
  required
}: NumberFieldWideProps) => (
  <div className="px-4 space-y-1 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:py-4">
    <div>
      <label
        htmlFor={name}
        className="block text-sm font-medium text-gray-900 dark:text-white sm:mt-px sm:pt-2"
      >
        {label} {required && <span className="text-gray-500">*</span>}
      </label>
    </div>
    <div className="sm:col-span-2">
      <Field
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
                ? "focus:ring-red-500 focus:border-red-500 border-red-500"
                : "focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700",
              "block w-full shadow-sm dark:bg-gray-800 sm:text-sm dark:text-white rounded-md"
            )}
            placeholder={placeholder}
          />
        )}
      </Field>
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
}

export const SwitchGroupWide = ({
  name,
  label,
  description,
  defaultValue
}: SwitchGroupWideProps) => (
  <ul className="mt-2 px-4 divide-y divide-gray-200 dark:divide-gray-700">
    <Switch.Group as="li" className="py-4 flex items-center justify-between">
      <div className="flex flex-col">
        <Switch.Label as="p" className="text-sm font-medium text-gray-900 dark:text-white"
          passive>
          {label}
        </Switch.Label>
        {description && (
          <Switch.Description className="text-sm text-gray-500 dark:text-gray-700">
            {description}
          </Switch.Description>
        )}
      </div>

      <Field
        name={name}
        defaultValue={defaultValue as boolean}
        type="checkbox"
      >
        {({ field, form }: FieldProps) => (
          <Switch
            {...field}
            type="button"
            value={field.value}
            checked={field.checked ?? false}
            onChange={(value: unknown) => {
              form.setFieldValue(field?.name ?? "", value);
            }}
            className={classNames(
              field.value ? "bg-blue-500 dark:bg-blue-500" : "bg-gray-200 dark:bg-gray-500",
              "ml-4 relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            )}
          >
            <span className="sr-only">Use setting</span>
            <span
              aria-hidden="true"
              className={classNames(
                field.value ? "translate-x-5" : "translate-x-0",
                "inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200"
              )}
            />
          </Switch>
        )}
      </Field>
    </Switch.Group>
  </ul>
);

interface SwitchGroupWideRedProps {
  name: string;
  label: string;
  description?: string;
  defaultValue?: boolean;
  className?: string;
}

export const SwitchGroupWideRed = ({
  name,
  label,
  description,
  defaultValue
}: SwitchGroupWideRedProps) => (
  <ul className="mt-2 px-4 divide-y divide-gray-200 dark:divide-gray-700">
    <Switch.Group as="li" className="py-4 flex items-center justify-between">
      <div className="flex flex-col">
        <Switch.Label as="p" className="text-sm font-medium text-gray-900 dark:text-white"
          passive>
          {label}
        </Switch.Label>
        {description && (
          <Switch.Description className="text-sm text-gray-500 dark:text-gray-700">
            {description}
          </Switch.Description>
        )}
      </div>

      <Field
        name={name}
        defaultValue={defaultValue as boolean}
        type="checkbox"
      >
        {({ field, form }: FieldProps) => (
          <Switch
            {...field}
            type="button"
            value={field.value}
            checked={field.checked ?? false}
            onChange={(value: unknown) => {
              form.setFieldValue(field?.name ?? "", value);
            }}
            className={classNames(
              field.value ? "bg-blue-500 dark:bg-blue-500" : "bg-red-500 dark:bg-red-500",
              "ml-4 relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            )}
          >
            <span className="sr-only">Use setting</span>
            <span
              aria-hidden="true"
              className={classNames(
                field.value ? "translate-x-5" : "translate-x-0",
                "inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200"
              )}
            />
          </Switch>
        )}
      </Field>
    </Switch.Group>
  </ul>
);

