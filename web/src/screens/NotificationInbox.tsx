/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment, useEffect, useMemo, useRef, useState } from "react";
import type { ComponentType, ReactNode, SVGProps } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useSearch } from "@tanstack/react-router";
import {
  Listbox,
  ListboxButton,
  ListboxOption,
  ListboxOptions,
  Menu,
  MenuButton,
  MenuItem,
  MenuItems,
  Transition
} from "@headlessui/react";
import {
  ArrowTopRightOnSquareIcon,
  BellIcon,
  CheckCircleIcon,
  CheckIcon,
  ChevronDownIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  Cog6ToothIcon,
  EllipsisHorizontalIcon,
  ExclamationTriangleIcon,
  TrashIcon
} from "@heroicons/react/24/outline";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";

import { APIClient } from "@api/APIClient";
import { NotificationKeys } from "@api/query_keys";
import { NotificationInboxQueryOptions } from "@api/queries";
import { ConfirmModal } from "@components/modals";
import { ExternalLink } from "@components/ExternalLink";
import { INBOX_EVENT_STYLES, INBOX_FILTER_EVENTS } from "@components/notifications/inboxEvents";
import type { InboxFilterEvent } from "@components/notifications/inboxEvents";
import toast from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { classNames, IsEmptyDate, simplifyDate } from "@utils";
import { paginationRange } from "@utils/pagination";

const DEFAULT_PAGE_SIZE = 25;
const PAGE_SIZES = [10, 25, 50, 100] as const;

type PageSize = typeof PAGE_SIZES[number];

type PendingAction =
  | { kind: "markRead"; ids: number[] }
  | { kind: "delete"; ids: number[] }
  | { kind: "markAllRead"; filtered: boolean }
  | { kind: "clearAll" };

type InboxSearch = {
  page?: number;
  pageSize?: PageSize;
  unread?: boolean;
  event?: InboxFilterEvent;
};

