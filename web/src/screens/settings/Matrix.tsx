/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment, MouseEvent, useCallback, useEffect, useRef } from "react";
import { useMutation, useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { PlusIcon } from "@heroicons/react/24/solid";
import { Menu, MenuButton, MenuItem, MenuItems, Transition } from "@headlessui/react";
import {
  ArrowPathIcon,
  ArrowRightEndOnRectangleIcon,
  ArrowsPointingInIcon,
  ArrowsPointingOutIcon,
  ArrowRightStartOnRectangleIcon,
  EllipsisHorizontalIcon,
  ExclamationCircleIcon,
  PencilSquareIcon,
  TrashIcon
} from "@heroicons/react/24/outline";
import { useTranslation } from "react-i18next";

import { classNames, simplifyDate } from "@utils";
import { MatrixLoginForm, MatrixNetworkAddForm, MatrixNetworkUpdateForm } from "@forms";
import { useToggle } from "@hooks/hooks";
import { APIClient } from "@api/APIClient";
import { MatrixKeys } from "@api/query_keys";
import { MatrixQueryOptions } from "@api/queries";
import { EmptySimple } from "@components/emptystates";
import { DeleteModal } from "@components/modals";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { Checkbox } from "@components/Checkbox";
import { RingResizeSpinner } from "@components/Icons";
import { SettingsContext } from "@utils/Context";
import { useMatrixHealthSync, useMatrixRoomWithHistory } from "@hooks/useMatrixEvents";

import { Section } from "./_components";

function MatrixSettings() {
  const { t } = useTranslation("settings");
  const [addNetworkIsOpen, toggleAddNetwork] = useToggle(false);
  const queryClient = useQueryClient();

  const matrixQuery = useSuspenseQuery(MatrixQueryOptions());
  const networks = matrixQuery.data;

  // The stream carries no login state, so a pending login from the last poll is kept.
  const onHealth = useCallback((event: MatrixHealthEvent) => {
    queryClient.setQueryData<MatrixNetwork[]>(MatrixKeys.lists(), (current) =>
      current?.map((network) => network.id !== event.network ? network : {
        ...network,
        health: { ...event.health, login: network.health?.login },
        rooms: network.rooms.map((room) => ({ ...room, health: event.rooms[room.name] ?? room.health }))
      })
    );
  }, [queryClient]);

  useMatrixHealthSync(onHealth);

  return (
    <Section
      title={t("matrix.listTitle")}
      description={t("matrix.listDescription")}
      rightSide={
        <button
          type="button"
          onClick={toggleAddNetwork}
          className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer"
        >
          <PlusIcon className="h-5 w-5 mr-1" />
          {t("matrix.addNew")}
        </button>
      }
    >
      <MatrixNetworkAddForm isOpen={addNetworkIsOpen} toggle={toggleAddNetwork} />

      {networks.length > 0 ? (
        <ul className="mt-6 min-w-full relative text-sm">
          <li className="grid grid-cols-12 gap-4 border-b border-gray-200 dark:border-gray-700 text-xs font-medium text-gray-500 dark:text-gray-400">
            <div className="col-span-2 md:col-span-1 pl-2 sm:px-3 py-3 text-left uppercase tracking-wider">
              {t("matrix.enabled")}
            </div>
            <div className="col-span-9 md:col-span-3 px-11 py-3 text-left uppercase tracking-wider">
              {t("matrix.network")}
            </div>
            <div className="hidden md:flex col-span-4 px-6 py-3 text-left uppercase tracking-wider">
              {t("matrix.homeserver")}
            </div>
            <div className="hidden md:flex col-span-3 px-5 py-3 text-left uppercase tracking-wider">
              {t("matrix.user")}
            </div>
          </li>
          {networks.map((network) => (
            <ListItem key={network.id} network={network} />
          ))}
        </ul>
      ) : (
        <EmptySimple
          title={t("matrix.noNetworks")}
          subtitle={t("matrix.noNetworksSubtitle")}
          buttonText={t("matrix.addNewNetwork")}
          buttonAction={toggleAddNetwork}
        />
      )}
    </Section>
  );
}

interface ListItemProps {
  network: MatrixNetwork;
}

function ListItem({ network }: ListItemProps) {
  const { t } = useTranslation("settings");
  const [updateIsOpen, toggleUpdate] = useToggle(false);
  const [loginIsOpen, toggleLogin] = useToggle(false);
  const [expanded, toggleExpanded] = useToggle(false);

  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: (network: MatrixNetwork) => APIClient.matrix.update(network),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: MatrixKeys.lists() });

      toast.custom((toastInstance) => (
        <Toast
          type="success"
          body={t("matrix.toggleSuccess", {
            name: network.name,
            state: variables.enabled ? t("matrix.enabledState") : t("matrix.disabledState")
          })}
          t={toastInstance}
        />
      ));
    }
  });

  const onToggleMutation = (newState: boolean) => {
    updateMutation.mutate({ ...network, enabled: newState });
  };

  const health = network.health;
  const loggedIn = network.user_id !== "";
  const needsLogin = !loggedIn || health?.login_required === true;

  // Drawers do not stack, so the edit panel hands over to the login panel.
  const loginFromEdit = () => {
    toggleUpdate();
    toggleLogin();
  };

  return (
    <li>
      <MatrixNetworkUpdateForm isOpen={updateIsOpen} toggle={toggleUpdate} data={network} onLogin={loginFromEdit} />
      <MatrixLoginForm isOpen={loginIsOpen} toggle={toggleLogin} data={network} />

      <div
        className={classNames(
          "grid grid-cols-12 gap-2 lg:gap-4 items-center mt-1 p-2.5 rounded-md transition cursor-pointer",
          network.enabled && !health?.healthy
            ? "bg-red-200 hover:bg-red-275 dark:bg-red-900 dark:hover:bg-red-800"
            : "hover:bg-gray-200 dark:hover:bg-gray-700"
        )}
        onClick={(e) => {
          if (e.defaultPrevented) {
            return;
          }

          toggleExpanded();
        }}
      >
        <div className="col-span-2 md:col-span-1 flex pl-1 sm:pl-2.5 text-gray-500 dark:text-gray-400">
          <Checkbox name="enabled" value={network.enabled} setValue={onToggleMutation} />
        </div>
        <div className="col-span-9 md:col-span-3 items-center pl-8 font-medium text-gray-900 dark:text-white">
          <div className="flex items-center">
            <NetworkStatus network={network} />
            <div className="block text-sm truncate">{network.name}</div>
          </div>
          {network.enabled && health?.error && (
            <div className="ml-6 text-xs text-red-600 dark:text-red-400 truncate" title={health.error}>
              {health.error}
            </div>
          )}
          {needsLogin && (
            <div className="md:hidden ml-6 mt-1">
              <LoginButton onClick={toggleLogin} />
            </div>
          )}
        </div>
        <div className="hidden md:flex col-span-4 md:pl-6 text-gray-500 dark:text-gray-400">
          <p className="block text-sm truncate">{network.homeserver}</p>
        </div>
        <div className="hidden md:flex col-span-3 md:pl-6 items-center gap-3 text-gray-500 dark:text-gray-400">
          {loggedIn && <p className="block text-sm truncate">{network.user_id}</p>}
          {needsLogin && <LoginButton onClick={toggleLogin} />}
        </div>
        <div className="col-span-1 text-gray-500 dark:text-gray-400">
          <ListItemDropdown network={network} toggleUpdate={toggleUpdate} toggleLogin={toggleLogin} />
        </div>
      </div>

      {expanded && (
        <div className="px-4 py-4 flex bg-gray-100 dark:bg-gray-775 rounded-b-md">
          <div className="min-w-full">
            {network.rooms.length > 0 ? (
              <ul>
                <li className="grid grid-cols-12 gap-4 text-xs font-medium text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-700">
                  <div className="col-span-5 sm:px-6 py-3 text-left uppercase tracking-wider truncate">
                    {t("matrix.room")}
                  </div>
                  <div className="hidden sm:flex col-span-3 sm:px-6 py-3 text-left uppercase tracking-wider truncate">
                    {t("matrix.joinedSince")}
                  </div>
                  <div className="col-span-6 sm:col-span-3 sm:px-6 py-3 text-left uppercase tracking-wider truncate">
                    {t("matrix.lastAnnounce")}
                  </div>
                </li>
                {network.rooms.map((room) => (
                  <RoomItem key={room.id} network={network} room={room} />
                ))}
              </ul>
            ) : (
              <div className="flex text-center justify-center py-4 dark:text-gray-500">
                <p>{t("matrix.noRooms")}</p>
              </div>
            )}
          </div>
        </div>
      )}
    </li>
  );
}

