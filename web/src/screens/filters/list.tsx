import {Dispatch, FC, Fragment, MouseEventHandler, useReducer, useRef, useState} from "react";
import {Link} from "react-router-dom";
import {toast} from "react-hot-toast";
import {Listbox, Menu, Switch, Transition} from "@headlessui/react";
import {useMutation, useQuery, useQueryClient} from "react-query";
import {
  ArrowsRightLeftIcon,
  CheckIcon,
  ChevronDownIcon,
  DocumentDuplicateIcon,
  EllipsisHorizontalIcon,
  PencilSquareIcon,
  TrashIcon
} from "@heroicons/react/24/outline";

import {queryClient} from "../../App";
import {classNames} from "../../utils";
import {FilterAddForm} from "../../forms";
import {useToggle} from "../../hooks/hooks";
import {APIClient} from "../../api/APIClient";
import Toast from "../../components/notifications/Toast";
import {EmptyListState} from "../../components/emptystates";
import {DeleteModal} from "../../components/modals";

type FilterListState = {
  indexerFilter: string[],
  sortOrder: string;
  status: string;
};

const initialState: FilterListState = {
  indexerFilter: [],
  sortOrder: "",
  status: ""
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
  const [createFilterIsOpen, toggleCreateFilter] = useToggle(false);

  return (
    <main>
      <FilterAddForm isOpen={createFilterIsOpen} toggle={toggleCreateFilter} />

      <header className="py-10">
        <div className="max-w-screen-xl mx-auto px-4 sm:px-6 lg:px-8 flex justify-between">
          <h1 className="text-3xl font-bold text-black dark:text-white">
            Filters
          </h1>
          <div className="flex-shrink-0">
            <button
              type="button"
              className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 dark:bg-blue-600 hover:bg-indigo-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
              onClick={toggleCreateFilter}
            >
              Add new
            </button>
          </div>
        </div>
      </header>

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
  const [{ indexerFilter, sortOrder, status }, dispatchFilter] =
    useReducer(FilterListReducer, initialState);

  const { error, data } = useQuery(
    ["filters", indexerFilter, sortOrder],
    () => APIClient.filters.find(indexerFilter, sortOrder),
    { refetchOnWindowFocus: false }
  );

  if (error) {
    return (<p>An error has occurred: </p>);
  }

  const filtered = filteredData(data ?? [], status);

  return (
    <div className="max-w-screen-xl mx-auto pb-12 px-4 sm:px-6 lg:px-8 relative">
      <div className="align-middle min-w-full rounded-t-md rounded-b-lg shadow-lg bg-gray-50 dark:bg-gray-800">
        <div className="flex justify-between px-4 bg-gray-50 dark:bg-gray-800  border-b border-gray-200 dark:border-gray-700">
          <div className="flex gap-4">
            <StatusButton data={filtered.all} label="All" value="" currentValue={status} dispatch={dispatchFilter} />
            <StatusButton data={filtered.enabled} label="Enabled" value="enabled" currentValue={status} dispatch={dispatchFilter} />
            <StatusButton data={filtered.disabled} label="Disabled" value="disabled" currentValue={status} dispatch={dispatchFilter} />
          </div>

          <div className="flex items-center gap-5">
            <IndexerSelectFilter dispatch={dispatchFilter} />
            <SortSelectFilter dispatch={dispatchFilter} />
          </div>
        </div>

        {data && data.length > 0 ? (
          <ol className="min-w-full">
            {filtered.filtered.map((filter: Filter, idx) => (
              <FilterListItem filter={filter} key={filter.id} idx={idx} />
            ))}
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
  filter: Filter;
  onToggle: (newState: boolean) => void;
}

const FilterItemDropdown = ({
  filter,
  onToggle
}: FilterItemDropdownProps) => {
  const cancelModalButtonRef = useRef(null);

  const queryClient = useQueryClient();

  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);
  const deleteMutation = useMutation(
    (id: number) => APIClient.filters.delete(id),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(["filters"]);
        queryClient.invalidateQueries(["filters", filter.id]);

        toast.custom((t) => <Toast type="success" body={`Filter ${filter?.name} was deleted`} t={t} />);
      }
    }
  );

  const duplicateMutation = useMutation(
    (id: number) => APIClient.filters.duplicate(id),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(["filters"]);

        toast.custom((t) => <Toast type="success" body={`Filter ${filter?.name} duplicated`} t={t} />);
      }
    }
  );

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
          className="absolute right-0 w-56 mt-2 origin-top-right bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700 rounded-md shadow-lg ring-1 ring-black ring-opacity-10 focus:outline-none"
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
  idx: number;
}

