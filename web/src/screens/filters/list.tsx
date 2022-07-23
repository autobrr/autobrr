import { Fragment, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { toast } from "react-hot-toast";
import { Menu, Switch, Transition } from "@headlessui/react";
import { useMutation, useQuery, useQueryClient } from "react-query";
import {
  TrashIcon,
  PencilAltIcon,
  SwitchHorizontalIcon,
  DotsHorizontalIcon, DuplicateIcon
} from "@heroicons/react/outline";

import { queryClient } from "../../App";
import { classNames } from "../../utils";
import { FilterAddForm } from "../../forms";
import { useToggle } from "../../hooks/hooks";
import { APIClient } from "../../api/APIClient";
import Toast from "../../components/notifications/Toast";
import { EmptyListState } from "../../components/emptystates";
import { DeleteModal } from "../../components/modals";

export default function Filters() {
  const [createFilterIsOpen, toggleCreateFilter] = useToggle(false);

  const { isLoading, error, data } = useQuery(
    ["filters"],
    () => APIClient.filters.getAll(),
    { refetchOnWindowFocus: false }
  );

  if (isLoading)
    return null;

  if (error)
    return (<p>An error has occurred: </p>);

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

      <div className="max-w-screen-xl mx-auto pb-12 px-4 sm:px-6 lg:px-8 relative">
        {data && data.length > 0 ? (
          <FilterList filters={data} />
        ) : (
          <EmptyListState text="No filters here.." buttonText="Add new" buttonOnClick={toggleCreateFilter} />
        )}
      </div>
    </main>
  );
}

interface FilterListProps {
    filters: Filter[];
}

function FilterList({ filters }: FilterListProps) {
  return (
    <div className="overflow-x-auto align-middle min-w-full rounded-t-md rounded-b-lg shadow-lg">
      <table className="min-w-full">
        <thead className="bg-gray-50 dark:bg-gray-800 text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-700">
          <tr>
            {["Enabled", "Name", "Actions", "Indexers"].map((label) => (
              <th
                key={`th-${label}`}
                scope="col"
                className="px-4 pt-4 pb-3 text-left text-xs font-medium uppercase tracking-wider"
              >
                {label}
              </th>
            ))}
            <th scope="col" className="relative px-4 py-3">
              <span className="sr-only">Edit</span>
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
          {filters.map((filter: Filter, idx) => (
            <FilterListItem filter={filter} key={filter.id} idx={idx} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

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
        <DotsHorizontalIcon
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
                  <PencilAltIcon
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
                  <SwitchHorizontalIcon
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
                  <DuplicateIcon
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
    <tr
      key={filter.id}
      className={classNames(
        idx % 2 === 0 ?
          "bg-white dark:bg-[#2e2e31]" :
          "bg-gray-50 dark:bg-gray-800",
        "hover:bg-gray-100 dark:hover:bg-[#222225]"
      )}
    >
      <td
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
      </td>
      <td className="px-4 w-full whitespace-nowrap text-sm font-medium text-gray-900 dark:text-gray-100">
        <Link
          to={filter.id.toString()}
          className="hover:text-black dark:hover:text-gray-300 w-full py-4 flex"
        >
          {filter.name}
        </Link>
      </td>
      <td className="px-4 w-full whitespace-nowrap text-sm font-medium text-gray-900 dark:text-gray-100">
        <Link
          to={`${filter.id.toString()}/actions`}
          className="hover:text-black dark:hover:text-gray-300 w-full py-4 flex"
        >
          <span className={classNames(filter.actions_count == 0 ? "text-red-500" : "")}>{filter.actions_count}</span>
        </Link>
      </td>
      <td className="px-4 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
        {filter.indexers && filter.indexers.map((t) => (
          <span
            key={t.id}
            className="mr-2 inline-flex items-center px-2.5 py-0.5 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400"
          >
            {t.name}
          </span>
        ))}
      </td>
      <td className="px-4 py-4 whitespace-nowrap text-right text-sm font-medium">
        <FilterItemDropdown
          filter={filter}
          onToggle={toggleActive}
        />
      </td>
    </tr>
  );
}