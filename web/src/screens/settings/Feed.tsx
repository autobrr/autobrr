/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment, useRef, useState, useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Menu, Transition } from "@headlessui/react";
import { toast } from "react-hot-toast";
import {
  ArrowsRightLeftIcon,
  DocumentTextIcon,
  EllipsisHorizontalIcon,
  PencilSquareIcon,
  ForwardIcon,
  TrashIcon
} from "@heroicons/react/24/outline";

import { APIClient } from "@api/APIClient";
import { useToggle } from "@hooks/hooks";
import { baseUrl, classNames, IsEmptyDate, simplifyDate } from "@utils";
import Toast from "@components/notifications/Toast";
import { DeleteModal, ForceRunModal } from "@components/modals";
import { FeedUpdateForm } from "@forms/settings/FeedForms";
import { EmptySimple } from "@components/emptystates";
import { ImplementationBadges } from "./Indexer";
import { ArrowPathIcon } from "@heroicons/react/24/solid";
import { ExternalLink } from "@components/ExternalLink";
import { Section } from "./_components";
import { Checkbox } from "@components/Checkbox";

export const feedKeys = {
  all: ["feeds"] as const,
  lists: () => [...feedKeys.all, "list"] as const,
  // list: (indexers: string[], sortOrder: string) => [...feedKeys.lists(), { indexers, sortOrder }] as const,
  details: () => [...feedKeys.all, "detail"] as const,
  detail: (id: number) => [...feedKeys.details(), id] as const
};

interface SortConfig {
  key: keyof ListItemProps["feed"] | "enabled";
  direction: "ascending" | "descending";
}

function useSort(items: ListItemProps["feed"][], config?: SortConfig) {
  const [sortConfig, setSortConfig] = useState(config);

  const sortedItems = useMemo(() => {
    if (!sortConfig) {
      return items;
    }

    const sortableItems = [...items];

    sortableItems.sort((a, b) => {
      const aValue = sortConfig.key === "enabled" ? (a[sortConfig.key] ?? false) as number | boolean | string : a[sortConfig.key] as number | boolean | string;
      const bValue = sortConfig.key === "enabled" ? (b[sortConfig.key] ?? false) as number | boolean | string : b[sortConfig.key] as number | boolean | string;

      if (aValue < bValue) {
        return sortConfig.direction === "ascending" ? -1 : 1;
      }
      if (aValue > bValue) {
        return sortConfig.direction === "ascending" ? 1 : -1;
      }
      return 0;
    });

    return sortableItems;
  }, [items, sortConfig]);

  const requestSort = (key: keyof ListItemProps["feed"] | "enabled") => {
    let direction: "ascending" | "descending" = "ascending";
    if (
      sortConfig &&
      sortConfig.key === key &&
      sortConfig.direction === "ascending"
    ) {
      direction = "descending";
    }
    setSortConfig({ key, direction });
  };


  const getSortIndicator = (key: keyof ListItemProps["feed"]) => {
    if (!sortConfig || sortConfig.key !== key) {
      return "";
    }

    return sortConfig.direction === "ascending" ? "↑" : "↓";
  };

  return { items: sortedItems, requestSort, sortConfig, getSortIndicator };
}

function FeedSettings() {
  const { data } = useQuery({
    queryKey: feedKeys.lists(),
    queryFn: APIClient.feeds.find,
    refetchOnWindowFocus: false
  });

  const sortedFeeds = useSort(data || []);

  return (
    <Section
      title="Feeds"
      description="Manage RSS, Newznab, and Torznab feeds."
    >
      {data && data.length > 0 ? (
        <ul className="min-w-full relative">
          <li className="grid grid-cols-12 border-b border-gray-200 dark:border-gray-700">
            <div
              className="flex col-span-2 sm:col-span-1 pl-0 sm:pl-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
              onClick={() => sortedFeeds.requestSort("enabled")}>
              Enabled <span className="sort-indicator">{sortedFeeds.getSortIndicator("enabled")}</span>
            </div>
            <div
              className="col-span-5 pl-10 sm:pl-12 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
              onClick={() => sortedFeeds.requestSort("name")}>
              Name <span className="sort-indicator">{sortedFeeds.getSortIndicator("name")}</span>
            </div>
            <div
              className="hidden md:flex col-span-1 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
              onClick={() => sortedFeeds.requestSort("type")}>
              Type <span className="sort-indicator">{sortedFeeds.getSortIndicator("type")}</span>
            </div>
            <div
              className="hidden md:flex col-span-2 px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
              onClick={() => sortedFeeds.requestSort("last_run")}>
              Last run <span className="sort-indicator">{sortedFeeds.getSortIndicator("last_run")}</span>
            </div>
            <div
              className="hidden md:flex col-span-2 px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
              onClick={() => sortedFeeds.requestSort("next_run")}>
              Next run <span className="sort-indicator">{sortedFeeds.getSortIndicator("next_run")}</span>
            </div>
          </li>
          {sortedFeeds.items.map((feed) => (
            <ListItem key={feed.id} feed={feed} />
          ))}
        </ul>
      ) : (
        <EmptySimple title="No feeds" subtitle="Setup via indexers" />
      )}
    </Section>
  );
}

