import React, {
  FC,
  forwardRef,
  ReactNode
} from "react";
import {
  FieldError,
  UseFormRegister,
  Path,
  RegisterOptions, DeepMap
} from "react-hook-form";
import { classNames, get } from "../../utils";
import { useToggle } from "../../hooks/hooks";
import { EyeIcon, EyeOffIcon } from "@heroicons/react/solid";
import { ErrorMessage } from "@hookform/error-message";

export type FormErrorMessageProps = {
  className?: string;
  children: ReactNode;
};

export const FormErrorMessage: FC<FormErrorMessageProps> = ({
  children,
  className
}) => (
  <p
    className={classNames(
      "mt-1 text-sm text-left block text-red-600",
      className ?? ""
    )}
  >
    {children}
  </p>
);

export type InputType = "text" | "email" | "password";
export type InputAutoComplete = "username" | "current-password";
export type InputColumnWidth = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;

export type InputProps = {
  id: string;
  name: string;
  label: string;
  type?: InputType;
  className?: string;
  placeholder?: string;
  autoComplete?: InputAutoComplete;
  isHidden?: boolean;
  columnWidth?: InputColumnWidth;
};

// Using maps so that the full Tailwind classes can be seen for purging
// see https://tailwindcss.com/docs/optimizing-for-production#writing-purgeable-html

// const sizeMap: { [key in InputSize]: string } = {
//   medium: "p-3 text-base",
//   large: "p-4 text-base"
// };

export const Input: FC<InputProps> = forwardRef<HTMLInputElement, InputProps>(
  (
    {
      id,
      name,
      label,
      type ,
      className = "",
      placeholder,
      autoComplete,
      ...props
    },
    ref
  ) => {
    return (
      <input
        id={id}
        ref={ref}
        name={name}
        type={type}
        aria-label={label}
        placeholder={placeholder}
        className={className}
        autoComplete={autoComplete}
        {...props}
      />
    );
  }
);

export type FormInputProps<TFormValues> = {
  name: Path<TFormValues>;
  rules?: RegisterOptions;
  register?: UseFormRegister<TFormValues>;
  errors?: Partial<DeepMap<TFormValues, FieldError>>;
} & Omit<InputProps, "name">;

export const TextInput = <TFormValues extends Record<string, unknown>>({
  name,
  register,
  rules,
  errors,
  isHidden,
  columnWidth,
  ...props
}: FormInputProps<TFormValues>): JSX.Element => {
  // If the name is in a FieldArray, it will be 'fields.index.fieldName' and errors[name] won't return anything, so we are using lodash get
  const errorMessages = get(errors, name);
  const hasError = !!(errors && errorMessages);

  return (
    <div
      className={classNames(
        isHidden ? "hidden" : "",
        columnWidth ? `col-span-${columnWidth}` : "col-span-12"
      )}
    >
      {props.label && (
        <label htmlFor={name} className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
          {props.label}
        </label>
      )}
      <div>
        <Input
          name={name}
          aria-invalid={hasError}
          className={classNames(
            "mt-2 block w-full dark:bg-gray-800 dark:text-gray-100 rounded-md",
            hasError ? "focus:ring-red-500 focus:border-red-500 border-red-500" : "focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700"
          )}
          {...props}
          {...(register && register(name, rules))}
        />
        <ErrorMessage
          errors={errors}
          name={name as any}
          render={({ message }) => (
            <FormErrorMessage>{message}</FormErrorMessage>
          )}
        />
      </div>
    </div>
  );
};

export const PasswordInput = <TFormValues extends Record<string, unknown>>({
  name,
  register,
  rules,
  errors,
  isHidden,
  columnWidth,
  ...props
}: FormInputProps<TFormValues>): JSX.Element => {
  const [isVisible, toggleVisibility] = useToggle(false);

  // If the name is in a FieldArray, it will be 'fields.index.fieldName' and errors[name] won't return anything, so we are using lodash get
  const errorMessages = get(errors, name);
  const hasError = !!(errors && errorMessages);

  return (
    <div
      className={classNames(
        isHidden ? "hidden" : "",
        columnWidth ? `col-span-${columnWidth}` : "col-span-12"
      )}
    >
      {props.label && (
        <label htmlFor={name} className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
          {props.label}
        </label>
      )}
      <div>
        <div className="sm:col-span-2 relative">
          <Input
            name={name}
            aria-invalid={hasError}
            type={isVisible ? "text" : "password"}
            className={classNames(
              "mt-2 block w-full dark:bg-gray-800 dark:text-gray-100 rounded-md",
              hasError ? "focus:ring-red-500 focus:border-red-500 border-red-500" : "focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700"
            )}
            {...props}
            {...(register && register(name, rules))}
          />
          <div className="absolute inset-y-0 right-0 px-3 flex items-center" onClick={toggleVisibility}>
            {!isVisible ? <EyeIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" /> : <EyeOffIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" />}
          </div>
        </div>
        <ErrorMessage
          errors={errors}
          name={name as any}
          render={({ message }) => (
            <FormErrorMessage>{message}</FormErrorMessage>
          )}
        />
      </div>
    </div>
  );
};

