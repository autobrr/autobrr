import {Field} from "react-final-form";
import React from "react";
import Error from "./Error";
import {classNames} from "../../styles/utils";

interface Props {
    name: string;
    label?: string;
    placeholder?: string;
    className?: string;
    required?: boolean;
}

const TextAreaWide: React.FC<Props> = ({name, label, placeholder, required, className}) => (
    <div
        className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
        <div>

            <label htmlFor={name} className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2">
                {label} {required && <span className="text-gray-500">*</span>}
            </label>
        </div>
        <div className="sm:col-span-2">
            <Field
                name={name}
                render={({input, meta}) => (
                    <textarea
                        {...input}
                        id={name}
                        className={classNames(meta.touched && meta.error ? "focus:ring-red-500 focus:border-red-500 border-red-500" : "focus:ring-indigo-500 focus:border-indigo-500 border-gray-300", "block w-full shadow-sm sm:text-sm   rounded-md")}
                        placeholder={placeholder}
                    />
                )}
            />
            <Error name={name} classNames="block text-red-500 mt-2"/>
        </div>
    </div>
)

export default TextAreaWide;
