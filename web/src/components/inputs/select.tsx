import { Fragment } from "react";
import { MultiSelect as RMSC}  from "react-multi-select-component";
import { Transition, Listbox } from "@headlessui/react";
import { CheckIcon, SelectorIcon } from '@heroicons/react/solid';
import { Action, DownloadClient } from "../../domain/interfaces";
import { classNames, COL_WIDTHS } from "../../utils";
import { Field } from "formik";

interface MultiSelectProps {
    label?: string;
    options?: [] | any;
    name: string;
    className?: string;
    columns?: COL_WIDTHS;
}

const MultiSelect: React.FC<MultiSelectProps> = ({
    name,
    label,
    options,
    className,
    columns,
}) => (
    <div
        className={classNames(
            columns ? `col-span-${columns}` : "col-span-12"
        )}
    >
        <label
            className="block mb-2 text-xs font-bold tracking-wide text-gray-700 uppercase dark:text-gray-200"
            htmlFor={label}
        >
            {label}
        </label>

        <Field name={name} type="select" multiple={true}>
            {({
                field,
                form: { setFieldValue },
            }: any) => (
                <RMSC
                    {...field}
                    type="select"
                    options={options}
                    labelledBy={name}
                    value={field.value && field.value.map((item: any) => options.find((o: any) => o.value === item))}
                    onChange={(values: any) => {
                        let am = values && values.map((i: any) => i.value)

                        setFieldValue(field.name, am)
                    }}
                    className="dark:bg-gray-700 dark"
                />
            )}
        </Field>
    </div>
);

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
                                <Listbox.Label className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
                                    Client
                                </Listbox.Label>
                                <div className="mt-2 relative">
                                    <Listbox.Button className="bg-white dark:bg-gray-800 relative w-full border border-gray-300 dark:border-gray-700 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 dark:text-gray-200 sm:text-sm">
                                        <span className="block truncate">
                                            {field.value
                                                ? clients.find((c) => c.id === field.value)!.name
                                                : "Choose a client"}
                                        </span>
                                        {/*<span className="block truncate">Choose a client</span>*/}
                                        <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
                                            <SelectorIcon
                                                className="h-5 w-5 text-gray-400 dark:text-gray-300"
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
                                            className="absolute z-10 mt-1 w-full bg-white dark:bg-gray-800 shadow-lg max-h-60 rounded-md py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm"
                                        >
                                            {clients
                                                .filter((c) => c.type === action.type)
                                                .map((client: any) => (
                                                    <Listbox.Option
                                                        key={client.id}
                                                        className={({ active }) => classNames(
                                                            active
                                                                ? "text-white dark:text-gray-100 bg-indigo-600 dark:bg-gray-800"
                                                                : "text-gray-900 dark:text-gray-300",
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
                                                                            active ? "text-white dark:text-gray-100" : "text-indigo-600 dark:text-gray-700",
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

interface SelectFieldOption {
    label: string;
    value: string;
}

interface SelectFieldProps {
    name: string;
    label: string;
    optionDefaultText: string;
    options: SelectFieldOption[];
}

function Select({ name, label, optionDefaultText, options }: SelectFieldProps) {
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

export { MultiSelect, DownloadClientSelect, Select }