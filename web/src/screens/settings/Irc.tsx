import React, {useEffect} from "react";
import {IrcNetworkAddForm} from "../../forms";
import {useToggle} from "../../hooks/hooks";
import {useQuery} from "react-query";
import IrcNetworkUpdateForm from "../../forms/settings/IrcNetworkUpdateForm";
import {Switch} from "@headlessui/react";
import {classNames} from "../../styles/utils";
import EmptySimple from "../../components/empty/EmptySimple";
import APIClient from "../../api/APIClient";

interface IrcNetwork {
    id: number;
    name: string;
    enabled: boolean;
    addr: string;
    nick: string;
    username: string;
    realname: string;
    pass: string;
    // connect_commands: string;
}

function IrcSettings() {
    const [addNetworkIsOpen, toggleAddNetwork] = useToggle(false)

    useEffect(() => {
    }, []);

    const { data } = useQuery<any[], Error>('networks', APIClient.irc.getNetworks,
        {
            refetchOnWindowFocus: false
        }
    )

    return (
        <div className="divide-y divide-gray-200 lg:col-span-9">

            {addNetworkIsOpen &&
            <IrcNetworkAddForm isOpen={addNetworkIsOpen} toggle={toggleAddNetwork}/>
            }

            <div className="py-6 px-4 sm:p-6 lg:pb-8">
                <div className="-ml-4 -mt-4 flex justify-between items-center flex-wrap sm:flex-nowrap">
                    <div className="ml-4 mt-4">
                        <h3 className="text-lg leading-6 font-medium text-gray-900">IRC</h3>
                        <p className="mt-1 text-sm text-gray-500">
                            IRC networks and channels.
                        </p>
                    </div>
                    <div className="ml-4 mt-4 flex-shrink-0">
                        <button
                            type="button"
                            onClick={toggleAddNetwork}
                            className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                        >
                            Add new
                        </button>
                    </div>
                </div>

                <div className="flex flex-col mt-6">
                    {data && data.length > 0 ?
                        <div className="-my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
                            <div className="py-2 align-middle inline-block min-w-full sm:px-6 lg:px-8">
                                <div className="shadow overflow-hidden border-b border-gray-200 sm:rounded-lg">
                                    <table className="min-w-full divide-y divide-gray-200">
                                        <thead className="bg-gray-50">
                                        <tr>
                                            <th
                                                scope="col"
                                                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                                            >
                                                Enabled
                                            </th>
                                            <th
                                                scope="col"
                                                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                                            >
                                                Network
                                            </th>
                                            <th
                                                scope="col"
                                                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                                            >
                                                Addr
                                            </th>
                                            <th
                                                scope="col"
                                                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                                            >
                                                Nick
                                            </th>
                                            <th scope="col" className="relative px-6 py-3">
                                                <span className="sr-only">Edit</span>
                                            </th>
                                        </tr>
                                        </thead>
                                        <tbody>
                                        {data && data.map((network: IrcNetwork, idx) => (
                                            <ListItem key={idx} idx={idx} network={network}/>
                                        ))}
                                        </tbody>
                                    </table>
                                </div>
                            </div>
                        </div>
                        : <EmptySimple title="No networks" subtitle="Add a new network" buttonText="New network" buttonAction={toggleAddNetwork}/>
                    }
                </div>
            </div>
        </div>
    )
}

const ListItem = ({ idx, network }: any) => {
    const [updateIsOpen, toggleUpdate] = useToggle(false)

    return (
        <tr key={network.name} className={idx % 2 === 0 ? 'bg-white' : 'bg-gray-50'}>
            {updateIsOpen && <IrcNetworkUpdateForm isOpen={updateIsOpen} toggle={toggleUpdate} network={network} />}
            <td className="px-6 py-4 whitespace-nowrap">
                <Switch
                    checked={network.enabled}
                    onChange={toggleUpdate}
                    className={classNames(
                        network.enabled ? 'bg-teal-500' : 'bg-gray-200',
                        'relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-light-blue-500'
                    )}
                >
                    <span className="sr-only">Enable</span>
                    <span
                        aria-hidden="true"
                        className={classNames(
                            network.enabled ? 'translate-x-5' : 'translate-x-0',
                            'inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200'
                        )}
                    />
                </Switch>
            </td>
            <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{network.name}</td>
            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{network.addr} {network.tls && <span className="ml-2 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">TLS</span>}</td>
            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{network.nick}</td>
            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <span className="text-indigo-600 hover:text-indigo-900 cursor-pointer" onClick={toggleUpdate}>
                    Edit
                </span>
            </td>
        </tr>

    )
}

export default IrcSettings;
