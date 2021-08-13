import React, {Fragment, useEffect } from "react";
import {useMutation} from "react-query";
import {Action, DownloadClient, Filter} from "../../domain/interfaces";
import {queryClient} from "../../App";
import {sleep} from "../../utils/utils";
import {CheckIcon, ExclamationIcon, SelectorIcon, XIcon} from "@heroicons/react/solid";
import {Dialog, Listbox, RadioGroup, Transition} from "@headlessui/react";
import {classNames} from "../../styles/utils";
import {Field, Form} from "react-final-form";
import DEBUG from "../../components/debug";
import APIClient from "../../api/APIClient";

interface radioFieldsetOption {
    label: string;
    description: string;
    value: string;
}

const actionTypeOptions: radioFieldsetOption[] = [
    {label: "Test", description: "A simple action to test a filter.", value: "TEST"},
    {label: "Watch dir", description: "Add filtered torrents to a watch directory", value: "WATCH_FOLDER"},
    {label: "Exec", description: "Run a custom command after a filter match", value: "EXEC"},
    {label: "qBittorrent", description: "Add torrents directly to qBittorrent", value: "QBITTORRENT"},
    {label: "Deluge", description: "Add torrents directly to Deluge", value: "DELUGE"},
];

interface props {
    filter: Filter;
    isOpen: boolean;
    toggle: any;
    clients: DownloadClient[];
}

