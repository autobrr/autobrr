import { useQuery } from "react-query";

import { classNames, IsEmptyDate, simplifyDate } from "../../utils";
import { IrcNetworkAddForm, IrcNetworkUpdateForm } from "../../forms";
import { useToggle } from "../../hooks/hooks";
import { APIClient } from "../../api/APIClient";
import { EmptySimple } from "../../components/emptystates";
import { ExclamationCircleIcon } from "@heroicons/react/outline";
import { LockClosedIcon, LockOpenIcon } from "@heroicons/react/solid";

export const IrcSettings = () => {
  const [addNetworkIsOpen, toggleAddNetwork] = useToggle(false);

  const { data } = useQuery("networks", () => APIClient.irc.getNetworks(), {
    refetchOnWindowFocus: false,
    // Refetch every 3 seconds
    refetchInterval: 3000
  });

  return (
    <div className="lg:col-span-9">
      <IrcNetworkAddForm isOpen={addNetworkIsOpen} toggle={toggleAddNetwork} />

      <div className="py-6 px-4 sm:p-6 lg:pb-8">
        <div className="-ml-4 -mt-4 flex justify-between items-center flex-wrap sm:flex-nowrap">
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
              className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 dark:bg-blue-600 hover:bg-indigo-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
            >
              Add new
            </button>
          </div>
        </div>

        <div className="flex flex-col mt-10">
          <ol className="flex gap-2 divide-x divide-gray-200 dark:divide-gray-700">
            <li className="flex items-center">
              <span
                className="mr-3 flex h-4 w-4 relative"
                title="Network healthy"
              >
                <span className="animate-ping inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
                <span className="inline-flex absolute rounded-full h-4 w-4 bg-green-500" />
              </span>
              <span className="text-sm text-gray-800 dark:text-gray-500">Network healthy</span>
            </li>

            <li className="flex items-center pl-2">
              <span
                className="mr-3 flex items-center"
                title="Network unhealthy"
              >
                <ExclamationCircleIcon className="h-4 w-4 text-yellow-400 hover:text-yellow-600" />
              </span>
              <span className="text-sm text-gray-800 dark:text-gray-500">Network unhealthy</span>
            </li>

            <li className="flex items-center pl-2">
              <span
                className="mr-3 flex h-4 w-4 rounded-full opacity-75 bg-gray-500"
                title="Network disabled"
              >
              </span>
              <span className="text-sm text-gray-800 dark:text-gray-500">Network disabled</span>
            </li>
          </ol>
        </div>

        {data && data.length > 0 ? (
          <section className="mt-6 light:bg-white dark:bg-gray-800 light:shadow sm:rounded-md">
            <ol className="min-w-full">
              <li className="grid grid-cols-12 gap-4 border-b border-gray-200 dark:border-gray-700">
                <div className="col-span-3 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Network
                </div>
                <div className="col-span-5 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Server
                </div>
                <div className="col-span-3 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Nick
                </div>
              </li>
              {data &&
                data.map((network, idx) => (
                  <ListItem key={idx} idx={idx} network={network} />
                ))}
            </ol>
          </section>
        ) : (
          <EmptySimple
            title="No networks"
            subtitle="Add a new network"
            buttonText="New network"
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
}

const ListItem = ({ idx, network }: ListItemProps) => {
  const [updateIsOpen, toggleUpdate] = useToggle(false);
  const [edit, toggleEdit] = useToggle(false);

  return (
    <li key={idx}>
      <div className={classNames("grid grid-cols-12 gap-2 lg:gap-4 items-center py-4", network.enabled && !network.healthy ? "bg-red-50 dark:bg-red-900 hover:bg-red-100 dark:hover:bg-red-800" : "hover:bg-gray-50 dark:hover:bg-gray-700 ")}>
        <IrcNetworkUpdateForm
          isOpen={updateIsOpen}
          toggle={toggleUpdate}
          network={network}
        />

        <div
          className="col-span-3 items-center sm:px-6 text-sm font-medium text-gray-900 dark:text-white cursor-pointer"
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
            <div className="overflow-x-auto flex">
              {network.name}
            </div>
          </div>
        </div>
        <div
          className="col-span-5 sm:px-6 text-sm text-gray-500 dark:text-gray-400 cursor-pointer"
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
            <p className="break-all">
              {network.server}:{network.port}
            </p>
          </div>
        </div>
        {network.nickserv && network.nickserv.account ? (
          <div
            className="col-span-3 items-center sm:px-6 text-sm text-gray-500 dark:text-gray-400 cursor-pointer"
            onClick={toggleEdit}
          >
            <div className="overflow-x-auto flex">
              {network.nickserv.account}
            </div>
          </div>
        ) : (
          <div className="col-span-3" />
        )}
        <div className="col-span-1 text-sm text-gray-500 dark:text-gray-400">
          <span
            className="text-indigo-600 dark:text-gray-300 hover:text-indigo-900 cursor-pointer"
            onClick={toggleUpdate}
          >
            Edit
          </span>
        </div>
      </div>
      {edit && (
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
                      <div className="col-span-4 flex items-center sm:px-6 ">
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
                      <div className="col-span-4 flex items-center sm:px-6 ">
                        <span title={simplifyDate(c.monitoring_since)}>
                          {IsEmptyDate(c.monitoring_since)}
                        </span>
                      </div>
                      <div className="col-span-4 flex items-center sm:px-6 ">
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
