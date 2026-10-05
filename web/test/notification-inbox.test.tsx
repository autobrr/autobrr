/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createMemoryHistory, createRootRouteWithContext, createRoute, createRouter, RouterProvider } from "@tanstack/react-router";
import { act, cleanup, fireEvent, render, renderHook, screen, within } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import { NotificationUpdateForm } from "@forms/settings/NotificationForms";
import { NotificationInbox } from "@screens/NotificationInbox";
import { InboxMenu } from "@components/header/InboxMenu";
import { useInboxEvents } from "@hooks/useInbox";
import { AlertKeys, NotificationKeys } from "@api/query_keys";
import { SettingsContext } from "@utils/Context";
import i18n from "@app/i18n";
import { NotificationInboxRoute } from "@app/routes";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

const emptyFields = {
  message: "",
  release_name: "",
  indexer: "",
  filter_name: "",
  filter_id: 0,
  action: "",
  action_client: "",
  rejections: [],
  url: ""
};

const messages: InboxMessage[] = [
  {
    ...emptyFields,
    id: 2,
    event: "PUSH_ERROR",
    title: "Push Error",
    release_name: "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
    indexer: "MockIndexer",
    filter_name: "TV",
    filter_id: 4,
    action: "Send to Sonarr",
    action_client: "Sonarr",
    rejections: ["error pushing to client"],
    read_at: null,
    created_at: new Date().toISOString()
  },
  {
    ...emptyFields,
    id: 1,
    event: "IRC_DISCONNECTED",
    title: "IRC Disconnected",
    message: "P2P-Network",
    read_at: new Date().toISOString(),
    created_at: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString()
  }
];

const renderInbox = (path = "/notifications", component: () => ReactNode = NotificationInbox) => {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const root = createRootRouteWithContext<{ queryClient: QueryClient }>()();
  const auth = createRoute({ getParentRoute: () => root, id: "auth" });
  const authenticated = createRoute({ getParentRoute: () => auth, id: "authenticated-routes" });
  const inbox = createRoute({
    getParentRoute: () => authenticated,
    path: "notifications",
    component,
    validateSearch: NotificationInboxRoute.options.validateSearch
  });
  const router = createRouter({
    routeTree: root.addChildren([auth.addChildren([authenticated.addChildren([inbox])])]),
    history: createMemoryHistory({ initialEntries: [path] }),
    context: { queryClient }
  });

  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  );
};

const response = (overrides: Partial<InboxResponse> = {}): InboxResponse => ({
  data: messages,
  count: 2,
  all_count: 2,
  unread_count: 1,
  ...overrides
});

test("inbox shows tab counts and asks for unread messages", async () => {
  const list = vi.spyOn(APIClient.notifications.inbox, "list").mockResolvedValue(response());

  renderInbox();

  const unreadTab = await screen.findByRole("button", { name: /^Unread\s*1$/ });
  expect(screen.getByRole("button", { name: /^All\s*2$/ }).getAttribute("aria-pressed")).toBe("true");
  expect(list).toHaveBeenLastCalledWith({ limit: 25, offset: 0, unread: false, event: undefined });

  list.mockResolvedValue(response({ data: [], count: 0, unread_count: 0 }));
  await act(async () => {
    fireEvent.click(unreadTab);
  });

  expect(list).toHaveBeenLastCalledWith({ limit: 25, offset: 0, unread: true, event: undefined });
  expect(await screen.findByText("Show all notifications")).toBeTruthy();
});

test("a failed load shows an error instead of an empty inbox and can be retried", async () => {
  const list = vi.spyOn(APIClient.notifications.inbox, "list").mockRejectedValue(new Error("boom"));

  renderInbox("/notifications?unread=true");

  expect(await screen.findByText("Couldn't load notifications")).toBeTruthy();
  expect(screen.queryByText("You're all caught up")).toBeNull();

  list.mockResolvedValue(response());
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
  });

  expect(await screen.findByText("P2P-Network")).toBeTruthy();
  expect(screen.queryByText("Couldn't load notifications")).toBeNull();
});

