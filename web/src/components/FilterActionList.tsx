import {Action, DownloadClient} from "../domain/interfaces";
import React, {Fragment, useEffect, useRef } from "react";
import {Dialog, Listbox, Switch, Transition} from '@headlessui/react'
import {classNames} from "../styles/utils";
import {CheckIcon, ChevronRightIcon, ExclamationIcon, SelectorIcon,} from "@heroicons/react/solid";
import {useToggle} from "../hooks/hooks";
import {useMutation} from "react-query";
import {Field, Form} from "react-final-form";
import {TextField} from "./inputs";
import DEBUG from "./debug";
import APIClient from "../api/APIClient";
import {queryClient} from "../App";

interface radioFieldsetOption {
    label: string;
    value: string;
}

const actionTypeOptions: radioFieldsetOption[] = [
    {label: "Test", value: "TEST"},
    {label: "Watch dir", value: "WATCH_FOLDER"},
    {label: "Exec", value: "EXEC"},
    {label: "qBittorrent", value: "QBITTORRENT"},
    {label: "Deluge", value: "DELUGE"},
];

interface FilterListProps {
    actions: Action[];
    clients: DownloadClient[];
    filterID: number;
}

export function FilterActionList({actions, clients, filterID}: FilterListProps) {
    useEffect(() => {
        // console.log("render list")
    }, [])

    return (
        <div className="bg-white shadow overflow-hidden sm:rounded-md">
            <ul className="divide-y divide-gray-200">
                {actions.map((action, idx) => (
                    <ListItem action={action} clients={clients} filterID={filterID} key={action.id} idx={idx} />
                ))}
            </ul>
        </div>
    )
}

interface ListItemProps {
    action: Action;
    clients: DownloadClient[];
    filterID: number;
    idx: number;
}