export const NotificationInbox = () => {
  const { t } = useTranslation("common");
  const queryClient = useQueryClient();
  const cancelModalButtonRef = useRef<HTMLInputElement | null>(null);
  const [pending, setPending] = useState<PendingAction | null>(null);
  const [confirmIsOpen, setConfirmIsOpen] = useState(false);
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const selectionAnchor = useRef<number | null>(null);

  const search = useSearch({ from: "/auth/authenticated-routes/notifications" as const });
  const navigate = useNavigate({ from: "/notifications" });
  const page = search.page ?? 0;
  const pageSize = search.pageSize ?? DEFAULT_PAGE_SIZE;
  const unreadOnly = search.unread ?? false;

  const { data, isPending, isPlaceholderData, isError, refetch } = useQuery(NotificationInboxQueryOptions({
    limit: pageSize,
    offset: page * pageSize,
    unread: unreadOnly,
    event: search.event
  }));
  const messages = useMemo(() => data?.data ?? [], [data]);
  const pageCount = Math.max(1, Math.ceil((data?.count ?? 0) / pageSize));

  // Deleting the last page's messages or opening a stale link leaves page past the end.
  const outOfRange = !!data && data.count > 0 && page >= pageCount;
  useEffect(() => {
    if (outOfRange) {
      navigate({ search: (prev) => ({ ...prev, page: pageCount > 1 ? pageCount - 1 : undefined }), replace: true });
    }
  }, [outOfRange, pageCount, navigate]);

  // Live updates refetch the page, so only the selected ids still on it count.
  const selectedIds = useMemo(() => messages.filter(m => selected.has(m.id)).map(m => m.id), [messages, selected]);
  const allSelected = messages.length > 0 && selectedIds.length === messages.length;

  const onMutated = () => {
    setSelected(new Set());
    selectionAnchor.current = null;
    queryClient.invalidateQueries({ queryKey: NotificationKeys.inbox.all() });
  };

  const markReadMutation = useMutation({
    mutationFn: (ids: number[]) => APIClient.notifications.inbox.markRead(ids),
    onSuccess: onMutated
  });

  const deleteMutation = useMutation({
    mutationFn: (ids: number[]) => APIClient.notifications.inbox.delete(ids),
    onSuccess: onMutated
  });

  const deleteAllMutation = useMutation({
    mutationFn: () => APIClient.notifications.inbox.deleteAll(),
    onSuccess: () => {
      onMutated();

      toast.custom((toastInstance) => <Toast type="success" body={t("inbox.cleared")} t={toastInstance} />);
      navigate({ search: (prev) => ({ ...prev, page: undefined }), replace: true });
    }
  });

  const confirmAction = (action: PendingAction) => {
    setPending(action);
    setConfirmIsOpen(true);
  };

  const runPending = () => {
    switch (pending?.kind) {
      case "markRead":
        markReadMutation.mutate(pending.ids);
        break;
      case "delete":
        deleteMutation.mutate(pending.ids);
        break;
      case "markAllRead":
        markReadMutation.mutate([]);
        break;
      case "clearAll":
        deleteAllMutation.mutate();
        break;
    }
  };

  const updateSearch = (next: InboxSearch, replace: boolean) => {
    setSelected(new Set());
    selectionAnchor.current = null;
    navigate({ search: (prev) => ({ ...prev, ...next }), replace });
  };

  // Shift-click applies the clicked row's new state to every row between it and the last clicked row.
  const toggleSelected = (id: number, shiftKey: boolean) => {
    const index = messages.findIndex(m => m.id === id);
    const anchorIndex = shiftKey && selectionAnchor.current !== null
      ? messages.findIndex(m => m.id === selectionAnchor.current)
      : -1;
    const from = anchorIndex === -1 ? index : Math.min(anchorIndex, index);
    const to = anchorIndex === -1 ? index : Math.max(anchorIndex, index);
    const select = !selected.has(id);

    setSelected((prev) => {
      const next = new Set(prev);
      for (const message of messages.slice(from, to + 1)) {
        if (select) {
          next.add(message.id);
        } else {
          next.delete(message.id);
        }
      }
      return next;
    });
    selectionAnchor.current = id;
  };

  const toggleAll = () => {
    setSelected(allSelected ? new Set() : new Set(messages.map(m => m.id)));
  };

  return (
    <main>
      {pending && (
        <ConfirmModal
          isOpen={confirmIsOpen}
          isLoading={false}
          toggle={() => setConfirmIsOpen(false)}
          buttonRef={cancelModalButtonRef}
          onConfirm={runPending}
          {...confirmContent(pending, t)}
        />
      )}

      <div className="flex justify-between items-center flex-row flex-wrap gap-4 my-6 max-w-(--breakpoint-xl) mx-auto px-4 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold text-black dark:text-white">{t("inbox.title")}</h1>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => confirmAction({ kind: "markAllRead", filtered: search.event !== undefined })}
            disabled={!data?.unread_count || markReadMutation.isPending}
            className="inline-flex items-center bg-white dark:bg-gray-700 py-2 px-3 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 disabled:opacity-50 cursor-pointer disabled:cursor-not-allowed"
          >
            <CheckIcon className="h-4 w-4 mr-1.5" aria-hidden="true" />
            {t("inbox.markAllRead")}
          </button>
          <HeaderMenu onClearAll={() => confirmAction({ kind: "clearAll" })} />
        </div>
      </div>

      <div className="max-w-(--breakpoint-xl) mx-auto pb-12 px-2 sm:px-6 lg:px-8">
        <div className="align-middle min-w-full rounded-lg shadow-table bg-gray-50 dark:bg-gray-800 border border-gray-250 dark:border-gray-775">
          <div className="rounded-t-lg flex items-center justify-between gap-4 px-4 bg-gray-125 dark:bg-gray-850 border-b border-gray-200 dark:border-gray-750">
            <div className="flex items-center gap-4">
              <input
                type="checkbox"
                ref={(el) => {
                  if (el) {
                    el.indeterminate = selectedIds.length > 0 && !allSelected;
                  }
                }}
                checked={allSelected}
                onChange={toggleAll}
                disabled={messages.length === 0}
                aria-label={t("inbox.selectAll")}
                title={t("inbox.selectAll")}
                className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-800 cursor-pointer disabled:cursor-not-allowed"
              />
              {selectedIds.length > 0 ? (
                <span className="py-4 text-xs font-bold tracking-wider text-gray-950 dark:text-gray-100">
                  {t("inbox.selected", { selected: selectedIds.length, total: messages.length })}
                </span>
              ) : (
                <div className="flex gap-4" role="group" aria-label={t("inbox.filterLabel")}>
                  <TabButton active={!unreadOnly} count={data?.all_count} onClick={() => updateSearch({ unread: undefined, page: undefined }, true)}>
                    {t("inbox.filterAll")}
                  </TabButton>
                  <TabButton active={unreadOnly} count={data?.unread_count} highlight onClick={() => updateSearch({ unread: true, page: undefined }, true)}>
                    {t("inbox.filterUnread")}
                  </TabButton>
                </div>
              )}
            </div>

            {selectedIds.length > 0 ? (
              <div className="flex items-center gap-2 py-2">
                <BulkButton
                  onClick={() => confirmAction({ kind: "markRead", ids: selectedIds })}
                  disabled={markReadMutation.isPending}
                  icon={CheckIcon}
                >
                  {t("inbox.markRead")}
                </BulkButton>
                <BulkButton
                  onClick={() => confirmAction({ kind: "delete", ids: selectedIds })}
                  disabled={deleteMutation.isPending}
                  icon={TrashIcon}
                  destructive
                >
                  {t("inbox.delete")}
                </BulkButton>
              </div>
            ) : (
              <EventTypeFilter
                value={search.event}
                onChange={(event) => updateSearch({ event, page: undefined }, true)}
              />
            )}
          </div>

          {isPending || outOfRange || (isPlaceholderData && messages.length === 0) ? (
            <InboxSkeleton />
          ) : isError && !data ? (
            <EmptyInbox
              icon={ExclamationTriangleIcon}
              title={t("inbox.loadError")}
              description={t("inbox.loadErrorDescription")}
              action={
                <button
                  type="button"
                  onClick={() => refetch()}
                  className="text-sm font-medium text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 cursor-pointer"
                >
                  {t("inbox.retry")}
                </button>
              }
            />
          ) : messages.length > 0 ? (
            <ul className="min-w-full divide-y divide-gray-150 dark:divide-gray-775">
              {messages.map((message, idx) => (
                <InboxListItem
                  key={message.id}
                  message={message}
                  idx={idx}
                  selected={selected.has(message.id)}
                  onToggleSelected={toggleSelected}
                  onMarkRead={(id) => confirmAction({ kind: "markRead", ids: [id] })}
                  onDelete={(id) => confirmAction({ kind: "delete", ids: [id] })}
                />
              ))}
            </ul>
          ) : unreadOnly ? (
            <EmptyInbox
              icon={CheckCircleIcon}
              title={t("inbox.caughtUp")}
              description={t("inbox.caughtUpDescription")}
              action={
                <button
                  type="button"
                  onClick={() => updateSearch({ unread: undefined, page: undefined }, true)}
                  className="text-sm font-medium text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 cursor-pointer"
                >
                  {t("inbox.showAll")}
                </button>
              }
            />
          ) : (
            <EmptyInbox
              icon={BellIcon}
              title={t("inbox.empty")}
              description={t("inbox.emptyDescription")}
              action={
                <Link
                  to="/settings/notifications"
                  className="text-sm font-medium text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 cursor-pointer"
                >
                  {t("inbox.configure")}
                </Link>
              }
            />
          )}
        </div>

        {(pageCount > 1 || pageSize !== DEFAULT_PAGE_SIZE) && (
          <InboxPagination
            page={page}
            pageCount={pageCount}
            pageSize={pageSize}
            onPageChange={(next) => updateSearch({ page: next === 0 ? undefined : next }, false)}
            onPageSizeChange={(size) => updateSearch({ pageSize: size === DEFAULT_PAGE_SIZE ? undefined : size, page: undefined }, false)}
          />
        )}
      </div>
    </main>
  );
};

