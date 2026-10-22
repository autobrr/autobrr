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
  Cog6ToothIcon,
  EllipsisHorizontalIcon,
  ExclamationTriangleIcon,
  TrashIcon
} from "@heroicons/react/24/outline";
import { ChevronDownIcon } from "@heroicons/react/24/solid";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";

import { APIClient } from "@api/APIClient";
import { NotificationKeys } from "@api/query_keys";
import { NotificationInboxQueryOptions } from "@api/queries";
import { TablePagination } from "@components/data-table";
import { ConfirmModal } from "@components/modals";
import { ExternalLink } from "@components/ExternalLink";
import { INBOX_EVENT_STYLES, INBOX_FILTER_EVENTS, inboxEventLabel, inboxHeadline, inboxRejectionClass } from "@components/notifications/inboxEvents";
import type { InboxFilterEvent } from "@components/notifications/inboxEvents";
import { InboxAge } from "@components/notifications/InboxAge";
import { InboxEventIcon } from "@components/notifications/InboxEventIcon";
import toast from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { classNames } from "@utils";

const DEFAULT_PAGE_SIZE = 25;
const PAGE_SIZES = [10, 25, 50, 100] as const;
const COLUMN_COUNT = 5;

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

      <div className="mt-6 mb-4 max-w-(--breakpoint-xl) mx-auto px-4 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold text-black dark:text-white">{t("inbox.title")}</h1>
      </div>

      <div className="max-w-(--breakpoint-xl) mx-auto pb-6 px-2 sm:px-6 lg:pb-16 lg:px-8">
        <div className="flex flex-col gap-3 mb-6 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-stretch gap-3">
            <div className="flex items-center gap-1 p-1 rounded-lg shadow-md bg-white dark:bg-gray-800 text-sm" role="group" aria-label={t("inbox.filterLabel")}>
              <TabButton active={!unreadOnly} count={data?.all_count} onClick={() => updateSearch({ unread: undefined, page: undefined }, true)}>
                {t("inbox.filterAll")}
              </TabButton>
              <TabButton active={unreadOnly} count={data?.unread_count} highlight onClick={() => updateSearch({ unread: true, page: undefined }, true)}>
                {t("inbox.filterUnread")}
              </TabButton>
            </div>
            <button
              type="button"
              onClick={() => confirmAction({ kind: "markAllRead", filtered: search.event !== undefined })}
              disabled={!data?.unread_count || markReadMutation.isPending}
              title={t("inbox.markAllRead")}
              className={classNames(TOOLBAR_BUTTON, "gap-1.5 px-2 sm:px-3 text-sm whitespace-nowrap")}
            >
              <CheckIcon className="h-5 w-5 sm:h-4 sm:w-4" aria-hidden="true" />
              <span className="sr-only sm:not-sr-only">{t("inbox.markAllRead")}</span>
            </button>
          </div>
          <div className="flex items-stretch gap-3">
            <EventTypeFilter
              value={search.event}
              onChange={(event) => updateSearch({ event, page: undefined }, true)}
            />
            <MoreActionsMenu onClearAll={() => confirmAction({ kind: "clearAll" })} />
          </div>
        </div>

        <div className="bg-white dark:bg-gray-800 border border-gray-250 dark:border-gray-775 shadow-table rounded-md overflow-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-750">
            <thead className="bg-gray-100 dark:bg-gray-850">
              <tr className="h-10">
                <th scope="col" className="w-px pl-3 pr-2 sm:pl-5 sm:pr-3">
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
                    className="block h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-800 cursor-pointer disabled:cursor-not-allowed"
                  />
                </th>
                {selectedIds.length > 0 ? (
                  <th scope="col" colSpan={COLUMN_COUNT - 1} className="py-0 pl-2 pr-3 sm:pl-3 sm:pr-5">
                    <div className="flex items-center justify-between gap-3">
                      <span className="text-xs font-bold tracking-wider whitespace-nowrap text-gray-950 dark:text-gray-100">
                        {t("inbox.selected", { selected: selectedIds.length, total: messages.length })}
                      </span>
                      <div className="flex items-center gap-2">
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
                    </div>
                  </th>
                ) : (
                  <>
                    <ColumnHeader className="hidden sm:table-cell w-px">{t("inbox.columns.age")}</ColumnHeader>
                    <ColumnHeader className="px-2 sm:px-3">{t("inbox.columns.message")}</ColumnHeader>
                    <ColumnHeader className="hidden sm:table-cell w-px">{t("inbox.columns.event")}</ColumnHeader>
                    <ColumnHeader className="w-px pl-1 pr-3 sm:pl-3 sm:pr-5"><span className="sr-only sm:not-sr-only sm:whitespace-nowrap">{t("inbox.columns.actions")}</span></ColumnHeader>
                  </>
                )}
              </tr>
            </thead>

            <tbody className="divide-y divide-gray-150 dark:divide-gray-750">
              {isPending || outOfRange || (isPlaceholderData && messages.length === 0) ? (
                <InboxSkeleton />
              ) : isError && !data ? (
                <EmptyInbox
                  colSpan={COLUMN_COUNT}
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
                messages.map((message) => (
                  <InboxRow
                    key={message.id}
                    message={message}
                    selected={selected.has(message.id)}
                    onToggleSelected={toggleSelected}
                    onMarkRead={(id) => confirmAction({ kind: "markRead", ids: [id] })}
                    onDelete={(id) => confirmAction({ kind: "delete", ids: [id] })}
                  />
                ))
              ) : unreadOnly ? (
                <EmptyInbox
                  colSpan={COLUMN_COUNT}
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
                  colSpan={COLUMN_COUNT}
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
            </tbody>
          </table>

          {/* A non-default page size keeps the footer on an empty page so the size can be changed back. */}
          {!isPending && !outOfRange && !(isError && !data) && (messages.length > 0 || pageSize !== DEFAULT_PAGE_SIZE) && (
            <TablePagination
              pageIndex={page}
              pageCount={pageCount}
              pageSize={pageSize}
              pageSizes={PAGE_SIZES}
              onPageChange={(next) => updateSearch({ page: next === 0 ? undefined : next }, false)}
              onPageSizeChange={(size) => updateSearch({ pageSize: size === DEFAULT_PAGE_SIZE ? undefined : size as PageSize, page: undefined }, false)}
            />
          )}
        </div>
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

// Matches the ListboxButton in EventTypeFilter so the toolbar controls read as one row.
const TOOLBAR_BUTTON = "inline-flex items-center justify-center rounded-lg shadow-md bg-white dark:bg-gray-800 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-blue-500 disabled:opacity-50 disabled:hover:text-gray-600 dark:disabled:hover:text-gray-400 cursor-pointer disabled:cursor-not-allowed";

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
      "inline-flex items-center gap-1.5 px-3 py-1 rounded-md transition cursor-pointer",
      active
        ? "font-medium bg-gray-100 text-gray-900 dark:bg-gray-700 dark:text-white"
        : "text-gray-600 dark:text-gray-400 hover:bg-gray-100 hover:text-gray-900 dark:hover:bg-gray-750 dark:hover:text-gray-200"
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

interface ColumnHeaderProps {
  className?: string;
  children: ReactNode;
}

const ColumnHeader = ({ className, children }: ColumnHeaderProps) => (
  <th
    scope="col"
    className={classNames(
      "px-3 py-3 text-xs font-medium tracking-wider text-left uppercase whitespace-nowrap text-gray-600 dark:text-gray-400",
      className ?? ""
    )}
  >
    {children}
  </th>
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
      "inline-flex items-center gap-1 whitespace-nowrap bg-white dark:bg-gray-800 py-1 px-2 border border-gray-300 dark:border-gray-700 rounded-md shadow-xs text-xs font-medium hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 cursor-pointer disabled:cursor-not-allowed",
      destructive ? "text-red-600 dark:text-red-400" : "text-gray-700 dark:text-gray-200"
    )}
  >
    <Icon className="h-4 w-4" aria-hidden="true" />
    <span className="sr-only sm:not-sr-only">{children}</span>
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
      <div className="relative flex-1 sm:flex-none sm:w-56">
        <ListboxButton className="relative w-full py-2 pl-3 pr-10 text-left bg-white dark:bg-gray-800 rounded-lg shadow-md cursor-pointer dark:text-gray-400 sm:text-sm">
          <span className="block truncate">
            {value
              ? `${t("common:inbox.eventFilter")}: ${t(`options:event.${value}.label`)}`
              : t("common:inbox.eventFilter")}
          </span>
          <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
            <ChevronDownIcon className="w-5 h-5 ml-2 -mr-1 text-gray-600 dark:text-gray-400" aria-hidden="true" />
          </span>
        </ListboxButton>
        <Transition
          as={Fragment}
          leave="transition ease-in duration-100"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <ListboxOptions className="absolute z-10 w-full mt-1 overflow-auto text-base bg-white dark:bg-gray-800 rounded-md shadow-lg max-h-72 border border-black/5 dark:border-gray-700/40 focus:outline-hidden sm:text-sm">
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
        "cursor-pointer select-none relative py-2 pl-10 pr-4",
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
            <span className="absolute inset-y-0 left-0 flex items-center pl-3 text-gray-500 dark:text-gray-400">
              <CheckIcon className="w-5 h-5" aria-hidden="true" />
            </span>
          )}
        </>
      )}
    </ListboxOption>
  );
};

