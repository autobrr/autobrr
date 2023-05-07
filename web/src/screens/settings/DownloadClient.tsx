/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, useMemo } from "react";
import { Switch } from "@headlessui/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import toast from "react-hot-toast";

import { useToggle } from "@hooks/hooks";
import { classNames } from "@utils";
import { DownloadClientAddForm, DownloadClientUpdateForm } from "@forms";
import { EmptySimple } from "@components/emptystates";
import { APIClient } from "@api/APIClient";
import { DownloadClientTypeNameMap } from "@domain/constants";
import Toast from "@components/notifications/Toast";

export const clientKeys = {
  all: ["download_clients"] as const,
  lists: () => [...clientKeys.all, "list"] as const,
  // list: (indexers: string[], sortOrder: string) => [...clientKeys.lists(), { indexers, sortOrder }] as const,
  details: () => [...clientKeys.all, "detail"] as const,
  detail: (id: number) => [...clientKeys.details(), id] as const
};

interface DLSettingsItemProps {
    client: DownloadClient;
    idx: number;
}

interface ListItemProps {
  clients: DownloadClient;
}

interface SortConfig {
  key: keyof ListItemProps["clients"] | "enabled";
  direction: "ascending" | "descending";
}

function useSort(items: ListItemProps["clients"][], config?: SortConfig) {
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

  const requestSort = (key: keyof ListItemProps["clients"]) => {
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

  const getSortIndicator = (key: keyof ListItemProps["clients"]) => {
    if (!sortConfig || sortConfig.key !== key) {
      return "";
    }
    
    return sortConfig.direction === "ascending" ? "↑" : "↓";
  };

  return { items: sortedItems, requestSort, sortConfig, getSortIndicator };
}

function DownloadClientSettingsListItem({ client }: DLSettingsItemProps) {
  const [updateClientIsOpen, toggleUpdateClient] = useToggle(false);

  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (client: DownloadClient) => APIClient.download_clients.update(client).then(() => client),
    onSuccess: (client: DownloadClient) => {
      toast.custom(t => <Toast type="success" body={`${client.name} was ${client.enabled ? "enabled" : "disabled"} successfully.`} t={t} />);
      queryClient.invalidateQueries({ queryKey: clientKeys.lists() });
    }
  });

  const onToggleMutation = (newState: boolean) => {
    mutation.mutate({
      ...client,
      enabled: newState
    });
  };

  return (
    <li key={client.name}>
      <div className="grid grid-cols-12 items-center py-2">
        <DownloadClientUpdateForm
          client={client}
          isOpen={updateClientIsOpen}
          toggle={toggleUpdateClient}
        />
        <div className="col-span-2 sm:col-span-1 px-6 flex items-center sm:px-6">
          <Switch
            checked={client.enabled}
            onChange={onToggleMutation}
            className={classNames(
              client.enabled ? "bg-blue-500" : "bg-gray-200 dark:bg-gray-600",
              "relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            )}
          >
            <span className="sr-only">Use setting</span>
            <span
              aria-hidden="true"
              className={classNames(
                client.enabled ? "translate-x-5" : "translate-x-0",
                "inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200"
              )}
            />
          </Switch>
        </div>
        <div className="col-span-8 sm:col-span-4 lg:col-span-4 pl-12 pr-6 py-3 block flex-col text-sm font-medium text-gray-900 dark:text-white truncate" title={client.name}>{client.name}</div>
        <div className="hidden sm:block col-span-4 pr-6 py-3 text-left items-center whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 truncate" title={client.host}>{client.host}</div>
        <div className="hidden sm:block col-span-2 py-3 text-left items-center text-sm text-gray-500 dark:text-gray-400">{DownloadClientTypeNameMap[client.type]}</div>
        <div className="col-span-1 pl-0.5 whitespace-nowrap text-center text-sm font-medium">
          <span className="text-blue-600 dark:text-gray-300 hover:text-blue-900 cursor-pointer" onClick={toggleUpdateClient}>
            Edit
          </span>
        </div>
      </div>
    </li>
  );
}

function DownloadClientSettings() {
  const [addClientIsOpen, toggleAddClient] = useToggle(false);

  const { error, data } = useQuery({
    queryKey: clientKeys.lists(),
    queryFn: APIClient.download_clients.getAll,
    refetchOnWindowFocus: false
  });

  const sortedClients = useSort(data || []);

  if (error) {
    return <p>Failed to fetch download clients</p>;
  }

  return (
    <div className="lg:col-span-9">
      <DownloadClientAddForm isOpen={addClientIsOpen} toggle={toggleAddClient} />

      <div className="py-6 px-2 lg:pb-8">
        <div className="px-4 -ml-4 -mt-4 flex justify-between items-center flex-wrap sm:flex-nowrap">
          <div className="ml-4 mt-4">
            <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">Clients</h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Manage download clients.
            </p>
          </div>
          <div className="ml-4 mt-4 flex-shrink-0">
            <button
              type="button"
              className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
              onClick={toggleAddClient}
            >
              Add new
            </button>
          </div>
        </div>

        <div className="flex flex-col mt-6 px-4">
          {sortedClients.items.length > 0
            ? <section className="light:bg-white dark:bg-gray-800 light:shadow sm:rounded-sm">
              <ol className="min-w-full relative">
                <li className="grid grid-cols-12 border-b border-gray-200 dark:border-gray-700">
                  <div className="flex col-span-2 sm:col-span-1 px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
                    onClick={() => sortedClients.requestSort("enabled")}>
                    Enabled <span className="sort-indicator">{sortedClients.getSortIndicator("enabled")}</span>
                  </div>
                  <div 
                    className="col-span-6 sm:col-span-4 lg:col-span-4 pl-12 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
                    onClick={() => sortedClients.requestSort("name")}
                  >
                    Name <span className="sort-indicator">{sortedClients.getSortIndicator("name")}</span>
                  </div>
                  <div
                    className="hidden sm:flex col-span-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
                    onClick={() => sortedClients.requestSort("host")}
                  >
                    Host <span className="sort-indicator">{sortedClients.getSortIndicator("host")}</span>
                  </div>
                  <div className="hidden sm:flex col-span-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
                    onClick={() => sortedClients.requestSort("type")}
                  >
                    Type <span className="sort-indicator">{sortedClients.getSortIndicator("type")}</span>
                  </div>
                </li>
                {sortedClients.items.map((client, idx) => (
                  <DownloadClientSettingsListItem client={client} idx={idx} key={idx} />
                ))}
              </ol>
            </section>
            : <EmptySimple title="No download clients" subtitle="" buttonText="Add new client" buttonAction={toggleAddClient} />
          }
        </div>
      </div>
    </div>
  );
}

export default DownloadClientSettings;