const confirmContent = (action: PendingAction, t: TFunction) => {
  switch (action.kind) {
    case "markRead":
      return {
        title: t("inbox.markRead"),
        text: action.ids.length === 1 ? t("inbox.confirmMarkReadOne") : t("inbox.confirmMarkReadMany", { selected: action.ids.length }),
        confirmLabel: t("inbox.markRead"),
        tone: "primary" as const
      };
    case "delete":
      return {
        title: t("inbox.deleteTitle"),
        text: action.ids.length === 1 ? t("inbox.confirmDeleteOne") : t("inbox.confirmDeleteMany", { selected: action.ids.length }),
        confirmLabel: t("inbox.delete"),
        tone: "danger" as const
      };
    case "markAllRead":
      return {
        title: t("inbox.markAllRead"),
        text: action.filtered ? t("inbox.confirmMarkAllReadFiltered") : t("inbox.confirmMarkAllRead"),
        confirmLabel: t("inbox.markAllRead"),
        tone: "primary" as const
      };
    case "clearAll":
      return {
        title: t("inbox.clearTitle"),
        text: t("inbox.clearText"),
        confirmLabel: t("inbox.clearAll"),
        tone: "danger" as const
      };
  }
};

interface TabButtonProps {
  active: boolean;
  count?: number;
  highlight?: boolean;
  onClick: () => void;
  children: ReactNode;
}

