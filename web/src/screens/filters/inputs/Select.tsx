import { Fragment } from "react";
import { Field } from "formik";
import { Listbox, Transition } from "@headlessui/react";
import { CheckIcon, SelectorIcon } from "@heroicons/react/solid";
import { classNames } from "../../../styles/utils";

interface Option {
    label: string;
    value: string;
}

interface props {
    name: string;
    label: string;
    optionDefaultText: string;
    options: Option[];
}

function Select({ name, label, optionDefaultText, options }: props) {
    return (
        <div className="col-span-6">
            <Field name={name} type="select">
                {({
                    field,
                    form: { setFieldValue },
                }: any) => (
                    <Listbox
                        value={field.value}
                        onChange={(value: any) => setFieldValue(field?.name, value)}
                    >
                        {({ open }) => (
                            <>
                                <Listbox.Label className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
                                    {label}
                                </Listbox.Label>
                                <div className="mt-2 relative">
                                    <Listbox.Button className="bg-white dark:bg-gray-800 relative w-full border border-gray-300 dark:border-gray-700 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 dark:text-gray-200 sm:text-sm">
                                        <span className="block truncate">
                                            {field.value
                                                ? options.find((c) => c.value === field.value)!.label
                                                : optionDefaultText
                                            }
                                        </span>
                                        <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
                                            <SelectorIcon
                                                className="h-5 w-5 text-gray-400 dark:text-gray-300"
                                                aria-hidden="true"
                                            />
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
                                            className="absolute z-10 mt-1 w-full bg-white dark:bg-gray-800 shadow-lg max-h-60 rounded-md py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm"
                                        >
                                            {options.map((opt) => (
                                                <Listbox.Option
                                                    key={opt.value}
                                                    className={({ active }) =>
                                                        classNames(
                                                            active
                                                                ? "text-white dark:text-gray-100 bg-indigo-600 dark:bg-gray-800"
                                                                : "text-gray-900 dark:text-gray-300",
                                                            "cursor-default select-none relative py-2 pl-3 pr-9"
                                                        )
                                                    }
                                                    value={opt.value}
                                                >
                                                    {({ selected, active }) => (
                                                        <>
                                                            <span
                                                                className={classNames(
                                                                    selected ? "font-semibold" : "font-normal",
                                                                    "block truncate"
                                                                )}
                                                            >
                                                                {opt.label}
                                                            </span>

                                                            {selected ? (
                                                                <span
                                                                    className={classNames(
                                                                        active ? "text-white dark:text-gray-100" : "text-indigo-600 dark:text-gray-700",
                                                                        "absolute inset-y-0 right-0 flex items-center pr-4"
                                                                    )}
                                                                >
                                                                    <CheckIcon
                                                                        className="h-5 w-5"
                                                                        aria-hidden="true"
                                                                    />
                                                                </span>
                                                            ) : null}
                                                        </>
                                                    )}
                                                </Listbox.Option>
                                            ))}
                                        </Listbox.Options>
                                    </Transition>
                                </div>
                            </>
                        )}
                    </Listbox>
                )}
            </Field>
        </div>
    );
}

export default Select;
