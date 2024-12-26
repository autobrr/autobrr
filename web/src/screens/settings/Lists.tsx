/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { PlusIcon } from "@heroicons/react/24/solid";

import { useToggle } from "@hooks/hooks";
import { APIClient } from "@api/APIClient";
import { ListKeys } from "@api/query_keys";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { Checkbox } from "@components/Checkbox";
import { ListsQueryOptions } from "@api/queries";
import { Section } from "@screens/settings/_components";
import { EmptySimple } from "@components/emptystates";
import { ListAddForm, ListUpdateForm } from "@forms";
import { FC } from "react";
import { Link } from "@tanstack/react-router";
import { ListTypeNameMap } from "@domain/constants";

function ListsSettings() {
  const [addFormIsOpen, toggleAddList] = useToggle(false);

  const listsQuery = useSuspenseQuery(ListsQueryOptions())
  const lists = listsQuery.data

  return (
    <Section
      title="Lists"
      description="Lists can automatically update your filters from arrs or other sources."
      rightSide={
        <button
          type="button"
          onClick={toggleAddList}
          className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
        >
          <PlusIcon className="h-5 w-5 mr-1"/>
          Add new
        </button>
      }
    >
      <ListAddForm isOpen={addFormIsOpen} toggle={toggleAddList} />

      <div className="flex flex-col">
        {lists.length ? (
          <ul className="min-w-full relative">
            <li className="grid grid-cols-12 border-b border-gray-200 dark:border-gray-700">
              <div
                className="flex col-span-2 sm:col-span-1 pl-0 sm:pl-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 hover:text-gray-800 hover:dark:text-gray-250 transition-colors uppercase tracking-wider cursor-pointer"
              >
                Enabled
              </div>
              <div
                className="col-span-5 sm:col-span-4 pl-12 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 hover:text-gray-800 hover:dark:text-gray-250 transition-colors uppercase tracking-wider cursor-pointer"
              >
                Name
              </div>
              <div
                className="hidden md:flex col-span-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 hover:text-gray-800 hover:dark:text-gray-250 transition-colors uppercase tracking-wider cursor-pointer"
              >
                Filters
              </div>
              <div
                className="hidden md:flex col-span-1 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 hover:text-gray-800 hover:dark:text-gray-250 transition-colors uppercase tracking-wider cursor-pointer"
              >
                Type
              </div>
            </li>
            {lists.map((list) => (
              <ListItem list={list} key={list.id}/>
            ))}
          </ul>
        ) : (
          <EmptySimple
            title="No lists"
            subtitle=""
            buttonText="Add new list"
            buttonAction={toggleAddList}
          />
        )}
      </div>
    </Section>
  );
}

interface FilterPillProps {
  filter: ListFilter;
}

const FilterPill: FC<FilterPillProps> = ({ filter }) => (
  <Link
    className="hidden sm:inline-flex items-center px-2 py-0.5 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400 hover:dark:bg-gray-750 hover:bg-gray-700"
    to={`/filters/$filterId`}
    params={{ filterId: filter.id }}
  >
    {filter.name}
  </Link>
);

export default ListsSettings;

interface ListItemProps {
  list: List;
}

function ListItem({ list }: ListItemProps) {
  const [isOpen, toggleUpdate] = useToggle(false);

  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: (req: List) => APIClient.lists.update(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ListKeys.lists() });

      toast.custom(t => <Toast type="success" body={`List ${list.name} was ${list.enabled ? "disabled" : "enabled"} successfully.`} t={t} />);
    },
    onError: () => {
      toast.custom((t) => <Toast type="error" body="List state could not be updated" t={t} />);
    }
  });

  const onToggleMutation = (newState: boolean) => {
    updateMutation.mutate({
      ...list,
      enabled: newState
    });
  };

  return (
    <li>
      <ListUpdateForm isOpen={isOpen} toggle={toggleUpdate} data={list} />

      <div className="grid grid-cols-12 items-center py-1.5">
        <div className="col-span-2 sm:col-span-1 flex pl-1 sm:pl-5 items-center">
          <Checkbox value={list.enabled ?? false} setValue={onToggleMutation}/>
        </div>
        <div
          className="col-span-5 sm:col-span-4 pl-12 sm:pr-6 py-3 block flex-col text-sm font-medium text-gray-900 dark:text-white truncate">
          {list.name}
        </div>
        <div
          className="hidden md:block col-span-4 pr-6 py-3 space-x-1 text-left items-center whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 truncate">
          {/*{list.filters.map(filter => <FilterPill filter={filter} key={filter.id} />)}*/}
          <ListItemFilters filters={list.filters} />
        </div>
        <div
          className="hidden md:block col-span-2 pr-6 py-3 text-left items-center whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 truncate">
          {ListTypeNameMap[list.type]}
        </div>
        <div className="col-span-1 flex first-letter:px-6 py-3 whitespace-nowrap text-right text-sm font-medium">
          <span
            className="col-span-1 px-6 text-blue-600 dark:text-gray-300 hover:text-blue-900 dark:hover:text-blue-500 cursor-pointer"
            onClick={toggleUpdate}
          >
            Edit
          </span>
        </div>
      </div>
    </li>
  );
}

interface ListItemFiltersProps {
  filters: ListFilter[];
}

const ListItemFilters = ({ filters }: ListItemFiltersProps) => {
  if (!filters.length) {
    return null;
  }

  const res = filters.slice(2);

  return (
    <div className="flex flex-row gap-1">
      <FilterPill filter={filters[0]} />
      {filters.length > 1 ? (
        <FilterPill filter={filters[1]} />
      ) : null}
      {filters.length > 2 ? (
        <span
          className="mr-2 inline-flex items-center px-2 py-0.5 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400"
          title={res.map(v => v.name).toString()}
        >
          +{filters.length - 2}
        </span>
      ) : null}
    </div>
  );
}