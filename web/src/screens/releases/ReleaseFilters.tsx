/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import { Column } from "@tanstack/react-table";
import { Listbox, ListboxButton, ListboxOption, ListboxOptions, Transition } from "@headlessui/react";
import { DebounceInput } from "react-debounce-input";
import { CheckIcon, ChevronDownIcon } from "@heroicons/react/24/solid";

import { classNames } from "@utils";
import { PushStatusOptions } from "@domain/constants";
import { ReleasesIndexersQueryOptions } from "@api/queries";

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
        <ListboxButton className="relative w-full py-2 pl-3 pr-10 text-left bg-white dark:bg-gray-800 rounded-lg shadow-md cursor-pointer dark:text-gray-400 sm:text-sm">
          <span className="block truncate">{label}</span>
          <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
            <ChevronDownIcon
              className="w-5 h-5 ml-2 -mr-1 text-gray-600 hover:text-gray-600"
              aria-hidden="true"
            />
          </span>
        </ListboxButton>
        <Transition
          as={React.Fragment}
          leave="transition ease-in duration-100"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <ListboxOptions
            className="absolute z-10 w-full mt-1 overflow-auto text-base bg-white dark:bg-gray-800 rounded-md shadow-lg max-h-60 border border-opacity-5 border-black dark:border-gray-700 dark:border-opacity-40 focus:outline-none sm:text-sm"
          >
            <FilterOption label="All" value="" />
            {children}
          </ListboxOptions>
        </Transition>
      </div>
    </Listbox>
  </div>
);

export const IndexerSelectColumnFilter = ({ column }: { column: Column<Release, unknown> }) => {
  const { data, isSuccess } = useQuery(ReleasesIndexersQueryOptions());

  // Assign indexer name based on the filterValue (indexer.identifier)
  const currentIndexerName = data?.find(indexer => indexer.identifier === column.getFilterValue())?.name ?? "Indexer";

  return (
    <ListboxFilter
      id={column.id}
      key={column.id}
      label={currentIndexerName}
      currentValue={column.getFilterValue() as string || ""}
      onChange={newValue => column.setFilterValue(newValue || undefined)}
    >
      {isSuccess && data && data?.map((indexer, idx) => (
        <FilterOption key={idx} label={indexer.name} value={indexer.identifier} />
      ))}
    </ListboxFilter>
  );
};

interface FilterOptionProps {
  label: string;
  value?: string;
}

const FilterOption = ({ label, value }: FilterOptionProps) => (
  <ListboxOption
    className={({ focus }) => classNames(
      "cursor-pointer select-none relative py-2 pl-10 pr-4",
      focus ? "text-black dark:text-gray-200 bg-gray-100 dark:bg-gray-900" : "text-gray-700 dark:text-gray-400"
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
  </ListboxOption>
);

export const PushStatusSelectColumnFilter = ({ column }: { column: Column<Release, unknown> }) => {
  // React.useEffect(() => {
  //   if (initialFilterValue) {
  //     setFilter(initialFilterValue);
  //   }
  // }, [initialFilterValue, setFilter]);

  const label = column.getFilterValue() ? PushStatusOptions.find((o) => o.value === column.getFilterValue() && o.value)?.label : "Push status";

  return (
    <div className="mr-3" key={column.id}>
      <ListboxFilter
        id={column.id}
        label={label ?? "Push status"}
        currentValue={column.getFilterValue() as string ?? ""}
        onChange={value => {
          column.setFilterValue(value || undefined);
        }}
      >
        {PushStatusOptions.map((status, idx) => (
          <FilterOption key={idx} value={status.value} label={status.label} />
        ))}
      </ListboxFilter>
    </div>
  );
};

export const SearchColumnFilter = ({ column }: { column: Column<Release, unknown> }) => {
  return (
    <div className="flex-1 mr-3 mt-1" key={column.id}>
      <DebounceInput
        minLength={2}
        value={column.getFilterValue() as string || undefined}
        debounceTimeout={500}
        onChange={e => {
          // Set undefined to remove the filter entirely
          column.setFilterValue(e.target.value || undefined)
        }}
        id="filter"
        type="text"
        autoComplete="off"
        className="relative w-full py-2 pl-3 pr-10 text-left bg-white dark:bg-gray-800 rounded-lg shadow-md dark:text-gray-400 sm:text-sm border-none"
        placeholder="Search releases..."
      />
    </div>
  );
};
