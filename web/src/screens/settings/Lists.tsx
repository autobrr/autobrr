/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { FC, Fragment, useRef } from "react";
import { Link } from "@tanstack/react-router";
import { useMutation, useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { Menu, MenuButton, MenuItem, MenuItems, Transition } from "@headlessui/react";
import { PlusIcon } from "@heroicons/react/24/solid";
import { ClipboardDocumentIcon, EllipsisHorizontalIcon, ForwardIcon, PencilSquareIcon, TrashIcon } from "@heroicons/react/24/outline";

import { useToggle } from "@hooks/hooks";
import { APIClient } from "@api/APIClient";
import { FeedKeys, ListKeys } from "@api/query_keys";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { Checkbox } from "@components/Checkbox";
import { ListsQueryOptions } from "@api/queries";
import { Section } from "@screens/settings/_components";
import { EmptySimple } from "@components/emptystates";
import { ListAddForm, ListUpdateForm } from "@forms";
import { ListTypeNameMap } from "@domain/constants";
import { classNames, IsErrorWithMessage } from "@utils";
import { DeleteModal } from "@components/modals";

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
          className="relative inline-flex items-center px-4 py-2 cursor-pointer border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
        >
          <PlusIcon className="h-5 w-5 mr-1"/>
          Add new
        </button>
      }
    >
      <ListAddForm isOpen={addFormIsOpen} toggle={toggleAddList} />

      <div className="flex flex-col">
        {lists.length > 0 ? (
          <ul className="min-w-full relative">
            <li className="grid grid-cols-12 border-b border-gray-200 dark:border-gray-700">
              <div
                className="col-span-2 sm:col-span-1 pl-0 sm:pl-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
              >
                Enabled
              </div>
              <div
                className="col-span-6 sm:col-span-4 lg:col-span-4 pl-10 sm:pl-12 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
              >
                Name
              </div>
              <div
                className="hidden sm:flex col-span-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
              >
                Filters
              </div>
              <div
                className="hidden sm:flex col-span-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
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
    className="hidden sm:inline-flex items-center px-2 py-0.5 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400 dark:hover:bg-gray-750 hover:bg-gray-700"
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
  const cancelModalButtonRef = useRef(null);
  const [isOpen, toggleUpdate] = useToggle(false);
  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);

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

  const refreshMutation = useMutation({
    mutationFn: (id: number) => APIClient.lists.refreshList(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: FeedKeys.lists() });
      toast.custom((t) => <Toast type="success" body={`List ${list?.name} was fetched successfully.`} t={t} />);
    },
    onError: (error: unknown) => {
      let errorMessage = 'An unknown error occurred';
      if (IsErrorWithMessage(error)) {
        errorMessage = error.message;
      }

      toast.custom((t) => <Toast type="error" body={`Failed to refresh ${list?.name}. Error: ${errorMessage}`} t={t} />, {
        duration: 10000
      });
    }
  });

  const deleteMutation = useMutation({
    mutationFn: (listID: number) => APIClient.lists.delete(listID),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ListKeys.lists() });

      toast.custom((t) => <Toast type="success" body={`${list.name} was deleted.`} t={t}/>);
    }
  });

  return (
    <li>
      <ListUpdateForm isOpen={isOpen} toggle={toggleUpdate} data={list} />

      <div className="grid grid-cols-12 items-center py-2">
        <div className="col-span-2 sm:col-span-1 pl-1 py-0.5 sm:pl-6 flex items-center">
          <Checkbox value={list.enabled ?? false} setValue={onToggleMutation}/>
        </div>
        <div
          className="col-span-8 sm:col-span-4 lg:col-span-4 pl-10 sm:pl-12 pr-6 block flex-col text-sm font-medium text-gray-900 dark:text-white truncate">
          {list.name}
        </div>
        <div
          className="hidden sm:block col-span-4 pr-6 text-left items-center whitespace-nowrap text-sm text-gray-600 dark:text-gray-400">
          {/*{list.filters.map(filter => <FilterPill filter={filter} key={filter.id} />)}*/}
          <ListItemFilters filters={list.filters} />
        </div>
        <div
          className="hidden sm:block col-span-2 text-left items-center text-sm text-gray-600 dark:text-gray-400">
          {ListTypeNameMap[list.type]}
        </div>
        <div className="col-span-1 pl-0.5 whitespace-nowrap text-center text-sm font-medium">

          <Menu as="div">
            <DeleteModal
              isOpen={deleteModalIsOpen}
              isLoading={deleteMutation.isPending}
              toggle={toggleDeleteModal}
              buttonRef={cancelModalButtonRef}
              deleteAction={() => {
                deleteMutation.mutate(list.id);
                toggleDeleteModal();
              }}
              title={`Remove list: ${list.name}`}
              text="Are you sure you want to remove this list? This action cannot be undone."
            />
            <MenuButton className="px-4 py-2 cursor-pointer">
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
                    {({ focus }) => (
                      <button
                        className={classNames(
                          focus ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                          "font-medium cursor-pointer group flex rounded-md items-center w-full px-2 py-2 text-sm"
                        )}
                        onClick={() => toggleUpdate()}
                      >
                        <PencilSquareIcon
                          className={classNames(
                            focus ? "text-white" : "text-blue-500",
                            "w-5 h-5 mr-2"
                          )}
                          aria-hidden="true"
                        />
                        Edit
                      </button>
                    )}
                  </MenuItem>
                  <MenuItem>
                    {({ focus }) => (
                      <button
                        className={classNames(
                          focus ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                          "font-medium cursor-pointer group flex rounded-md items-center w-full px-2 py-2 text-sm"
                        )}
                        onClick={() => {
                          navigator.clipboard.writeText(String(list.id));
                          toast.custom(t => <Toast type="success" body={`List ID ${list.id} copied to clipboard.`} t={t} />);
                        }}
                      >
                        <ClipboardDocumentIcon
                          className={classNames(
                            focus ? "text-white" : "text-blue-500",
                            "w-5 h-5 mr-2"
                          )}
                          aria-hidden="true"
                        />
                        Copy List ID
                      </button>
                    )}
                  </MenuItem>
                </div>
                <div className="px-1 py-1">
                  <MenuItem>
                    {({ focus }) => (
                      <button
                        onClick={() => refreshMutation.mutate(list.id)}
                        className={classNames(
                          focus ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                          "font-medium cursor-pointer group flex rounded-md items-center w-full px-2 py-2 text-sm"
                        )}
                      >
                        <ForwardIcon
                          className={classNames(
                            focus ? "text-white" : "text-blue-500",
                            "w-5 h-5 mr-2"
                          )}
                          aria-hidden="true"
                        />
                        Refresh
                      </button>
                    )}
                  </MenuItem>
                </div>
                <div className="px-1 py-1">
                  <MenuItem>
                    {({ focus }) => (
                      <button
                        className={classNames(
                          focus ? "bg-red-600 text-white" : "text-gray-900 dark:text-gray-300",
                          "font-medium cursor-pointer group flex rounded-md items-center w-full px-2 py-2 text-sm"
                        )}
                        onClick={() => toggleDeleteModal()}
                      >
                        <TrashIcon
                          className={classNames(
                            focus ? "text-white" : "text-red-500",
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

  const res = filters.slice(1);

  return (
    <div className="flex flex-row gap-1 truncate">
      <FilterPill filter={filters[0]} />
      {filters.length > 1 ? (
        <span
          className="mr-2 inline-flex items-center px-2 py-0.5 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-400"
          title={res.map((v) => v.name).toString()}
        >
          +{filters.length - 1}
        </span>
      ) : null}
    </div>
  );
}