const MoreActionsMenu = ({ onClearAll }: { onClearAll: () => void }) => {
  const { t } = useTranslation("common");

  return (
    <Menu as="div" className="relative flex">
      <MenuButton
        className={classNames(TOOLBAR_BUTTON, "px-2")}
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
          className="w-56 mt-1 z-10 divide-y divide-gray-100 dark:divide-gray-750 rounded-md shadow-lg bg-white dark:bg-gray-800 border border-gray-250 dark:border-gray-775 focus:outline-hidden"
        >
          <MenuItem>
            {({ focus }) => (
              <Link
                to="/settings/notifications"
                className={classNames(
                  focus ? "bg-gray-100 dark:bg-gray-600" : "",
                  "flex items-center w-full transition rounded-t-md px-2 py-2 text-sm text-gray-900 dark:text-gray-200 cursor-pointer"
                )}
              >
                <Cog6ToothIcon className="w-5 h-5 mr-1 text-gray-700 dark:text-gray-400" aria-hidden="true" />
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
                  focus ? "bg-gray-100 dark:bg-gray-600" : "",
                  "flex items-center w-full transition rounded-b-md px-2 py-2 text-sm text-red-600 dark:text-red-400 cursor-pointer"
                )}
              >
                <TrashIcon className="w-5 h-5 mr-1" aria-hidden="true" />
                {t("inbox.clearAll")}
              </button>
            )}
          </MenuItem>
        </MenuItems>
      </Transition>
    </Menu>
  );
};

