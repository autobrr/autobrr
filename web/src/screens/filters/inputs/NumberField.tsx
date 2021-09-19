import { Field } from "formik";
import React from "react";
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

        <Field name={name} type="number">
            {({
                field,
                form: { touched, errors },
                meta,
            }: any) => (
                <div className="sm:col-span-2">
                    <input
                        type="number"
                        {...field}
                        className={classNames(
                            meta.touched && meta.error
                                ? "focus:ring-red-500 focus:border-red-500 border-red-500"
                                : "focus:ring-indigo-500 focus:border-indigo-500 border-gray-300",
                            "block w-full shadow-sm sm:text-sm rounded-md"
                        )}
                        placeholder={placeholder}
                    />
                    {meta.touched && meta.error && (
                        <div className="error">{meta.error}</div>
                    )}
                </div>

            )}
        </Field>
    </div>
);

export default NumberField;
