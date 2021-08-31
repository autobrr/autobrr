import React, { Fragment } from "react";
import { useMutation, useQuery } from "react-query";
import { Channel, Indexer, IndexerSchema, IndexerSchemaSettings, Network } from "../../domain/interfaces";
import { sleep } from "../../utils/utils";
import { XIcon } from "@heroicons/react/solid";
import { Dialog, Transition } from "@headlessui/react";
import { Field, Form } from "react-final-form";
import DEBUG from "../../components/debug";
import Select from "react-select";
import { queryClient } from "../../App";
import { SwitchGroup, TextFieldWide } from "../../components/inputs";
import APIClient from "../../api/APIClient";
import { NumberFieldWide, PasswordFieldWide } from "../../components/inputs/wide";
import { toast } from 'react-hot-toast'
import Toast from '../../components/notifications/Toast';
interface props {
    isOpen: boolean;
    toggle: any;
}

function IndexerAddForm({ isOpen, toggle }: props) {
    const { data } = useQuery<IndexerSchema[], Error>('indexerSchema', APIClient.indexers.getSchema,
        {
            enabled: isOpen,
            refetchOnWindowFocus: false
        }
    )

    const mutation = useMutation((indexer: Indexer) => APIClient.indexers.create(indexer), {
        onSuccess: () => {
            queryClient.invalidateQueries(['indexer']);
            toast.custom((t) => <Toast type="success" body="Indexer was added" t={t} />)
            sleep(1500)
            toggle()
        },
        onError: () => {
            toast.custom((t) => <Toast type="error" body="Indexer could not be added" t={t} />)
        }
    })

    const ircMutation = useMutation((network: Network) => APIClient.irc.createNetwork(network), {
        onSuccess: (data) => {
            console.log("irc mutation: ", data);

            // queryClient.invalidateQueries(['indexer']);
            // sleep(1500)

            // toggle()
        }
    })

    const onSubmit = (formData: any) => {
        let ind = data && data.find(i => i.identifier === formData.identifier)

        if (!ind) {
            return
        }

        let channels: Channel[] = []
        if (ind.irc.channels.length) {
            ind.irc.channels.forEach(element => {
                channels.push({ name: element })
            });
        }

        const network: Network = {
            name: ind.name,
            enabled: false,
            server: formData.irc.server,
            port: formData.irc.port,
            tls: formData.irc.tls,
            nickserv: formData.irc.nickserv,
            invite_command: formData.irc.invite_command,
            settings: formData.irc.settings,
            channels: channels,
        }

        console.log("network: ", network);


        mutation.mutate(formData, {
            onSuccess: (data) => {
                // create irc 
                ircMutation.mutate(network)
            }
        })

    };

    const renderSettingFields = (indexer: string) => {
        if (indexer !== "") {
            let ind = data && data.find(i => i.identifier === indexer)

            return (
                <div key="opt">
                    {ind && ind.settings && ind.settings.map((f: any, idx: number) => {
                        switch (f.type) {
                            case "text":
                                return (
                                    <TextFieldWide name={`settings.${f.name}`} label={f.label} key={idx} help={f.help} defaultValue=""/>
                                )
                            case "secret":
                                return (
                                    <PasswordFieldWide name={`settings.${f.name}`} label={f.label} key={idx} help={f.help} defaultValue="" />
                                )
                        }
                    })}
                    <div hidden={true}>
                        <TextFieldWide name={`name`} label="Name" defaultValue={ind?.name} />
                    </div>
                </div>
            )
        }
    }

    const renderIrcSettingFields = (indexer: string) => {

        if (indexer !== "") {
            let ind = data && data.find(i => i.identifier === indexer)

            return (
                <Fragment>
                    {ind && ind.irc && ind.irc.settings && (
                        <div className="border-t border-gray-200 py-5">
                            <div className="px-6 space-y-1">
                                <Dialog.Title className="text-lg font-medium text-gray-900">IRC</Dialog.Title>
                                <p className="text-sm text-gray-500">
                                    Networks, channels and invite commands are configured automatically.
                                </p>
                            </div>
                            {ind.irc.settings.map((f: IndexerSchemaSettings, idx: number) => {
                                switch (f.type) {
                                    case "text":
                                        return <TextFieldWide name={`irc.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} />
                                    case "secret":
                                        return <PasswordFieldWide name={`irc.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} defaultValue={f.default} />
                                }
                            })}

                            <div hidden={true}>
                                <TextFieldWide name={`irc.server`} label="Server" defaultValue={ind.irc.server} />
                                <NumberFieldWide name={`irc.port`} label="Port" defaultValue={ind.irc.port} />
                                <SwitchGroup name="irc.tls" label="TLS" defaultValue={ind.irc.tls} />
                            </div>
                        </div>
                    )}
                </Fragment>
            )
        }
    }

    return (
        <Transition.Root show={isOpen} as={Fragment}>
            <Dialog as="div" static className="fixed inset-0 overflow-hidden" open={isOpen} onClose={toggle}>
                <div className="absolute inset-0 overflow-hidden">
                    <Dialog.Overlay className="absolute inset-0" />

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
                                        enabled: true,
                                        identifier: "",
                                        irc: {}
                                    }}
                                    onSubmit={onSubmit}
                                >
                                    {({ handleSubmit, values }) => {
                                        return (
                                            <form className="h-full flex flex-col bg-white shadow-xl overflow-y-scroll"
                                                onSubmit={handleSubmit}>
                                                <div className="flex-1">
                                                    <div className="px-4 py-6 bg-gray-50 sm:px-6">
                                                        <div className="flex items-start justify-between space-x-3">
                                                            <div className="space-y-1">
                                                                <Dialog.Title
                                                                    className="text-lg font-medium text-gray-900">Add
                                                                    indexer</Dialog.Title>
                                                                <p className="text-sm text-gray-500">
                                                                    Add indexer.
                                                                </p>
                                                            </div>
                                                            <div className="h-7 flex items-center">
                                                                <button
                                                                    type="button"
                                                                    className="bg-white rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                                                                    onClick={toggle}
                                                                >
                                                                    <span className="sr-only">Close panel</span>
                                                                    <XIcon className="h-6 w-6" aria-hidden="true" />
                                                                </button>
                                                            </div>
                                                        </div>
                                                    </div>

                                                    <div
                                                        className="py-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">

                                                        <div
                                                            className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                                                            <div>
                                                                <label
                                                                    htmlFor="identifier"
                                                                    className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                                                >
                                                                    Indexer
                                                                </label>
                                                            </div>
                                                            <div className="sm:col-span-2">
                                                                <Field
                                                                    name="identifier"
                                                                    parse={val => val && val.value}
                                                                    format={val => data && data.find((o: any) => o.value === val)}
                                                                    render={({ input, meta }) => (
                                                                        <React.Fragment>
                                                                            <Select {...input}
                                                                                isClearable={true}
                                                                                placeholder="Choose an indexer"

                                                                                options={data && data.sort((a, b): any => a.name.localeCompare(b.name)).map(v => ({
                                                                                    label: v.name,
                                                                                    value: v.identifier
                                                                                }))} />
                                                                        </React.Fragment>
                                                                    )}
                                                                />
                                                            </div>
                                                        </div>

                                                        <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                                                            <SwitchGroup name="enabled" label="Enabled" />
                                                        </div>


                                                        {renderSettingFields(values.identifier)}

                                                    </div>

                                                    {renderIrcSettingFields(values.identifier)}
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

                                                <DEBUG values={values} />
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

export default IndexerAddForm;
