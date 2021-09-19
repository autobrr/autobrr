import React, { Fragment } from "react";
import { Transition, Listbox } from "@headlessui/react";
import { CheckIcon, SelectorIcon } from '@heroicons/react/solid';
import { Action, DownloadClient } from "../../../domain/interfaces";
import { classNames } from "../../../styles/utils";
import { Field } from "formik";

interface DownloadClientSelectProps {
    name: string;
    action: Action;
    clients: DownloadClient[];
}

export default function DownloadClientSelect({
    name, action, clients,
}: DownloadClientSelectProps) {
    return (
        <div className="col-span-6 sm:col-span-6">
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
                                <Listbox.Label className="block text-xs font-bold text-gray-700 uppercase tracking-wide">
                                    Client
                                </Listbox.Label>
                                <div className="mt-2 relative">
                                    <Listbox.Button className="bg-white relative w-full border border-gray-300 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm">
                                        <span className="block truncate">
                                            {field.value
                                                ? clients.find((c) => c.id === field.value)!.name
                                                : "Choose a client"}
                                        </span>
                                        {/*<span className="block truncate">Choose a client</span>*/}
                                        <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
                                            <SelectorIcon
                                                className="h-5 w-5 text-gray-400"
                                                aria-hidden="true" />
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
                                            {clients
                                                .filter((c) => c.type === action.type)
                                                .map((client: any) => (
                                                    <Listbox.Option
                                                        key={client.id}
                                                        className={({ active }) => classNames(
                                                            active
                                                                ? "text-white bg-indigo-600"
                                                                : "text-gray-900",
                                                            "cursor-default select-none relative py-2 pl-3 pr-9"
                                                        )}
                                                        value={client.id}
                                                    >
                                                        {({ selected, active }) => (
                                                            <>
                                                                <span
                                                                    className={classNames(
                                                                        selected ? "font-semibold" : "font-normal",
                                                                        "block truncate"
                                                                    )}
                                                                >
                                                                    {client.name}
                                                                </span>

                                                                {selected ? (
                                                                    <span
                                                                        className={classNames(
                                                                            active ? "text-white" : "text-indigo-600",
                                                                            "absolute inset-y-0 right-0 flex items-center pr-4"
                                                                        )}
                                                                    >
                                                                        <CheckIcon
                                                                            className="h-5 w-5"
                                                                            aria-hidden="true" />
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