const TabButton = ({ active, count, highlight = false, onClick, children }: TabButtonProps) => (
  <button
    type="button"
    aria-pressed={active}
    onClick={onClick}
    className={classNames(
      "cursor-pointer",
      "inline-flex items-center gap-1.5 py-4 text-left text-xs tracking-wider transition border-b-2",
      active
        ? "font-bold border-blue-500 dark:text-gray-100 text-gray-950"
        : "font-medium border-transparent text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200"
    )}
  >
    {children}
    {count !== undefined && (
      <span
        className={classNames(
          "rounded-full px-1.5 py-px text-[11px] font-semibold leading-4 tabular-nums",
          highlight && count > 0
            ? "bg-blue-600 text-white"
            : "bg-gray-200 text-gray-700 dark:bg-gray-750 dark:text-gray-300"
        )}
      >
        {count}
      </span>
    )}
  </button>
);

interface BulkButtonProps {
  onClick: () => void;
  disabled: boolean;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  destructive?: boolean;
  children: ReactNode;
}

const BulkButton = ({ onClick, disabled, icon: Icon, destructive = false, children }: BulkButtonProps) => (
  <button
    type="button"
    onClick={onClick}
    disabled={disabled}
    className={classNames(
      "inline-flex items-center bg-white dark:bg-gray-800 py-1.5 px-2.5 border border-gray-300 dark:border-gray-700 rounded-md shadow-xs text-xs font-medium hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 cursor-pointer disabled:cursor-not-allowed",
      destructive ? "text-red-600 dark:text-red-400" : "text-gray-700 dark:text-gray-200"
    )}
  >
    <Icon className="h-4 w-4 mr-1" aria-hidden="true" />
    {children}
  </button>
);

interface EventTypeFilterProps {
  value?: InboxFilterEvent;
  onChange: (event?: InboxFilterEvent) => void;
}

