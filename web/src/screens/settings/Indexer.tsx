/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, useMemo } from "react";
import toast from "react-hot-toast";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Switch } from "@headlessui/react";

import Toast from "@components/notifications/Toast";
import { IndexerAddForm, IndexerUpdateForm } from "@forms";
import { useToggle } from "@hooks/hooks";
import { classNames } from "@utils";
import { EmptySimple } from "@components/emptystates";
import { APIClient } from "@api/APIClient";
import { componentMapType } from "@forms/settings/DownloadClientForms";

export const indexerKeys = {
  all: ["indexers"] as const,
  lists: () => [...indexerKeys.all, "list"] as const,
  // list: (indexers: string[], sortOrder: string) => [...indexerKeys.lists(), { indexers, sortOrder }] as const,
  details: () => [...indexerKeys.all, "detail"] as const,
  detail: (id: number) => [...indexerKeys.details(), id] as const
};

interface SortConfig {
  key: keyof ListItemProps["indexer"] | "enabled";
  direction: "ascending" | "descending";
}

function useSort(items: ListItemProps["indexer"][], config?: SortConfig) {
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

  const requestSort = (key: keyof ListItemProps["indexer"]) => {
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

  const getSortIndicator = (key: keyof ListItemProps["indexer"]) => {
    if (!sortConfig || sortConfig.key !== key) {
      return "";
    }

    return sortConfig.direction === "ascending" ? "↑" : "↓";
  };

  return { items: sortedItems, requestSort, sortConfig, getSortIndicator };
}

const ImplementationBadgeIRC = () => (
  <span className="mr-2 inline-flex items-center px-2.5 py-0.5 rounded-md text-sm font-medium bg-green-200 dark:bg-green-400 text-green-800 dark:text-green-800">
    IRC
  </span>
);

const ImplementationBadgeTorznab = () => (
  <span className="inline-flex items-center px-2.5 py-0.5 rounded-md text-sm font-medium bg-orange-200 dark:bg-orange-400 text-orange-800 dark:text-orange-800">
    Torznab
  </span>
);

const ImplementationBadgeNewznab = () => (
  <span className="inline-flex items-center px-2.5 py-0.5 rounded-md text-sm font-medium bg-blue-200 dark:bg-blue-400 text-blue-800 dark:text-blue-800">
    Newznab
  </span>
);

const ImplementationBadgeRSS = () => (
  <span className="inline-flex items-center px-2.5 py-0.5 rounded-md text-sm font-medium bg-amber-200 dark:bg-amber-400 text-amber-800 dark:text-amber-800">
    RSS
  </span>
);

export const ImplementationBadges: componentMapType = {
  irc: <ImplementationBadgeIRC />,
  torznab: <ImplementationBadgeTorznab />,
  newznab: <ImplementationBadgeNewznab />,
  rss: <ImplementationBadgeRSS />
};

interface ListItemProps {
  indexer: IndexerDefinition;
}

const ListItem = ({ indexer }: ListItemProps) => {
  const [updateIsOpen, toggleUpdate] = useToggle(false);

  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: (enabled: boolean) => APIClient.indexers.toggleEnable(indexer.id, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: indexerKeys.lists() });
      toast.custom((t) => <Toast type="success" body={`${indexer.name} was updated successfully`} t={t} />);
    }
  });

  const onToggleMutation = (newState: boolean) => {
    // backend is rejecting when ending the whole object
    updateMutation.mutate(newState);
  };

  return (
    <li>
      <div className="grid grid-cols-12 items-center py-1.5">
        <IndexerUpdateForm
          isOpen={updateIsOpen}
          toggle={toggleUpdate}
          indexer={indexer}
        />
        <div className="col-span-2 sm:col-span-1 flex px-6 items-center sm:px-6">
          <Switch
            onClick={(e) => e.stopPropagation()}
            checked={indexer.enabled ?? false}
            onChange={onToggleMutation}
            className={classNames(
              indexer.enabled ? "bg-blue-500" : "bg-gray-200 dark:bg-gray-600",
              "relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            )}
          >
            <span className="sr-only">Enable</span>
            <span
              aria-hidden="true"
              className={classNames(
                indexer.enabled ? "translate-x-5" : "translate-x-0",
                "inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200"
              )}
            />
          </Switch>
        </div>
        <div className="col-span-7 sm:col-span-8 pl-12 sm:pr-6 py-3 block flex-col text-sm font-medium text-gray-900 dark:text-white truncate">
          {indexer.name}
        </div>
        <div className="hidden md:block col-span-2 pr-6 py-3 text-left items-center whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 truncate">
          {ImplementationBadges[indexer.implementation]}
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
};

function IndexerSettings() {
  const [addIndexerIsOpen, toggleAddIndexer] = useToggle(false);

  const { error, data } = useQuery({
    queryKey: indexerKeys.lists(),
    queryFn: APIClient.indexers.getAll,
    refetchOnWindowFocus: false
  });

  const sortedIndexers = useSort(data || []);

  if (error) {
    return (<p>An error has occurred</p>);
  }

  return (
    <div className="lg:col-span-9">
      <IndexerAddForm isOpen={addIndexerIsOpen} toggle={toggleAddIndexer} />

      <div className="py-6 px-4 sm:p-6 lg:pb-8">
        <div className="-ml-4 -mt-4 flex justify-between items-center flex-wrap sm:flex-nowrap">
          <div className="ml-4 mt-4">
            <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">
              Indexers
            </h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Indexer settings for IRC, RSS, Newznab, and Torznab based indexers.<br />
              Generic feeds can be added here by selecting the Generic indexer.
            </p>
          </div>
          <div className="ml-4 mt-4 flex-shrink-0">
            <button
              type="button"
              onClick={toggleAddIndexer}
              className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
            >
              Add new
            </button>
          </div>
        </div>

        <div className="flex flex-col mt-6">
          {data && data.length > 0 ? (
            <section className="light:bg-white dark:bg-gray-800 light:shadow sm:rounded-md">
              <ol className="min-w-full relative">
                <li className="grid grid-cols-12 border-b border-gray-200 dark:border-gray-700">
                  <div
                    className="flex col-span-2 sm:col-span-1 px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
                    onClick={() => sortedIndexers.requestSort("enabled")}
                  >
  Enabled <span className="sort-indicator">{sortedIndexers.getSortIndicator("enabled")}</span>
                  </div>
                  <div
                    className="col-span-7 sm:col-span-8 pl-12 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
                    onClick={() => sortedIndexers.requestSort("name")}
                  >
  Name <span className="sort-indicator">{sortedIndexers.getSortIndicator("name")}</span>
                  </div>
                  <div
                    className="hidden md:flex col-span-1 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
                    onClick={() => sortedIndexers.requestSort("implementation")}
                  >
  Implementation <span className="sort-indicator">{sortedIndexers.getSortIndicator("implementation")}</span>
                  </div>
                </li>
                {sortedIndexers.items.map((indexer) => (
                  <ListItem indexer={indexer} key={indexer.id} />
                ))}
              </ol>
            </section>
          ) : (
            <EmptySimple
              title="No indexers"
              subtitle=""
              buttonText="Add new indexer"
              buttonAction={toggleAddIndexer}
            />
          )}
        </div>
      </div>
    </div>
  );
}

export default IndexerSettings;
