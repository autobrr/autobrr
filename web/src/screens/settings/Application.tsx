import React, { useState } from "react";
import { Switch } from "@headlessui/react";
import { classNames } from "../../utils";
// import {useRecoilState} from "recoil";
// import {configState} from "../../state/state";
import { useQuery } from "react-query";
import { Config } from "../../domain/interfaces";
import APIClient from "../../api/APIClient";

function ApplicationSettings() {
    const [isDebug, setIsDebug] = useState(true)
    // const [config] = useRecoilState(configState)

    const { isLoading, data } = useQuery<Config, Error>(['config'], () => APIClient.config.get(),
        {
            retry: false,
            refetchOnWindowFocus: false,
            onError: err => {
                console.log(err)
            }
        },
    )

    return (
        <form className="divide-y divide-gray-200 dark:divide-gray-700 lg:col-span-9" action="#" method="POST">
            <div className="py-6 px-4 sm:p-6 lg:pb-8">
                <div>
                    <h2 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">Application</h2>
                    <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                        Application settings. Change in config.toml and restart to take effect.
                    </p>
                </div>

                {!isLoading && data && (

                    <div className="mt-6 grid grid-cols-12 gap-6">
                        <div className="col-span-6 sm:col-span-4">
                            <label htmlFor="host" className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
                                Host
                            </label>
                            <input
                                type="text"
                                name="host"
                                id="host"
                                value={data.host}
                                disabled={true}
                                className="mt-2 block w-full dark:bg-gray-800 border border-gray-300 dark:border-gray-700 border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:text-gray-100 sm:text-sm"
                            />
                        </div>

                        <div className="col-span-6 sm:col-span-4">
                            <label htmlFor="port" className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
                                Port
                            </label>
                            <input
                                type="text"
                                name="port"
                                id="port"
                                value={data.port}
                                disabled={true}
                                className="mt-2 block w-full dark:bg-gray-800 border border-gray-300 dark:border-gray-700 border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:text-gray-100 sm:text-sm"
                            />
                        </div>

                        <div className="col-span-6 sm:col-span-4">
                            <label htmlFor="base_url" className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
                                Base url
                            </label>
                            <input
                                type="text"
                                name="base_url"
                                id="base_url"
                                value={data.base_url}
                                disabled={true}
                                className="mt-2 block w-full dark:bg-gray-800 border border-gray-300 dark:border-gray-700 border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:text-gray-100 sm:text-sm"
                            />
                        </div>
                    </div>
                )}
            </div>

            <div className="pt-6 pb-6 divide-y divide-gray-200 dark:divide-gray-700">
                <div className="px-4 sm:px-6">
                    <ul className="mt-2 divide-y divide-gray-200">
                        <Switch.Group as="li" className="py-4 flex items-center justify-between">
                            <div className="flex flex-col">
                                <Switch.Label as="p" className="text-sm font-medium text-gray-900 dark:text-white" passive>
                                    Debug
                                </Switch.Label>
                                <Switch.Description className="text-sm text-gray-500 dark:text-gray-400">
                                    Enable debug mode to get more logs.
                                </Switch.Description>
                            </div>
                            <Switch
                                checked={isDebug}
                                disabled={true}
                                onChange={setIsDebug}
                                className={classNames(
                                    isDebug ? 'bg-teal-500 dark:bg-blue-500' : 'bg-gray-200 dark:bg-gray-700',
                                    'ml-4 relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500'
                                )}
                            >
                                <span className="sr-only">Use setting</span>
                                <span
                                    aria-hidden="true"
                                    className={classNames(
                                        isDebug ? 'translate-x-5' : 'translate-x-0',
                                        'inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200'
                                    )}
                                />
                            </Switch>
                        </Switch.Group>
                    </ul>
                </div>
                {/*<div className="mt-4 py-4 px-4 flex justify-end sm:px-6">*/}
                {/*    <button*/}
                {/*        type="button"*/}
                {/*        className="bg-white border border-gray-300 rounded-md shadow-sm py-2 px-4 inline-flex justify-center text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"*/}
                {/*    >*/}
                {/*        Cancel*/}
                {/*    </button>*/}
                {/*    <button*/}
                {/*        type="submit"*/}
                {/*        className="ml-5 bg-indigo-700 border border-transparent rounded-md shadow-sm py-2 px-4 inline-flex justify-center text-sm font-medium text-white hover:bg-indigo-800 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"*/}
                {/*    >*/}
                {/*        Save*/}
                {/*    </button>*/}
                {/*</div>*/}
            </div>
        </form>

    )
}

export default ApplicationSettings;