const EventTypeFilter = ({ value, onChange }: EventTypeFilterProps) => {
  const { t } = useTranslation(["common", "options"]);

  return (
    <Listbox value={value ?? ""} onChange={(next: string) => onChange(next === "" ? undefined : next as InboxFilterEvent)}>
      <div className="relative">
        <ListboxButton className="relative w-full py-2 pr-5 text-left text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 cursor-pointer">
          <span className="block truncate">
            {value
              ? `${t("common:inbox.eventFilter")}: ${t(`options:event.${value}.label`)}`
              : t("common:inbox.eventFilter")}
          </span>
          <span className="absolute inset-y-0 right-0 flex items-center pointer-events-none">
            <ChevronDownIcon className="w-3 h-3" aria-hidden="true" />
          </span>
        </ListboxButton>
        <Transition
          as={Fragment}
          leave="transition ease-in duration-100"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <ListboxOptions className="w-56 absolute z-10 mt-1 right-0 overflow-auto text-base bg-white dark:bg-gray-800 rounded-md shadow-lg max-h-72 border border-black/5 dark:border-gray-700/40 focus:outline-hidden sm:text-sm">
            <EventOption label={t("common:inbox.allEvents")} value="" />
            {INBOX_FILTER_EVENTS.map((event) => (
              <EventOption key={event} label={t(`options:event.${event}.label`)} value={event} />
            ))}
          </ListboxOptions>
        </Transition>
      </div>
    </Listbox>
  );
};

interface EventOptionProps {
  label: string;
  value: string;
}

const EventOption = ({ label, value }: EventOptionProps) => {
  const style = value ? INBOX_EVENT_STYLES[value as NotificationEvent] : undefined;

  return (
    <ListboxOption
      className={({ focus }) => classNames(
        "cursor-pointer select-none relative py-2 pl-4 pr-9",
        focus ? "text-black dark:text-gray-200 bg-gray-100 dark:bg-gray-900" : "text-gray-700 dark:text-gray-400"
      )}
      value={value}
    >
      {({ selected }) => (
        <>
          <span className={classNames("flex items-center gap-2 truncate", selected ? "font-medium text-black dark:text-white" : "font-normal")}>
            {style && <style.icon className={classNames("h-4 w-4 shrink-0", style.text)} aria-hidden="true" />}
            {label}
          </span>
          {selected && (
            <span className="absolute inset-y-0 right-0 flex items-center pr-3 text-gray-500 dark:text-gray-400">
              <CheckIcon className="w-5 h-5" aria-hidden="true" />
            </span>
          )}
        </>
      )}
    </ListboxOption>
  );
};

const HeaderMenu = ({ onClearAll }: { onClearAll: () => void }) => {
  const { t } = useTranslation("common");

  return (
    <Menu as="div" className="relative">
      <MenuButton
        className="inline-flex items-center bg-white dark:bg-gray-700 p-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer"
        title={t("inbox.moreActions")}
      >
        <span className="sr-only">{t("inbox.moreActions")}</span>
        <EllipsisHorizontalIcon className="h-5 w-5" aria-hidden="true" />
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
          anchor={{ to: "bottom end", padding: "8px" }}
          className="w-56 mt-1 bg-white dark:bg-gray-825 divide-y divide-gray-200 dark:divide-gray-750 rounded-md shadow-lg border border-gray-250 dark:border-gray-750 focus:outline-hidden z-10"
        >
          <MenuItem>
            {({ focus }) => (
              <Link
                to="/settings/notifications"
                className={classNames(
                  focus ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                  "font-medium group flex rounded-t-md items-center w-full px-3 py-2 text-sm cursor-pointer"
                )}
              >
                <Cog6ToothIcon className={classNames(focus ? "text-white" : "text-blue-500", "w-5 h-5 mr-2")} aria-hidden="true" />
                {t("inbox.configure")}
              </Link>
            )}
          </MenuItem>
          <MenuItem>
            {({ focus }) => (
              <button
                type="button"
                onClick={onClearAll}
                className={classNames(
                  focus ? "bg-red-600 text-white" : "text-gray-900 dark:text-gray-300",
                  "font-medium group flex rounded-b-md items-center w-full px-3 py-2 text-sm cursor-pointer"
                )}
              >
                <TrashIcon className={classNames(focus ? "text-white" : "text-red-500", "w-5 h-5 mr-2")} aria-hidden="true" />
                {t("inbox.clearAll")}
              </button>
            )}
          </MenuItem>
        </MenuItems>
      </Transition>
    </Menu>
  );
};

