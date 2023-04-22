import { Dispatch, FC, Fragment, MouseEventHandler, useReducer, useRef, useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { toast } from "react-hot-toast";
import { Listbox, Menu, Switch, Transition } from "@headlessui/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FormikValues } from "formik";
import { useCallback } from "react";
import { Tooltip } from "react-tooltip";
import {
  ArrowsRightLeftIcon,
  CheckIcon,
  ChevronDownIcon,
  PlusIcon,
  DocumentDuplicateIcon,
  EllipsisHorizontalIcon,
  PencilSquareIcon,
  ChatBubbleBottomCenterTextIcon,
  TrashIcon
} from "@heroicons/react/24/outline";
import { ArrowDownTrayIcon } from "@heroicons/react/24/solid";

import { FilterListContext, FilterListState } from "@utils/Context";
import { classNames } from "@utils";
import { FilterAddForm } from "@forms";
import { useToggle } from "@hooks/hooks";
import { APIClient } from "@api/APIClient";
import Toast from "@components/notifications/Toast";
import { EmptyListState } from "@components/emptystates";
import { DeleteModal } from "@components/modals";

export const filterKeys = {
  all: ["filters"] as const,
  lists: () => [...filterKeys.all, "list"] as const,
  list: (indexers: string[], sortOrder: string) => [...filterKeys.lists(), { indexers, sortOrder }] as const,
  details: () => [...filterKeys.all, "detail"] as const,
  detail: (id: number) => [...filterKeys.details(), id] as const
};

enum ActionType {
  INDEXER_FILTER_CHANGE = "INDEXER_FILTER_CHANGE",
  INDEXER_FILTER_RESET = "INDEXER_FILTER_RESET",
  SORT_ORDER_CHANGE = "SORT_ORDER_CHANGE",
  SORT_ORDER_RESET = "SORT_ORDER_RESET",
  STATUS_CHANGE = "STATUS_RESET",
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
  case ActionType.INDEXER_FILTER_CHANGE:
    return { ...state, indexerFilter: action.payload };
  case ActionType.INDEXER_FILTER_RESET:
    return { ...state, indexerFilter: [] };
  case ActionType.SORT_ORDER_CHANGE:
    return { ...state, sortOrder: action.payload };
  case ActionType.SORT_ORDER_RESET:
    return { ...state, sortOrder: "" };
  case ActionType.STATUS_CHANGE:
    return { ...state, status: action.payload };
  case ActionType.STATUS_RESET:
    return { ...state };
  default:
    throw new Error(`Unhandled action type: ${action}`);
  }
};

