import {useToggle} from "../../hooks/hooks";
import {useQuery} from "react-query";
import React, {useEffect} from "react";
import {IndexerAddForm, IndexerUpdateForm} from "../../forms";
import {Indexer} from "../../domain/interfaces";
import {Switch} from "@headlessui/react";
import {classNames} from "../../styles/utils";
import EmptySimple from "../../components/empty/EmptySimple";
import APIClient from "../../api/APIClient";

const ListItem = ({ indexer }: any) => {
    const [updateIsOpen, toggleUpdate] = useToggle(false)

 return (
     <tr key={indexer.name}>
         {updateIsOpen && <IndexerUpdateForm isOpen={updateIsOpen} toggle={toggleUpdate} indexer={indexer} />}
        <td className="px-6 py-4 whitespace-nowrap">
            <Switch
                checked={indexer.enabled}
                onChange={toggleUpdate}
                className={classNames(
                    indexer.enabled ? 'bg-teal-500' : 'bg-gray-200',
                    'relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-light-blue-500'
                )}
            >
                <span className="sr-only">Enable</span>
                <span
                    aria-hidden="true"
                    className={classNames(
                        indexer.enabled ? 'translate-x-5' : 'translate-x-0',
                        'inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200'
                    )}
                />
            </Switch>
        </td>
         <td className="px-6 py-4 w-full whitespace-nowrap text-sm font-medium text-gray-900">{indexer.name}</td>
        <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
            <span className="text-indigo-600 hover:text-indigo-900 cursor-pointer" onClick={toggleUpdate}>
                Edit
            </span>
        </td>
    </tr>
)
}

function IndexerSettings() {
    const [addIndexerIsOpen, toggleAddIndexer] = useToggle(false)

    const {error, data} = useQuery<any[], Error>('indexer', APIClient.indexers.getAll,
        {
            refetchOnWindowFocus: false
        }
    )

    useEffect(() => {
    }, []);

    if (error) return (<p>An error has occurred</p>)

    return (
        <div className="divide-y divide-gray-200 lg:col-span-9">

            <IndexerAddForm isOpen={addIndexerIsOpen} toggle={toggleAddIndexer} />

            <div className="py-6 px-4 sm:p-6 lg:pb-8">
                <div className="-ml-4 -mt-4 flex justify-between items-center flex-wrap sm:flex-nowrap">
                    <div className="ml-4 mt-4">
                        <h3 className="text-lg leading-6 font-medium text-gray-900">Indexers</h3>
                        <p className="mt-1 text-sm text-gray-500">
                            Indexer settings.
                        </p>
                    </div>
                    <div className="ml-4 mt-4 flex-shrink-0">
                        <button
                            type="button"
                            onClick={toggleAddIndexer}
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
                                                Name
                                            </th>
                                            <th scope="col" className="relative px-6 py-3">
                                                <span className="sr-only">Edit</span>
                                            </th>
                                        </tr>
                                        </thead>
                                        <tbody className="bg-white divide-y divide-gray-200">
                                        {data && data.map((indexer: Indexer, idx: number) => (
                                            <ListItem indexer={indexer} key={idx}/>
                                        ))}
                                        </tbody>
                                    </table>
                                </div>
                            </div>
                        </div>
                        : <EmptySimple title="No indexers" subtitle="Add a new indexer" buttonText="New indexer" buttonAction={toggleAddIndexer}/>
                    }
                </div>

            </div>
        </div>
    )
}

export default IndexerSettings;