interface InboxListItemProps {
  message: InboxMessage;
  idx: number;
  selected: boolean;
  onToggleSelected: (id: number, shiftKey: boolean) => void;
  onMarkRead: (id: number) => void;
  onDelete: (id: number) => void;
}

const InboxListItem = ({ message, idx, selected, onToggleSelected, onMarkRead, onDelete }: InboxListItemProps) => {
  const { t } = useTranslation(["common", "options"]);
  const unread = message.read_at === null;
  const style = INBOX_EVENT_STYLES[message.event] ?? INBOX_EVENT_STYLES.TEST;
  const Icon = style.icon;
  const eventLabel = message.event === "TEST"
    ? message.title
    : t(`options:event.${message.event}.label`, { defaultValue: message.title });
  const headline = message.release_name || message.message || eventLabel;
  const action = message.action_client && message.action_client !== message.action
    ? `${message.action} (${message.action_client})`
    : message.action;
  const headlineClass = unread ? "font-semibold text-gray-900 dark:text-white" : "font-medium text-gray-700 dark:text-gray-300";

  const meta: ReactNode[] = [
    <time key="time" dateTime={message.created_at} title={simplifyDate(message.created_at)}>
      {IsEmptyDate(message.created_at)}
    </time>
  ];
  if (message.release_name && message.message) {
    meta.push(<span key="message">{message.message}</span>);
  }
  if (message.indexer) {
    meta.push(<span key="indexer" title={t("common:inbox.fields.indexer")}>{message.indexer}</span>);
  }
  if (message.filter_name) {
    meta.push(message.filter_id > 0 ? (
      <Link
        key="filter"
        to="/filters/$filterId"
        params={{ filterId: message.filter_id }}
        title={t("common:inbox.fields.filter")}
        className="hover:text-blue-600 dark:hover:text-blue-400 hover:underline cursor-pointer"
      >
        {message.filter_name}
      </Link>
    ) : (
      <span key="filter" title={t("common:inbox.fields.filter")}>{message.filter_name}</span>
    ));
  }
  if (action) {
    meta.push(<span key="action" title={t("common:inbox.fields.action")}>{action}</span>);
  }
  if (message.rejections.length > 0) {
    meta.push(
      <span
        key="rejections"
        title={message.rejections.join("\n")}
        className={classNames("truncate", message.event === "PUSH_REJECTED" ? "text-amber-700 dark:text-amber-400" : "text-red-600 dark:text-red-400")}
      >
        {message.rejections.join(", ")}
      </span>
    );
  }
  if (message.url) {
    meta.push(
      <ExternalLink
        key="url"
        href={message.url}
        className="inline-flex items-center gap-0.5 text-blue-600 dark:text-blue-400 hover:underline"
      >
        {t("common:inbox.viewReleaseNotes")}
        <ArrowTopRightOnSquareIcon className="h-3 w-3" aria-hidden="true" />
      </ExternalLink>
    );
  }

  return (
    <li
      className={classNames(
        "group flex items-start gap-3 px-4 py-2.5 transition last:rounded-b-lg",
        selected
          ? "bg-blue-50 dark:bg-blue-500/10"
          : idx % 2 === 0 ? "bg-white dark:bg-gray-800" : "bg-gray-75 dark:bg-gray-825"
      )}
    >
      <input
        type="checkbox"
        checked={selected}
        onChange={(e) => onToggleSelected(message.id, (e.nativeEvent as MouseEvent).shiftKey)}
        onMouseDown={(e) => {
          // Keep the browser from selecting the text between the two clicked rows.
          if (e.shiftKey) {
            e.preventDefault();
          }
        }}
        aria-label={t("common:inbox.selectMessage", { title: headline })}
        className="mt-1 h-4 w-4 shrink-0 rounded border-gray-300 text-blue-600 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-800 cursor-pointer"
      />
      <Icon className={classNames("mt-0.5 h-5 w-5 shrink-0", style.text)} aria-hidden="true" />

      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
          {message.release_name ? (
            <Link
              to="/releases"
              search={{ q: message.release_name }}
              title={t("common:inbox.searchRelease")}
              className={classNames("min-w-0 break-all text-sm hover:text-blue-600 dark:hover:text-blue-400 cursor-pointer", headlineClass)}
            >
              {headline}
            </Link>
          ) : (
            <span className={classNames("min-w-0 break-words text-sm", headlineClass)}>
              {headline}
            </span>
          )}
          {headline !== eventLabel && (
            <span className={classNames("inline-flex items-center rounded-full px-2 py-px text-xs font-medium ring-1 ring-inset", style.label)}>
              {eventLabel}
            </span>
          )}
          {unread && (
            <span className="h-2 w-2 shrink-0 rounded-full bg-blue-500">
              <span className="sr-only">{t("common:inbox.unread")}</span>
            </span>
          )}
        </div>
        <p className="mt-0.5 flex min-w-0 flex-wrap items-center gap-x-1.5 text-xs text-gray-500 dark:text-gray-400">
          {meta.map((part, partIdx) => (
            <Fragment key={partIdx}>
              {partIdx > 0 && <span aria-hidden="true">·</span>}
              {part}
            </Fragment>
          ))}
        </p>
      </div>

      <div className="flex w-14 shrink-0 self-center items-center justify-end sm:opacity-0 sm:group-hover:opacity-100 sm:group-focus-within:opacity-100 transition-opacity">
        {unread && (
          <button
            type="button"
            onClick={() => onMarkRead(message.id)}
            title={t("common:inbox.markRead")}
            className="rounded-md p-1 text-gray-500 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 hover:text-gray-900 dark:hover:text-white focus:outline-hidden focus-visible:ring-2 focus-visible:ring-blue-500 cursor-pointer"
          >
            <span className="sr-only">{t("common:inbox.markRead")}</span>
            <CheckIcon className="h-4 w-4" aria-hidden="true" />
          </button>
        )}
        <button
          type="button"
          onClick={() => onDelete(message.id)}
          title={t("common:inbox.delete")}
          className="rounded-md p-1 text-gray-500 dark:text-gray-400 hover:bg-red-100 dark:hover:bg-red-500/15 hover:text-red-700 dark:hover:text-red-400 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-red-500 cursor-pointer"
        >
          <span className="sr-only">{t("common:inbox.delete")}</span>
          <TrashIcon className="h-4 w-4" aria-hidden="true" />
        </button>
      </div>
    </li>
  );
};

