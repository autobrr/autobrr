/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Dispatch, FC, Fragment, MouseEventHandler, useCallback, useEffect, useReducer, useRef, useState } from "react";
import { Link } from '@tanstack/react-router'
import {
  Listbox,
  ListboxButton,
  ListboxOption,
  ListboxOptions,
  Menu,
  MenuButton,
  MenuItem,
  MenuItems,
  Transition
} from "@headlessui/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowsRightLeftIcon,
  ArrowUpOnSquareIcon,
  ChatBubbleBottomCenterTextIcon,
  CheckIcon,
  ChevronDownIcon,
  DocumentDuplicateIcon,
  EllipsisHorizontalIcon,
  PencilSquareIcon,
  PlusIcon, SparklesIcon,
  TrashIcon
} from "@heroicons/react/24/outline";
import { ArrowDownTrayIcon } from "@heroicons/react/24/solid";

import { FilterListContext, FilterListState } from "@utils/Context";
import { classNames, CopyTextToClipboard } from "@utils";
import { FilterAddForm } from "@forms";
import { useToggle } from "@hooks/hooks";
import { APIClient } from "@api/APIClient";
import { FilterKeys } from "@api/query_keys";
import { FiltersQueryOptions, IndexersOptionsQueryOptions } from "@api/queries";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { EmptyListState } from "@components/emptystates";
import { DeleteModal } from "@components/modals";

import { Importer } from "./Importer";
import { Tooltip } from "@components/tooltips/Tooltip";
import { Checkbox } from "@components/Checkbox";
import { RingResizeSpinner } from "@components/Icons";

enum ActionType {
  INDEXER_FILTER_CHANGE = "INDEXER_FILTER_CHANGE",
  INDEXER_FILTER_RESET = "INDEXER_FILTER_RESET",
  SORT_ORDER_CHANGE = "SORT_ORDER_CHANGE",
  SORT_ORDER_RESET = "SORT_ORDER_RESET",
  STATUS_CHANGE = "STATUS_CHANGE",
  STATUS_RESET = "STATUS_RESET"
}

type Actions =
  | { type: ActionType.STATUS_CHANGE; payload: string }
  | { type: ActionType.STATUS_RESET; payload: "" }
  | { type: ActionType.SORT_ORDER_CHANGE; payload: string }
  | { type: ActionType.SORT_ORDER_RESET; payload: "" }
  | { type: ActionType.INDEXER_FILTER_CHANGE; payload: string[] }
  | { type: ActionType.INDEXER_FILTER_RESET; payload: [] };

const FilterListReducer = (state: FilterListState, action: Actions): FilterListState => {
  switch (action.type) {
  case ActionType.INDEXER_FILTER_CHANGE: {
    return { ...state, indexerFilter: action.payload };
  }
  case ActionType.INDEXER_FILTER_RESET: {
    return { ...state, indexerFilter: [] };
  }
  case ActionType.SORT_ORDER_CHANGE: {
    return { ...state, sortOrder: action.payload };
  }
  case ActionType.SORT_ORDER_RESET: {
    return { ...state, sortOrder: "" };
  }
  case ActionType.STATUS_CHANGE: {
    return { ...state, status: action.payload };
  }
  case ActionType.STATUS_RESET: {
    return { ...state, status: "" };
  }
  default: {
    throw new Error(`Unhandled action type: ${action}`);
  }
  }
};