export default function Filters() {
  const queryClient = useQueryClient();

  const [createFilterIsOpen, setCreateFilterIsOpen] = useState(false);
  const toggleCreateFilter = () => {
    setCreateFilterIsOpen(!createFilterIsOpen);
  };  

  const [showImportModal, setShowImportModal] = useState(false);
  const [importJson, setImportJson] = useState("");
  
  // This function handles the import of a filter from a JSON string
  const handleImportJson = async () => {
    try {
      const importedData = JSON.parse(importJson);
  
      // Extract the filter data and name from the imported object
      const importedFilter = importedData.data;
      const filterName = importedData.name;
  
      // Check if the required properties are present and add them with default values if they are missing
      const requiredProperties = ["resolutions", "sources", "codecs", "containers"];
      requiredProperties.forEach((property) => {
        if (!importedFilter.hasOwnProperty(property)) {
          importedFilter[property] = [];
        }
      });
  
      // Fetch existing filters from the API
      const existingFilters = await APIClient.filters.getAll();
  
      // Create a unique filter title by appending an incremental number if title is taken by another filter
      let nameCounter = 0;
      let uniqueFilterName = filterName;
      while (existingFilters.some((filter) => filter.name === uniqueFilterName)) {
        nameCounter++;
        uniqueFilterName = `${filterName}-${nameCounter}`;
      }
  
      // Create a new filter using the API
      const newFilter: Filter = {
        ...importedFilter,
        name: uniqueFilterName
      };
  
      await APIClient.filters.create(newFilter);
  
      // Update the filter list
      queryClient.invalidateQueries({ queryKey: filterKeys.lists() });
  
      toast.custom((t) => <Toast type="success" body="Filter imported successfully." t={t} />);
      setShowImportModal(false);
    } catch (error) {
      // Log the error and show an error toast message
      console.error("Error:", error);
      toast.custom((t) => <Toast type="error" body="Failed to import JSON data. Please check your input." t={t} />);
    }
  };
  
  return (
    <main>
      <FilterAddForm isOpen={createFilterIsOpen} toggle={toggleCreateFilter} />
      <header className="py-10">
        <div className="max-w-screen-xl mx-auto px-4 sm:px-6 lg:px-8 flex justify-between">
          <h1 className="text-3xl font-bold text-black dark:text-white">Filters</h1>
          <div className="relative">
            <Menu>
              {({ open }) => (
                <>
                  <button
                    className="relative inline-flex items-center px-4 py-2 shadow-sm text-sm font-medium rounded-l-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                    onClick={(e: { stopPropagation: () => void; }) => {
                      if (!open) {
                        e.stopPropagation();
                        toggleCreateFilter();
                      }
                    }}
                  >
                    <PlusIcon className="h-5 w-5 mr-1" />
                    Add Filter
                  </button>
                  <Menu.Button className="relative inline-flex items-center px-2 py-2 border-l border-spacing-1 dark:border-black shadow-sm text-sm font-medium rounded-r-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500">
                    <ChevronDownIcon className="h-5 w-5" />
                  </Menu.Button>
                  <Transition
                    show={open}
                    enter="transition ease-out duration-100 transform"
                    enterFrom="opacity-0 scale-95"
                    enterTo="opacity-100 scale-100"
                    leave="transition ease-in duration-75 transform"
                    leaveFrom="opacity-100 scale-100"
                    leaveTo="opacity-0 scale-95"
                  >
                    <Menu.Items className="absolute right-0 mt-0.5 w-46 bg-white dark:bg-gray-700 rounded-md shadow-lg">
                      <Menu.Item>
                        {({ active }) => (
                          <button
                            type="button"
                            className={`${
                              active
                                ? "bg-gray-50 dark:bg-gray-600"
                                : ""
                            } w-full text-left py-2 px-4 text-sm font-medium text-gray-700 dark:text-gray-200 rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500`}
                            onClick={() => setShowImportModal(true)}
                          >
                            Import Filter
                          </button>
                        )}
                      </Menu.Item>
                    </Menu.Items>
                  </Transition>
                </>
              )}
            </Menu>
          </div>
        </div>
      </header>

      {showImportModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="w-1/2 md:w-1/2 bg-white dark:bg-gray-800 p-6 rounded-md shadow-lg">
            <h2 className="text-lg font-medium mb-4 text-black dark:text-white">Import Filter JSON</h2>
            <textarea
              className="form-input block w-full resize-y rounded-md border-gray-300 dark:bg-gray-800 dark:border-gray-600 shadow-sm text-sm font-medium text-gray-700 dark:text-white focus:outline-none focus:ring-2  focus:ring-blue-500 dark:focus:ring-blue-500 mb-4"
              placeholder="Paste JSON data here"
              value={importJson}
              onChange={(event) => setImportJson(event.target.value)}
              style={{ minHeight: "30vh", maxHeight: "50vh" }}
            />
            <div className="flex justify-end">
              <button
                type="button"
                className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                onClick={() => setShowImportModal(false)}
              >
              Cancel
              </button>
              <button
                type="button"
                className="ml-4 relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                onClick={handleImportJson}
              >
              Import
              </button>
            </div>
          </div>
        </div>
      )}
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

  const { data, error } = useQuery({
    queryKey: filterKeys.list(indexerFilter, sortOrder),
    queryFn: ({ queryKey }) => APIClient.filters.find(queryKey[2].indexers, queryKey[2].sortOrder),
    refetchOnWindowFocus: false
  });

  useEffect(() => {
    FilterListContext.set({ indexerFilter, sortOrder, status });
  }, [indexerFilter, sortOrder, status]);

  if (error) {
    return <p>An error has occurred:</p>;
  }

  const filtered = filteredData(data ?? [], status);

  return (
    <div className="max-w-screen-xl mx-auto pb-12 px-4 sm:px-6 lg:px-8 relative">
      <div className="align-middle min-w-full rounded-t-lg rounded-b-lg shadow-lg bg-gray-50 dark:bg-gray-800">
        <div className="rounded-t-lg flex justify-between px-4 bg-gray-50 dark:bg-gray-800  border-b border-gray-200 dark:border-gray-700">
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

        {data && data.length > 0 ? (
          <ol className="min-w-full">
            {filtered.filtered.length > 0
              ? filtered.filtered.map((filter: Filter, idx) => (
                <FilterListItem filter={filter} values={filter} key={filter.id} idx={idx} />
              ))

              : <EmptyListState text={`No ${status} filters`} />
            }
          </ol>
        ) : (
          <EmptyListState text="No filters here.." buttonText="Add new" buttonOnClick={toggleCreateFilter} />
        )}
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
        currentValue == value ? "font-bold border-b-2 border-blue-500 dark:text-gray-100 text-gray-900" : "font-medium text-gray-600 dark:text-gray-400",
        "py-4 pb-4 text-left text-xs tracking-wider"
      )}
      onClick={setFilter}
      value={value}
    >
      {data?.length ?? 0} {label}
    </button>
  );
};