interface InboxPaginationProps {
  page: number;
  pageCount: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: PageSize) => void;
}

const InboxPagination = ({ page, pageCount, pageSize, onPageChange, onPageSizeChange }: InboxPaginationProps) => {
  const { t } = useTranslation("common");
  const items = paginationRange(page, pageCount);

  return (
    <div className="mt-4 flex flex-col items-center gap-3 sm:grid sm:grid-cols-[1fr_auto_1fr]">
      <div className="hidden sm:block" />
      <nav className="flex flex-wrap items-center justify-center gap-1 text-sm" aria-label={t("releaseTable.pagination")}>
        <button
          type="button"
          onClick={() => onPageChange(page - 1)}
          disabled={page === 0}
          className="inline-flex items-center rounded-md px-2 py-1 text-blue-600 dark:text-blue-400 hover:bg-gray-200 dark:hover:bg-gray-800 disabled:text-gray-400 dark:disabled:text-gray-600 disabled:hover:bg-transparent cursor-pointer disabled:cursor-not-allowed"
        >
          <ChevronLeftIcon className="h-4 w-4 mr-0.5" aria-hidden="true" />
          {t("inbox.previous")}
        </button>
        {items.map((item, idx) => item === "gap" ? (
          <span key={`gap-${idx}`} className="px-1 text-gray-500 dark:text-gray-400" aria-hidden="true">…</span>
        ) : (
          <button
            key={item}
            type="button"
            onClick={() => onPageChange(item)}
            aria-current={item === page ? "page" : undefined}
            className={classNames(
              "min-w-8 rounded-md px-2 py-1 tabular-nums",
              item === page
                ? "bg-blue-600 font-medium text-white"
                : "text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-800 cursor-pointer"
            )}
          >
            {item + 1}
          </button>
        ))}
        <button
          type="button"
          onClick={() => onPageChange(page + 1)}
          disabled={page + 1 >= pageCount}
          className="inline-flex items-center rounded-md px-2 py-1 text-blue-600 dark:text-blue-400 hover:bg-gray-200 dark:hover:bg-gray-800 disabled:text-gray-400 dark:disabled:text-gray-600 disabled:hover:bg-transparent cursor-pointer disabled:cursor-not-allowed"
        >
          {t("inbox.next")}
          <ChevronRightIcon className="h-4 w-4 ml-0.5" aria-hidden="true" />
        </button>
      </nav>
      <label className="sm:justify-self-end">
        <span className="sr-only">{t("releaseTable.itemsPerPage")}</span>
        <select
          className="py-1 pl-2 pr-8 text-sm block w-full border-gray-300 rounded-md shadow-xs cursor-pointer transition-colors dark:bg-gray-800 dark:border-gray-600 dark:text-gray-400 dark:hover:text-gray-200 focus:border-blue-300 focus:ring-3 focus:ring-blue-200 focus:ring-opacity-50"
          value={pageSize}
          onChange={(e) => onPageSizeChange(Number(e.target.value) as PageSize)}
        >
          {PAGE_SIZES.map((size) => (
            <option key={size} value={size}>
              {t("releaseTable.entries", { count: size })}
            </option>
          ))}
        </select>
      </label>
    </div>
  );
};