export function Filters() {
  const [createFilterIsOpen, setCreateFilterIsOpen] = useState(false);
  const toggleCreateFilter = () => {
    setCreateFilterIsOpen(!createFilterIsOpen);
  };

  const [showImportModal, setShowImportModal] = useState(false);

  return (
    <main>
      <FilterAddForm isOpen={createFilterIsOpen} toggle={toggleCreateFilter} />
      <Importer
        isOpen={showImportModal}
        setIsOpen={setShowImportModal}
      />

      <div className="flex justify-between items-center flex-row flex-wrap my-6 max-w-(--breakpoint-xl) mx-auto px-4 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold text-black dark:text-white">Filters</h1>
        <Menu as="div" className="relative">
          {({ open }) => (
            <>
              <button
                className="relative inline-flex items-center px-4 py-2 shadow-xs text-sm font-medium rounded-l-md transition text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                onClick={(e: { stopPropagation: () => void; }) => {
                  if (!open) {
                    e.stopPropagation();
                    toggleCreateFilter();
                  }
                }}
              >
                <PlusIcon className="h-5 w-5 mr-1" />
                Create Filter
              </button>
              <MenuButton className="relative inline-flex items-center px-2 py-2 border-l border-spacing-1 dark:border-black shadow-xs text-sm font-medium rounded-r-md transition text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500">
                <ChevronDownIcon className="h-5 w-5" />
              </MenuButton>
              <Transition
                show={open}
                as={Fragment}
                enter="transition ease-out duration-100 transform"
                enterFrom="opacity-0 scale-95"
                enterTo="opacity-100 scale-100"
                leave="transition ease-in duration-75 transform"
                leaveFrom="opacity-100 scale-100"
                leaveTo="opacity-0 scale-95"
              >
                <MenuItems className="absolute z-10 right-0 mt-0.5 bg-white dark:bg-gray-700 rounded-md shadow-lg">
                  <MenuItem>
                    {({ active }) => (
                      <button
                        type="button"
                        className={classNames(
                          active ? "bg-gray-50 dark:bg-gray-600" : "",
                          "flex items-center w-full text-left py-2 px-3 text-sm font-medium text-gray-700 dark:text-gray-200 rounded-md focus:outline-hidden"
                        )}
                        onClick={() => setShowImportModal(true)}
                      >
                        <ArrowUpOnSquareIcon className="mr-1 w-4 h-4" />
                        <span>Import filter</span>
                      </button>
                    )}
                  </MenuItem>
                </MenuItems>
              </Transition>
            </>
          )}
        </Menu>
      </div>

      <FilterList toggleCreateFilter={toggleCreateFilter} />
    </main>
  );
}

function filteredData(data: Filter[], status: string) {
  let filtered: Filter[];

  const enabledItems = data?.filter(f => f.enabled);
  const disabledItems = data?.filter(f => !f.enabled);

  if (status === "enabled") {
    filtered = enabledItems;
  } else if (status === "disabled") {
    filtered = disabledItems;
  } else {
    filtered = data;
  }

  return {
    all: data,
    filtered: filtered,
    enabled: enabledItems,
    disabled: disabledItems
  };
}

function FilterList({ toggleCreateFilter }: any) {
  const filterListState = FilterListContext.useValue();

  const [{ indexerFilter, sortOrder, status }, dispatchFilter] = useReducer(
    FilterListReducer,
    filterListState
  );

  const { isLoading, data, error } = useQuery(FiltersQueryOptions(indexerFilter, sortOrder));

  useEffect(() => {
    FilterListContext.set({ indexerFilter, sortOrder, status });
  }, [indexerFilter, sortOrder, status]);

  if (error) {
    return <p>An error has occurred:</p>;
  }

  const filtered = filteredData(data ?? [], status);

  return (
    <div className="max-w-(--breakpoint-xl) mx-auto pb-12 px-2 sm:px-6 lg:px-8 relative">
      <div className="align-middle min-w-full rounded-t-lg rounded-b-lg shadow-table bg-gray-50 dark:bg-gray-800 border border-gray-250 dark:border-gray-775">
        <div className="rounded-t-lg flex justify-between px-4 bg-gray-125 dark:bg-gray-850 border-b border-gray-200 dark:border-gray-750">
          <div className="flex gap-4">
            <StatusButton data={filtered.all} label="All" value="" currentValue={status} dispatch={dispatchFilter} />
            <StatusButton data={filtered.enabled} label="Enabled" value="enabled" currentValue={status} dispatch={dispatchFilter} />
            <StatusButton data={filtered.disabled} label="Disabled" value="disabled" currentValue={status} dispatch={dispatchFilter} />
          </div>

          <div className="flex items-center gap-5">
            <div className="hidden md:flex"><IndexerSelectFilter dispatch={dispatchFilter} /></div>
            <SortSelectFilter dispatch={dispatchFilter} />
          </div>
        </div>

        {isLoading
          ? <div className="flex items-center justify-center py-64"><RingResizeSpinner className="text-blue-500 size-24"/></div>
          : data && data.length > 0 ? (
              <ul className="min-w-full divide-y divide-gray-150 dark:divide-gray-775">
                {filtered.filtered.length > 0
                  ? filtered.filtered.map((filter: Filter, idx) => <FilterListItem filter={filter} key={filter.id} idx={idx}/>)
                  : <EmptyListState text={`No ${status} filters`}/>
                }
              </ul>
            ) : (
              <EmptyListState text="No filters here.." buttonText="Add new" buttonOnClick={toggleCreateFilter}/>
            )
        }
      </div>
    </div>
  );
}