interface InboxRowProps {
  message: InboxMessage;
  selected: boolean;
  onToggleSelected: (id: number, shiftKey: boolean) => void;
  onMarkRead: (id: number) => void;
  onDelete: (id: number) => void;
}

const InboxRow = ({ message, selected, onToggleSelected, onMarkRead, onDelete }: InboxRowProps) => {
  const { t } = useTranslation(["common", "options"]);
  const unread = message.read_at === null;
  const eventLabel = inboxEventLabel(message, t);
  const headline = inboxHeadline(message, eventLabel);
  const style = INBOX_EVENT_STYLES[message.event] ?? INBOX_EVENT_STYLES.TEST;
  const headlineClass = unread ? "font-semibold text-gray-900 dark:text-white" : "font-medium text-gray-700 dark:text-gray-300";

  return (
    <tr className={selected ? "bg-blue-50 dark:bg-blue-500/10" : undefined}>
      <td className="w-px pl-3 pr-2 sm:pl-5 sm:pr-3">
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
          className="block h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-800 cursor-pointer"
        />
      </td>

      <td className="hidden sm:table-cell w-px px-3 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
        <InboxAge createdAt={message.created_at} />
      </td>

      {/* max-w-0 lets this column take the remaining width and truncate instead of widening the table. */}
      <td className="w-full max-w-0 px-2 py-2 sm:px-3">
        <div className="flex items-start gap-2 min-w-0">
          {/* The event column is hidden below sm, so the icon moves here; aria-label keeps the name for screen readers. */}
          <style.icon role="img" aria-label={eventLabel} className={classNames("sm:hidden mt-0.5 h-4 w-4 shrink-0", style.text)} />
          <span className={classNames("mt-1.5 h-2 w-2 shrink-0 rounded-full", unread ? "bg-blue-500" : "")}>
            {unread && <span className="sr-only">{t("common:inbox.unread")}</span>}
          </span>
          <div className="min-w-0 flex-1">
            {message.release_name ? (
              <Link
                to="/releases"
                search={{ q: message.release_name }}
                title={`${headline}\n${t("common:inbox.searchRelease")}`}
                className={classNames("line-clamp-2 break-all sm:line-clamp-none sm:truncate text-sm hover:text-blue-600 dark:hover:text-blue-400 cursor-pointer", headlineClass)}
              >
                {headline}
              </Link>
            ) : (
              <span className={classNames("line-clamp-2 break-words sm:line-clamp-none sm:truncate text-sm", headlineClass)} title={headline}>
                {headline}
              </span>
            )}
            <MessageDetails message={message} />
          </div>
        </div>
      </td>

      <td className="hidden sm:table-cell w-px px-3">
        <div className="flex">
          <InboxEventIcon event={message.event} label={eventLabel} showLabel />
        </div>
      </td>

      <td className="w-px pl-1 pr-3 sm:pl-3 sm:pr-5 whitespace-nowrap">
        <div className="flex items-center gap-0.5 sm:gap-1">
          {unread ? (
            <button
              type="button"
              onClick={() => onMarkRead(message.id)}
              title={t("common:inbox.markRead")}
              className="rounded-md p-1.5 sm:p-1 text-gray-500 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 hover:text-gray-900 dark:hover:text-white focus:outline-hidden focus-visible:ring-2 focus-visible:ring-blue-500 cursor-pointer"
            >
              <span className="sr-only">{t("common:inbox.markRead")}</span>
              <CheckIcon className="h-5 w-5" aria-hidden="true" />
            </button>
          ) : (
            <span className="p-1.5 sm:p-1" aria-hidden="true">
              <span className="block h-5 w-5" />
            </span>
          )}
          <button
            type="button"
            onClick={() => onDelete(message.id)}
            title={t("common:inbox.delete")}
            className="rounded-md p-1.5 sm:p-1 text-gray-500 dark:text-gray-400 hover:bg-red-100 dark:hover:bg-red-500/15 hover:text-red-700 dark:hover:text-red-400 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-red-500 cursor-pointer"
          >
            <span className="sr-only">{t("common:inbox.delete")}</span>
            <TrashIcon className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>
      </td>
    </tr>
  );
};