interface LoginButtonProps {
  onClick: () => void;
}

const LoginButton = ({ onClick }: LoginButtonProps) => {
  const { t } = useTranslation("settings");

  return (
    <button
      type="button"
      onClick={(e) => {
        // the row toggles its rooms on click
        e.preventDefault();
        onClick();
      }}
      className="shrink-0 inline-flex items-center px-2.5 py-1 border border-transparent shadow-xs text-xs font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer"
    >
      {t("matrix.login")}
    </button>
  );
};

interface NetworkStatusProps {
  network: MatrixNetwork;
}

const NetworkStatus = ({ network }: NetworkStatusProps) => {
  const { t } = useTranslation("settings");

  if (!network.enabled) {
    return <span className="mr-3 flex h-3 w-3 rounded-full opacity-75 bg-gray-500" title={t("matrix.networkDisabled")} />;
  }

  if (network.health?.healthy) {
    return (
      <span
        className="mr-3 flex h-3 w-3 relative"
        title={t("matrix.connectedSince", { time: simplifyDate(network.health.connected_since) })}
      >
        <span className="animate-ping inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
        <span className="inline-flex absolute rounded-full h-3 w-3 bg-green-500" />
      </span>
    );
  }

  return (
    <span className="mr-3 flex items-center" title={network.health?.error ?? t("matrix.networkUnhealthy")}>
      <ExclamationCircleIcon className="h-4 w-4 text-yellow-400 hover:text-yellow-600" />
    </span>
  );
};

