/*
 * Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useToggle } from "@hooks/hooks.ts";
import { useMutation, useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { PlusIcon } from "@heroicons/react/24/solid";
import { useTranslation } from "react-i18next";

import { APIClient } from "@api/APIClient";
import { ProxyKeys } from "@api/query_keys";
import { ProxiesQueryOptions } from "@api/queries";
import { Section } from "./_components";
import { EmptySimple } from "@components/emptystates";
import { Checkbox } from "@components/Checkbox";
import { ProxyAddForm, ProxyUpdateForm } from "@forms/settings/ProxyForms";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";

interface ListItemProps {
  proxy: Proxy;
}

function ListItem({ proxy }: ListItemProps) {
  const { t } = useTranslation("settings");
  const [isOpen, toggleUpdate] = useToggle(false);

  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: (req: Proxy) => APIClient.proxy.update(req),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ProxyKeys.lists() });

      toast.custom((toastInstance) => (
        <Toast
          type="success"
          body={t("listScreens.proxies.toggleSuccess", {
            name: proxy.name,
            state: variables.enabled
              ? t("listScreens.proxies.enabledState")
              : t("listScreens.proxies.disabledState")
          })}
          t={toastInstance}
        />
      ));
    },
    onError: () => {
      toast.custom((toastInstance) => <Toast type="error" body={t("listScreens.proxies.toggleError")} t={toastInstance} />);
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
        <div className="col-span-7 pl-12 sm:pr-6 py-3 block flex-col text-sm font-medium text-gray-900 dark:text-white truncate">
          {proxy.name}
        </div>
        <div className="hidden md:block col-span-2 pr-6 py-3 text-left items-center whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 truncate">
          {proxy.type}
        </div>
        <div className="col-span-2 flex first-letter:px-6 py-3 whitespace-nowrap justify-end text-sm font-medium">
          <span
            className="col-span-2 px-6 text-blue-600 dark:text-gray-300 hover:text-blue-900 dark:hover:text-blue-500 cursor-pointer"
            onClick={toggleUpdate}
          >
            {t("listScreens.common.edit")}
          </span>
        </div>
      </div>
    </li>
  );
}

function ProxySettings() {
  const { t } = useTranslation("settings");
  const [addProxyIsOpen, toggleAddProxy] = useToggle(false);

  const proxiesQuery = useSuspenseQuery(ProxiesQueryOptions())
  const proxies = proxiesQuery.data

  return (
    <Section
      title={t("listScreens.proxies.title")}
      description={t("listScreens.proxies.description")}
      rightSide={
        <button
          type="button"
          onClick={toggleAddProxy}
          className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
        >
          <PlusIcon className="h-5 w-5 mr-1"/>
          {t("listScreens.common.addNew")}
        </button>
      }
    >
      <ProxyAddForm isOpen={addProxyIsOpen} toggle={toggleAddProxy} />

      <div className="flex flex-col">
        {proxies.length ? (
          <ul className="min-w-full relative">
            <li className="grid grid-cols-12 border-b border-gray-200 dark:border-gray-700">
              <div
                className="flex col-span-2 sm:col-span-1 pl-0 sm:pl-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-250 transition-colors uppercase tracking-wider cursor-pointer"
                // onClick={() => sortedIndexers.requestSort("enabled")}
              >
                {t("listScreens.common.enabled")}
                {/*<span className="sort-indicator">{sortedIndexers.getSortIndicator("enabled")}</span>*/}
              </div>
              <div
                className="col-span-7 pl-12 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-250 transition-colors uppercase tracking-wider cursor-pointer"
                // onClick={() => sortedIndexers.requestSort("name")}
              >
                {t("listScreens.common.name")}
                {/*<span className="sort-indicator">{sortedIndexers.getSortIndicator("name")}</span>*/}
              </div>
              <div
                className="hidden md:flex col-span-1 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-250 transition-colors uppercase tracking-wider cursor-pointer"
                // onClick={() => sortedIndexers.requestSort("implementation")}
              >
                {t("listScreens.common.type")}
                {/*<span className="sort-indicator">{sortedIndexers.getSortIndicator("implementation")}</span>*/}
              </div>
            </li>
            {proxies.map((proxy) => (
              <ListItem proxy={proxy} key={proxy.id}/>
            ))}
          </ul>
        ) : (
          <EmptySimple
            title={t("listScreens.proxies.noItems")}
            subtitle=""
            buttonText={t("listScreens.proxies.addNewItem")}
            buttonAction={toggleAddProxy}
          />
        )}
      </div>
    </Section>
  );
}

export default ProxySettings;