function FilterActionAddForm({filter, isOpen, toggle, clients}: props) {
    const mutation = useMutation((action: Action) => APIClient.actions.create(action), {
        onSuccess: () => {
            queryClient.invalidateQueries(['filter', filter.id]);
            sleep(500).then(() => toggle())
        }
    })

    useEffect(() => {
        // console.log("render add action form", clients)
    }, []);

    const onSubmit = (data: any) => {
        // TODO clear data depending on type
        mutation.mutate(data)
    };

    const TypeForm = (values: any) => {
        switch (values.type) {
            case "TEST":
                return (
                    <div className="p-4">
                        <div className="rounded-md bg-yellow-50 p-4">
                            <div className="flex">
                                <div className="flex-shrink-0">
                                    <ExclamationIcon className="h-5 w-5 text-yellow-400" aria-hidden="true"/>
                                </div>
                                <div className="ml-3">
                                    <h3 className="text-sm font-medium text-yellow-800">Notice</h3>
                                    <div className="mt-2 text-sm text-yellow-700">
                                        <p>
                                            The test action does nothing except to show if the filter works.
                                        </p>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                )
            case "WATCH_FOLDER":
                return (
                    <div className="">
                        <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                            <div>
                                <label
                                    htmlFor="watch_folder"
                                    className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                >
                                    Watch dir
                                </label>
                            </div>
                            <div className="sm:col-span-2">
                                <Field name="watch_folder">
                                    {({input, meta}) => (
                                        <div className="sm:col-span-2">
                                            <input
                                                type="text"
                                                {...input}
                                                className="block w-full shadow-sm sm:text-sm focus:ring-indigo-500 focus:border-indigo-500 border-gray-300 rounded-md"
                                                placeholder="Watch directory eg. /home/user/watch_folder"
                                            />
                                            {meta.touched && meta.error &&
                                            <span>{meta.error}</span>}
                                        </div>
                                    )}
                                </Field>
                            </div>
                        </div>
                    </div>
                )
            case "EXEC":
                return (
                    <div className="">
                        <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                            <div>
                                <label
                                    htmlFor="exec_cmd"
                                    className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                >
                                    Program
                                </label>
                            </div>
                            <div className="sm:col-span-2">
                                <Field name="exec_cmd">
                                    {({input, meta}) => (
                                        <div className="sm:col-span-2">
                                            <input
                                                type="text"
                                                {...input}
                                                className="block w-full shadow-sm sm:text-sm focus:ring-indigo-500 focus:border-indigo-500 border-gray-300 rounded-md"
                                                placeholder="Path to program eg. /bin/test"
                                            />
                                            {meta.touched && meta.error &&
                                            <span>{meta.error}</span>}
                                        </div>
                                    )}
                                </Field>
                            </div>
                        </div>

                        <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                            <div>
                                <label
                                    htmlFor="exec_args"
                                    className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                >
                                    Arguments
                                </label>
                            </div>
                            <div className="sm:col-span-2">
                                <Field name="exec_args">
                                    {({input, meta}) => (
                                        <div className="sm:col-span-2">
                                            <input
                                                type="text"
                                                {...input}
                                                className="block w-full shadow-sm sm:text-sm focus:ring-indigo-500 focus:border-indigo-500 border-gray-300 rounded-md"
                                                placeholder="Arguments eg. --test"
                                            />
                                            {meta.touched && meta.error &&
                                            <span>{meta.error}</span>}
                                        </div>
                                    )}
                                </Field>
                            </div>
                        </div>
                    </div>

                )
            case "QBITTORRENT":

                return (
                    <div>

                        <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                            {/*// TODO change available clients to match only selected action type. eg qbittorrent or deluge*/}

                            <Field
                                name="client_id"
                                type="select"
                                render={({input}) => (
                                    <Listbox value={input.value} onChange={input.onChange}>
                                        {({open}) => (
                                            <>
                                                <Listbox.Label
                                            className="block text-sm font-medium text-gray-700">Client</Listbox.Label>
                                        <div className="mt-1 relative">
                                            <Listbox.Button
                                                className="bg-white relative w-full border border-gray-300 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm">
                                                <span className="block truncate">{input.value ? clients.find(c => c.id === input.value)!.name : "Choose a client"}</span>
                                                {/*<span className="block truncate">Choose a client</span>*/}
                                                <span
                                                    className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
                <SelectorIcon className="h-5 w-5 text-gray-400" aria-hidden="true"/>
              </span>
                                            </Listbox.Button>

                                            <Transition
                                                show={open}
                                                as={Fragment}
                                                leave="transition ease-in duration-100"
                                                leaveFrom="opacity-100"
                                                leaveTo="opacity-0"
                                            >
                                                <Listbox.Options
                                                    static
                                                    className="absolute z-10 mt-1 w-full bg-white shadow-lg max-h-60 rounded-md py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm"
                                                >
                                                    {clients.filter((c) => c.type === values.type).map((client: any) => (
                                                        <Listbox.Option
                                                            key={client.id}
                                                            className={({active}) =>
                                                                classNames(
                                                                    active ? 'text-white bg-indigo-600' : 'text-gray-900',
                                                                    'cursor-default select-none relative py-2 pl-3 pr-9'
                                                                )
                                                            }
                                                            value={client.id}
                                                        >
                                                            {({selected, active}) => (
                                                                <>
                        <span className={classNames(selected ? 'font-semibold' : 'font-normal', 'block truncate')}>
                          {client.name}
                        </span>

                                                                    {selected ? (
                                                                        <span
                                                                            className={classNames(
                                                                                active ? 'text-white' : 'text-indigo-600',
                                                                                'absolute inset-y-0 right-0 flex items-center pr-4'
                                                                            )}
                                                                        >
                            <CheckIcon className="h-5 w-5" aria-hidden="true"/>
                          </span>
                                                                    ) : null}
                                                                </>
                                                            )}
                                                        </Listbox.Option>
                                                    ))}
                                                </Listbox.Options>
                                            </Transition>
                                        </div>
                                    </>
                                )}
                            </Listbox>
                                )} />

                        </div>

                        <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                            <div>
                                <label
                                    htmlFor="category"
                                    className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                >
                                    Category
                                </label>
                            </div>
                            <div className="sm:col-span-2">
                                <Field name="category">
                                    {({input, meta}) => (
                                        <div className="sm:col-span-2">
                                            <input
                                                type="text"
                                                {...input}
                                                className="block w-full shadow-sm sm:text-sm focus:ring-indigo-500 focus:border-indigo-500 border-gray-300 rounded-md"
                                                // placeholder="Arguments eg. --test"
                                            />
                                            {meta.touched && meta.error &&
                                            <span>{meta.error}</span>}
                                        </div>
                                    )}
                                </Field>
                            </div>
                        </div>

                        <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                            <div>
                                <label
                                    htmlFor="tags"
                                    className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                >
                                    Tags
                                </label>
                            </div>
                            <div className="sm:col-span-2">
                                <Field name="tags">
                                    {({input, meta}) => (
                                        <div className="sm:col-span-2">
                                            <input
                                                type="text"
                                                {...input}
                                                className="block w-full shadow-sm sm:text-sm focus:ring-indigo-500 focus:border-indigo-500 border-gray-300 rounded-md"
                                                placeholder="Comma separated eg. 4k,remux"
                                            />
                                            {meta.touched && meta.error &&
                                            <span>{meta.error}</span>}
                                        </div>
                                    )}
                                </Field>
                            </div>
                        </div>

                        <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                            <div>
                                <label
                                    htmlFor="save_path"
                                    className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                >
                                    Save path. <br/><span className="text-gray-500">if left blank and category is selected it will use category path</span>
                                </label>
                            </div>
                            <div className="sm:col-span-2">
                                <Field name="save_path">
                                    {({input, meta}) => (
                                        <div className="sm:col-span-2">
                                            <input
                                                type="text"
                                                {...input}
                                                className="block w-full shadow-sm sm:text-sm focus:ring-indigo-500 focus:border-indigo-500 border-gray-300 rounded-md"
                                            />
                                            {meta.touched && meta.error &&
                                            <span>{meta.error}</span>}
                                        </div>
                                    )}
                                </Field>
                            </div>
                        </div>

                        <div className="divide-y px-4 divide-gray-200 pt-8 space-y-6 sm:pt-10 sm:space-y-5">
                            <div>
                                <h3 className="text-lg leading-6 font-medium text-gray-900">Limit speeds</h3>
                                <p className="mt-1 max-w-2xl text-sm text-gray-500">
                                    Limit download and upload speed for torrents in this filter. In KB/s.
                                </p>
                            </div>
                            <div className="space-y-6 sm:space-y-5 divide-y divide-gray-200">
                                <div className="pt-6 sm:pt-5">

                                    <div className="space-y-1 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:py-5">
                                        <div>
                                            <label
                                                htmlFor="limit_download_speed"
                                                className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                            >
                                                Limit download speed
                                            </label>
                                        </div>
                                        <div className="sm:col-span-2">
                                            <Field name="limit_download_speed">
                                                {({input, meta}) => (
                                                    <div className="sm:col-span-2">
                                                        <input
                                                            type="number"
                                                            {...input}
                                                            className="block w-full shadow-sm sm:text-sm focus:ring-indigo-500 focus:border-indigo-500 border-gray-300 rounded-md"
                                                        />
                                                        {meta.touched && meta.error &&
                                                        <span>{meta.error}</span>}
                                                    </div>
                                                )}
                                            </Field>
                                        </div>
                                    </div>
                                </div>
                            </div>


                            <div className="space-y-6 sm:space-y-5 divide-y divide-gray-200">
                                <div className="pt-6 sm:pt-5">

                                    <div className="space-y-1 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:py-5">
                                        <div>
                                            <label
                                                htmlFor="limit_upload_speed"
                                                className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                            >
                                                Limit upload speed
                                            </label>
                                        </div>
                                        <div className="sm:col-span-2">
                                            <Field name="limit_upload_speed">
                                                {({input, meta}) => (
                                                    <div className="sm:col-span-2">
                                                        <input
                                                            type="number"
                                                            {...input}
                                                            className="block w-full shadow-sm sm:text-sm focus:ring-indigo-500 focus:border-indigo-500 border-gray-300 rounded-md"
                                                        />
                                                        {meta.touched && meta.error &&
                                                        <span>{meta.error}</span>}
                                                    </div>
                                                )}
                                            </Field>
                                        </div>
                                    </div>
                                </div>
                            </div>

                        </div>
                    </div>
                )
            case "DELUGE":
                return (
                    <div>
                        {/*TODO choose client*/}

                        <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                        <Field
                            name="client_id"
                            type="select"
                            render={({input}) => (
                                <Listbox value={input.value} onChange={input.onChange}>
                                    {({open}) => (
                                        <>
                                            <Listbox.Label
                                                className="block text-sm font-medium text-gray-700">Client</Listbox.Label>
                                            <div className="mt-1 relative">
                                                <Listbox.Button
                                                    className="bg-white relative w-full border border-gray-300 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm">
                                                    <span className="block truncate">{input.value ? clients.find(c => c.id === input.value)!.name : "Choose a client"}</span>
                                                    {/*<span className="block truncate">Choose a client</span>*/}
                                                    <span
                                                        className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
                <SelectorIcon className="h-5 w-5 text-gray-400" aria-hidden="true"/>
              </span>
                                                </Listbox.Button>

                                                <Transition
                                                    show={open}
                                                    as={Fragment}
                                                    leave="transition ease-in duration-100"
                                                    leaveFrom="opacity-100"
                                                    leaveTo="opacity-0"
                                                >
                                                    <Listbox.Options
                                                        static
                                                        className="absolute z-10 mt-1 w-full bg-white shadow-lg max-h-60 rounded-md py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm"
                                                    >
                                                        {clients.filter((c) => c.type === values.type).map((client: any) => (
                                                            <Listbox.Option
                                                                key={client.id}
                                                                className={({active}) =>
                                                                    classNames(
                                                                        active ? 'text-white bg-indigo-600' : 'text-gray-900',
                                                                        'cursor-default select-none relative py-2 pl-3 pr-9'
                                                                    )
                                                                }
                                                                value={client.id}
                                                            >
                                                                {({selected, active}) => (
                                                                    <>
                        <span className={classNames(selected ? 'font-semibold' : 'font-normal', 'block truncate')}>
                          {client.name}
                        </span>

                                                                        {selected ? (
                                                                            <span
                                                                                className={classNames(
                                                                                    active ? 'text-white' : 'text-indigo-600',
                                                                                    'absolute inset-y-0 right-0 flex items-center pr-4'
                                                                                )}
                                                                            >
                            <CheckIcon className="h-5 w-5" aria-hidden="true"/>
                          </span>
                                                                        ) : null}
                                                                    </>
                                                                )}
                                                            </Listbox.Option>
                                                        ))}
                                                    </Listbox.Options>
                                                </Transition>
                                            </div>
                                        </>
                                    )}
                                </Listbox>
                            )} />
                        </div>

                        <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                            <div>
                                <label
                                    htmlFor="label"
                                    className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                >
                                    Label
                                </label>
                            </div>
                            <div className="sm:col-span-2">
                                <Field name="label">
                                    {({input, meta}) => (
                                        <div className="sm:col-span-2">
                                            <input
                                                type="text"
                                                {...input}
                                                className="block w-full shadow-sm sm:text-sm focus:ring-indigo-500 focus:border-indigo-500 border-gray-300 rounded-md"
                                            />
                                            {meta.touched && meta.error &&
                                            <span>{meta.error}</span>}
                                        </div>
                                    )}
                                </Field>
                            </div>
                        </div>

                        <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                            <div>
                                <label
                                    htmlFor="save_path"
                                    className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                >
                                    Save path
                                </label>
                            </div>
                            <div className="sm:col-span-2">
                                <Field name="save_path">
                                    {({input, meta}) => (
                                        <div className="sm:col-span-2">
                                            <input
                                                type="text"
                                                {...input}
                                                className="block w-full shadow-sm sm:text-sm focus:ring-indigo-500 focus:border-indigo-500 border-gray-300 rounded-md"
                                            />
                                            {meta.touched && meta.error &&
                                            <span>{meta.error}</span>}
                                        </div>
                                    )}
                                </Field>
                            </div>
                        </div>

                        <div className="divide-y px-4 divide-gray-200 pt-8 space-y-6 sm:pt-10 sm:space-y-5">
                            <div>
                                <h3 className="text-lg leading-6 font-medium text-gray-900">Limit speeds</h3>
                                <p className="mt-1 max-w-2xl text-sm text-gray-500">
                                    Limit download and upload speed for torrents in this filter. In KB/s.
                                </p>
                            </div>
                            <div className="space-y-6 sm:space-y-5 divide-y divide-gray-200">
                                <div className="pt-6 sm:pt-5">

                                    <div className="space-y-1 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:py-5">
                                        <div>
                                            <label
                                                htmlFor="limit_download_speed"
                                                className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                            >
                                                Limit download speed
                                            </label>
                                        </div>
                                        <div className="sm:col-span-2">
                                            <Field name="limit_download_speed">
                                                {({input, meta}) => (
                                                    <div className="sm:col-span-2">
                                                        <input
                                                            type="number"
                                                            {...input}
                                                            className="block w-full shadow-sm sm:text-sm focus:ring-indigo-500 focus:border-indigo-500 border-gray-300 rounded-md"
                                                        />
                                                        {meta.touched && meta.error &&
                                                        <span>{meta.error}</span>}
                                                    </div>
                                                )}
                                            </Field>
                                        </div>
                                    </div>
                                </div>
                            </div>


                            <div className="space-y-6 sm:space-y-5 divide-y divide-gray-200">
                                <div className="pt-6 sm:pt-5">

                                    <div className="space-y-1 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:py-5">
                                        <div>
                                            <label
                                                htmlFor="limit_upload_speed"
                                                className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                            >
                                                Limit upload speed
                                            </label>
                                        </div>
                                        <div className="sm:col-span-2">
                                            <Field name="limit_upload_speed">
                                                {({input, meta}) => (
                                                    <div className="sm:col-span-2">
                                                        <input
                                                            type="number"
                                                            {...input}
                                                            className="block w-full shadow-sm sm:text-sm focus:ring-indigo-500 focus:border-indigo-500 border-gray-300 rounded-md"
                                                        />
                                                        {meta.touched && meta.error &&
                                                        <span>{meta.error}</span>}
                                                    </div>
                                                )}
                                            </Field>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                )
            default:
                return (
                    <div className="p-4">
                        <div className="rounded-md bg-yellow-50 p-4">
                            <div className="flex">
                                <div className="flex-shrink-0">
                                    <ExclamationIcon className="h-5 w-5 text-yellow-400" aria-hidden="true"/>
                                </div>
                                <div className="ml-3">
                                    <h3 className="text-sm font-medium text-yellow-800">Notice</h3>
                                    <div className="mt-2 text-sm text-yellow-700">
                                        <p>
                                            The test action does nothing except to show if the filter works.
                                        </p>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                )
        }
    }

    return (
        <Transition.Root show={isOpen} as={Fragment}>
            <Dialog as="div" static className="fixed inset-0 overflow-hidden" open={isOpen} onClose={toggle}>
                <div className="absolute inset-0 overflow-hidden">
                    <Dialog.Overlay className="absolute inset-0"/>

                    <div className="fixed inset-y-0 right-0 pl-10 max-w-full flex sm:pl-16">
                        <Transition.Child
                            as={Fragment}
                            enter="transform transition ease-in-out duration-500 sm:duration-700"
                            enterFrom="translate-x-full"
                            enterTo="translate-x-0"
                            leave="transform transition ease-in-out duration-500 sm:duration-700"
                            leaveFrom="translate-x-0"
                            leaveTo="translate-x-full"
                        >
                            <div className="w-screen max-w-2xl">
                                <Form
                                    initialValues={{
                                        name: "",
                                        enabled: false,
                                        type: "TEST",
                                        watch_folder: "",
                                        exec_cmd: "",
                                        exec_args: "",
                                        category: "",
                                        tags: "",
                                        label: "",
                                        save_path: "",
                                        paused: false,
                                        ignore_rules: false,
                                        limit_upload_speed: 0,
                                        limit_download_speed: 0,
                                        filter_id: filter.id,
                                        client_id: null,
                                    }}
                                    onSubmit={onSubmit}
                                >
                                    {({handleSubmit, values}) => {
                                        return (
                                            <form className="h-full flex flex-col bg-white shadow-xl overflow-y-scroll"
                                                  onSubmit={handleSubmit}>
                                                <div className="flex-1">
                                                    {/* Header */}
                                                    <div className="px-4 py-6 bg-gray-50 sm:px-6">
                                                        <div className="flex items-start justify-between space-x-3">
                                                            <div className="space-y-1">
                                                                <Dialog.Title
                                                                    className="text-lg font-medium text-gray-900">Add
                                                                    action</Dialog.Title>
                                                                <p className="text-sm text-gray-500">
                                                                    Add filter action.
                                                                </p>
                                                            </div>
                                                            <div className="h-7 flex items-center">
                                                                <button
                                                                    type="button"
                                                                    className="bg-white rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                                                                    onClick={toggle}
                                                                >
                                                                    <span className="sr-only">Close panel</span>
                                                                    <XIcon className="h-6 w-6" aria-hidden="true"/>
                                                                </button>
                                                            </div>
                                                        </div>
                                                    </div>

                                                    {/* Divider container */}
                                                    <div
                                                        className="py-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                                                        <div
                                                            className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                                                            <div>
                                                                <label
                                                                    htmlFor="name"
                                                                    className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                                                >
                                                                    Action name
                                                                </label>
                                                            </div>
                                                            <Field name="name">
                                                                {({input, meta}) => (
                                                                    <div className="sm:col-span-2">
                                                                        <input
                                                                            type="text"
                                                                            {...input}
                                                                            className="block w-full shadow-sm sm:text-sm focus:ring-indigo-500 focus:border-indigo-500 border-gray-300 rounded-md"
                                                                        />
                                                                        {meta.touched && meta.error &&
                                                                        <span>{meta.error}</span>}
                                                                    </div>
                                                                )}
                                                            </Field>
                                                        </div>

                                                        <fieldset>
                                                            <div
                                                                className="space-y-2 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:items-start sm:px-6 sm:py-5">
                                                                <div>
                                                                    <legend
                                                                        className="text-sm font-medium text-gray-900">Type
                                                                    </legend>
                                                                </div>
                                                                <div className="space-y-5 sm:col-span-2">
                                                                    <div className="space-y-5 sm:mt-0">
                                                                        <Field
                                                                            name="type"
                                                                            type="radio"
                                                                            render={({input}) => (
                                                                                <RadioGroup value={values.type} onChange={input.onChange}>
                                                                                    <RadioGroup.Label className="sr-only">Privacy setting</RadioGroup.Label>
                                                                                    <div className="bg-white rounded-md -space-y-px">
                                                                                        {actionTypeOptions.map((setting, settingIdx) => (
                                                                                            <RadioGroup.Option
                                                                                                key={setting.value}
                                                                                                value={setting.value}
                                                                                                className={({checked}) =>
                                                                                                    classNames(
                                                                                                        settingIdx === 0 ? 'rounded-tl-md rounded-tr-md' : '',
                                                                                                        settingIdx === actionTypeOptions.length - 1 ? 'rounded-bl-md rounded-br-md' : '',
                                                                                                        checked ? 'bg-indigo-50 border-indigo-200 z-10' : 'border-gray-200',
                                                                                                        'relative border p-4 flex cursor-pointer focus:outline-none'
                                                                                                    )
                                                                                                }
                                                                                            >
                                                                                                {({
                                                                                                      active,
                                                                                                      checked
                                                                                                  }) => (
                                                                                                    <Fragment>
                                                                                                        <span
                                                                                                            className={classNames(
                                                                                                                                    checked ? 'bg-indigo-600 border-transparent' : 'bg-white border-gray-300',
                                                                                                                active ? 'ring-2 ring-offset-2 ring-indigo-500' : '',
                                                                                                                'h-4 w-4 mt-0.5 cursor-pointer rounded-full border flex items-center justify-center'
                                                                                                            )}
                                                                                                            aria-hidden="true"
                                                                                                        >
                                                                                                          <span className="rounded-full bg-white w-1.5 h-1.5"/>
                                                                                                        </span>
                                                                                                        <div
                                                                                                            className="ml-3 flex flex-col">
                                                                                                            <RadioGroup.Label
                                                                                                                as="span"
                                                                                                                className={classNames(checked ? 'text-indigo-900' : 'text-gray-900', 'block text-sm font-medium')}
                                                                                                            >
                                                                                                                {setting.label}
                                                                                                            </RadioGroup.Label>
                                                                                                            <RadioGroup.Description
                                                                                                                as="span"
                                                                                                                className={classNames(checked ? 'text-indigo-700' : 'text-gray-500', 'block text-sm')}
                                                                                                            >
                                                                                                                {setting.description}
                                                                                                            </RadioGroup.Description>
                                                                                                        </div>
                                                                                                    </Fragment>
                                                                                                )}
                                                                                            </RadioGroup.Option>
                                                                                        ))}
                                                                                    </div>
                                                                                </RadioGroup>

                                                                            )}
                                                                        />

                                                                    </div>
                                                                </div>
                                                            </div>
                                                        </fieldset>

                                                        {TypeForm(values)}

                                                    </div>
                                                </div>

                                                <div
                                                    className="flex-shrink-0 px-4 border-t border-gray-200 py-5 sm:px-6">
                                                    <div className="space-x-3 flex justify-end">
                                                        <button
                                                            type="button"
                                                            className="bg-white py-2 px-4 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                                                            onClick={toggle}
                                                        >
                                                            Cancel
                                                        </button>
                                                        <button
                                                            type="submit"
                                                            className="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                                                        >
                                                            Save
                                                        </button>
                                                    </div>
                                                </div>

                                                <DEBUG values={values}/>
                                            </form>
                                        )
                                    }}
                                </Form>
                            </div>

                        </Transition.Child>
                    </div>
                </div>
            </Dialog>
        </Transition.Root>
    )
}

export default FilterActionAddForm;