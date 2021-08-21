import { Field } from "react-final-form";
import React from "react";
import Error from "../Error";
import { classNames } from "../../../styles/utils";

interface Props {
  name: string;
  label?: string;
  placeholder?: string;
  className?: string;
  required?: boolean;
}

const NumberField: React.FC<Props> = ({
  name,
  label,
  placeholder,
  required,
  className,
}) => (
  <div className="col-span-12 sm:col-span-6">
    <label htmlFor={name} className="block text-sm font-medium text-gray-700">
      {label}
    </label>

    <Field name={name} parse={(v) => v & parseInt(v, 10)}>
      {({ input, meta }) => (
        <div className="sm:col-span-2">
          <input
            type="number"
            {...input}
            className={classNames(
              meta.touched && meta.error
                ? "focus:ring-red-500 focus:border-red-500 border-red-500"
                : "focus:ring-indigo-500 focus:border-indigo-500 border-gray-300",
              "block w-full shadow-sm sm:text-sm rounded-md"
            )}
            placeholder={placeholder}
          />
          <Error name={name} classNames="block text-red-500 mt-2" />
        </div>
      )}
    </Field>
  </div>
);

export default NumberField;
