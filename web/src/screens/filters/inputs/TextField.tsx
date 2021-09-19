import { Field } from "formik";
import React from "react";
import { classNames } from "../../../styles/utils";

type COL_WIDTHS = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;

interface Props {
    name: string;
    label?: string;
    placeholder?: string;
    columns?: COL_WIDTHS;
    className?: string;
    autoComplete?: string;
}

const TextField: React.FC<Props> = ({ name, label, placeholder, columns, className, autoComplete }) => (
    <div
        className={classNames(
            columns ? `col-span-${columns}` : "col-span-12"
        )}
    >
        {label && (
            <label htmlFor={name} className="block text-xs font-bold text-gray-700 uppercase tracking-wide">
                {label}
            </label>
        )}
        <Field name={name}>
            {({
                field,
                meta,
            }: any) => (
                <div>
                    <input
                        {...field}
                        id={name}
                        type="text"
                        autoComplete={autoComplete}
                        className="mt-2 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-light-blue-500 focus:border-light-blue-500 sm:text-sm"
                        placeholder={placeholder}
                    />

                    {meta.touched && meta.error && (
                        <div className="error">{meta.error}</div>
                    )}
                </div>
            )}
        </Field>
    </div>
)

export default TextField;