test("selected messages can be marked read and deleted in bulk", async () => {
  vi.spyOn(APIClient.notifications.inbox, "list").mockResolvedValue(response());
  const markRead = vi.spyOn(APIClient.notifications.inbox, "markRead").mockResolvedValue(undefined as never);
  const remove = vi.spyOn(APIClient.notifications.inbox, "delete").mockResolvedValue(undefined as never);

  renderInbox();

  await screen.findByText("P2P-Network");
  fireEvent.click(screen.getByLabelText("Select P2P-Network"));
  expect(screen.getByText("1 of 2 selected")).toBeTruthy();

  fireEvent.click(screen.getByLabelText("Select all on this page"));
  expect(screen.getByText("2 of 2 selected")).toBeTruthy();

  fireEvent.click(screen.getAllByRole("button", { name: "Mark as read" })[0]);
  const markDialog = await screen.findByRole("dialog");
  expect(within(markDialog).getByText("Mark 2 selected notifications as read?")).toBeTruthy();
  expect(markRead).not.toHaveBeenCalled();

  await act(async () => {
    fireEvent.click(within(markDialog).getByRole("button", { name: "Mark as read" }));
  });
  expect(markRead.mock.calls[0][0]).toEqual([2, 1]);
  expect(await screen.findByRole("button", { name: /^All\s*2$/ })).toBeTruthy();

  fireEvent.click(screen.getByLabelText("Select P2P-Network"));
  fireEvent.click(screen.getAllByRole("button", { name: "Delete" })[0]);
  const deleteDialog = await screen.findByRole("dialog");
  expect(within(deleteDialog).getByText("Delete this notification? This cannot be undone.")).toBeTruthy();

  await act(async () => {
    fireEvent.click(within(deleteDialog).getByRole("button", { name: "Delete" }));
  });
  expect(remove.mock.calls[0][0]).toEqual([1]);
});

test("row actions do nothing when the confirmation is cancelled", async () => {
  vi.spyOn(APIClient.notifications.inbox, "list").mockResolvedValue(response());
  const markRead = vi.spyOn(APIClient.notifications.inbox, "markRead").mockResolvedValue(undefined as never);
  const remove = vi.spyOn(APIClient.notifications.inbox, "delete").mockResolvedValue(undefined as never);

  renderInbox();

  await screen.findByText("P2P-Network");
  const rows = screen.getAllByRole("listitem");

  fireEvent.click(within(rows[0]).getByRole("button", { name: "Mark as read" }));
  expect(within(await screen.findByRole("dialog")).getByText("Mark this notification as read?")).toBeTruthy();
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  });

  fireEvent.click(within(rows[1]).getByRole("button", { name: "Delete" }));
  await act(async () => {
    fireEvent.click(within(await screen.findByRole("dialog")).getByRole("button", { name: "Cancel" }));
  });

  expect(markRead).not.toHaveBeenCalled();
  expect(remove).not.toHaveBeenCalled();
});

test("mark all as read and clear all ask first and close after confirming", async () => {
  vi.spyOn(APIClient.notifications.inbox, "list").mockResolvedValue(response());
  const markRead = vi.spyOn(APIClient.notifications.inbox, "markRead").mockResolvedValue(undefined as never);
  const deleteAll = vi.spyOn(APIClient.notifications.inbox, "deleteAll").mockResolvedValue(undefined as never);

  renderInbox();

  await screen.findByText("P2P-Network");
  fireEvent.click(screen.getByRole("button", { name: "Mark all as read" }));
  const markDialog = await screen.findByRole("dialog");
  expect(within(markDialog).getByText("Every unread notification will be marked as read.")).toBeTruthy();
  await act(async () => {
    fireEvent.click(within(markDialog).getByRole("button", { name: "Mark all as read" }));
  });
  expect(markRead.mock.calls[0][0]).toEqual([]);

  fireEvent.click(screen.getByRole("button", { name: "More actions" }));
  fireEvent.click(await screen.findByRole("menuitem", { name: "Clear all" }));
  const clearDialog = await screen.findByRole("dialog");
  expect(within(clearDialog).getByText("Clear notifications")).toBeTruthy();
  await act(async () => {
    fireEvent.click(within(clearDialog).getByRole("button", { name: "Clear all" }));
  });

  expect(deleteAll).toHaveBeenCalledTimes(1);
  await vi.waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
});

test("shift-click selects and clears a range of messages", async () => {
  const rows: InboxMessage[] = ["Alpha", "Bravo", "Charlie", "Delta", "Echo"].map((name, idx) => ({
    ...emptyFields,
    id: 10 - idx,
    event: "IRC_DISCONNECTED",
    title: "IRC Disconnected",
    message: name,
    read_at: null,
    created_at: new Date().toISOString()
  }));
  vi.spyOn(APIClient.notifications.inbox, "list").mockResolvedValue(response({ data: rows, count: 5, all_count: 5, unread_count: 5 }));

  renderInbox();

  const checkbox = async (name: string) => await screen.findByLabelText(`Select ${name}`) as HTMLInputElement;
  const checked = async () => Promise.all(["Alpha", "Bravo", "Charlie", "Delta", "Echo"].map(async (name) => (await checkbox(name)).checked));

  fireEvent.click(await checkbox("Bravo"));
  fireEvent.click(await checkbox("Delta"), { shiftKey: true });
  expect(await checked()).toEqual([false, true, true, true, false]);
  expect(screen.getByText("3 of 5 selected")).toBeTruthy();

  fireEvent.click(await checkbox("Echo"), { shiftKey: true });
  expect(await checked()).toEqual([false, true, true, true, true]);

  fireEvent.click(await checkbox("Charlie"), { shiftKey: true });
  expect(await checked()).toEqual([false, true, false, false, false]);
});