interface ListItemProps {
  feed: Feed;
}

function ListItem({ feed }: ListItemProps) {
  const [updateFormIsOpen, toggleUpdateForm] = useToggle(false);

  const [enabled, setEnabled] = useState(feed.enabled);
  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: (status: boolean) => APIClient.feeds.toggleEnable(feed.id, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: feedKeys.lists() });
      queryClient.invalidateQueries({ queryKey: feedKeys.detail(feed.id) });

      toast.custom((t) => <Toast type="success" body={`${feed.name} was ${!enabled ? "disabled" : "enabled"} successfully.`} t={t} />);
    }
  });

  const toggleActive = (status: boolean) => {
    setEnabled(status);
    updateMutation.mutate(status);
  };

  return (
    <li key={feed.id} className="text-gray-500 dark:text-gray-400">
      <FeedUpdateForm isOpen={updateFormIsOpen} toggle={toggleUpdateForm} feed={feed} />

      <div className="grid grid-cols-12 items-center">
        <div className="col-span-2 sm:col-span-1 pl-1 sm:pl-5 flex items-center">
          <Checkbox
            value={feed.enabled}
            setValue={toggleActive}
          />
        </div>
        <div className="col-span-8 sm:col-span-5 pl-10 sm:pl-12 py-3 flex flex-col text-sm font-medium text-gray-900 dark:text-white">
          <span>{feed.name}</span>
          <span className="text-gray-900 dark:text-gray-500 text-xs">
            {feed.indexer}
          </span>
        </div>
        <div className="hidden md:flex col-span-1 py-3 items-center">
          {ImplementationBadges[feed.type.toLowerCase()]}
        </div>
        <div className="hidden md:flex col-span-2 py-3 items-center sm:px-4 text-sm font-medium text-gray-900 dark:text-gray-500">
          <span title={simplifyDate(feed.last_run)}>
            {IsEmptyDate(feed.last_run)}
          </span>
        </div>
        <div className="hidden md:flex col-span-2 py-3 items-center sm:px-4 text-sm font-medium text-gray-900 dark:text-gray-500">
          <span title={simplifyDate(feed.next_run)}>
            {IsEmptyDate(feed.next_run)}
          </span>
        </div>
        <div className="col-span-1 flex justify-center items-center sm:px-6">
          <FeedItemDropdown
            feed={feed}
            onToggle={toggleActive}
            toggleUpdate={toggleUpdateForm}
          />
        </div>
      </div>
    </li>
  );
}

interface FeedItemDropdownProps {
  feed: Feed;
  onToggle: (newState: boolean) => void;
  toggleUpdate: () => void;
}

