import React, {Fragment, useEffect} from "react";
import {useMutation, useQuery} from "react-query";
import {Indexer} from "../../domain/interfaces";
import {sleep} from "../../utils/utils";
import {XIcon} from "@heroicons/react/solid";
import {Dialog, Transition} from "@headlessui/react";
import {Field, Form} from "react-final-form";
import DEBUG from "../../components/debug";
import Select from "react-select";
import {queryClient} from "../../index";
import { SwitchGroup } from "../../components/inputs";
import APIClient from "../../api/APIClient";

interface props {
    isOpen: boolean;
    toggle: any;
}

function IndexerAddForm({isOpen, toggle}: props) {
    const {data} = useQuery<any[], Error>('indexerSchema', APIClient.indexers.getSchema,
        {
            enabled: isOpen,
            refetchOnWindowFocus: false
        }
    )

    const mutation = useMutation((indexer: Indexer) => APIClient.indexers.create(indexer), {
        onSuccess: () => {
            queryClient.invalidateQueries(['indexer']);
            sleep(1500)

            toggle()
        }
    })

    const onSubmit = (data: any) => {
        mutation.mutate(data)
    };

    const renderSettingFields = (indexer: string) => {
        if (indexer !== "") {
            // let ind = data.find(i => i.implementation_name === indexer)
            let ind = data && data.find(i => i.identifier === indexer)

            return (
                <div key="opt">
                {ind && ind.settings && ind.settings.map((f: any, idx: number) => {
                    switch (f.type) {
                        case "text":
                           return (
                                <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5" key={idx}>
                                    <div>
                                        <label
                                            htmlFor={f.name}
                                            className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                        >
                                            {f.label}
                                        </label>
                                    </div>
                                    <div className="sm:col-span-2">
                                        <Field name={"settings."+f.name}>
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
                               )
                    }
                })}
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
                                        enabled: true,
                                        identifier: "",
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
                                                                    Name
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

                                                        <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                                                            <SwitchGroup name="enabled" label="Enabled" />
                                                        </div>

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
                                                                    render={({input, meta}) => (
                                                                        <React.Fragment>
                                                                            <Select {...input}
                                                                                    isClearable={true}
                                                                                    placeholder="Choose an indexer"
                                                                                    options={data && data.sort((a,b): any => a.name.localeCompare(b.name)).map(v => ({
                                                                                        label: v.name,
                                                                                        value: v.identifier
                                                                                        // value: v.implementation_name
                                                                                    }))}/>
                                                                            {/*<Error name={input.name} classNames="text-red mt-2 block" />*/}
                                                                        </React.Fragment>
                                                                    )}
                                                                />
                                                            </div>
                                                        </div>

                                                        {renderSettingFields(values.identifier)}

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

export default IndexerAddForm;
