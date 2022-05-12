import { useQuery } from "react-query";

import {
    simplifyDate,
    IsEmptyDate
} from "../../utils";
import {
    IrcNetworkAddForm,
    IrcNetworkUpdateForm
} from "../../forms";
import { useToggle } from "../../hooks/hooks";
import { APIClient } from "../../api/APIClient";
import { EmptySimple } from "../../components/emptystates";

export const IrcSettings = () => {
    const [addNetworkIsOpen, toggleAddNetwork] = useToggle(false)

    const { data } = useQuery(
        "networks",
        APIClient.irc.getNetworks,
        {
          refetchOnWindowFocus: false,
          // Refetch every 3 seconds
          refetchInterval: 3000
        }
    );

    return (
        <div className="divide-y divide-gray-200 lg:col-span-9">
            <IrcNetworkAddForm isOpen={addNetworkIsOpen} toggle={toggleAddNetwork} />

            <div className="py-6 px-4 sm:p-6 lg:pb-8">
                <div className="-ml-4 -mt-4 flex justify-between items-center flex-wrap sm:flex-nowrap">
                    <div className="ml-4 mt-4">
                        <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">IRC</h3>
                        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                            IRC networks and channels. Click on a network to view channel status.
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

                {data && data.length > 0 ? (
                    <section className="mt-6 light:bg-white dark:bg-gray-800 light:shadow sm:rounded-md">
                        <ol className="min-w-full">
                            <li className="grid grid-cols-12 gap-4 border-b border-gray-200 dark:border-gray-700">
                                {/* <div className="col-span-1 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Enabled</div> */}
                                <div className="col-span-3 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Network</div>
                                <div className="col-span-4 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Server</div>
                                <div className="col-span-4 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Nick</div>
                            </li>

                            {data && data.map((network, idx) => (
                                <ListItem key={idx} idx={idx} network={network} />
                            ))}
                        </ol>
                    </section>
                ) : <EmptySimple title="No networks" subtitle="Add a new network" buttonText="New network" buttonAction={toggleAddNetwork} />}
            </div>
        </div>
    )
}

interface ListItemProps {
    idx: number;
    network: IrcNetworkWithHealth;
}

const ListItem = ({ idx, network }: ListItemProps) => {
    const [updateIsOpen, toggleUpdate] = useToggle(false)
    const [edit, toggleEdit] = useToggle(false);

    return (

        <li key={idx} >
            <div className="grid grid-cols-12 gap-4 items-center hover:bg-gray-50 dark:hover:bg-gray-700 py-4">
                <IrcNetworkUpdateForm isOpen={updateIsOpen} toggle={toggleUpdate} network={network} />
 
                <div className="col-span-3 items-center sm:px-6 text-sm font-medium text-gray-900 dark:text-white cursor-pointer" onClick={toggleEdit}>
                    <span className="relative inline-flex items-center">
                        {
                            network.enabled ? (
                                network.connected ? (
                                    <span className="mr-3 flex h-3 w-3 relative" title={`Connected since: ${simplifyDate(network.connected_since)}`}>
                                        <span className="animate-ping inline-flex h-full w-full rounded-full bg-green-400 opacity-75"/>
                                        <span className="inline-flex absolute rounded-full h-3 w-3 bg-green-500"/>
                                    </span>
                                ) : <span className="mr-3 flex h-3 w-3 rounded-full opacity-75 bg-red-400" />
                            ) : <span className="mr-3 flex h-3 w-3 rounded-full opacity-75 bg-gray-500" />
                        }
                        {network.name}
                    </span>
                </div>

                <div className="col-span-4 flex justify-between items-center sm:px-6 text-sm text-gray-500 dark:text-gray-400 cursor-pointer" onClick={toggleEdit}>{network.server}:{network.port} {network.tls && <span className="ml-2 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 dark:bg-green-300 text-green-800 dark:text-green-900">TLS</span>}</div>
                {network.nickserv && network.nickserv.account ? (
                    <div className="col-span-4 items-center sm:px-6 text-sm text-gray-500 dark:text-gray-400 cursor-pointer" onClick={toggleEdit}>{network.nickserv.account}</div>
                ) : null}
                <div className="col-span-1 text-sm text-gray-500 dark:text-gray-400">
                    <span className="text-indigo-600 dark:text-gray-300 hover:text-indigo-900 cursor-pointer" onClick={toggleUpdate}>
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
                                    <div className="col-span-4 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Channel</div>
                                    <div className="col-span-4 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Monitoring since</div>
                                    <div className="col-span-4 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Last announce</div>
                                </li>
                                {network.channels.map(c => (
                                    <li key={c.id} className="text-gray-500 dark:text-gray-400">
                                        <div className="grid grid-cols-12 gap-4 items-center py-4">
                                            <div className="col-span-4 flex items-center sm:px-6 ">
                                                <span className="relative inline-flex items-center">
                                                    {
                                                        network.enabled ? (
                                                            c.monitoring ? (
                                                                <span className="mr-3 flex h-3 w-3 relative" title="monitoring">
                                                                    <span className="animate-ping inline-flex h-full w-full rounded-full bg-green-400 opacity-75"/>
                                                                    <span className="inline-flex absolute rounded-full h-3 w-3 bg-green-500"/>
                                                                </span>
                                                            ) : <span className="mr-3 flex h-3 w-3 rounded-full opacity-75 bg-red-400" />
                                                        ) : <span className="mr-3 flex h-3 w-3 rounded-full opacity-75 bg-gray-500" />
                                                    }
                                                    {c.name}
                                                </span>
                                            </div>
                                            <div className="col-span-4 flex items-center sm:px-6 ">
                                                <span className="" title={simplifyDate(c.monitoring_since)}>{IsEmptyDate(c.monitoring_since)}</span>
                                            </div>
                                            <div className="col-span-4 flex items-center sm:px-6 ">
                                                <span className="" title={simplifyDate(c.last_announce)}>{IsEmptyDate(c.last_announce)}</span>
                                            </div>
                                        </div>
                                    </li>
                                ))}
                            </ol>
                        ) : <div className="flex text-center justify-center py-4 dark:text-gray-500"><p>No channels!</p></div>}
                    </div>
                </div>
            )}
        </li>
    )
}