const MessageDetails = ({ message }: { message: InboxMessage }) => {
  const { t } = useTranslation("common");
  const label = "text-gray-500 dark:text-gray-400";
  const action = message.action_client && message.action_client !== message.action
    ? `${message.action} (${message.action_client})`
    : message.action;

  const parts: ReactNode[] = [];
  if (message.release_name && message.message) {
    parts.push(<span key="message">{message.message}</span>);
  }
  if (message.indexer) {
    parts.push(<Fragment key="indexer"><span className={label}>{t("inbox.fields.indexer")}:</span> <span>{message.indexer}</span></Fragment>);
  }
  if (message.filter_name) {
    parts.push(
      <Fragment key="filter">
        <span className={label}>{t("inbox.fields.filter")}:</span>{" "}
        {message.filter_id > 0 ? (
          <Link
            to="/filters/$filterId"
            params={{ filterId: message.filter_id }}
            className="hover:text-blue-600 dark:hover:text-blue-400 hover:underline cursor-pointer"
          >
            {message.filter_name}
          </Link>
        ) : (
          <span>{message.filter_name}</span>
        )}
      </Fragment>
    );
  }
  if (action) {
    parts.push(<Fragment key="action"><span className={label}>{t("inbox.fields.action")}:</span> <span>{action}</span></Fragment>);
  }
  if (message.rejections.length > 0) {
    parts.push(
      <span key="rejections" title={message.rejections.join("\n")} className={inboxRejectionClass(message.event)}>
        {message.rejections.join(", ")}
      </span>
    );
  }
  if (message.url) {
    parts.push(
      <ExternalLink
        key="url"
        href={message.url}
        className="inline-flex items-center gap-0.5 text-blue-600 dark:text-blue-400 hover:underline"
      >
        {t("inbox.viewReleaseNotes")}
        <ArrowTopRightOnSquareIcon className="h-3 w-3" aria-hidden="true" />
      </ExternalLink>
    );
  }

  return (
    <div className="flex gap-x-3 min-w-0 text-xs text-gray-900 dark:text-gray-300">
      <InboxAge createdAt={message.created_at} className={classNames("sm:hidden shrink-0", label)} />
      {parts.length > 0 && (
        <div className="truncate">
          {/* The trailing space keeps screen readers from running the parts together. */}
          {parts.map((part, idx) => (
            <span key={idx} className="mr-2 last:mr-0">{part}{" "}</span>
          ))}
        </div>
      )}
    </div>
  );
};

