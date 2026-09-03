/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMemo, useRef, useState } from "react";
import { useMutation, useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { PlusIcon } from "@heroicons/react/24/solid";
import { ArchiveBoxXMarkIcon, TrashIcon } from "@heroicons/react/24/outline";
import { Trans, useTranslation } from "react-i18next";

import { useToggle } from "@hooks/hooks";
import { APIClient } from "@api/APIClient";
import { FilterKeys, IndexerKeys } from "@api/query_keys";
import { IndexerDeprecationsQueryOptions, IndexersOptionsQueryOptions, IndexersQueryOptions } from "@api/queries";
import { Checkbox } from "@components/Checkbox";
import { ExternalLink } from "@components/ExternalLink";
import { DeleteModal } from "@components/modals";
import toast from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { EmptySimple } from "@components/emptystates";
import { IndexerAddForm, IndexerUpdateForm } from "@forms";
import { componentMapType } from "@forms/settings/DownloaderForms";

import { Section } from "./_components";

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
  <span className="inline-flex items-center px-2.5 py-0.5 rounded-md text-sm font-medium bg-orange-200 dark:bg-orange-400 text-orange-800 dark:text-amber-900">
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

const ImplementationBadges: componentMapType = {
  irc: <ImplementationBadgeIRC />,
  torznab: <ImplementationBadgeTorznab />,
  newznab: <ImplementationBadgeNewznab />,
  rss: <ImplementationBadgeRSS />
};

export const ImplementationBadge = ({ implementation }: { implementation: string }) => ImplementationBadges[implementation.toLowerCase()] ?? null;

interface ListItemProps {
  indexer: IndexerDefinition;
}

const ListItem = ({ indexer }: ListItemProps) => {
  const { t } = useTranslation("settings");
  const [updateIsOpen, toggleUpdate] = useToggle(false);

  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: (enabled: boolean) => APIClient.indexers.toggleEnable(indexer.id, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: IndexerKeys.lists() });
      toast.custom((toastItem) => <Toast type="success" body={t("listScreens.indexers.updated", { name: indexer.name })} t={toastItem} />);
    }
  });

  const onToggleMutation = (newState: boolean) => {
    updateMutation.mutate(newState);
  };

  if (!indexer) {
    return null;
  }

  return (
    <li>
      <div className="grid grid-cols-12 items-center py-1.5">
        <IndexerUpdateForm
          isOpen={updateIsOpen}
          toggle={toggleUpdate}
          data={indexer}
        />
        <div className="col-span-2 sm:col-span-1 flex pl-1 sm:pl-5 items-center">
          <Checkbox name="enabled" value={indexer.enabled ?? false} setValue={onToggleMutation} />
        </div>
        <div className="col-span-7 pl-6 sm:pl-12 sm:pr-6 py-3 block flex-col text-sm font-medium text-gray-900 dark:text-white truncate">
          {indexer.name}
        </div>
        <div className="hidden md:block col-span-2 pr-6 py-3 text-left items-center whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 truncate">
          {ImplementationBadges[indexer.implementation]}
        </div>
        <div className="col-span-3 sm:col-span-2 flex first-letter:px-6 py-3 whitespace-nowrap justify-end text-sm font-medium">
          <span
            className="col-span-3 sm:col-span-2 px-6 text-blue-600 dark:text-gray-300 hover:text-blue-900 dark:hover:text-blue-500 cursor-pointer"
            onClick={toggleUpdate}
          >
            {t("listScreens.common.edit")}
          </span>
        </div>
      </div>
    </li>
  );
};