function ListItem({action, clients, filterID, idx}: ListItemProps) {
    const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false)
    const [edit, toggleEdit] = useToggle(false)

    const deleteMutation = useMutation((actionID: number) => APIClient.actions.delete(actionID), {
        onSuccess: () => {
            queryClient.invalidateQueries(['filter',filterID]);
            toggleDeleteModal()
        }
    })

    const enabledMutation = useMutation((actionID: number) => APIClient.actions.toggleEnable(actionID), {
        onSuccess: () => {
            queryClient.invalidateQueries(['filter',filterID]);
        }
    })

    const updateMutation = useMutation((action: Action) => APIClient.actions.update(action), {
        onSuccess: () => {
            queryClient.invalidateQueries(['filter',filterID]);
        }
    })

    const toggleActive = () => {
        enabledMutation.mutate(action.id)
    }

    useEffect(() => {
    }, [action])

    const cancelButtonRef = useRef(null)

    const deleteAction = () => {
        deleteMutation.mutate(action.id)
    }

    const onSubmit = (action: Action) => {
        // TODO clear data depending on type
        updateMutation.mutate(action)
    };

    const TypeForm = (action: Action) => {
        switch (action.type) {
            case "TEST":
                return (
                    <div className="py-4">
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
            case "EXEC":
                return (
                    <div>
                        <div className="mt-6 grid grid-cols-12 gap-6">
                            <TextField name="exec_cmd" label="Command" columns={6} placeholder="Path to program eg. /bin/test"/>
                            <TextField name="exec_args" label="Arguments" columns={6} placeholder="Arguments eg. --test"/>
                        </div>
                    </div>
                )
            case "WATCH_FOLDER":
                return (
                    <div className="mt-6 grid grid-cols-12 gap-6">
                        <TextField name="watch_folder" label="Watch folder" columns={6} placeholder="Watch directory eg. /home/user/rwatch"/>
                    </div>
                )
            case "QBITTORRENT":
                return (
                    <div className="w-full">
                        <div className="mt-6 grid grid-cols-12 gap-6">

                            <div className="col-span-6 sm:col-span-6">
                                <Field
                                    name="client_id"
                                    type="select"
                                    render={({input}) => (
                                        <Listbox value={input.value} onChange={input.onChange}>
                                            {({open}) => (
                                                <>
                                                    <Listbox.Label
                                                        className="block text-xs font-bold text-gray-700 uppercase tracking-wide">Client</Listbox.Label>
                                                    <div className="mt-2 relative">
                                                        <Listbox.Button
                                                            className="bg-white relative w-full border border-gray-300 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm">
                                                            <span
                                                                className="block truncate">{input.value ? clients.find(c => c.id === input.value)!.name : "Choose a client"}</span>
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
                                                                {clients.filter((c) => c.type === action.type).map((client: any) => (
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
                                    )}/>
                            </div>

                            <div className="col-span-6 sm:col-span-6">
                                <TextField name="save_path" label="Save path" columns={6}/>
                            </div>
                        </div>

                        <div className="mt-6 grid grid-cols-12 gap-6">
                            <TextField name="category" label="Category" columns={6}/>
                            <TextField name="tags" label="Tags" columns={6}/>
                        </div>

                        <div className="mt-6 grid grid-cols-12 gap-6">
                            <div className="col-span-12 sm:col-span-6">
                                <label htmlFor="first_name" className="block text-sm font-medium text-gray-700">
                                    Limit upload speed (kb/s)
                                </label>

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

                            <div className="col-span-12 sm:col-span-6">
                                <label htmlFor="first_name" className="block text-sm font-medium text-gray-700">
                                    Limit download speed (kb/s)
                                </label>

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
                )
            case "DELUGE":
                return (
                    <div>
                        <div className="mt-6 grid grid-cols-12 gap-6">
                            <div className="col-span-12 sm:col-span-6">
                                <Field
                                    name="client_id"
                                    type="select"
                                    render={({input}) => (
                                        <Listbox value={input.value} onChange={input.onChange}>
                                            {({open}) => (
                                                <>
                                                    <Listbox.Label
                                                        className="block text-xs font-bold text-gray-700 uppercase tracking-wide">Client</Listbox.Label>
                                                    <div className="mt-2 relative">
                                                        <Listbox.Button
                                                            className="bg-white relative w-full border border-gray-300 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm">
                                                            <span
                                                                className="block truncate">{input.value ? clients.find(c => c.id === input.value)!.name : "Choose a client"}</span>
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
                                                                {clients.filter((c) => c.type === action.type).map((client: any) => (
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
                                    )}/>

                            </div>
                            <div className="col-span-12 sm:col-span-6">
                                <TextField name="save_path" label="Save path" columns={6}/>
                            </div>
                        </div>

                        <div className="mt-6 col-span-12 sm:col-span-6">
                            <TextField name="label" label="Label" columns={6}/>
                        </div>

                        <div className="mt-6 grid grid-cols-12 gap-6">
                            <div className="col-span-12 sm:col-span-6">
                                <label htmlFor="first_name" className="block text-sm font-medium text-gray-700">
                                    Limit upload speed (kb/s)
                                </label>

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

                            <div className="col-span-12 sm:col-span-6">
                                <label htmlFor="first_name" className="block text-sm font-medium text-gray-700">
                                    Limit download speed (kb/s)
                                </label>

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
                )

            default:
                return <p>default</p>
        }
    }

    return (
        <li key={action.id}>
            <div className={classNames(idx % 2 === 0 ? 'bg-white' : 'bg-gray-50', "flex items-center sm:px-6 hover:bg-gray-50")}>
                <Switch
                    checked={action.enabled}
                    onChange={toggleActive}
                    className={classNames(
                        action.enabled ? 'bg-teal-500' : 'bg-gray-200',
                        'z-10 relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-light-blue-500'
                    )}
                >
                    <span className="sr-only">Use setting</span>
                    <span
                        aria-hidden="true"
                        className={classNames(
                            action.enabled ? 'translate-x-5' : 'translate-x-0',
                            'inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200'
                        )}
                    />
                </Switch>
                <button className="px-4 py-4 w-full flex block" onClick={toggleEdit}>
                    <div className="min-w-0 flex-1 sm:flex sm:items-center sm:justify-between">
                        <div className="truncate">
                            <div className="flex text-sm">
                                <p className="ml-4 font-medium text-indigo-600 truncate">{action.name}</p>
                            </div>
                        </div>
                        <div className="mt-4 flex-shrink-0 sm:mt-0 sm:ml-5">
                            <div className="flex overflow-hidden -space-x-1">
                                <span className="text-sm font-normal text-gray-500">{action.type}</span>
                            </div>
                        </div>
                    </div>
                    <div className="ml-5 flex-shrink-0">
                        <ChevronRightIcon className="h-5 w-5 text-gray-400" aria-hidden="true"/>
                    </div>
                </button>
            </div>

            {edit &&
            <div className="px-4 py-4 flex items-center sm:px-6">
                <Transition.Root show={deleteModalIsOpen} as={Fragment}>
                    <Dialog
                        as="div"
                        static
                        className="fixed z-10 inset-0 overflow-y-auto"
                        initialFocus={cancelButtonRef}
                        open={deleteModalIsOpen}
                        onClose={toggleDeleteModal}
                    >
                        <div
                            className="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
                            <Transition.Child
                                as={Fragment}
                                enter="ease-out duration-300"
                                enterFrom="opacity-0"
                                enterTo="opacity-100"
                                leave="ease-in duration-200"
                                leaveFrom="opacity-100"
                                leaveTo="opacity-0"
                            >
                                <Dialog.Overlay className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"/>
                            </Transition.Child>

                            {/* This element is to trick the browser into centering the modal contents. */}
                            <span className="hidden sm:inline-block sm:align-middle sm:h-screen" aria-hidden="true">
            &#8203;
          </span>
                            <Transition.Child
                                as={Fragment}
                                enter="ease-out duration-300"
                                enterFrom="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
                                enterTo="opacity-100 translate-y-0 sm:scale-100"
                                leave="ease-in duration-200"
                                leaveFrom="opacity-100 translate-y-0 sm:scale-100"
                                leaveTo="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
                            >
                                <div
                                    className="inline-block align-bottom bg-white rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
                                    <div className="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
                                        <div className="sm:flex sm:items-start">
                                            <div
                                                className="mx-auto flex-shrink-0 flex items-center justify-center h-12 w-12 rounded-full bg-red-100 sm:mx-0 sm:h-10 sm:w-10">
                                                <ExclamationIcon className="h-6 w-6 text-red-600" aria-hidden="true"/>
                                            </div>
                                            <div className="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left">
                                                <Dialog.Title as="h3"
                                                              className="text-lg leading-6 font-medium text-gray-900">
                                                    Remove filter action
                                                </Dialog.Title>
                                                <div className="mt-2">
                                                    <p className="text-sm text-gray-500">
                                                        Are you sure you want to remove this action?
                                                        This action cannot be undone.
                                                    </p>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                    <div className="bg-gray-50 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
                                        <button
                                            type="button"
                                            className="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 bg-red-600 text-base font-medium text-white hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:ml-3 sm:w-auto sm:text-sm"
                                            onClick={deleteAction}
                                        >
                                            Remove
                                        </button>
                                        <button
                                            type="button"
                                            className="mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 shadow-sm px-4 py-2 bg-white text-base font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 sm:mt-0 sm:ml-3 sm:w-auto sm:text-sm"
                                            onClick={toggleDeleteModal}
                                            ref={cancelButtonRef}
                                        >
                                            Cancel
                                        </button>
                                    </div>
                                </div>
                            </Transition.Child>
                        </div>
                    </Dialog>
                </Transition.Root>

                <Form
                    initialValues={{
                        id: action.id,
                        name: action.name,
                        enabled: action.enabled,
                        type: action.type,
                        watch_folder: action.watch_folder,
                        exec_cmd: action.exec_cmd,
                        exec_args: action.exec_args,
                        category: action.category,
                        tags: action.tags,
                        label: action.label,
                        save_path: action.save_path,
                        paused: action.paused,
                        ignore_rules: action.ignore_rules,
                        limit_upload_speed: action.limit_upload_speed || 0,
                        limit_download_speed: action.limit_download_speed || 0,
                        filter_id: action.filter_id,
                        client_id: action.client_id,
                    }}
                    onSubmit={onSubmit}
                >
                    {({handleSubmit, values}) => {
                        return (
                            <form onSubmit={handleSubmit} className="w-full">

                                <div className="mt-6 grid grid-cols-12 gap-6">
                                    <div className="col-span-6">

                                        <Field
                                            name="type"
                                            type="select"
                                            render={({input}) => (
                                                <Listbox value={input.value} onChange={input.onChange}>
                                                    {({open}) => (
                                                        <>
                                                            <Listbox.Label
                                                                className="block text-xs font-bold text-gray-700 uppercase tracking-wide">Type</Listbox.Label>
                                                            <div className="mt-2 relative">
                                                                <Listbox.Button
                                                                    className="bg-white relative w-full border border-gray-300 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm">
                                                                    <span
                                                                        className="block truncate">{input.value ? actionTypeOptions.find(c => c.value === input.value)!.label : "Choose a type"}</span>
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
                                                                        {actionTypeOptions.map((opt) => (
                                                                            <Listbox.Option
                                                                                key={opt.value}
                                                                                className={({active}) =>
                                                                                    classNames(
                                                                                        active ? 'text-white bg-indigo-600' : 'text-gray-900',
                                                                                        'cursor-default select-none relative py-2 pl-3 pr-9'
                                                                                    )
                                                                                }
                                                                                value={opt.value}
                                                                            >
                                                                                {({selected, active}) => (
                                                                                    <>
                        <span className={classNames(selected ? 'font-semibold' : 'font-normal', 'block truncate')}>
                          {opt.label}
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
                                            )}/>
                                    </div>

                                    <TextField name="name" label="Name" columns={6}/>

                                </div>

                                {TypeForm(values)}

                                <div className="pt-6 divide-y divide-gray-200">
                                    <div className="mt-4 pt-4 flex justify-between">
                                        <button
                                            type="button"
                                            className="inline-flex items-center justify-center px-4 py-2 border border-transparent font-medium rounded-md text-red-700 bg-red-100 hover:bg-red-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:text-sm"
                                            onClick={toggleDeleteModal}
                                        >
                                            Remove
                                        </button>

                                        <div>
                                            <button
                                                type="button"
                                                className="bg-white border border-gray-300 rounded-md shadow-sm py-2 px-4 inline-flex justify-center text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-light-blue-500"
                                            >
                                                Cancel
                                            </button>
                                            <button
                                                type="submit"
                                                className="ml-4 relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                                            >
                                                Save
                                            </button>
                                        </div>
                                    </div>
                                </div>

                                <DEBUG values={values}/>
                            </form>

                        )
                    }}
                </Form>
            </div>
            }
        </li>
    )
}