interface EmptyInboxProps {
  colSpan: number;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  title: string;
  description: string;
  action: ReactNode;
}

const EmptyInbox = ({ colSpan, icon: Icon, title, description, action }: EmptyInboxProps) => (
  <tr>
    <td colSpan={colSpan}>
      <div className="flex flex-col items-center px-4 py-16 text-center">
        <span className="flex size-12 items-center justify-center rounded-full bg-gray-100 dark:bg-gray-750 text-gray-500 dark:text-gray-400">
          <Icon className="h-6 w-6" aria-hidden="true" />
        </span>
        <h3 className="mt-4 text-sm font-semibold text-gray-900 dark:text-white">{title}</h3>
        <p className="mt-1 max-w-sm text-sm text-gray-500 dark:text-gray-400">{description}</p>
        <div className="mt-4">{action}</div>
      </div>
    </td>
  </tr>
);

const InboxSkeleton = () => (
  <>
    {Array.from({ length: 5 }, (_, idx) => (
      <tr key={idx} className="animate-pulse" aria-hidden="true">
        <td className="w-px pl-3 pr-2 sm:pl-5 sm:pr-3"><span className="block h-4 w-4 rounded bg-gray-200 dark:bg-gray-750" /></td>
        <td className="hidden sm:table-cell px-3"><span className="block h-3 w-16 rounded bg-gray-200 dark:bg-gray-750" /></td>
        <td className="px-2 py-3 sm:px-3">
          <div className="space-y-2">
            <div className="h-3 w-2/3 rounded bg-gray-200 dark:bg-gray-750" />
            <div className="h-2.5 w-1/3 rounded bg-gray-200 dark:bg-gray-750" />
          </div>
        </td>
        <td className="hidden sm:table-cell px-3"><span className="block h-3 w-24 rounded bg-gray-200 dark:bg-gray-750" /></td>
        <td className="pl-1 pr-3 sm:pl-3 sm:pr-5" />
      </tr>
    ))}
  </>
);