interface FilterItemDropdownProps {
  values: FormikValues;
  filter: Filter;
  onToggle: (newState: boolean) => void;
}

const FilterItemDropdown = ({ filter, onToggle }: FilterItemDropdownProps) => {

  // This function handles the export of a filter to a JSON string
  const handleExportJson = useCallback(async (discordFormat = false) => {    try {
      type CompleteFilterType = {
        id: number;
        name: string;
        created_at: Date;
        updated_at: Date;
        indexers: any;
        actions: any;
        actions_count: any;
        external_script_enabled: any;
        external_script_cmd: any;
        external_script_args: any;
        external_script_expect_status: any;
        external_webhook_enabled: any;
        external_webhook_host: any;
        external_webhook_data: any;
        external_webhook_expect_status: any;
      };
  
      const completeFilter = await APIClient.filters.getByID(filter.id) as Partial<CompleteFilterType>;
  
      // Extract the filter name and remove unwanted properties
      const title = completeFilter.name;
      delete completeFilter.name;
      delete completeFilter.id;
      delete completeFilter.created_at;
      delete completeFilter.updated_at;
      delete completeFilter.actions_count;
      delete completeFilter.indexers;
      delete completeFilter.actions;
      delete completeFilter.external_script_enabled;
      delete completeFilter.external_script_cmd;
      delete completeFilter.external_script_args;
      delete completeFilter.external_script_expect_status;
      delete completeFilter.external_webhook_enabled;
      delete completeFilter.external_webhook_host;
      delete completeFilter.external_webhook_data;
      delete completeFilter.external_webhook_expect_status;
  
      // Remove properties with default values from the exported filter to minimize the size of the JSON string
      ["enabled", "priority", "smart_episode", "resolutions", "sources", "codecs", "containers"].forEach((key) => {
        const value = completeFilter[key as keyof CompleteFilterType];
        if (["enabled", "priority", "smart_episode"].includes(key) && (value === false || value === 0)) {
          delete completeFilter[key as keyof CompleteFilterType];
        } else if (["resolutions", "sources", "codecs", "containers"].includes(key) && Array.isArray(value) && value.length === 0) {
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

      const copyTextToClipboard = (text: string) => {
        const textarea = document.createElement("textarea");
        textarea.style.position = "fixed";
        textarea.style.opacity = "0";
        textarea.value = text;
        document.body.appendChild(textarea);
        textarea.focus();
        textarea.select();
  
        try {
          const successful = document.execCommand("copy");
          if (successful) {
            toast.custom((t) => <Toast type="success" body="Filter copied to clipboard." t={t} />);
          } else {
            toast.custom((t) => <Toast type="error" body="Failed to copy JSON to clipboard." t={t} />);
          }
        } catch (err) {
          console.error("Unable to copy text", err);
          toast.custom((t) => <Toast type="error" body="Failed to copy JSON to clipboard." t={t} />);
        }
  
        document.body.removeChild(textarea);
      };
  
      if (navigator.clipboard) {
        navigator.clipboard.writeText(finalJson).then(() => {
          toast.custom((t) => <Toast type="success" body="Filter copied to clipboard." t={t} />);
        }, () => {
          toast.custom((t) => <Toast type="error" body="Failed to copy JSON to clipboard." t={t} />);
        });
      } else {
        copyTextToClipboard(finalJson);
      }
  
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
      queryClient.invalidateQueries({ queryKey: filterKeys.lists() });
      queryClient.invalidateQueries({ queryKey: filterKeys.detail(filter.id) });

      toast.custom((t) => <Toast type="success" body={`Filter ${filter?.name} was deleted`} t={t} />);
    }
  });

  const duplicateMutation = useMutation({
    mutationFn: (id: number) => APIClient.filters.duplicate(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: filterKeys.lists() });

      toast.custom((t) => <Toast type="success" body={`Filter ${filter?.name} duplicated`} t={t} />);
    }
  });

  return (
    <Menu as="div">
      <DeleteModal
        isOpen={deleteModalIsOpen}
        toggle={toggleDeleteModal}
        buttonRef={cancelModalButtonRef}
        deleteAction={() => {
          deleteMutation.mutate(filter.id);
          toggleDeleteModal();
        }}
        title={`Remove filter: ${filter.name}`}
        text="Are you sure you want to remove this filter? This action cannot be undone."
      />
      <Menu.Button className="px-4 py-2">
        <EllipsisHorizontalIcon
          className="w-5 h-5 text-gray-700 hover:text-gray-900 dark:text-gray-100 dark:hover:text-gray-400"
          aria-hidden="true"
        />
      </Menu.Button>
      <Transition
        as={Fragment}
        enter="transition ease-out duration-100"
        enterFrom="transform opacity-0 scale-95"
        enterTo="transform opacity-100 scale-100"
        leave="transition ease-in duration-75"
        leaveFrom="transform opacity-100 scale-100"
        leaveTo="transform opacity-0 scale-95"
      >
        <Menu.Items
          className="absolute right-0 w-56 mt-2 origin-top-right bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700 rounded-md shadow-lg ring-1 ring-black ring-opacity-10 focus:outline-none z-10"
        >
          <div className="px-1 py-1">
            <Menu.Item>
              {({ active }) => (
                <Link
                  to={filter.id.toString()}
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
            </Menu.Item>
            <Menu.Item>
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
            </Menu.Item>
            <Menu.Item>
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
            </Menu.Item>
            <Menu.Item>
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
            </Menu.Item>
            <Menu.Item>
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
            </Menu.Item>
          </div>
          <div className="px-1 py-1">
            <Menu.Item>
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
            </Menu.Item>
          </div>
        </Menu.Items>
      </Transition>
    </Menu>
  );
};

interface FilterListItemProps {
  filter: Filter;
  values: FormikValues;
  idx: number;
}

function FilterListItem({ filter, values, idx }: FilterListItemProps) {
  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: (status: boolean) => APIClient.filters.toggleEnable(filter.id, status),
    onSuccess: () => {
      toast.custom((t) => <Toast type="success" body={`${filter.name} was ${!filter.enabled ? "disabled" : "enabled"} successfully`} t={t} />);

      // We need to invalidate both keys here.
      // The filters key is used on the /filters page,
      // while the ["filter", filter.id] key is used on the details page.
      queryClient.invalidateQueries({ queryKey: filterKeys.lists() });
      queryClient.invalidateQueries({ queryKey: filterKeys.detail(filter.id) });
    }
  });

  const toggleActive = (status: boolean) => {
    updateMutation.mutate(status);
  };

  return (
    <li
      key={filter.id}
      className={classNames(
        "flex items-center hover:bg-gray-100 dark:hover:bg-[#222225] rounded-b-lg",
        idx % 2 === 0 ?
          "bg-white dark:bg-[#2e2e31]" :
          "bg-gray-50 dark:bg-gray-800"
      )}
    >
      <span
        className="px-4 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-100"
      >
        <Switch
          checked={filter.enabled}
          onChange={toggleActive}
          className={classNames(
            filter.enabled ? "bg-blue-500 dark:bg-blue-500" : "bg-gray-200 dark:bg-gray-700",
            "relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          )}
        >
          <span className="sr-only">Use setting</span>
          <span
            aria-hidden="true"
            className={classNames(
              filter.enabled ? "translate-x-5" : "translate-x-0",
              "inline-block h-5 w-5 rounded-full bg-white dark:bg-gray-200 shadow transform ring-0 transition ease-in-out duration-200"
            )}
          />
        </Switch>
      </span>
      <div className="py-2 flex flex-col overflow-hidden w-full justify-center">
        <span className="w-full break-words whitespace-wrap text-sm font-bold text-gray-900 dark:text-gray-100">
          <Link
            to={filter.id.toString()}
            className="hover:text-black dark:hover:text-gray-300"
          >
            {filter.name}
          </Link>
        </span>
        <div className="flex items-center">
          <span className="mr-2 break-words whitespace-nowrap text-xs font-medium text-gray-600 dark:text-gray-400">
            Priority: {filter.priority}
          </span>
          <span className="whitespace-nowrap text-xs font-medium text-gray-600 dark:text-gray-400">
            <Link
              to={`${filter.id.toString()}/actions`}
              className="hover:text-black dark:hover:text-gray-300"
            >
              <span
                id={`tooltip-actions-${filter.id}`}
                className="flex items-center hover:cursor-pointer"
              >
                <span className={classNames(filter.actions_count == 0 ? "text-red-500" : "")}>
                  <span
                    className={
                      classNames(
                        filter.actions_count == 0 ? "hover:text-red-400 dark:hover:text-red-400" : ""
                      )
                    }
                  >
        Actions: {filter.actions_count}
                  </span>
                </span>
                {filter.actions_count === 0 && (
                  <>
                    <span className="mr-2 ml-2 flex h-3 w-3 relative">
                      <span className="animate-ping inline-flex h-full w-full rounded-full dark:bg-red-500 bg-red-400 opacity-75" />
                      <span
                        className="inline-flex absolute rounded-full h-3 w-3 dark:bg-red-500 bg-red-400"
                      />
                    </span>
                    <span className="text-sm text-gray-800 dark:text-gray-500">
                      <Tooltip style={{ width: "350px", fontSize: "12px", textTransform: "none", fontWeight: "normal", borderRadius: "0.375rem", backgroundColor: "#34343A", color: "#fff", opacity: "1", whiteSpace: "pre-wrap", overflow: "hidden", textOverflow: "ellipsis" }} delayShow={100} delayHide={150} data-html={true} place="right" data-tooltip-id={`tooltip-actions-${filter.id}`} html="<p>You need to setup an action in the filter otherwise you will not get any snatches.</p>" />
                    </span>
                  </>
                )}
              </span>
            </Link>
          </span>
        </div>
      </div>
      <span className="hidden md:flex px-4 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
        <FilterIndexers indexers={filter.indexers} />
      </span>
      <span className="min-w-fit px-4 py-4 whitespace-nowrap text-right text-sm font-medium">
        <FilterItemDropdown
          values={values}
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
    key={indexer.id}
    className="hidden sm:inline-flex mr-2 items-center px-2.5 py-0.5 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400"
  >
    {indexer.name}
  </span>
);

interface FilterIndexersProps {
  indexers: Indexer[];
}

function FilterIndexers({ indexers }: FilterIndexersProps) {
  if (indexers.length <= 2) {
    return (
      <>
        {indexers.length > 0
          ? indexers.map((indexer, idx) => (
            <IndexerTag key={idx} indexer={indexer} />
          ))
          : <span className="hidden sm:flex text-red-400 dark:text-red-800 p-1 text-xs tracking-wide rounded border border-red-400 dark:border-red-700 bg-red-100 dark:bg-red-400">NO INDEXERS SELECTED</span>
        }
      </>
    );
  }

  const res = indexers.slice(2);

  return (
    <>
      <IndexerTag indexer={indexers[0]} />
      <IndexerTag indexer={indexers[1]} />
      <span
        className="mr-2 inline-flex items-center px-2.5 py-0.5 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400"
        title={res.map(v => v.name).toString()}
      >
          +{indexers.length - 2}
      </span>
    </>
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
      <Listbox.Button className="relative w-full py-2 pr-5 text-left dark:text-gray-400 text-sm">
        <span className="block truncate">{label}</span>
        <span className="absolute inset-y-0 right-0 flex items-center pointer-events-none">
          <ChevronDownIcon
            className="w-3 h-3 text-gray-600 hover:text-gray-600"
            aria-hidden="true"
          />
        </span>
      </Listbox.Button>
      <Transition
        as={Fragment}
        leave="transition ease-in duration-100"
        leaveFrom="opacity-100"
        leaveTo="opacity-0"
      >
        <Listbox.Options
          className="w-52 absolute z-10 mt-1 right-0 overflow-auto text-base bg-white dark:bg-gray-800 rounded-md shadow-lg max-h-60 border border-opacity-5 border-black dark:border-gray-700 dark:border-opacity-40 focus:outline-none sm:text-sm"
        >
          {children}
        </Listbox.Options>
      </Transition>
    </div>
  </Listbox>
);

// a unique option from a list
const IndexerSelectFilter = ({ dispatch }: any) => {
  const { data, isSuccess } = useQuery({
    queryKey: ["filters","indexers_options"],
    queryFn: () => APIClient.indexers.getOptions(),
    keepPreviousData: true,
    staleTime: Infinity
  });

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
      label="Indexer"
      currentValue={""}
      onChange={setFilter}
    >
      <FilterOption label="All" />
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
  <Listbox.Option
    className={({ active }) => classNames(
      "cursor-pointer select-none relative py-2 px-4",
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

export const SortSelectFilter = ({ dispatch }: any) => {
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
    { label: "Priority lowest", value: "priority-asc" }
  ];

  // Render a multi-select box
  return (
    <ListboxFilter
      id="sort"
      key="sort-select"
      label="Sort"
      currentValue={""}
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