function FilterListItem({ filter, idx }: FilterListItemProps) {
  const [enabled, setEnabled] = useState(filter.enabled);

  const updateMutation = useMutation(
    (status: boolean) => APIClient.filters.toggleEnable(filter.id, status),
    {
      onSuccess: () => {
        toast.custom((t) => <Toast type="success" body={`${filter.name} was ${enabled ? "disabled" : "enabled"} successfully`} t={t} />);

        // We need to invalidate both keys here.
        // The filters key is used on the /filters page,
        // while the ["filter", filter.id] key is used on the details page.
        queryClient.invalidateQueries(["filters"]);
        queryClient.invalidateQueries(["filters", filter?.id]);
      }
    }
  );

  const toggleActive = (status: boolean) => {
    setEnabled(status);
    updateMutation.mutate(status);
  };

  return (
    <li
      key={filter.id}
      className={classNames(
        "flex items-center hover:bg-gray-100 dark:hover:bg-[#222225]",
        idx % 2 === 0 ?
          "bg-white dark:bg-[#2e2e31]" :
          "bg-gray-50 dark:bg-gray-800"
      )}
    >
      <span
        className="px-4 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-100"
      >
        <Switch
          checked={enabled}
          onChange={toggleActive}
          className={classNames(
            enabled ? "bg-teal-500 dark:bg-blue-500" : "bg-gray-200 dark:bg-gray-700",
            "relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          )}
        >
          <span className="sr-only">Use setting</span>
          <span
            aria-hidden="true"
            className={classNames(
              enabled ? "translate-x-5" : "translate-x-0",
              "inline-block h-5 w-5 rounded-full bg-white dark:bg-gray-200 shadow transform ring-0 transition ease-in-out duration-200"
            )}
          />
        </Switch>
      </span>
      <div className="flex flex-col w-full justify-center">
        <span className="whitespace-nowrap text-sm font-bold text-gray-900 dark:text-gray-100">
          <Link
            to={filter.id.toString()}
            className="hover:text-black dark:hover:text-gray-300"
          >
            {filter.name}
          </Link>
        </span>
        <div className="flex-col">
          <span className="mr-2 whitespace-nowrap text-xs font-medium text-gray-600 dark:text-gray-400">
            Priority: {filter.priority}
          </span>
          <span className="whitespace-nowrap text-xs font-medium text-gray-600 dark:text-gray-400">
            <Link
              to={`${filter.id.toString()}/actions`}
              className="hover:text-black dark:hover:text-gray-300"
            >
              <span className={classNames(filter.actions_count == 0 ? "text-red-500" : "")}>Actions: {filter.actions_count}</span>
            </Link>
          </span>
        </div>
      </div>
      <span className="px-4 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
        <FilterIndexers indexers={filter.indexers} />
      </span>
      <span className="px-4 py-4 whitespace-nowrap text-right text-sm font-medium">
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
    key={indexer.id}
    className="mr-2 inline-flex items-center px-2.5 py-0.5 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400"
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
          : <span className="text-red-400 dark:text-red-800 p-1 text-xs tracking-wide rounded border border-red-400 dark:border-red-700 bg-red-100 dark:bg-red-400">NO INDEXERS SELECTED</span>
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
  <div className="">
    <Listbox
      refName={id}
      value={currentValue}
      onChange={onChange}
    >
      <div className="relative">
        <Listbox.Button className="relative w-full py-2 pr-5 text-left cursor-default dark:text-gray-400 sm:text-sm">
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
            className="w-48 absolute z-10 w-full mt-1 right-0 overflow-auto text-base bg-white dark:bg-gray-800 rounded-md shadow-lg max-h-60 border border-opacity-5 border-black dark:border-gray-700 dark:border-opacity-40 focus:outline-none sm:text-sm"
          >
            {children}
          </Listbox.Options>
        </Transition>
      </div>
    </Listbox>
  </div>
);

// a unique option from a list
const IndexerSelectFilter = ({ dispatch }: any) => {
  const { data, isSuccess } = useQuery(
    "release_indexers",
    () => APIClient.indexers.getOptions(),
    {
      keepPreviousData: true,
      staleTime: Infinity
    }
  );

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