test("rows link the release and filter", async () => {
  vi.spyOn(APIClient.notifications.inbox, "list").mockResolvedValue(response());

  renderInbox();

  const release = (await screen.findByText("Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP")).closest("a");
  expect(release?.getAttribute("href")).toBe("/releases?q=Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP");
  expect(screen.getByText("TV").closest("a")?.getAttribute("href")).toBe("/filters/4");
  expect(screen.getByText("Send to Sonarr (Sonarr)")).toBeTruthy();
  expect(screen.getByText("error pushing to client")).toBeTruthy();
  expect(screen.getByText("IRC Disconnected")).toBeTruthy();
});

test("pagination only shows when there is more than one page", async () => {
  const list = vi.spyOn(APIClient.notifications.inbox, "list").mockResolvedValue(response());

  renderInbox();

  await screen.findByText("P2P-Network");
  expect(screen.queryByRole("navigation", { name: "Pagination" })).toBeNull();

  cleanup();
  list.mockResolvedValue(response({ count: 60, all_count: 60 }));
  renderInbox();

  const nav = await screen.findByRole("navigation", { name: "Pagination" });
  expect(nav.textContent).toContain("123");

  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "2" }));
  });
  expect(list).toHaveBeenLastCalledWith({ limit: 25, offset: 25, unread: false, event: undefined });
});

test("a page past the end redirects to the last page without flashing the empty state", async () => {
  let resolveLastPage: (value: InboxResponse) => void = () => {};
  const list = vi.spyOn(APIClient.notifications.inbox, "list").mockImplementation(({ offset }) => offset === 25
    ? new Promise((resolve) => { resolveLastPage = resolve; })
    : Promise.resolve(response({ data: [], count: 30, all_count: 30 })));

  // The empty state would only render for a frame before the redirect, so record every node that is ever added.
  let emptyStateRendered = false;
  const observer = new MutationObserver((records) => {
    emptyStateRendered ||= records.some((record) => [...record.addedNodes].some((node) => node.textContent?.includes("No notifications")));
  });
  observer.observe(document.body, { childList: true, subtree: true });

  renderInbox("/notifications?page=5");

  await vi.waitFor(() => expect(list).toHaveBeenLastCalledWith({ limit: 25, offset: 25, unread: false, event: undefined }));
  expect(list).toHaveBeenCalledWith({ limit: 25, offset: 125, unread: false, event: undefined });

  await act(async () => {
    resolveLastPage(response({ count: 30, all_count: 30 }));
  });
  expect(await screen.findByText("P2P-Network")).toBeTruthy();
  expect(screen.getByRole("button", { name: "2" }).getAttribute("aria-current")).toBe("page");

  observer.disconnect();
  expect(emptyStateRendered).toBe(false);
});

test.each([
  ["pageSize=0", { limit: 25, offset: 0 }],
  ["pageSize=7", { limit: 25, offset: 0 }],
  ["pageSize=50&page=-1", { limit: 50, offset: 0 }],
  ["pageSize=50&page=1.5", { limit: 50, offset: 0 }],
  ["pageSize=50&page=1", { limit: 50, offset: 50 }]
])("pagination search %s falls back to supported values", async (search, params) => {
  const list = vi.spyOn(APIClient.notifications.inbox, "list").mockResolvedValue(response({ count: 120, all_count: 120 }));

  renderInbox(`/notifications?${search}`);

  const nav = await screen.findByRole("navigation", { name: "Pagination" });
  expect(list).toHaveBeenLastCalledWith({ ...params, unread: false, event: undefined });
  expect(nav.textContent).not.toContain("Infinity");
  expect((screen.getByRole("combobox", { name: "Items Per Page" }) as HTMLSelectElement).value).toBe(String(params.limit));
});