const FeedItemDropdown = ({
  feed,
  onToggle,
  toggleUpdate
}: FeedItemDropdownProps) => {
  const cancelModalButtonRef = useRef(null);
  const cancelCacheModalButtonRef = useRef(null);

  const queryClient = useQueryClient();

  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);
  const [deleteCacheModalIsOpen, toggleDeleteCacheModal] = useToggle(false);
  const [forceRunModalIsOpen, toggleForceRunModal] = useToggle(false);

  const deleteMutation = useMutation({
    mutationFn: (id: number) => APIClient.feeds.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: feedKeys.lists() });
      queryClient.invalidateQueries({ queryKey: feedKeys.detail(feed.id) });

      toast.custom((t) => <Toast type="success" body={`Feed ${feed?.name} was deleted`} t={t} />);
    }
  });

  const deleteCacheMutation = useMutation({
    mutationFn: (id: number) => APIClient.feeds.deleteCache(id),
    onSuccess: () => {
      toast.custom((t) => <Toast type="success" body={`Feed ${feed?.name} cache was cleared!`} t={t} />);
    }
  });

  const forceRunMutation = useMutation({
    mutationFn: (id: number) => APIClient.feeds.forceRun(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: feedKeys.lists() });
      toast.custom((t) => <Toast type="success" body={`Feed ${feed?.name} was force run successfully.`} t={t} />);
      toggleForceRunModal(); 
    },
    onError: (error: any) => {
      toast.custom((t) => <Toast type="error" body={`Failed to force run ${feed?.name}. Error: ${error.message}`} t={t} />, {
        duration: 10000
      });    
      toggleForceRunModal(); 
    }
  });


  return (
    <Menu as="div">
      <DeleteModal
        isOpen={deleteModalIsOpen}
        isLoading={deleteMutation.isLoading}
        toggle={toggleDeleteModal}
        buttonRef={cancelModalButtonRef}
        deleteAction={() => {
          deleteMutation.mutate(feed.id);
          toggleDeleteModal();
        }}
        title={`Remove feed: ${feed.name}`}
        text="Are you sure you want to remove this feed? This action cannot be undone."
      />
      <DeleteModal
        isOpen={deleteCacheModalIsOpen}
        isLoading={deleteMutation.isLoading}
        toggle={toggleDeleteCacheModal}
        buttonRef={cancelCacheModalButtonRef}
        deleteAction={() => {
          deleteCacheMutation.mutate(feed.id);
        }}
        title={`Remove feed cache: ${feed.name}`}
        text="Are you sure you want to remove the feed cache? This action cannot be undone."
      />
      <ForceRunModal
        isOpen={forceRunModalIsOpen}
        isLoading={forceRunMutation.isLoading}
        toggle={toggleForceRunModal}
        buttonRef={cancelModalButtonRef}
        forceRunAction={() => {
          forceRunMutation.mutate(feed.id);
          toggleForceRunModal();
        }}
        title={`Force run feed: ${feed.name}`}
        text={`Are you sure you want to force run the ${feed.name} feed? Respecting RSS interval rules is crucial to avoid potential IP bans.`}
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
          className="absolute right-0 w-56 mt-2 origin-top-right bg-white dark:bg-gray-825 divide-y divide-gray-200 dark:divide-gray-750 rounded-md shadow-lg border border-gray-250 dark:border-gray-750 focus:outline-none z-10"
        >
          <div className="px-1 py-1">
            <Menu.Item>
              {({ active }) => (
                <button
                  className={classNames(
                    active ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                    "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                  )}
                  onClick={() => toggleUpdate()}
                >
                  <PencilSquareIcon
                    className={classNames(
                      active ? "text-white" : "text-blue-500",
                      "w-5 h-5 mr-2"
                    )}
                    aria-hidden="true"
                  />
                  Edit
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
                  onClick={() => onToggle(!feed.enabled)}
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
          </div>
          <div className="px-1 py-1">
            <Menu.Item>
              {({ active }) => (
                <button
                  onClick={() => toggleForceRunModal()}
                  className={classNames(
                    active ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                    "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                  )}
                >
                  <ForwardIcon
                    className={classNames(
                      active ? "text-white" : "text-blue-500",
                      "w-5 h-5 mr-2"
                    )}
                    aria-hidden="true"
                  />
            Force run
                </button>
              )}
            </Menu.Item>
            <Menu.Item>
              <ExternalLink
                href={`${baseUrl()}api/feeds/${feed.id}/latest`}
                className="font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm text-gray-900 dark:text-gray-300 hover:bg-blue-600 hover:text-white"
              >
                <DocumentTextIcon
                  className="w-5 h-5 mr-2 text-blue-500 group-hover:text-white"
                  aria-hidden="true"
                />
                View latest run
              </ExternalLink>
            </Menu.Item>
            <Menu.Item>
              {({ active }) => (
                <button
                  className={classNames(
                    active ? "bg-red-600 text-white" : "text-gray-900 dark:text-gray-300",
                    "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                  )}
                  onClick={() => toggleDeleteCacheModal()}
                  title="Manually clear all feed cache"
                >
                  <ArrowPathIcon
                    className={classNames(
                      active ? "text-white" : "text-red-500",
                      "w-5 h-5 mr-2"
                    )}
                    aria-hidden="true"
                  />
                  Clear feed cache
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

export default FeedSettings;
