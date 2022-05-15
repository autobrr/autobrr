import * as React from "react";
import { useQuery } from "react-query";
import { Listbox, Transition } from "@headlessui/react";
import {
    CheckIcon,
    ChevronDownIcon
} from "@heroicons/react/solid";

import { APIClient } from "../../api/APIClient";
import { classNames } from "../../utils";
import { PushStatusOptions } from "../../domain/constants";
import {FilterProps} from "react-table";

interface ListboxFilterProps {
    id: string;
    label: string;
    currentValue: string;
    onChange: (newValue: string) => void;
    children: React.ReactNode;
}

const ListboxFilter = ({
    id,
    label,
    currentValue,
    onChange,
    children
}: ListboxFilterProps) => (
    <div className="w-48">
        <Listbox
            refName={id}
            value={currentValue}
            onChange={onChange}
        >
            <div className="relative mt-1">
                <Listbox.Button className="relative w-full py-2 pl-3 pr-10 text-left bg-white dark:bg-gray-800 rounded-lg shadow-md cursor-default dark:text-gray-400 sm:text-sm">
                    <span className="block truncate">{label}</span>
                    <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
                        <ChevronDownIcon
                            className="w-5 h-5 ml-2 -mr-1 text-gray-600 hover:text-gray-600"
                            aria-hidden="true"
                        />
                    </span>
                </Listbox.Button>
                <Transition
                    as={React.Fragment}
                    leave="transition ease-in duration-100"
                    leaveFrom="opacity-100"
                    leaveTo="opacity-0"
                >
                    <Listbox.Options
                        className="absolute w-full mt-1 overflow-auto text-base bg-white dark:bg-gray-800 rounded-md shadow-lg max-h-60 border border-opacity-5 border-black dark:border-gray-700 dark:border-opacity-40 focus:outline-none sm:text-sm"
                    >
                        <FilterOption label="All" />
                        {children}
                    </Listbox.Options>
                </Transition>
            </div>
        </Listbox>
    </div>
);

// a unique option from a list
export const IndexerSelectColumnFilter = ({
    column: { filterValue, setFilter, id }
}: FilterProps<object>) => {
    const { data, isSuccess } = useQuery(
        "release_indexers",
        () => APIClient.release.indexerOptions(),
        {
            keepPreviousData: true,
            staleTime: Infinity
        }
    );

    // Render a multi-select box
    return (
        <ListboxFilter
            id={id}
            label={filterValue ?? "Indexer"}
            currentValue={filterValue}
            onChange={setFilter}
        >
            {isSuccess && data?.map((indexer, idx) => (
                <FilterOption key={idx} label={indexer} value={indexer} />
            ))}
        </ListboxFilter>
    );
};

interface FilterOptionProps {
    label: string;
    value?: string;
}

const FilterOption = ({ label, value }: FilterOptionProps) => (
    <Listbox.Option
        className={({ active }) => classNames(
            "cursor-pointer select-none relative py-2 pl-10 pr-4",
            active ? "text-black dark:text-gray-200 bg-gray-100 dark:bg-gray-900" : "text-gray-700 dark:text-gray-400"
        )}
        value={value}
    >
        {({ selected }) => (
            <>
                <span
                    className={classNames(
                        "block truncate",
                        selected ? "font-medium text-black dark:text-white" : "font-normal"
                    )}
                >
                    {label}
                </span>
                {selected ? (
                    <span className="absolute inset-y-0 left-0 flex items-center pl-3 text-gray-500 dark:text-gray-400">
                        <CheckIcon className="w-5 h-5" aria-hidden="true" />
                    </span>
                ) : null}
            </>
        )}
    </Listbox.Option>
);

export const PushStatusSelectColumnFilter = ({
    column: { filterValue, setFilter, id }
}: FilterProps<object>) => {
    const label = filterValue ? PushStatusOptions.find((o) => o.value === filterValue && o.value)?.label : "Push status";
    return (
    <div className="mr-3">
        <ListboxFilter
            id={id}
            label={label ?? "Push status"}
            currentValue={filterValue}
            onChange={setFilter}
        >
            {PushStatusOptions.map((status, idx) => (
                <FilterOption key={idx} value={status.value} label={status.label} />
            ))}
        </ListboxFilter>
    </div>
);};