interface RoomItemProps {
  network: MatrixNetwork;
  room: MatrixRoom;
}

const RoomItem = ({ network, room }: RoomItemProps) => {
  const { t } = useTranslation("settings");
  const [viewRoom, toggleView] = useToggle(false);
  const joined = network.enabled && room.enabled && room.health?.joined;

  return (
    <li
      className={classNames(
        "mt-1 mb-2 text-gray-500 dark:text-gray-400 rounded-md",
        viewRoom ? "bg-gray-200 dark:bg-gray-800" : "hover:bg-gray-300 dark:hover:bg-gray-800"
      )}
    >
      <div className="grid grid-cols-12 gap-4 items-center py-3 cursor-pointer" onClick={toggleView}>
        <div className="col-span-5 flex flex-col md:px-6 pl-2 sm:pl-0">
          <span className="relative inline-flex items-center">
            <span
              className={classNames(
                "mr-3 flex h-3 w-3 rounded-full",
                joined ? "bg-green-500" : room.enabled && network.enabled ? "bg-yellow-400" : "bg-gray-500"
              )}
            />
            <span className="truncate" title={room.room_id}>{room.name}</span>
          </span>
          {room.health?.error && (
            <span className="ml-6 text-xs text-red-500 truncate" title={room.health.error}>
              {room.health.error}
            </span>
          )}
        </div>
        <div className="hidden sm:flex col-span-3 md:px-6" title={room.health?.joined_since}>
          {simplifyDate(room.health?.joined_since)}
        </div>
        <div className="col-span-6 sm:col-span-3 md:px-6" title={room.health?.last_announce}>
          {simplifyDate(room.health?.last_announce)}
        </div>
        <div className="col-span-1 flex items-center justify-end">
          <button type="button" className="hover:text-gray-500 px-2 mx-2 py-1 dark:bg-gray-800 rounded-sm cursor-pointer">
            {viewRoom ? t("matrix.hide") : t("matrix.view")}
          </button>
        </div>
      </div>
      {viewRoom && <RoomEvents key={`${network.id}:${room.name}`} networkId={network.id} room={room.name} />}
    </li>
  );
};

