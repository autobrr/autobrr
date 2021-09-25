import React from "react";
import { Field } from "formik";
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
        <label htmlFor={name} className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
            {label}
        </label>

        <Field name={name} type="number">
            {({
                field,
                meta,
            }: any) => (
                <div className="sm:col-span-2">
                    <input
                        type="number"
                        {...field}
                        className={classNames(
                            meta.touched && meta.error
                                ? "focus:ring-red-500 focus:border-red-500 border-red-500"
                                : "focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 border-gray-300",
                            "mt-2 block w-full dark:bg-gray-800 border border-gray-300 dark:border-gray-700 shadow-sm dark:text-gray-100 sm:text-sm rounded-md"
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
