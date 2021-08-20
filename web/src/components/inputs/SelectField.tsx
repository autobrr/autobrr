import {Field} from "react-final-form";
import {Listbox, Transition} from "@headlessui/react";
import {CheckIcon, SelectorIcon} from "@heroicons/react/solid";
import React, {Fragment} from "react";
import {classNames} from "../../styles/utils";


interface SelectOption {
    label: string;
    value: string;
}

interface props {
    name: string;
    label: string;
    optionDefaultText: string;
    options: SelectOption[];
}

function SelectField({name, label, optionDefaultText, options}: props) {
    return (
        <div className="col-span-6 sm:col-span-6">
            <Field
                name={name}
                type="select"
                render={({input}) => (
                    <Listbox value={input.value} onChange={input.onChange}>
                        {({open}) => (
                            <div
                                className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                                <Listbox.Label
                                    className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2">{label}</Listbox.Label>
                                <div className="mt-2 relative">
                                    <Listbox.Button
                                        className="bg-white relative w-full border border-gray-300 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm">
                                                                    <span
                                                                        className="block truncate">{input.value ? options.find(c => c.value === input.value)!.label : optionDefaultText}</span>
                                        <span
                                            className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
                <SelectorIcon className="h-5 w-5 text-gray-400" aria-hidden="true"/>
              </span>
                                    </Listbox.Button>

                                    <Transition
                                        show={open}
                                        as={Fragment}
                                        leave="transition ease-in duration-100"
                                        leaveFrom="opacity-100"
                                        leaveTo="opacity-0"
                                    >
                                        <Listbox.Options
                                            static
                                            className="absolute z-10 mt-1 w-full bg-white shadow-lg max-h-60 rounded-md py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm"
                                        >
                                            {options.map((opt) => (
                                                <Listbox.Option
                                                    key={opt.value}
                                                    className={({active}) =>
                                                        classNames(
                                                            active ? 'text-white bg-indigo-600' : 'text-gray-900',
                                                            'cursor-default select-none relative py-2 pl-3 pr-9'
                                                        )
                                                    }
                                                    value={opt.value}
                                                >
                                                    {({selected, active}) => (
                                                        <>
                        <span className={classNames(selected ? 'font-semibold' : 'font-normal', 'block truncate')}>
                          {opt.label}
                        </span>

                                                            {selected ? (
                                                                <span
                                                                    className={classNames(
                                                                        active ? 'text-white' : 'text-indigo-600',
                                                                        'absolute inset-y-0 right-0 flex items-center pr-4'
                                                                    )}
                                                                >
                            <CheckIcon className="h-5 w-5" aria-hidden="true"/>
                          </span>
                                                            ) : null}
                                                        </>
                                                    )}
                                                </Listbox.Option>
                                            ))}
                                        </Listbox.Options>
                                    </Transition>
                                </div>
                            </div>
                        )}
                    </Listbox>
                )}/>
        </div>
    )
}

export default SelectField;