function IndexerSettings() {
  const { t } = useTranslation("settings");
  const [addIndexerIsOpen, toggleAddIndexer] = useToggle(false);

  const indexersQuery = useSuspenseQuery(IndexersQueryOptions())
  const indexers = indexersQuery.data
  const sortedIndexers = useSort(indexers || []);

  // if (error) {
  //   return (<p>An error has occurred</p>);
  // }

  return (
    <Section
      title={t("listScreens.indexers.title")}
      description={
        <>
          {t("listScreens.indexers.description")}
          <br />
          <Trans
            i18nKey="listScreens.indexers.descriptionGeneric"
            ns="settings"
            components={{ strong: <span className="font-bold" /> }}
          />
        </>
      }
      rightSide={
        <button
          type="button"
          onClick={toggleAddIndexer}
          className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer"
        >
          <PlusIcon className="h-5 w-5 mr-1" />
          {t("listScreens.common.addNew")}
        </button>
      }
    >
      <IndexerAddForm isOpen={addIndexerIsOpen} toggle={toggleAddIndexer} />

      <div className="flex flex-col">
        {sortedIndexers.items.length ? (
          <ul className="min-w-full relative">
            <li className="grid grid-cols-12 border-b border-gray-200 dark:border-gray-700">
              <div
                className="flex col-span-2 sm:col-span-1 pl-0 sm:pl-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-250 transition-colors uppercase tracking-wider cursor-pointer"
                onClick={() => sortedIndexers.requestSort("enabled")}
              >
                {t("listScreens.common.enabled")} <span className="sort-indicator">{sortedIndexers.getSortIndicator("enabled")}</span>
              </div>
              <div
                className="col-span-7 pl-6 sm:pl-12 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-250 transition-colors uppercase tracking-wider cursor-pointer"
                onClick={() => sortedIndexers.requestSort("name")}
              >
                {t("listScreens.common.name")} <span className="sort-indicator">{sortedIndexers.getSortIndicator("name")}</span>
              </div>
              <div
                className="hidden md:flex col-span-1 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-250 transition-colors uppercase tracking-wider cursor-pointer"
                onClick={() => sortedIndexers.requestSort("implementation")}
              >
                {t("listScreens.indexers.implementation")} <span className="sort-indicator">{sortedIndexers.getSortIndicator("implementation")}</span>
              </div>
            </li>
            {sortedIndexers.items.map((indexer) => (
              <ListItem indexer={indexer} key={indexer.id} />
            ))}
          </ul>
        ) : (
          <EmptySimple
            title={t("listScreens.indexers.noItems")}
            subtitle=""
            buttonText={t("listScreens.indexers.addNewItem")}
            buttonAction={toggleAddIndexer}
          />
        )}
      </div>
    </Section>
  );
}