interface StatusButtonProps {
  data: unknown[];
  label: string;
  value: string;
  currentValue: string;
  dispatch: Dispatch<Actions>;
}

const StatusButton = ({ data, label, value, currentValue, dispatch }: StatusButtonProps) => {
  const setFilter: MouseEventHandler = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.preventDefault();
    if (value == undefined || value == "") {
      dispatch({ type: ActionType.STATUS_RESET, payload: "" });
    } else {
      dispatch({ type: ActionType.STATUS_CHANGE, payload: e.currentTarget.value });
    }
  };

  return (
    <button
      className={classNames(
        "py-4 pb-4 text-left text-xs tracking-wider transition border-b-2",
        currentValue === value
          ? "font-bold  border-blue-500 dark:text-gray-100 text-gray-950"
          : "font-medium border-transparent text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200"
      )}
      onClick={setFilter}
      value={value}
    >
      {data?.length ?? 0} {label}
    </button>
  );
};

interface FilterItemDropdownProps {
  filter: Filter;
  onToggle: (newState: boolean) => void;
}

const FilterItemDropdown = ({ filter, onToggle }: FilterItemDropdownProps) => {

  // This function handles the export of a filter to a JSON string
  const handleExportJson = useCallback(async (discordFormat = false) => {
    try {
      type CompleteFilterType = {
        id: number;
        name: string;
        created_at: Date;
        updated_at: Date;
        indexers: any;
        actions: any;
        actions_count: any;
        actions_enabled_count: number;
      };

      const completeFilter = await APIClient.filters.getByID(filter.id) as Partial<CompleteFilterType>;

      // Extract the filter name and remove unwanted properties
      const title = completeFilter.name;
      delete completeFilter.name;
      delete completeFilter.id;
      delete completeFilter.created_at;
      delete completeFilter.updated_at;
      delete completeFilter.actions_count;
      delete completeFilter.actions_enabled_count;
      delete completeFilter.indexers;
      delete completeFilter.actions;

      // Remove properties with default values from the exported filter to minimize the size of the JSON string
      ["enabled", "priority", "smart_episode", "resolutions", "sources", "codecs", "containers", "tags_match_logic", "except_tags_match_logic"].forEach((key) => {
        const value = completeFilter[key as keyof CompleteFilterType];
        if (["enabled", "priority", "smart_episode"].includes(key) && (value === false || value === 0)) {
          delete completeFilter[key as keyof CompleteFilterType];
        } else if (["resolutions", "sources", "codecs", "containers"].includes(key) && Array.isArray(value) && value.length === 0) {
          delete completeFilter[key as keyof CompleteFilterType];
        } else if (["tags_match_logic", "except_tags_match_logic"].includes(key) && value === "ANY") {
          delete completeFilter[key as keyof CompleteFilterType];
        }
      });

      // Create a JSON string from the filter data, including a name and version
      const json = JSON.stringify(
        {
          "name": title,
          "version": "1.0",
          data: completeFilter
        },
        null,
        4
      );

      const finalJson = discordFormat ? "```JSON\n" + json + "\n```" : json;

      // Asynchronously call copyTextToClipboard
      CopyTextToClipboard(finalJson)
        .then(() => {
          toast.custom((t) => <Toast type="success" body="Filter copied to clipboard!" t={t} />);

        })
        .catch((err) => {
          console.error("could not copy filter to clipboard", err);

          toast.custom((t) => <Toast type="error" body="Failed to copy JSON to clipboard." t={t} />);
        });
    } catch (error) {
      console.error(error);
      toast.custom((t) => <Toast type="error" body="Failed to get filter data." t={t} />);
    }
  }, [filter]);

  const cancelModalButtonRef = useRef(null);

  const queryClient = useQueryClient();

  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);

  const deleteMutation = useMutation({
    mutationFn: (id: number) => APIClient.filters.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: FilterKeys.lists() });
      queryClient.invalidateQueries({ queryKey: FilterKeys.detail(filter.id) });

      toast.custom((t) => <Toast type="success" body={`Filter ${filter?.name} was deleted`} t={t} />);
    }
  });

  const duplicateMutation = useMutation({
    mutationFn: (id: number) => APIClient.filters.duplicate(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: FilterKeys.lists() });

      toast.custom((t) => <Toast type="success" body={`Filter ${filter?.name} duplicated`} t={t} />);
    }
  });

  return (
    <Menu as="div">
      <DeleteModal
        isOpen={deleteModalIsOpen}
        isLoading={deleteMutation.isPending}
        toggle={toggleDeleteModal}
        buttonRef={cancelModalButtonRef}
        deleteAction={() => {
          deleteMutation.mutate(filter.id);
          toggleDeleteModal();
        }}
        title={`Remove filter: ${filter.name}`}
        text="Are you sure you want to remove this filter? This action cannot be undone."
      />
      <MenuButton className="px-4 py-2">
        <EllipsisHorizontalIcon
          className="w-5 h-5 text-gray-700 hover:text-gray-900 dark:text-gray-100 dark:hover:text-gray-400"
          aria-hidden="true"
        />
      </MenuButton>
      <Transition
        as={Fragment}
        enter="transition ease-out duration-100"
        enterFrom="transform opacity-0 scale-95"
        enterTo="transform opacity-100 scale-100"
        leave="transition ease-in duration-75"
        leaveFrom="transform opacity-100 scale-100"
        leaveTo="transform opacity-0 scale-95"
      >
        <MenuItems
          anchor={{ to: 'bottom end', padding: '8px' }} // padding: '8px' === m-2
          className="absolute w-56 bg-white dark:bg-gray-825 divide-y divide-gray-200 dark:divide-gray-750 rounded-md shadow-lg border border-gray-250 dark:border-gray-750 focus:outline-hidden z-10"
        >
          <div className="px-1 py-1">
            <MenuItem>
              {({ active }) => (
                <Link
                  // to={filter.id.toString()}
                  to="/filters/$filterId"
                  params={{
                    filterId: filter.id
                  }}
                  className={classNames(
                    active ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                    "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                  )}
                >
                  <PencilSquareIcon
                    className={classNames(
                      active ? "text-white" : "text-blue-500",
                      "w-5 h-5 mr-2"
                    )}
                    aria-hidden="true"
                  />
                  Edit
                </Link>
              )}
            </MenuItem>
            <MenuItem>
              {({ active }) => (
                <button
                  className={classNames(
                    active ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                    "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                  )}
                  onClick={() => handleExportJson(false)}                >
                  <ArrowDownTrayIcon
                    className={classNames(
                      active ? "text-white" : "text-blue-500",
                      "w-5 h-5 mr-2"
                    )}
                    aria-hidden="true"
                  />
                  Export JSON
                </button>
              )}
            </MenuItem>
            <MenuItem>
              {({ active }) => (
                <button
                  className={classNames(
                    active ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                    "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                  )}
                  onClick={() => handleExportJson(true)}
                >
                  <ChatBubbleBottomCenterTextIcon
                    className={classNames(
                      active ? "text-white" : "text-blue-500",
                      "w-5 h-5 mr-2"
                    )}
                    aria-hidden="true"
                  />
                  Export JSON to Discord
                </button>
              )}
            </MenuItem>
            <MenuItem>
              {({ active }) => (
                <button
                  className={classNames(
                    active ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                    "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                  )}
                  onClick={() => onToggle(!filter.enabled)}
                >
                  <ArrowsRightLeftIcon
                    className={classNames(
                      active ? "text-white" : "text-blue-500",
                      "w-5 h-5 mr-2"
                    )}
                    aria-hidden="true"
                  />
                  Toggle
                </button>
              )}
            </MenuItem>
            <MenuItem>
              {({ active }) => (
                <button
                  className={classNames(
                    active ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                    "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                  )}
                  onClick={() => duplicateMutation.mutate(filter.id)}
                >
                  <DocumentDuplicateIcon
                    className={classNames(
                      active ? "text-white" : "text-blue-500",
                      "w-5 h-5 mr-2"
                    )}
                    aria-hidden="true"
                  />
                  Duplicate
                </button>
              )}
            </MenuItem>
          </div>
          <div className="px-1 py-1">
            <MenuItem>
              {({ active }) => (
                <button
                  className={classNames(
                    active ? "bg-red-600 text-white" : "text-gray-900 dark:text-gray-300",
                    "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                  )}
                  onClick={() => toggleDeleteModal()}
                >
                  <TrashIcon
                    className={classNames(
                      active ? "text-white" : "text-red-500",
                      "w-5 h-5 mr-2"
                    )}
                    aria-hidden="true"
                  />
                  Delete
                </button>
              )}
            </MenuItem>
          </div>
        </MenuItems>
      </Transition>
    </Menu>
  );
};

interface FilterListItemProps {
  filter: Filter;
  idx: number;
}

function FilterListItem({ filter, idx }: FilterListItemProps) {
  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: (status: boolean) => APIClient.filters.toggleEnable(filter.id, status),
    onSuccess: () => {
      toast.custom((t) => <Toast type="success" body={`${filter.name} was ${filter.enabled ? "disabled" : "enabled"} successfully`} t={t} />);
      // We need to invalidate both keys here.
      // The filters key is used on the /filters page,
      // while the ["filter", filter.id] key is used on the details page.
      queryClient.invalidateQueries({ queryKey: FilterKeys.lists() });
      queryClient.invalidateQueries({ queryKey: FilterKeys.detail(filter.id) });
    }
  });

  const toggleActive = (status: boolean) => {
    updateMutation.mutate(status);
  };

  return (
    <li
      key={filter.id}
      className={classNames(
        "flex items-center transition last:rounded-b-md py-0.5",
        idx % 2 === 0
          ? "bg-white dark:bg-gray-800"
          : "bg-gray-75 dark:bg-gray-825"
      )}
    >
      <span
        className="pl-2 pr-4 sm:px-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-100"
      >
        <Checkbox
          value={filter.enabled}
          setValue={toggleActive}
        />
      </span>
      <div className="py-2 flex flex-col overflow-hidden w-full justify-center">
        <Link
          to="/filters/$filterId"
          params={{
            filterId: filter.id
          }}
          className="transition flex items-center w-full break-words whitespace-wrap text-sm font-bold text-gray-800 dark:text-gray-100 hover:text-black dark:hover:text-gray-350"
        >
          {filter.name} {filter.is_auto_updated && <SparklesIcon title="This filter is automatically updated by a list" className="ml-1 w-4 h-4 text-amber-500 dark:text-amber-400" aria-hidden="true"/>}
        </Link>
        <div className="flex items-center flex-wrap">
          <span className="mr-2 break-words whitespace-nowrap text-xs font-medium text-gray-600 dark:text-gray-400">
            Priority: {filter.priority !== 0 ? (
              <span className="text-gray-850 dark:text-gray-200">{filter.priority}</span>
            ) : filter.priority}
          </span>
          <span className="z-10 whitespace-nowrap text-xs font-medium text-gray-600 dark:text-gray-400">
            {filter.actions_count === 0 || filter.actions_enabled_count === 0 ? (
              <Tooltip
                label={
                  <Link
                    to="/filters/$filterId/actions"
                    params={{
                      filterId: filter.id
                    }}
                    className="flex items-center cursor-pointer hover:text-black dark:hover:text-gray-300"
                  >
                    <span className={filter.actions_count === 0 || filter.actions_enabled_count === 0 ? "text-red-500 hover:text-red-400 dark:hover:text-red-400" : ""}>
          Actions: {filter.actions_enabled_count}/{filter.actions_count}
                    </span>
                  </Link>
                }
              >
                {filter.actions_count === 0 ? (
                  <>
                    {"No actions defined. Set up actions to enable snatching."}
                  </>
                ) : filter.actions_enabled_count === 0 ? (
                  <>
                    {"You need to enable at least one action in the filter otherwise you will not get any snatches."}
                  </>
                ) : null}
              </Tooltip>
            ) : (
              <Link
                to="/filters/$filterId/actions"
                params={{
                  filterId: filter.id
                }}
                className="flex items-center cursor-pointer hover:text-black dark:hover:text-gray-300"
              >
                <span>
          Actions: {filter.actions_enabled_count}/{filter.actions_count}
                </span>
              </Link>
            )}
          </span>
        </div>
      </div>
      <span className="hidden md:flex px-4 whitespace-nowrap text-sm font-medium text-gray-900">
        <FilterIndexers indexers={filter.indexers} />
      </span>
      <span className="min-w-fit px-4 py-2 whitespace-nowrap text-right text-sm font-medium">
        <FilterItemDropdown
          filter={filter}
          onToggle={toggleActive}
        />
      </span>
    </li>
  );
}

interface IndexerTagProps {
  indexer: Indexer;
}

const IndexerTag: FC<IndexerTagProps> = ({ indexer }) => (
  <span
    className="hidden sm:inline-flex items-center px-2 py-0.5 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400"
  >
    {indexer.name}
  </span>
);

interface FilterIndexersProps {
  indexers: Indexer[];
}

function FilterIndexers({ indexers }: FilterIndexersProps) {
  if (!indexers.length) {
    return (
      <span className="hidden sm:inline-flex items-center px-2 py-1 rounded-md text-xs font-medium uppercase text-white bg-red-750">
        NO INDEXER
      </span>
    );
  }

  const res = indexers.slice(2);

  return (
    <div className="flex flex-row gap-1">
      <IndexerTag indexer={indexers[0]} />
      {indexers.length > 1 ? (
        <IndexerTag indexer={indexers[1]} />
      ) : null}
      {indexers.length > 2 ? (
        <span
          className="mr-2 inline-flex items-center px-2 py-0.5 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400"
          title={res.map(v => v.name).toString()}
        >
          +{indexers.length - 2}
        </span>
      ) : null}
    </div>
  );
}

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
  <Listbox
    refName={id}
    value={currentValue}
    onChange={onChange}
  >
    <div className="relative">
      <ListboxButton className="relative w-full py-2 pr-4 text-left dark:text-gray-400 text-sm">
        <span className="block truncate">{label}</span>
        <span className="absolute inset-y-0 right-0 flex items-center pointer-events-none">
          <ChevronDownIcon
            className="w-3 h-3"
            aria-hidden="true"
          />
        </span>
      </ListboxButton>
      <Transition
        as={Fragment}
        leave="transition ease-in duration-100"
        leaveFrom="opacity-100"
        leaveTo="opacity-0"
      >
        <ListboxOptions
          className="w-52 absolute z-10 mt-1 right-0 overflow-auto text-base bg-white dark:bg-gray-800 rounded-md shadow-lg max-h-60 border border-opacity-5 border-black dark:border-gray-700 dark:border-opacity-40 focus:outline-hidden sm:text-sm"
        >
          {children}
        </ListboxOptions>
      </Transition>
    </div>
  </Listbox>
);

// a unique option from a list
const IndexerSelectFilter = ({ dispatch }: any) => {
  const filterListState = FilterListContext.useValue();

  const { data, isSuccess } = useQuery(IndexersOptionsQueryOptions());

  const setFilter = (value: string) => {
    if (value == undefined || value == "") {
      dispatch({ type: ActionType.INDEXER_FILTER_RESET, payload: [] });
    } else {
      dispatch({ type: ActionType.INDEXER_FILTER_CHANGE, payload: [value] });
    }
  };

  // Render a multi-select box
  return (
    <ListboxFilter
      id="1"
      key="indexer-select"
      label={data && filterListState.indexerFilter[0] ? `Indexer: ${data.find(i => i.identifier == filterListState.indexerFilter[0])?.name}` : "Indexer"}
      currentValue={filterListState.indexerFilter[0] ?? ""}
      onChange={setFilter}
    >
      <FilterOption label="All" value="" />
      {isSuccess && data?.map((indexer, idx) => (
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
    className={({ active }) => classNames(
      "cursor-pointer select-none relative py-2 px-4",
      active ? "text-black dark:text-gray-200 bg-gray-100 dark:bg-gray-900" : "text-gray-700 dark:text-gray-400"
    )}
    value={value}
  >
    {({ selected }) => (
      <div className="flex justify-between">
        <span
          className={classNames(
            "block truncate",
            selected ? "font-medium text-black dark:text-white" : "font-normal"
          )}
        >
          {label}
        </span>
        {selected ? (
          <span className="absolute inset-y-0 right-0 flex items-center pr-3 text-gray-500 dark:text-gray-400">
            <CheckIcon className="w-5 h-5" aria-hidden="true" />
          </span>
        ) : null}
      </div>
    )}
  </ListboxOption>
);

export const SortSelectFilter = ({ dispatch }: any) => {
  const filterListState = FilterListContext.useValue();

  const setFilter = (value: string) => {
    if (value == undefined || value == "") {
      dispatch({ type: ActionType.SORT_ORDER_RESET, payload: "" });
    } else {
      dispatch({ type: ActionType.SORT_ORDER_CHANGE, payload: value });
    }
  };

  const options = [
    { label: "Name A-Z", value: "name-asc" },
    { label: "Name Z-A", value: "name-desc" },
    { label: "Priority highest", value: "priority-desc" },
    { label: "Priority lowest", value: "priority-asc" },
    { label: "Recently created first", value: "created_at-desc" },
    { label: "Recently created last", value: "created_at-asc" },
    { label: "Recently updated first", value: "updated_at-desc" },
    { label: "Recently updated last", value: "updated_at-asc" }
  ];

  // Render a multi-select box
  return (
    <ListboxFilter
      id="sort"
      key="sort-select"
      label={filterListState.sortOrder ? `Sort: ${options.find(o => o.value == filterListState.sortOrder)?.label}` : "Sort"}
      currentValue={filterListState.sortOrder ?? ""}
      onChange={setFilter}
    >
      <>
        <FilterOption label="Reset" />
        {options.map((f, idx) =>
          <FilterOption key={idx} label={f.label} value={f.value} />
        )}
      </>
    </ListboxFilter>
  );
};
