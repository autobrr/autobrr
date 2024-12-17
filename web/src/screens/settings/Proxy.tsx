/*
 * Copyright (c) 2021-2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useToggle } from "@hooks/hooks.ts";
import { useMutation, useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { PlusIcon } from "@heroicons/react/24/solid";
import { toast } from "react-hot-toast";

import { APIClient } from "@api/APIClient";
import { ProxyKeys } from "@api/query_keys";
import { ProxiesQueryOptions } from "@api/queries";
import { Section } from "./_components";
import { EmptySimple } from "@components/emptystates";
import { Checkbox } from "@components/Checkbox";
import { ProxyAddForm, ProxyUpdateForm } from "@forms/settings/ProxyForms";
import Toast from "@components/notifications/Toast";

interface ListItemProps {
  proxy: Proxy;
}

function ListItem({ proxy }: ListItemProps) {
  const [isOpen, toggleUpdate] = useToggle(false);

  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: (req: Proxy) => APIClient.proxy.update(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ProxyKeys.lists() });

      toast.custom(t => <Toast type="success" body={`Proxy ${proxy.name} was ${proxy.enabled ? "enabled" : "disabled"} successfully.`} t={t} />);
    },
    onError: () => {
      toast.custom((t) => <Toast type="error" body="Proxy state could not be updated" t={t} />);
    }
  });

  const onToggleMutation = (newState: boolean) => {
    updateMutation.mutate({
      ...proxy,
      enabled: newState
    });
  };

  return (
    <li>
      <ProxyUpdateForm isOpen={isOpen} toggle={toggleUpdate} data={proxy} />

      <div className="grid grid-cols-12 items-center py-1.5">
        <div className="col-span-2 sm:col-span-1 flex pl-1 sm:pl-5 items-center">
          <Checkbox value={proxy.enabled ?? false} setValue={onToggleMutation} />
        </div>
        <div className="col-span-7 sm:col-span-8 pl-12 sm:pr-6 py-3 block flex-col text-sm font-medium text-gray-900 dark:text-white truncate">
          {proxy.name}
        </div>
        <div className="hidden md:block col-span-2 pr-6 py-3 text-left items-center whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 truncate">
          {proxy.type}
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

function ProxySettings() {
  const [addProxyIsOpen, toggleAddProxy] = useToggle(false);

  const proxiesQuery = useSuspenseQuery(ProxiesQueryOptions())
  const proxies = proxiesQuery.data

  return (
    <Section
      title="Proxies"
      description={
        <>
          Proxies that can be used with Indexers, feeds and IRC.<br/>
        </>
      }
      rightSide={
        <button
          type="button"
          onClick={toggleAddProxy}
          className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
        >
          <PlusIcon className="h-5 w-5 mr-1"/>
          Add new
        </button>
      }
    >
      <ProxyAddForm isOpen={addProxyIsOpen} toggle={toggleAddProxy} />

      <div className="flex flex-col">
        {proxies.length ? (
          <ul className="min-w-full relative">
            <li className="grid grid-cols-12 border-b border-gray-200 dark:border-gray-700">
              <div
                className="flex col-span-2 sm:col-span-1 pl-0 sm:pl-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 hover:text-gray-800 hover:dark:text-gray-250 transition-colors uppercase tracking-wider cursor-pointer"
                // onClick={() => sortedIndexers.requestSort("enabled")}
              >
                Enabled
                {/*<span className="sort-indicator">{sortedIndexers.getSortIndicator("enabled")}</span>*/}
              </div>
              <div
                className="col-span-7 sm:col-span-8 pl-12 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 hover:text-gray-800 hover:dark:text-gray-250 transition-colors uppercase tracking-wider cursor-pointer"
                // onClick={() => sortedIndexers.requestSort("name")}
              >
                Name
                {/*<span className="sort-indicator">{sortedIndexers.getSortIndicator("name")}</span>*/}
              </div>
              <div
                className="hidden md:flex col-span-1 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 hover:text-gray-800 hover:dark:text-gray-250 transition-colors uppercase tracking-wider cursor-pointer"
                // onClick={() => sortedIndexers.requestSort("implementation")}
              >
                Type
                {/*<span className="sort-indicator">{sortedIndexers.getSortIndicator("implementation")}</span>*/}
              </div>
            </li>
            {proxies.map((proxy) => (
              <ListItem proxy={proxy} key={proxy.id}/>
            ))}
          </ul>
        ) : (
          <EmptySimple
            title="No proxies"
            subtitle=""
            buttonText="Add new proxy"
            buttonAction={toggleAddProxy}
          />
        )}
      </div>
    </Section>
  );
}

export default ProxySettings;