function DeprecatedIndexers() {
  const { t } = useTranslation("settings");
  const queryClient = useQueryClient();

  const cancelModalButtonRef = useRef(null);
  const [pendingAction, setPendingAction] = useState<
    | { type: "prune"; identifiers: string[]; name?: string }
    | { type: "purge"; id: number; name: string }
    | null
  >(null);

  const optionsQuery = useSuspenseQuery(IndexersOptionsQueryOptions());
  const deprecationsQuery = useSuspenseQuery(IndexerDeprecationsQueryOptions());

  const archived = useMemo(
    () => (optionsQuery.data || []).filter((indexer) => indexer.archived),
    [optionsQuery.data]
  );

  const metaByIdentifier = useMemo(
    () => new Map((deprecationsQuery.data || []).map((d) => [d.identifier, d])),
    [deprecationsQuery.data]
  );

  const totalFilterUsage = useMemo(
    () => archived.reduce((sum, indexer) => sum + (metaByIdentifier.get(indexer.identifier)?.filter_count ?? 0), 0),
    [archived, metaByIdentifier]
  );

  const pruneMutation = useMutation({
    mutationFn: (identifiers: string[]) => APIClient.filters.pruneDeprecatedIndexers(identifiers),
    onSuccess: (res) => {
      toast.custom((tt) => (
        <Toast
          type="success"
          body={t("listScreens.indexers.deprecated.pruneSuccess", { count: res?.removed ?? 0 })}
          t={tt}
        />
      ));
      queryClient.invalidateQueries({ queryKey: FilterKeys.all });
      queryClient.invalidateQueries({ queryKey: IndexerKeys.deprecations() });
    },
    onError: () => {
      toast.custom((tt) => (
        <Toast type="error" body={t("listScreens.indexers.deprecated.pruneError")} t={tt} />
      ));
    }
  });

  const purgeMutation = useMutation({
    mutationFn: (id: number) => APIClient.indexers.deleteArchived(id),
    onSuccess: () => {
      toast.custom((tt) => (
        <Toast type="success" body={t("listScreens.indexers.deprecated.purgeSuccess")} t={tt} />
      ));
      queryClient.invalidateQueries({ queryKey: IndexerKeys.options() });
      queryClient.invalidateQueries({ queryKey: IndexerKeys.deprecations() });
    },
    onError: () => {
      toast.custom((tt) => (
        <Toast type="error" body={t("listScreens.indexers.deprecated.purgeError")} t={tt} />
      ));
    }
  });

  const closeModal = () => setPendingAction(null);
  const modalIsLoading = pruneMutation.isPending || purgeMutation.isPending;
  const modalTitle = pendingAction?.type === "purge"
    ? t("listScreens.indexers.deprecated.purgeTitle", { name: pendingAction.name })
    : pendingAction?.name
      ? t("listScreens.indexers.deprecated.pruneOneTitle", { name: pendingAction.name })
      : t("listScreens.indexers.deprecated.pruneTitle");
  const modalText = pendingAction?.type === "purge"
    ? t("listScreens.indexers.deprecated.purgeText")
    : pendingAction?.name
      ? t("listScreens.indexers.deprecated.pruneOneText", { name: pendingAction.name })
      : t("listScreens.indexers.deprecated.pruneText");

  if (!archived.length) {
    return null;
  }

  return (
    <div className="pt-6">
      <Section
        title={t("listScreens.indexers.deprecated.title")}
        description={t("listScreens.indexers.deprecated.description")}
        rightSide={
          totalFilterUsage > 0 ? (
            <button
              type="button"
              onClick={() => setPendingAction({ type: "prune", identifiers: [] })}
              disabled={pruneMutation.isPending}
              className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-red-600 dark:bg-red-600 hover:bg-red-700 dark:hover:bg-red-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-red-500 disabled:opacity-50 cursor-pointer disabled:cursor-not-allowed"
            >
              <ArchiveBoxXMarkIcon className="h-5 w-5 mr-1" />
              {t("listScreens.indexers.deprecated.pruneAll")}
            </button>
          ) : undefined
        }
      >
        <DeleteModal
          isOpen={pendingAction !== null}
          isLoading={modalIsLoading}
          toggle={closeModal}
          buttonRef={cancelModalButtonRef}
          deleteAction={() => {
            if (pendingAction?.type === "prune") {
              pruneMutation.mutate(pendingAction.identifiers);
            } else if (pendingAction?.type === "purge") {
              purgeMutation.mutate(pendingAction.id);
            }
          }}
          title={modalTitle}
          text={modalText}
        />

        <div className="flex flex-col">
          <ul className="min-w-full relative">
            {archived.map((indexer) => {
              const meta = metaByIdentifier.get(indexer.identifier);
              const usage = meta?.filter_count ?? 0;
              return (
                <li
                  key={indexer.id}
                  className="grid grid-cols-12 gap-2 items-center border-b border-gray-200 dark:border-gray-700 py-3"
                >
                  <div className="col-span-12 sm:col-span-3 pl-0 sm:pl-3 flex items-center gap-x-2">
                    <ArchiveBoxXMarkIcon className="h-4 w-4 shrink-0 text-amber-500 dark:text-amber-400" aria-hidden="true" />
                    <span className="text-sm font-medium text-gray-900 dark:text-white truncate">
                      {meta?.name || indexer.name}
                    </span>
                  </div>
                  <div className="col-span-8 sm:col-span-4 text-sm text-gray-500 dark:text-gray-400">
                    {meta?.reason || t("listScreens.indexers.deprecated.removed")}
                    {meta?.issue_url ? (
                      <>
                        {" "}
                        <ExternalLink href={meta.issue_url} className="text-blue-600 dark:text-blue-400 hover:underline">
                          {t("listScreens.indexers.deprecated.moreInfo")}
                        </ExternalLink>
                      </>
                    ) : null}
                  </div>
                  <div className="col-span-4 sm:col-span-2 text-right text-xs text-gray-500 dark:text-gray-400">
                    {t("listScreens.indexers.deprecated.usedByFilters", { count: usage })}
                  </div>
                  <div className="col-span-12 sm:col-span-3 pr-0 sm:pr-3 flex justify-end">
                    {usage > 0 ? (
                      <button
                        type="button"
                        onClick={() => setPendingAction({
                          type: "prune",
                          identifiers: [indexer.identifier],
                          name: meta?.name || indexer.name
                        })}
                        disabled={pruneMutation.isPending}
                        className="text-sm font-medium text-amber-700 hover:text-amber-900 dark:text-amber-400 dark:hover:text-amber-300 disabled:opacity-50 cursor-pointer disabled:cursor-not-allowed"
                      >
                        {t("listScreens.indexers.deprecated.pruneOne")}
                      </button>
                    ) : (
                      <button
                        type="button"
                        onClick={() => setPendingAction({
                          type: "purge",
                          id: indexer.id,
                          name: meta?.name || indexer.name
                        })}
                        disabled={purgeMutation.isPending}
                        className="inline-flex items-center text-sm font-medium text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 disabled:opacity-50 cursor-pointer disabled:cursor-not-allowed"
                      >
                        <TrashIcon className="h-4 w-4 mr-1" aria-hidden="true" />
                        {t("listScreens.indexers.deprecated.purge")}
                      </button>
                    )}
                  </div>
                </li>
              );
            })}
          </ul>
        </div>
      </Section>
    </div>
  );
}

function IndexerSettingsPage() {
  return (
    <div className="lg:col-span-9">
      <IndexerSettings />
      <DeprecatedIndexers />
    </div>
  );
}

export default IndexerSettingsPage;