interface RoomEventsProps {
  networkId: number;
  room: string;
}

const RoomEvents = ({ networkId, room }: RoomEventsProps) => {
  const { t } = useTranslation("settings");
  const scrollOnNewLog = SettingsContext.useSelector((s) => s.scrollOnNewLog);
  const { messages } = useMatrixRoomWithHistory(networkId, room, 1000);
  const [isFullscreen, toggleFullscreen] = useToggle(false);
  const logRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (scrollOnNewLog && logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [messages, scrollOnNewLog]);

  useEffect(() => {
    if (!isFullscreen) {
      return;
    }

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        toggleFullscreen();
      }
    };

    window.addEventListener("keydown", onKeyDown);
    document.body.classList.add("overflow-hidden");

    return () => {
      window.removeEventListener("keydown", onKeyDown);
      document.body.classList.remove("overflow-hidden");
    };
  }, [isFullscreen, toggleFullscreen]);

  return (
    <div
      className={classNames(
        "dark:bg-gray-800 rounded-lg shadow-lg p-2",
        isFullscreen ? "fixed top-0 left-0 right-0 bottom-0 w-screen h-screen z-50" : ""
      )}
    >
      <div className="flex relative">
        <button
          type="button"
          className="p-2 absolute top-2 right-2 mr-2 bg-gray-200 dark:bg-gray-800 hover:bg-gray-300 dark:hover:bg-gray-700 rounded-md cursor-pointer"
          onClick={toggleFullscreen}
        >
          {isFullscreen ? <ArrowsPointingInIcon className="w-5 h-5" /> : <ArrowsPointingOutIcon className="w-5 h-5" />}
        </button>
      </div>
      <div
        ref={logRef}
        className={classNames(
          "overflow-auto rounded-lg min-w-full bg-gray-100 dark:bg-gray-900",
          isFullscreen ? "max-w-full h-full p-2" : "px-2 py-1 aspect-2/1"
        )}
      >
        {messages.length === 0 && (
          <p className="py-4 text-center text-sm text-gray-500 dark:text-gray-500">{t("matrix.noMessages")}</p>
        )}
        {messages.map((message, idx) => (
          <div key={message.id || `${message.time}-${idx}`} className="flex items-center hover:bg-gray-200 dark:hover:bg-gray-800">
            <ReprocessAnnounceButton networkId={networkId} room={room} msg={message.msg} />
            <div className="flex-1">
              <span className="font-mono text-gray-500 dark:text-gray-500">
                <span className="dark:text-gray-600">
                  <span className="dark:text-gray-700">[{simplifyDate(message.time)}]</span> {message.sender}:
                </span> {message.msg}
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

interface ReprocessAnnounceButtonProps {
  networkId: number;
  room: string;
  msg: string;
}

const ReprocessAnnounceButton = ({ networkId, room, msg }: ReprocessAnnounceButtonProps) => {
  const { t } = useTranslation("settings");

  const reprocessMutation = useMutation({
    mutationFn: () => APIClient.matrix.reprocessAnnounce(networkId, room, msg),
    onSuccess: () => {
      toast.custom((toastInstance) => <Toast type="success" body={t("matrix.announceReprocessed")} t={toastInstance} />);
    },
    onError: () => {
      toast.custom((toastInstance) => <Toast type="error" body={t("matrix.announceReprocessFailed")} t={toastInstance} />);
    }
  });

  return (
    <button
      type="button"
      className="flex shrink-0 items-center justify-center size-5 mr-1 p-1 rounded-sm transition bg-gray-250 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 cursor-pointer disabled:cursor-not-allowed"
      onClick={() => reprocessMutation.mutate()}
      disabled={reprocessMutation.isPending}
      title={t("matrix.reprocessAnnounce")}
    >
      {reprocessMutation.isPending
        ? <RingResizeSpinner className="text-blue-500 iconHeight" aria-hidden="true" />
        : <ArrowPathIcon />}
    </button>
  );
};

interface ListItemDropdownProps {
  network: MatrixNetwork;
  toggleUpdate: () => void;
  toggleLogin: () => void;
}

const ListItemDropdown = ({ network, toggleUpdate, toggleLogin }: ListItemDropdownProps) => {
  const { t } = useTranslation("settings");
  const cancelModalButtonRef = useRef(null);
  const queryClient = useQueryClient();

  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);

  const deleteMutation = useMutation({
    mutationFn: (id: number) => APIClient.matrix.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MatrixKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("matrix.networkDeleted", { name: network.name })} t={toastInstance} />);
      toggleDeleteModal();
    }
  });

  const restartMutation = useMutation({
    mutationFn: (id: number) => APIClient.matrix.restart(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MatrixKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("matrix.networkRestarted", { name: network.name })} t={toastInstance} />);
    }
  });

  const logoutMutation = useMutation({
    mutationFn: (id: number) => APIClient.matrix.logout(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MatrixKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("matrix.loggedOut", { name: network.name })} t={toastInstance} />);
    }
  });

  const loggedIn = network.user_id !== "";

  return (
    <Menu
      as="div"
      onClick={(e: MouseEvent) => {
        e.preventDefault();
        e.stopPropagation();
      }}
    >
      <DeleteModal
        isOpen={deleteModalIsOpen}
        isLoading={deleteMutation.isPending}
        toggle={toggleDeleteModal}
        buttonRef={cancelModalButtonRef}
        deleteAction={() => deleteMutation.mutate(network.id)}
        title={t("matrix.removeNetworkTitle", { name: network.name })}
        text={t("matrix.removeNetworkText")}
      />
      <MenuButton className="px-4 py-2 cursor-pointer">
        <EllipsisHorizontalIcon
          className="w-5 h-5 text-gray-700 hover:text-gray-900 dark:text-gray-100 dark:hover:text-gray-400"
          aria-hidden="true"
        />
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
          className="absolute w-56 bg-white dark:bg-gray-825 divide-y divide-gray-200 dark:divide-gray-750 rounded-md shadow-lg border border-gray-250 dark:border-gray-750 focus:outline-hidden z-10"
        >
          <div className="px-1 py-1">
            <DropdownItem icon={PencilSquareIcon} label={t("matrix.edit")} onClick={toggleUpdate} />
            <DropdownItem icon={ArrowRightEndOnRectangleIcon} label={t("matrix.login")} onClick={toggleLogin} />
            <DropdownItem
              icon={ArrowPathIcon}
              label={t("matrix.restart")}
              onClick={() => restartMutation.mutate(network.id)}
              disabled={!network.enabled}
            />
            <DropdownItem
              icon={ArrowRightStartOnRectangleIcon}
              label={t("matrix.logout")}
              onClick={() => logoutMutation.mutate(network.id)}
              disabled={!loggedIn}
            />
          </div>
          <div className="px-1 py-1">
            <DropdownItem icon={TrashIcon} label={t("matrix.delete")} onClick={toggleDeleteModal} danger={true} />
          </div>
        </MenuItems>
      </Transition>
    </Menu>
  );
};

interface DropdownItemProps {
  icon: typeof TrashIcon;
  label: string;
  onClick: () => void;
  disabled?: boolean;
  danger?: boolean;
}

const DropdownItem = ({ icon: Icon, label, onClick, disabled, danger }: DropdownItemProps) => (
  <MenuItem disabled={disabled}>
    {({ active }) => (
      <button
        className={classNames(
          "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm cursor-pointer disabled:cursor-not-allowed",
          disabled
            ? "text-gray-600 dark:text-gray-500"
            : active
              ? danger ? "bg-red-600 text-white" : "bg-blue-600 text-white"
              : "text-gray-900 dark:text-gray-300"
        )}
        onClick={onClick}
        disabled={disabled}
      >
        <Icon
          className={classNames(
            "w-5 h-5 mr-2",
            disabled ? "text-gray-600 dark:text-gray-500" : active ? "text-white" : danger ? "text-red-500" : "text-blue-500"
          )}
          aria-hidden="true"
        />
        {label}
      </button>
    )}
  </MenuItem>
);

export default MatrixSettings;
