import React, { useState } from "react";
import { Switch } from "@headlessui/react";
import { EmptyListState } from "../../components/EmptyListState";

import {
    Link,
} from "react-router-dom";
import { Filter } from "../../domain/interfaces";
import { useToggle } from "../../hooks/hooks";
import { useQuery } from "react-query";
import { classNames } from "../../styles/utils";
import { FilterAddForm } from "../../forms";
import APIClient from "../../api/APIClient";

export default function Filters() {
    const [createFilterIsOpen, toggleCreateFilter] = useToggle(false)

    const { isLoading, error, data } = useQuery<Filter[], Error>('filter', APIClient.filters.getAll,
        {
            refetchOnWindowFocus: false
        }
    );

    if (isLoading) {
        return null
    }

    if (error) return (<p>'An error has occurred: '</p>)

    return (
        <main className="-mt-48 ">
            <FilterAddForm isOpen={createFilterIsOpen} toggle={toggleCreateFilter} />

            <header className="py-10">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex justify-between">
                    <h1 className="text-3xl font-bold text-white capitalize">Filters</h1>

                    <div className="flex-shrink-0">
                        <button
                            type="button"
                            className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                            onClick={toggleCreateFilter}
                        >
                            Add new
                        </button>
                    </div>
                </div>
            </header>

            <div className="max-w-7xl mx-auto pb-12 px-4 sm:px-6 lg:px-8">
                <div className="bg-white rounded-lg shadow">
                    <div className="relative inset-0 py-3 px-3 sm:px-3 lg:px-3 h-full">
                        {data && data.length > 0 ? <FilterList filters={data} /> :
                            <EmptyListState text="No filters here.." buttonText="Add new" buttonOnClick={toggleCreateFilter} />}
                    </div>
                </div>
            </div>
        </main>
    )
}

interface FilterListProps {
    filters: Filter[];
}

function FilterList({ filters }: FilterListProps) {
    return (
        <div className="flex flex-col">
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
                                    <th
                                        scope="col"
                                        className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                                    >
                                        Indexers
                                    </th>
                                    <th scope="col" className="relative px-6 py-3">
                                        <span className="sr-only">Edit</span>
                                    </th>
                                </tr>
                            </thead>
                            <tbody className="bg-white divide-y divide-gray-200">
                                {filters.map((filter: Filter, idx) => (
                                    <FilterListItem filter={filter} key={idx} idx={idx} />
                                ))}
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    )
}

interface FilterListItemProps {
    filter: Filter;
    idx: number;
}

function FilterListItem({ filter, idx }: FilterListItemProps) {
    const [enabled, setEnabled] = useState(filter.enabled)

    const toggleActive = (status: boolean) => {
        console.log(status)
        setEnabled(status)
        // call api
    }

    return (
        <tr key={filter.name}
            className={idx % 2 === 0 ? 'bg-white' : 'bg-gray-50'}>
            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                <Switch
                    checked={enabled}
                    onChange={toggleActive}
                    className={classNames(
                        enabled ? 'bg-teal-500' : 'bg-gray-200',
                        'relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-light-blue-500'
                    )}
                >
                    <span className="sr-only">Use setting</span>
                    <span
                        aria-hidden="true"
                        className={classNames(
                            enabled ? 'translate-x-5' : 'translate-x-0',
                            'inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200'
                        )}
                    />
                </Switch>
            </td>
            <td className="px-6 py-4 w-full whitespace-nowrap text-sm font-medium text-gray-900">{filter.name}</td>
            <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{filter.indexers && filter.indexers.map(t =>
                <span key={t.id} className="mr-2 inline-flex items-center px-2.5 py-0.5 rounded-md text-sm font-medium bg-gray-100 text-gray-800">{t.name}</span>)}</td>
            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <Link to={`filters/${filter.id.toString()}`} className="text-indigo-600 hover:text-indigo-900">
                    Edit
                </Link>
            </td>
        </tr>
    )
}