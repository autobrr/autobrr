import { useMutation, useQuery, useQueryClient } from "react-query";
import { classNames, IsEmptyDate, simplifyDate } from "../../utils";
import { IrcNetworkAddForm, IrcNetworkUpdateForm } from "../../forms";
import { useToggle } from "../../hooks/hooks";
import { APIClient } from "../../api/APIClient";
import { EmptySimple } from "../../components/emptystates";
import { LockClosedIcon, LockOpenIcon } from "@heroicons/react/24/solid";
import { Menu, Switch, Transition } from "@headlessui/react";
import { Fragment, useRef } from "react";
import { DeleteModal } from "../../components/modals";
import { useState, useMemo } from "react";
import { toast } from "react-hot-toast";
import Toast from "../../components/notifications/Toast";
import {
  ArrowsPointingInIcon,
  ArrowsPointingOutIcon,
  EllipsisHorizontalIcon,
  ExclamationCircleIcon,
  PencilSquareIcon,
  TrashIcon
} from "@heroicons/react/24/outline";

interface SortConfig {
  key: keyof ListItemProps["network"] | "enabled";
  direction: "ascending" | "descending";
}

function useSort(items: ListItemProps["network"][], config?: SortConfig) {
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

  const requestSort = (key: keyof ListItemProps["network"]) => {
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
  

  const getSortIndicator = (key: keyof ListItemProps["network"]) => {
    if (!sortConfig || sortConfig.key !== key) {
      return "";
    }
    
    return sortConfig.direction === "ascending" ? "↑" : "↓";
  };

  return { items: sortedItems, requestSort, sortConfig, getSortIndicator };
}

const IrcSettings = () => {
  const [expandNetworks, toggleExpand] = useToggle(false);
  const [addNetworkIsOpen, toggleAddNetwork] = useToggle(false);

  const { data } = useQuery("networks", () => APIClient.irc.getNetworks(), {
    refetchOnWindowFocus: false,
    // Refetch every 3 seconds
    refetchInterval: 3000
  });

  const sortedNetworks = useSort(data || []);

  return (
    <div className="lg:col-span-9">
      <IrcNetworkAddForm isOpen={addNetworkIsOpen} toggle={toggleAddNetwork} />

      <div className="py-6 px-4 md:p-6 lg:pb-8">
        <div className="-ml-4 -mt-4 flex justify-between items-center flex-wrap md:flex-nowrap">
          <div className="ml-4 mt-4">
            <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">
              IRC
            </h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              IRC networks and channels. Click on a network to view channel
              status.
            </p>
          </div>
          <div className="ml-4 mt-4 flex-shrink-0">
            <button
              type="button"
              onClick={toggleAddNetwork}
              className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              Add new
            </button>
          </div>
        </div>

        <div className="flex justify-between flex-col md:flex-row mt-10 px-1">
          <ol className="flex flex-col md:flex-row md:gap-2 pb-4 md:pb-0 md:divide-x md:divide-gray-200 md:dark:divide-gray-700">
            <li className="flex items-center">
              <span
                className="mr-2 flex h-4 w-4 relative"
                title="Network healthy"
              >
                <span className="animate-ping inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
                <span className="inline-flex absolute rounded-full h-4 w-4 bg-green-500" />
              </span>
              <span className="text-sm text-gray-800 dark:text-gray-500">Network healthy</span>
            </li>

            <li className="flex items-center md:pl-2">
              <span
                className="mr-2 flex h-4 w-4 rounded-full opacity-75 bg-yellow-400 over:text-yellow-600"
                title="Network unhealthy"
              />
              <span className="text-sm text-gray-800 dark:text-gray-500">Network unhealthy</span>
            </li>

            <li className="flex items-center md:pl-2">
              <span
                className="mr-2 flex h-4 w-4 rounded-full opacity-75 bg-gray-500"
                title="Network disabled"
              >
              </span>
              <span className="text-sm text-gray-800 dark:text-gray-500">Network disabled</span>
            </li>
          </ol>
          <div className="flex gap-x-2">
            <button className="flex items-center text-sm text-gray-800 dark:text-gray-400 p-1 px-2 rounded shadow bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600" onClick={toggleExpand} title={expandNetworks ? "collapse" : "expand"}>
              {expandNetworks
                ? <span className="flex items-center">Collapse <ArrowsPointingInIcon className="ml-1 w-4 h-4"/></span>
                : <span className="flex items-center">Expand <ArrowsPointingOutIcon className="ml-1 w-4 h-4"/></span>
              }</button>
          </div>
        </div>

        {data && data.length > 0 ? (
          <section className="mt-6 light:bg-white dark:bg-gray-800 light:shadow md:rounded-md">
            <ol className="min-w-full relative">
              <li className="grid grid-cols-12 gap-4 border-b border-gray-200 dark:border-gray-700">
                <div className="flex col-span-2 md:col-span-1 px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
                  onClick={() => sortedNetworks.requestSort("enabled")}>
                    Enabled <span className="sort-indicator">{sortedNetworks.getSortIndicator("enabled")}</span>
                </div>
                <div className="col-span-10 md:col-span-3 px-8 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
                  onClick={() => sortedNetworks.requestSort("name")}>
                  Network <span className="sort-indicator">{sortedNetworks.getSortIndicator("name")}</span>
                </div>
                <div className="hidden md:flex col-span-4 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
                  onClick={() => sortedNetworks.requestSort("server")}>
                  Server <span className="sort-indicator">{sortedNetworks.getSortIndicator("server")}</span>
                </div>
                <div className="hidden md:flex col-span-3 px-5 lg:px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"
                  onClick={() => sortedNetworks.requestSort("nick")}>
                Nick <span className="sort-indicator">{sortedNetworks.getSortIndicator("nick")}</span>
                </div>
              </li>
              {data &&
                sortedNetworks.items.map((network, idx) => (
                  <ListItem key={idx} idx={idx} expanded={expandNetworks} network={network} />
                ))}
            </ol>
          </section>
        ) : (
          <EmptySimple
            title="No networks"
            subtitle="Normally set up via Indexers"
            buttonText="Add new network"
            buttonAction={toggleAddNetwork}
          />
        )}
      </div>
    </div>
  );
};

interface ListItemProps {
  idx: number;
  network: IrcNetworkWithHealth;
  expanded: boolean;
}

const ListItem = ({ idx, network, expanded }: ListItemProps) => {
  const [updateIsOpen, toggleUpdate] = useToggle(false);
  const [edit, toggleEdit] = useToggle(false);

  const queryClient = useQueryClient();

  const mutation = useMutation(
    (network: IrcNetwork) => APIClient.irc.updateNetwork(network),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(["networks"]);
        toast.custom((t) => <Toast type="success" body={`${network.name} was updated successfully`} t={t}/>);
      }
    }
  );

  const onToggleMutation = (newState: boolean) => {
    mutation.mutate({
      ...network,
      enabled: newState
    });
  };

  return (
    <li key={idx}>
      <div className={classNames("grid grid-cols-12 gap-2 lg:gap-4 items-center py-2", network.enabled && !network.healthy ? "bg-red-50 dark:bg-red-900 hover:bg-red-100 dark:hover:bg-red-800" : "hover:bg-gray-50 dark:hover:bg-gray-700 ")}>
        <IrcNetworkUpdateForm
          isOpen={updateIsOpen}
          toggle={toggleUpdate}
          network={network}
        />
        <div className="col-span-2 md:col-span-1 flex pl-5 text-sm text-gray-500 dark:text-gray-400 cursor-pointer">
          <Switch
            checked={network.enabled}
            onChange={onToggleMutation}
            className={classNames(
              network.enabled ? "bg-blue-500" : "bg-gray-200 dark:bg-gray-600",
              "items-center relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            )}
          >
            <span className="sr-only">Enable</span>
            <span
              aria-hidden="true"
              className={classNames(
                network.enabled ? "translate-x-5" : "translate-x-0",
                "inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200"
              )}
            />
          </Switch>
        </div>
        <div
          className="col-span-8 xs:col-span-3 md:col-span-3 items-center pl-8 text-sm font-medium text-gray-900 dark:text-white cursor-pointer"
          onClick={toggleEdit}
        >
          <div className="flex">
            <span className="relative inline-flex items-center ml-1">
              {network.enabled ? (
                network.healthy ? (
                  <span
                    className="mr-3 flex h-3 w-3 relative"
                    title={`Connected since: ${simplifyDate(network.connected_since)}`}
                  >
                    <span className="animate-ping inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
                    <span className="inline-flex absolute rounded-full h-3 w-3 bg-green-500" />
                  </span>
                ) : (
                  <span
                    className="mr-3 flex items-center"
                    title={network.connection_errors.toString()}
                  >
                    <ExclamationCircleIcon className="h-4 w-4 text-yellow-400 hover:text-yellow-600" />
                  </span>
                )
              ) : (
                <span className="mr-3 flex h-3 w-3 rounded-full opacity-75 bg-gray-500" />
              )}
            </span>
            <div className="block truncate">
              {network.name}
            </div>
          </div>
        </div>
        <div
          className="hidden md:flex col-span-4 md:pl-6 text-sm text-gray-500 dark:text-gray-400 cursor-pointer"
          onClick={toggleEdit}
        >
          <div
            className="overflow-x-auto flex items-center"
            title={network.tls ? "Secured using TLS" : "Insecure, not using TLS"}
          >
            <div className="min-h-2 min-w-2">
              {network.tls ? (
                <LockClosedIcon
                  className={classNames(
                    "mr-2 h-4 w-4",
                    network.enabled ? "text-green-600" : "text-gray-500"
                  )}
                />
              ) : (
                <LockOpenIcon className={classNames(
                  "mr-2 h-4 w-4",
                  network.enabled ? "text-red-500" : "text-yellow-500"
                )} />
              )}
            </div>
            <p className="block truncate">
              {network.server}:{network.port}
            </p>
          </div>
        </div>
        <div
          className="hidden md:flex col-span-3 items-center md:pl-6 text-sm text-gray-500 dark:text-gray-400 cursor-pointer"
          onClick={toggleEdit}
        >
          <div className="block truncate">
            {network.nick}
          </div>
        </div>
        <div className="col-span-1 text-sm text-gray-500 dark:text-gray-400">
          <ListItemDropdown network={network} toggleUpdate={toggleUpdate} />
        </div>
      </div>
      {(edit || expanded) && (
        <div className="px-4 py-4 flex border-b border-x-0 dark:border-gray-600 dark:bg-gray-700">
          <div className="min-w-full">
            {network.channels.length > 0 ? (
              <ol>
                <li className="grid grid-cols-12 gap-4 border-b border-gray-200 dark:border-gray-700">
                  <div className="col-span-4 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Channel
                  </div>
                  <div className="col-span-4 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Monitoring since
                  </div>
                  <div className="col-span-4 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Last announce
                  </div>
                </li>
                {network.channels.map((c) => (
                  <li key={c.id} className="text-gray-500 dark:text-gray-400">
                    <div className="grid grid-cols-12 gap-4 items-center py-4">
                      <div className="col-span-4 flex items-center md:px-6 ">
                        <span className="relative inline-flex items-center">
                          {network.enabled ? (
                            c.monitoring ? (
                              <span
                                className="mr-3 flex h-3 w-3 relative"
                                title="monitoring"
                              >
                                <span className="animate-ping inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
                                <span className="inline-flex absolute rounded-full h-3 w-3 bg-green-500" />
                              </span>
                            ) : (
                              <span className="mr-3 flex h-3 w-3 rounded-full opacity-75 bg-red-400" />
                            )
                          ) : (
                            <span className="mr-3 flex h-3 w-3 rounded-full opacity-75 bg-gray-500" />
                          )}
                          {c.name}
                        </span>
                      </div>
                      <div className="col-span-4 flex items-center md:px-6 ">
                        <span title={simplifyDate(c.monitoring_since)}>
                          {IsEmptyDate(c.monitoring_since)}
                        </span>
                      </div>
                      <div className="col-span-4 flex items-center md:px-6 ">
                        <span title={simplifyDate(c.last_announce)}>
                          {IsEmptyDate(c.last_announce)}
                        </span>
                      </div>
                    </div>
                  </li>
                ))}
              </ol>
            ) : (
              <div className="flex text-center justify-center py-4 dark:text-gray-500">
                <p>No channels!</p>
              </div>
            )}
          </div>
        </div>
      )}
    </li>
  );
};

interface ListItemDropdownProps {
  network: IrcNetwork;
  toggleUpdate: () => void;
}

const ListItemDropdown = ({
  network,
  toggleUpdate
}: ListItemDropdownProps) => {
  const cancelModalButtonRef = useRef(null);

  const queryClient = useQueryClient();

  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);
  const deleteMutation = useMutation(
    (id: number) => APIClient.irc.deleteNetwork(id),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(["networks"]);
        queryClient.invalidateQueries(["networks", network.id]);

        toast.custom((t) => <Toast type="success" body={`Network ${network.name} was deleted`} t={t}/>);

        toggleDeleteModal();
      }
    }
  );

  const restartMutation = useMutation(
    (id: number) => APIClient.irc.restartNetwork(id),
    {
      onSuccess: () => {
        toast.custom((t) => <Toast type="success"
          body={`${network.name} was successfully restarted`}
          t={t}/>);

        queryClient.invalidateQueries(["networks"]);
        queryClient.invalidateQueries(["networks", network.id]);
      }
    }
  );

  const restart = (id: number) => {
    restartMutation.mutate(id);
  };

  return (
    <Menu as="div">
      <DeleteModal
        isOpen={deleteModalIsOpen}
        toggle={toggleDeleteModal}
        buttonRef={cancelModalButtonRef}
        deleteAction={() => {
          deleteMutation.mutate(network.id);
          toggleDeleteModal();
        }}
        title={`Remove network: ${network.name}`}
        text="Are you sure you want to remove this network? This action cannot be undone."
      />
      <Menu.Button className="px-4 py-2">
        <EllipsisHorizontalIcon
          className="w-5 h-5 text-gray-700 hover:text-gray-900 dark:text-gray-100 dark:hover:text-gray-400"
          aria-hidden="true"
        />
      </Menu.Button>
      <Transition
        as={Fragment}
        enter="transition ease-out duration-100"
        enterFrom="transform opacity-0 scale-95"
        enterTo="transform opacity-100 scale-100"
        leave="transition ease-in duration-75"
        leaveFrom="transform opacity-100 scale-100"
        leaveTo="transform opacity-0 scale-95"
      >
        <Menu.Items
          className="absolute right-0 w-32 md:w-56 mt-2 origin-top-right bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700 rounded-md shadow-lg ring-1 ring-black ring-opacity-10 focus:outline-none"
        >
          <div className="px-1 py-1">
            <Menu.Item>
              {({ active }) => (
                <button
                  className={classNames(
                    active ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                    "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                  )}
                  onClick={() => toggleUpdate()}
                >
                  <PencilSquareIcon
                    className={classNames(
                      active ? "text-white" : "text-blue-500",
                      "w-5 h-5 mr-2"
                    )}
                    aria-hidden="true"
                  />
                  Edit
                </button>
              )}
            </Menu.Item>
            {/*<Menu.Item>*/}
            {/*  {({ active }) => (*/}
            {/*    <button*/}
            {/*      className={classNames(*/}
            {/*        active ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",*/}
            {/*        "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"*/}
            {/*      )}*/}
            {/*      onClick={() => onToggle(!network.enabled)}*/}
            {/*    >*/}
            {/*      <SwitchHorizontalIcon*/}
            {/*        className={classNames(*/}
            {/*          active ? "text-white" : "text-blue-500",*/}
            {/*          "w-5 h-5 mr-2"*/}
            {/*        )}*/}
            {/*        aria-hidden="true"*/}
            {/*      />*/}
            {/*      {network.enabled ? "Disable" : "Enable"}*/}
            {/*    </button>*/}
            {/*  )}*/}
            {/*</Menu.Item>*/}
            <Menu.Item>
              {({ active }) => (
                <button
                  className={classNames(
                    active ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                    "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                  )}
                  onClick={() => restart(network.id)}
                >
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className={classNames(
                    active ? "text-white" : "text-blue-500",
                    "w-5 h-5 mr-2"
                  )}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M5.636 5.636a9 9 0 1012.728 0M12 3v9" />
                  </svg>

                  Restart
                </button>
              )}
            </Menu.Item>
          </div>
          <div className="px-1 py-1">
            <Menu.Item>
              {({ active }) => (
                <button
                  className={classNames(
                    active ? "bg-red-600 text-white" : "text-gray-900 dark:text-gray-300",
                    "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                  )}
                  onClick={() => toggleDeleteModal()}
                >
                  <TrashIcon
                    className={classNames(
                      active ? "text-white" : "text-red-500",
                      "w-5 h-5 mr-2"
                    )}
                    aria-hidden="true"
                  />
                  Delete
                </button>
              )}
            </Menu.Item>
          </div>
        </Menu.Items>
      </Transition>
    </Menu>
  );
};

export default IrcSettings;
