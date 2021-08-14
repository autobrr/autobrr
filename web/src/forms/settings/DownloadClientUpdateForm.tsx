import {Fragment, useRef, useState} from "react";
import {useToggle} from "../../hooks/hooks";
import {useMutation} from "react-query";
import {DownloadClient} from "../../domain/interfaces";
import {queryClient} from "../../App";
import {Dialog, RadioGroup, Transition} from "@headlessui/react";
import {ExclamationIcon, XIcon} from "@heroicons/react/solid";
import {classNames} from "../../styles/utils";
import {Field, Form} from "react-final-form";
import DEBUG from "../../components/debug";
import {SwitchGroup} from "../../components/inputs";
import {DownloadClientTypeOptions} from "../../domain/constants";
import APIClient from "../../api/APIClient";
import {sleep} from "../../utils/utils";

function DownloadClientUpdateForm({client, isOpen, toggle}: any) {
    const [isTesting, setIsTesting] = useState(false)
    const [isSuccessfulTest, setIsSuccessfulTest] = useState(false)
    const [isErrorTest, setIsErrorTest] = useState(false)
    const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false)

    const mutation = useMutation((client: DownloadClient) => APIClient.download_clients.update(client), {
        onSuccess: () => {
            queryClient.invalidateQueries(['downloadClients']);

            toggle()
        }
    })

    const deleteMutation = useMutation((clientID: number) => APIClient.download_clients.delete(clientID), {
        onSuccess: () => {
            queryClient.invalidateQueries();
            toggleDeleteModal()
        }
    })

    const testClientMutation = useMutation((client: DownloadClient) => APIClient.download_clients.test(client), {
        onMutate: () => {
            setIsTesting(true)
            setIsErrorTest(false)
            setIsSuccessfulTest(false)
        },
        onSuccess: () => {
            sleep(1000).then(() => {
                setIsTesting(false)
                setIsSuccessfulTest(true)
            }).then(() => {
                sleep(2500).then(() => {
                    setIsSuccessfulTest(false)
                })
            })
        },
        onError: (error) => {
            setIsTesting(false)
            setIsErrorTest(true)
            sleep(2500).then(() => {
                setIsErrorTest(false)
            })
        },
    })

    const onSubmit = (data: any) => {
        mutation.mutate(data)
    };

    const cancelButtonRef = useRef(null)
    const cancelModalButtonRef = useRef(null)

    const deleteAction = () => {
        deleteMutation.mutate(client.id)
    }

    const testClient = (data: any) => {
        testClientMutation.mutate(data)
    }

    return (
        <Transition.Root show={isOpen} as={Fragment}>
            <Dialog as="div" static className="fixed inset-0 overflow-hidden" open={isOpen} onClose={toggle}
                    initialFocus={cancelButtonRef}>
                <Transition.Root show={deleteModalIsOpen} as={Fragment}>
                    <Dialog
                        as="div"
                        static
                        className="fixed z-10 inset-0 overflow-y-auto"
                        initialFocus={cancelModalButtonRef}
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
                                                    Remove client
                                                </Dialog.Title>
                                                <div className="mt-2">
                                                    <p className="text-sm text-gray-500">
                                                        Are you sure you want to remove this client?
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
                                        id: client.id,
                                        name: client.name,
                                        type: client.type,
                                        enabled: client.enabled,
                                        host: client.host,
                                        port: client.port,
                                        ssl: client.ssl,
                                        username: client.username,
                                        password: client.password
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
                                                                    className="text-lg font-medium text-gray-900">Edit
                                                                    client</Dialog.Title>
                                                                <p className="text-sm text-gray-500">
                                                                    Edit download client settings.
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

                                                        <div
                                                            className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                                                            <SwitchGroup name="enabled" label="Enabled"/>
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
                                                                                <RadioGroup value={values.type}
                                                                                            onChange={input.onChange}>
                                                                                    <RadioGroup.Label
                                                                                        className="sr-only">Privacy
                                                                                        setting</RadioGroup.Label>
                                                                                    <div
                                                                                        className="bg-white rounded-md -space-y-px">
                                                                                        {DownloadClientTypeOptions.map((setting, settingIdx) => (
                                                                                            <RadioGroup.Option
                                                                                                key={setting.value}
                                                                                                value={setting.value}
                                                                                                className={({checked}) =>
                                                                                                    classNames(
                                                                                                        settingIdx === 0 ? 'rounded-tl-md rounded-tr-md' : '',
                                                                                                        settingIdx === DownloadClientTypeOptions.length - 1 ? 'rounded-bl-md rounded-br-md' : '',
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
                                                                                                          <span
                                                                                                              className="rounded-full bg-white w-1.5 h-1.5"/>
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

                                                        <div>

                                                            <div
                                                                className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                                                                <div>
                                                                    <label
                                                                        htmlFor="host"
                                                                        className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                                                    >
                                                                        Host
                                                                    </label>
                                                                </div>
                                                                <Field name="host">
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


                                                            <div
                                                                className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                                                                <div>
                                                                    <label
                                                                        htmlFor="port"
                                                                        className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                                                    >
                                                                        Port
                                                                    </label>
                                                                </div>
                                                                <Field name="port" parse={(v) => v && parseInt(v, 10)}>
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

                                                            <div
                                                                className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                                                                <SwitchGroup name="ssl" label="SSL"/>
                                                            </div>

                                                            <div
                                                                className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                                                                <div>
                                                                    <label
                                                                        htmlFor="username"
                                                                        className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                                                    >
                                                                        Username
                                                                    </label>
                                                                </div>
                                                                <Field name="username">
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

                                                            <div
                                                                className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                                                                <div>
                                                                    <label
                                                                        htmlFor="password"
                                                                        className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2"
                                                                    >
                                                                        Password
                                                                    </label>
                                                                </div>
                                                                <Field name="password">
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
                                                    </div>
                                                </div>

                                                <div
                                                    className="flex-shrink-0 px-4 border-t border-gray-200 py-5 sm:px-6">
                                                    <div className="space-x-3 flex justify-between">
                                                        <button
                                                            type="button"
                                                            className="inline-flex items-center justify-center px-4 py-2 border border-transparent font-medium rounded-md text-red-700 bg-red-100 hover:bg-red-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:text-sm"
                                                            onClick={toggleDeleteModal}
                                                        >
                                                            Remove
                                                        </button>
                                                        <div className="flex">
                                                            <button
                                                                type="button"
                                                                className={classNames(isSuccessfulTest ? "text-green-500 border-green-500 bg-green-50" : (isErrorTest ? "text-red-500 border-red-500 bg-red-50" : "border-gray-300 text-gray-700 bg-white hover:bg-gray-50 focus:border-rose-700 active:bg-rose-700"), isTesting ? "cursor-not-allowed" : "", "mr-2 inline-flex items-center px-4 py-2 border font-medium rounded-md shadow-sm text-sm transition ease-in-out duration-150")}
                                                                disabled={isTesting}
                                                                onClick={() => testClient(values)}
                                                            >
                                                                {isTesting ?
                                                                    <svg
                                                                        className="animate-spin h-5 w-5 text-green-500"
                                                                        xmlns="http://www.w3.org/2000/svg" fill="none"
                                                                        viewBox="0 0 24 24">
                                                                        <circle className="opacity-25" cx="12" cy="12"
                                                                                r="10" stroke="currentColor"
                                                                                strokeWidth="4"></circle>
                                                                        <path className="opacity-75" fill="currentColor"
                                                                              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                                                                    </svg>
                                                                    : (isSuccessfulTest ? "OK!" : (isErrorTest ? "ERROR" : "Test"))
                                                                }
                                                            </button>

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

export default DownloadClientUpdateForm;