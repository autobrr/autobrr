import {Fragment, useRef} from "react";
import {useMutation } from "react-query";
import {Indexer} from "../../domain/interfaces";
import {sleep} from "../../utils/utils";
import {ExclamationIcon, XIcon} from "@heroicons/react/solid";
import {Dialog, Transition} from "@headlessui/react";
import {Field, Form} from "react-final-form";
import DEBUG from "../../components/debug";
import { SwitchGroup } from "../../components/inputs";
import {useToggle} from "../../hooks/hooks";
import APIClient from "../../api/APIClient";
import {queryClient} from "../../App";

interface props {
    isOpen: boolean;
    toggle: any;
    indexer: Indexer;
}

function IndexerUpdateForm({isOpen, toggle, indexer}: props) {
    const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false)

    const mutation = useMutation((indexer: Indexer) => APIClient.indexers.update(indexer), {
        onSuccess: () => {
            queryClient.invalidateQueries(['indexer']);
            sleep(1500)

            toggle()
        }
    })

    const deleteMutation = useMutation((id: number) => APIClient.indexers.delete(id), {
        onSuccess: () => {
            queryClient.invalidateQueries(['indexer']);
        }
    })

    const cancelModalButtonRef = useRef(null)

    const onSubmit = (data: any) => {
        // TODO clear data depending on type
        mutation.mutate(data)
    };

    const deleteAction = () => {
        deleteMutation.mutate(indexer.id)
    }

    const renderSettingFields = (settings: any[]) => {
        if (settings !== []) {

            return (
                <div key="opt">
                    {settings && settings.map((f: any, idx: number) => {
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

    // const setss = indexer.settings.reduce((o: any, obj: any) => ({ ...o, [obj.name]: obj.value }), {})
    // console.log("setts", setss)

    return (
        <Transition.Root show={isOpen} as={Fragment}>
            <Dialog as="div" static className="fixed inset-0 overflow-hidden" open={isOpen} onClose={toggle}>

                <Transition.Root show={deleteModalIsOpen} as={Fragment}>
                    <Dialog
                        as="div"
                        static
                        className="fixed z-10 inset-0 overflow-y-auto"
                        initialFocus={cancelModalButtonRef}
                        open={deleteModalIsOpen}
                        onClose={toggleDeleteModal}
                    >
                        <div className="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
                            <Transition.Child
                                as={Fragment}
                                enter="ease-out duration-300"
                                enterFrom="opacity-0"
                                enterTo="opacity-100"
                                leave="ease-in duration-200"
                                leaveFrom="opacity-100"
                                leaveTo="opacity-0"
                            >
                                <Dialog.Overlay className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity" />
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
                                <div className="inline-block align-bottom bg-white rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
                                    <div className="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
                                        <div className="sm:flex sm:items-start">
                                            <div className="mx-auto flex-shrink-0 flex items-center justify-center h-12 w-12 rounded-full bg-red-100 sm:mx-0 sm:h-10 sm:w-10">
                                                <ExclamationIcon className="h-6 w-6 text-red-600" aria-hidden="true" />
                                            </div>
                                            <div className="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left">
                                                <Dialog.Title as="h3" className="text-lg leading-6 font-medium text-gray-900">
                                                    Remove indexer
                                                </Dialog.Title>
                                                <div className="mt-2">
                                                    <p className="text-sm text-gray-500">
                                                        Are you sure you want to remove this indexer?
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
                                            ref={cancelModalButtonRef}
                                        >
                                            Cancel
                                        </button>
                                    </div>
                                </div>
                            </Transition.Child>
                        </div>
                    </Dialog>
                </Transition.Root>
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
                                        id: indexer.id,
                                        name: indexer.name,
                                        enabled: indexer.enabled,
                                        identifier: indexer.identifier,
                                        settings: indexer.settings.reduce((o: any, obj: any) => ({ ...o, [obj.name]: obj.value }), {}),
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
                                                                    className="text-lg font-medium text-gray-900">Update
                                                                    indexer</Dialog.Title>
                                                                <p className="text-sm text-gray-500">
                                                                    Update indexer.
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

                                                        {renderSettingFields(indexer.settings)}

                                                    </div>
                                                </div>

                                                <div className="flex-shrink-0 px-4 border-t border-gray-200 py-5 sm:px-6">
                                                    <div className="space-x-3 flex justify-between">
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
                                                                className="bg-white py-2 px-4 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                                                                onClick={toggle}
                                                            >
                                                                Cancel
                                                            </button>
                                                            <button
                                                                type="submit"
                                                                className="ml-4 inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
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

                        </Transition.Child>
                    </div>
                </div>
            </Dialog>
        </Transition.Root>
    )
}

export default IndexerUpdateForm;