interface EmptyInboxProps {
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  title: string;
  description: string;
  action: ReactNode;
}

const EmptyInbox = ({ icon: Icon, title, description, action }: EmptyInboxProps) => (
  <div className="flex flex-col items-center rounded-b-lg bg-white dark:bg-gray-800 px-4 py-16 text-center">
    <span className="flex size-12 items-center justify-center rounded-full bg-gray-100 dark:bg-gray-750 text-gray-500 dark:text-gray-400">
      <Icon className="h-6 w-6" aria-hidden="true" />
    </span>
    <h3 className="mt-4 text-sm font-semibold text-gray-900 dark:text-white">{title}</h3>
    <p className="mt-1 max-w-sm text-sm text-gray-500 dark:text-gray-400">{description}</p>
    <div className="mt-4">{action}</div>
  </div>
);

const InboxSkeleton = () => (
  <ul className="divide-y divide-gray-150 dark:divide-gray-775 animate-pulse" aria-hidden="true">
    {Array.from({ length: 5 }, (_, idx) => (
      <li key={idx} className={classNames("flex gap-3 px-4 py-3", idx % 2 === 0 ? "bg-white dark:bg-gray-800" : "bg-gray-75 dark:bg-gray-825")}>
        <span className="h-4 w-4 rounded bg-gray-200 dark:bg-gray-750" />
        <span className="h-5 w-5 rounded-full bg-gray-200 dark:bg-gray-750" />
        <div className="flex-1 space-y-2">
          <div className="h-3 w-2/3 rounded bg-gray-200 dark:bg-gray-750" />
          <div className="h-2.5 w-1/3 rounded bg-gray-200 dark:bg-gray-750" />
        </div>
      </li>
    ))}
  </ul>
);