test("mark all as read mentions hidden messages when an event filter is active", async () => {
  vi.spyOn(APIClient.notifications.inbox, "list").mockResolvedValue(response());

  renderInbox("/notifications?event=%22PUSH_ERROR%22");

  await screen.findByText("P2P-Network");
  expect(screen.getByText("Event type: Push Error")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Mark all as read" }));
  expect(within(await screen.findByRole("dialog")).getByText(/including ones hidden by the current filter/)).toBeTruthy();
});

test("the bell menu closes before asking to mark messages read", async () => {
  vi.spyOn(APIClient.notifications.inbox, "list").mockResolvedValue(response());
  const markRead = vi.spyOn(APIClient.notifications.inbox, "markRead").mockResolvedValue(undefined as never);

  renderInbox("/notifications", InboxMenu);

  fireEvent.click(await screen.findByTitle("Notifications"));
  fireEvent.click(await screen.findByRole("button", { name: "Mark all as read" }));

  const dialog = await screen.findByRole("dialog");
  await vi.waitFor(() => expect(screen.queryByText("View all notifications")).toBeNull());
  expect(markRead).not.toHaveBeenCalled();

  await act(async () => {
    fireEvent.click(within(dialog).getByRole("button", { name: "Mark all as read" }));
  });
  expect(markRead.mock.calls[0][0]).toEqual([]);
});

test("the built-in notification can not change type or be removed", () => {
  const builtin: ServiceNotification = {
    id: 1,
    name: "Built-in",
    enabled: true,
    type: "BUILTIN",
    events: ["PUSH_ERROR"]
  };

  render(
    <QueryClientProvider client={new QueryClient()}>
      <NotificationUpdateForm isOpen={true} toggle={() => {}} data={builtin} />
    </QueryClientProvider>
  );

  expect(screen.queryByText("Type")).toBeNull();
  expect(screen.queryByText("Remove")).toBeNull();
  expect(screen.getByText("Browser notifications")).toBeTruthy();
  expect(screen.queryByText("New Release")).toBeNull();
});

test("the inbox and alerts refetch every time the event stream connects", async () => {
  vi.useFakeTimers();
  vi.stubGlobal("EventSource", { CONNECTING: 0, OPEN: 1, CLOSED: 2 });
  const streams: EventSource[] = [];
  vi.spyOn(APIClient.events, "notifications").mockImplementation(() => {
    const es = { readyState: 1, addEventListener: vi.fn(), close: vi.fn() } as unknown as EventSource;
    streams.push(es);
    return es;
  });

  const queryClient = new QueryClient();
  const invalidate = vi.spyOn(queryClient, "invalidateQueries");
  const { unmount } = renderHook(() => useInboxEvents(), {
    wrapper: ({ children }) => <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  });

  streams[0].onopen?.(new Event("open"));
  expect(invalidate).toHaveBeenCalledTimes(2);
  expect(invalidate).toHaveBeenCalledWith({ queryKey: NotificationKeys.inbox.all() });
  expect(invalidate).toHaveBeenCalledWith({ queryKey: AlertKeys.all });

  // The browser retries a dropped connection on the same EventSource.
  streams[0].onopen?.(new Event("open"));
  expect(invalidate).toHaveBeenCalledTimes(4);

  // A failed handshake closes the stream and our backoff opens a new one.
  Object.assign(streams[0], { readyState: 2 });
  streams[0].onerror?.(new Event("error"));
  act(() => vi.advanceTimersByTime(1000));
  expect(streams).toHaveLength(2);

  streams[1].onopen?.(new Event("open"));
  expect(invalidate).toHaveBeenCalledTimes(6);

  unmount();
});

test("browser notifications for push events show the translated title, release, filter, action and rejections", async () => {
  vi.stubGlobal("EventSource", { CONNECTING: 0, OPEN: 1, CLOSED: 2 });
  const notification = vi.fn();
  vi.stubGlobal("Notification", Object.assign(notification, { permission: "granted" }));
  vi.spyOn(SettingsContext, "get").mockReturnValue({ ...SettingsContext.get(), browserNotifications: true });

  const listeners: Record<string, (event: MessageEvent) => void> = {};
  vi.spyOn(APIClient.events, "notifications").mockImplementation(() => ({
    readyState: 1,
    addEventListener: (type: string, listener: (event: MessageEvent) => void) => {
      listeners[type] = listener;
    },
    close: vi.fn()
  }) as unknown as EventSource);

  const queryClient = new QueryClient();
  const { unmount } = renderHook(() => useInboxEvents(), {
    wrapper: ({ children }) => <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  });

  listeners.NOTIFICATION(new MessageEvent("NOTIFICATION", { data: JSON.stringify(messages[0]) }));

  expect(notification).toHaveBeenCalledWith("Push Error", {
    body: "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP\nTV / Send to Sonarr\nerror pushing to client",
    tag: "autobrr-inbox-2"
  });

  await i18n.changeLanguage("de");
  try {
    listeners.NOTIFICATION(new MessageEvent("NOTIFICATION", { data: JSON.stringify(messages[0]) }));
    expect(notification).toHaveBeenLastCalledWith("Push Fehler", expect.anything());
  } finally {
    await i18n.changeLanguage("en");
    unmount();
  }
});
