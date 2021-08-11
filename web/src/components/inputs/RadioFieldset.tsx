import React from "react";
import {Field} from "react-final-form";

export interface radioFieldsetOption {
    label: string;
    description: string;
    value: string;
}

interface props {
    name: string;
    legend: string;
    options: radioFieldsetOption[];
}

const RadioFieldset: React.FC<props> = ({ name, legend,options }) => (
    <fieldset>
        <div className="space-y-2 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:items-start sm:px-6 sm:py-5">
            <div>
                <legend className="text-sm font-medium text-gray-900">{legend}</legend>
            </div>
            <div className="space-y-5 sm:col-span-2">
                <div className="space-y-5 sm:mt-0">

                    {options.map((opt, idx) => (
                        <div className="relative flex items-start" key={idx}>
                            <div className="absolute flex items-center h-5">
                                <Field
                                    name={name}
                                    type="radio"
                                    render={({input}) => (
                                        <input
                                            {...input}
                                            id={name}
                                            value={opt.value}
                                            // type="radio"
                                            checked={input.checked}
                                            className="focus:ring-indigo-500 h-4 w-4 text-indigo-600 border-gray-300"
                                        />
                                    )}
                                />
                            </div>
                            <div className="pl-7 text-sm">
                                <label htmlFor={opt.value} className="font-medium text-gray-900">
                                    {opt.label}
                                </label>
                                <p id={opt.value+"_description"} className="text-gray-500">
                                    {opt.description}
                                </p>
                            </div>
                        </div>
                    ))}

                </div>
            </div>
        </div>
    </fieldset>
)

export default RadioFieldset;
