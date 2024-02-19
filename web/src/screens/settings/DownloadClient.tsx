/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMemo, useState } from "react";
import { useMutation, useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { PlusIcon } from "@heroicons/react/24/solid";
import toast from "react-hot-toast";

import { useToggle } from "@hooks/hooks";
import { DownloadClientAddForm, DownloadClientUpdateForm } from "@forms";
import { EmptySimple } from "@components/emptystates";
import { APIClient } from "@api/APIClient";
import { DownloadClientKeys } from "@api/query_keys";
import { DownloadClientsQueryOptions } from "@api/queries";
import { ActionTypeNameMap } from "@domain/constants";
import Toast from "@components/notifications/Toast";
import { Checkbox } from "@components/Checkbox";

import { Section } from "./_components";

interface DLSettingsItemProps {
  client: DownloadClient;
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

function ListItem({ client }: DLSettingsItemProps) {
  const [updateClientIsOpen, toggleUpdateClient] = useToggle(false);

  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (client: DownloadClient) => APIClient.download_clients.update(client).then(() => client),
    onSuccess: (client: DownloadClient) => {
      toast.custom(t => <Toast type="success" body={`${client.name} was ${client.enabled ? "enabled" : "disabled"} successfully.`} t={t} />);
      queryClient.invalidateQueries({ queryKey: DownloadClientKeys.lists() });
    }
  });

  const onToggleMutation = (newState: boolean) => {
    mutation.mutate({
      ...client,
      enabled: newState
    });
  };

  return (
    <li>
      <div className="grid grid-cols-12 items-center py-2">
        <DownloadClientUpdateForm
          client={client}
          isOpen={updateClientIsOpen}
          toggle={toggleUpdateClient}
        />
        <div className="col-span-2 sm:col-span-1 pl-1 sm:pl-6 flex items-center">
          <Checkbox
            value={client.enabled}
            setValue={onToggleMutation}
          />
        </div>
        <div className="col-span-8 sm:col-span-4 lg:col-span-4 pl-10 sm:pl-12 pr-6 py-3 block flex-col text-sm font-medium text-gray-900 dark:text-white truncate" title={client.name}>{client.name}</div>
        <div className="hidden sm:block col-span-4 pr-6 py-3 text-left items-center whitespace-nowrap text-sm text-gray-600 dark:text-gray-400 truncate" title={client.host}>{client.host}</div>
        <div className="hidden sm:block col-span-2 py-3 text-left items-center text-sm text-gray-600 dark:text-gray-400">
          {ActionTypeNameMap[client.type]}
        </div>
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

  const downloadClientsQuery = useSuspenseQuery(DownloadClientsQueryOptions())

  const sortedClients = useSort(downloadClientsQuery.data || []);

  return (
    <Section
      title="Download Clients"
      description="Manage download clients."
      rightSide={
        <button
          type="button"
          className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
          onClick={toggleAddClient}
        >
          <PlusIcon className="h-5 w-5 mr-1" />
          Add new
        </button>
      }
    >
      <DownloadClientAddForm isOpen={addClientIsOpen} toggle={toggleAddClient} />

      <div className="flex flex-col">
        {sortedClients.items.length > 0 ? (
          <ul className="min-w-full relative">
            <li className="grid grid-cols-12 border-b border-gray-200 dark:border-gray-700">
              <div className="flex col-span-2 sm:col-span-1 pl-0 sm:pl-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
                onClick={() => sortedClients.requestSort("enabled")}>
                Enabled <span className="sort-indicator">{sortedClients.getSortIndicator("enabled")}</span>
              </div>
              <div
                className="col-span-6 sm:col-span-4 lg:col-span-4 pl-10 sm:pl-12 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
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
            {sortedClients.items.map((client) => (
              <ListItem key={client.id} client={client} />
            ))}
          </ul>
        ) : (
          <EmptySimple title="No download clients" subtitle="" buttonText="Add new client" buttonAction={toggleAddClient} />
        )}
      </div>
    </Section>
  );
}

export default DownloadClientSettings;
