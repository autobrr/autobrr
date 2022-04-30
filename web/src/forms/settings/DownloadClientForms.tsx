import { Fragment, useRef, useState } from "react";
import { useMutation } from "react-query";
import { Dialog, Transition } from "@headlessui/react";
import { XIcon } from "@heroicons/react/solid";
import { sleep, classNames } from "../../utils";
import { Form, Formik, useFormikContext } from "formik";
import DEBUG from "../../components/debug";
import { queryClient } from "../../App";
import { APIClient } from "../../api/APIClient";
import { DownloadClientTypeOptions } from "../../domain/constants";

import { toast } from 'react-hot-toast'
import Toast from '../../components/notifications/Toast';
import { useToggle } from "../../hooks/hooks";
import { DeleteModal } from "../../components/modals";
import { NumberFieldWide, PasswordFieldWide, SwitchGroupWide, TextFieldWide } from "../../components/inputs/input_wide";
import { RadioFieldsetWide } from "../../components/inputs/radio";

interface InitialValuesSettings {
    basic?: {
        auth: boolean;
        username: string;
        password: string;
    };
    rules?: {
        enabled?: boolean;
        ignore_slow_torrents?: boolean;
        download_speed_threshold?: number;
        max_active_downloads?: number;
    };
}

interface InitialValues {
    name: string;
    type: DownloadClientType;
    enabled: boolean;
    host: string;
    port: number;
    tls: boolean;
    tls_skip_verify: boolean;
    username: string;
    password: string;
    settings: InitialValuesSettings;
}


function FormFieldsDefault() {
    const {
        values: { tls },
    } = useFormikContext<InitialValues>();

    return (
        <Fragment>
            <TextFieldWide name="host" label="Host" help="Eg. client.domain.ltd, domain.ltd/client, domain.ltd:port" />

            <NumberFieldWide name="port" label="Port" help="WebUI port for qBittorrent and daemon port for Deluge" />

            <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200 dark:divide-gray-700">
                <SwitchGroupWide name="tls" label="TLS" />

                {tls && (
                    <Fragment>
                        <SwitchGroupWide name="tls_skip_verify" label="Skip TLS verification (insecure)" />
                    </Fragment>
                )}
            </div>

            <TextFieldWide name="username" label="Username" />
            <PasswordFieldWide name="password" label="Password" />
        </Fragment>
    );
}

function FormFieldsQbit() {
    const {
        values: { tls, settings },
    } = useFormikContext<InitialValues>();

    return (
        <Fragment>
            <TextFieldWide name="host" label="Host" help="Eg. client.domain.ltd, domain.ltd/client, domain.ltd:port" />

            <NumberFieldWide name="port" label="Port" help="WebUI port for qBittorrent and daemon port for Deluge" />

            <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200 dark:divide-gray-700">
                <SwitchGroupWide name="tls" label="TLS" />

                {tls && (
                    <Fragment>
                        <SwitchGroupWide name="tls_skip_verify" label="Skip TLS verification (insecure)" />
                    </Fragment>
                )}
            </div>

            <TextFieldWide name="username" label="Username" />
            <PasswordFieldWide name="password" label="Password" />

            <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                <SwitchGroupWide name="settings.basic.auth" label="Basic auth" />
            </div>

            {settings.basic?.auth === true && (
                <Fragment>
                    <TextFieldWide name="settings.basic.username" label="Username" />
                    <PasswordFieldWide name="settings.basic.password" label="Password" />
                </Fragment>
            )}
        </Fragment>
    );
}

function FormFieldsArr() {
    const {
        values: { settings },
    } = useFormikContext<InitialValues>();

    return (
        <Fragment>
            <TextFieldWide name="host" label="Host" help="Full url http(s)://domain.ltd and/or subdomain/subfolder" />

            <PasswordFieldWide name="settings.apikey" label="API key" />

            <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                <SwitchGroupWide name="settings.basic.auth" label="Basic auth" />
            </div>

            {settings.basic?.auth === true && (
                <Fragment>
                    <TextFieldWide name="settings.basic.username" label="Username" />
                    <PasswordFieldWide name="settings.basic.password" label="Password" />
                </Fragment>
            )}
        </Fragment>
    );
}

export const componentMap: any = {
    DELUGE_V1: <FormFieldsDefault />,
    DELUGE_V2: <FormFieldsDefault />,
    QBITTORRENT: <FormFieldsQbit />,
    RADARR: <FormFieldsArr />,
    SONARR: <FormFieldsArr />,
    LIDARR: <FormFieldsArr />,
    WHISPARR: <FormFieldsArr />,
};


function FormFieldsRulesBasic() {
    const {
        values: { settings },
    } = useFormikContext<InitialValues>();

    return (
        <div className="border-t border-gray-200 dark:border-gray-700 py-5">

            <div className="px-6 space-y-1">
                <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">Rules</Dialog.Title>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                    Manage max downloads.
                </p>
            </div>

            <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                <SwitchGroupWide name="settings.rules.enabled" label="Enabled" />
            </div>

            {settings && settings.rules?.enabled === true && (
                <Fragment>
                    <NumberFieldWide name="settings.rules.max_active_downloads" label="Max active downloads" />
                </Fragment>
            )}
        </div>
    );
}

function FormFieldsRules() {
    const {
        values: { settings },
    } = useFormikContext<InitialValues>();

    return (
        <div className="border-t border-gray-200 dark:border-gray-700 py-5">

            <div className="px-6 space-y-1">
                <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">Rules</Dialog.Title>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                    Manage max downloads etc.
                </p>
            </div>

            <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                <SwitchGroupWide name="settings.rules.enabled" label="Enabled" />
            </div>

            {settings.rules?.enabled === true && (
                <Fragment>
                    <NumberFieldWide name="settings.rules.max_active_downloads" label="Max active downloads" />
                    <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                        <SwitchGroupWide name="settings.rules.ignore_slow_torrents" label="Ignore slow torrents" />
                    </div>

                    {settings.rules?.ignore_slow_torrents === true && (
                        <Fragment>
                            <NumberFieldWide name="settings.rules.download_speed_threshold" label="Download speed threshold" placeholder="in KB/s" help="If download speed is below this when max active downloads is hit, download anyways. KB/s" />
                        </Fragment>
                    )}
                </Fragment>
            )}
        </div>
    );
}

export const rulesComponentMap: any = {
    DELUGE_V1: <FormFieldsRulesBasic />,
    DELUGE_V2: <FormFieldsRulesBasic />,
    QBITTORRENT: <FormFieldsRules />,
};

interface formButtonsProps {
    isSuccessfulTest: boolean;
    isErrorTest: boolean;
    isTesting: boolean;
    cancelFn: any;
    testFn: any;
    values: any;
    type: "CREATE" | "UPDATE";
    toggleDeleteModal?: any;
}

function DownloadClientFormButtons({ type, isSuccessfulTest, isErrorTest, isTesting, cancelFn, testFn, values, toggleDeleteModal }: formButtonsProps) {

    const test = () => {
        testFn(values)
    }

    return (
        <div className="flex-shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-5 sm:px-6">
            <div className={classNames(type === "CREATE" ? "justify-end" : "justify-between", "space-x-3 flex")}>
                {type === "UPDATE" && (
                    <button
                        type="button"
                        className="inline-flex items-center justify-center px-4 py-2 border border-transparent font-medium rounded-md text-red-700 dark:text-white bg-red-100 dark:bg-red-700 hover:bg-red-200 dark:hover:bg-red-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:text-sm"
                        onClick={toggleDeleteModal}
                    >
                        Remove
                    </button>
                )}
                <div className="flex">
                    <button
                        type="button"
                        className={classNames(
                            isSuccessfulTest
                                ? "text-green-500 border-green-500 bg-green-50"
                                : isErrorTest
                                    ? "text-red-500 border-red-500 bg-red-50"
                                    : "border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-400 bg-white dark:bg-gray-700 hover:bg-gray-50 focus:border-rose-700 active:bg-rose-700",
                            isTesting ? "cursor-not-allowed" : "",
                            "mr-2 inline-flex items-center px-4 py-2 border font-medium rounded-md shadow-sm text-sm transition ease-in-out duration-150 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                        )}
                        disabled={isTesting}
                        // onClick={() => testClient(values)}
                        onClick={test}
                    >
                        {isTesting ? (
                            <svg
                                className="animate-spin h-5 w-5 text-green-500"
                                xmlns="http://www.w3.org/2000/svg"
                                fill="none"
                                viewBox="0 0 24 24"
                            >
                                <circle
                                    className="opacity-25"
                                    cx="12"
                                    cy="12"
                                    r="10"
                                    stroke="currentColor"
                                    strokeWidth="4"
                                ></circle>
                                <path
                                    className="opacity-75"
                                    fill="currentColor"
                                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                                ></path>
                            </svg>
                        ) : isSuccessfulTest ? (
                            "OK!"
                        ) : isErrorTest ? (
                            "ERROR"
                        ) : (
                            "Test"
                        )}
                    </button>

                    <button
                        type="button"
                        className="mr-4 bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                        onClick={cancelFn}
                    >
                        Cancel
                    </button>
                    <button
                        type="submit"
                        className="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 dark:bg-blue-600 hover:bg-indigo-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                    >
                        {type === "CREATE" ? "Create" : "Save"}
                    </button>
                </div>
            </div>
        </div>
    )
}

export function DownloadClientAddForm({ isOpen, toggle }: any) {
    const [isTesting, setIsTesting] = useState(false);
    const [isSuccessfulTest, setIsSuccessfulTest] = useState(false);
    const [isErrorTest, setIsErrorTest] = useState(false);

    const mutation = useMutation(
        (client: DownloadClient) => APIClient.download_clients.create(client),
        {
            onSuccess: () => {
                queryClient.invalidateQueries(["downloadClients"]);
                toast.custom((t) => <Toast type="success" body="Client was added" t={t} />)

                toggle();
            },
            onError: () => {
                toast.custom((t) => <Toast type="error" body="Client could not be added" t={t} />)
            }
        }
    );

    const testClientMutation = useMutation(
        (client: DownloadClient) => APIClient.download_clients.test(client),
        {
            onMutate: () => {
                setIsTesting(true);
                setIsErrorTest(false);
                setIsSuccessfulTest(false);
            },
            onSuccess: () => {
                sleep(1000)
                    .then(() => {
                        setIsTesting(false);
                        setIsSuccessfulTest(true);
                    })
                    .then(() => {
                        sleep(2500).then(() => {
                            setIsSuccessfulTest(false);
                        });
                    });
            },
            onError: () => {
                console.log('not added')
                setIsTesting(false);
                setIsErrorTest(true);
                sleep(2500).then(() => {
                    setIsErrorTest(false);
                });
            },
        }
    );

    const onSubmit = (data: any) => {
        mutation.mutate(data);
    };

    const testClient = (data: any) => {
        testClientMutation.mutate(data);
    };

    const initialValues: InitialValues = {
        name: "",
        type: "QBITTORRENT",
        enabled: true,
        host: "",
        port: 10000,
        tls: false,
        tls_skip_verify: false,
        username: "",
        password: "",
        settings: {}
    }

    return (
        <Transition.Root show={isOpen} as={Fragment}>
            <Dialog
                as="div"
                static
                className="fixed inset-0 overflow-hidden"
                open={isOpen}
                onClose={toggle}
            >
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
                            <div className="w-screen max-w-2xl border-l dark:border-gray-700">
                                <Formik
                                    initialValues={initialValues}
                                    onSubmit={onSubmit}
                                >
                                    {({ handleSubmit, values }) => (
                                        <Form
                                            className="h-full flex flex-col bg-white dark:bg-gray-800 shadow-xl overflow-y-scroll"
                                            onSubmit={handleSubmit}
                                        >
                                            <div className="flex-1">
                                                <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
                                                    <div className="flex items-start justify-between space-x-3">
                                                        <div className="space-y-1">
                                                            <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">
                                                                Add client
                                                            </Dialog.Title>
                                                            <p className="text-sm text-gray-500 dark:text-gray-400">
                                                                Add download client.
                                                            </p>
                                                        </div>
                                                        <div className="h-7 flex items-center">
                                                            <button
                                                                type="button"
                                                                className="bg-white dark:bg-gray-800 rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                                                                onClick={toggle}
                                                            >
                                                                <span className="sr-only">Close panel</span>
                                                                <XIcon
                                                                    className="h-6 w-6"
                                                                    aria-hidden="true"
                                                                />
                                                            </button>
                                                        </div>
                                                    </div>
                                                </div>

                                                <div className="py-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y dark:divide-gray-700">
                                                    <TextFieldWide name="name" label="Name" />

                                                    <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200 dark:divide-gray-700">
                                                        <SwitchGroupWide name="enabled" label="Enabled" />
                                                    </div>

                                                    <RadioFieldsetWide
                                                        name="type"
                                                        legend="Type"
                                                        options={DownloadClientTypeOptions}
                                                    />

                                                    <div>{componentMap[values.type]}</div>
                                                </div>
                                            </div>

                                            {rulesComponentMap[values.type]}

                                            <DownloadClientFormButtons
                                                type="CREATE"
                                                isTesting={isTesting}
                                                isSuccessfulTest={isSuccessfulTest}
                                                isErrorTest={isErrorTest}
                                                cancelFn={toggle}
                                                testFn={testClient}
                                                values={values}
                                            />

                                            <DEBUG values={values} />
                                        </Form>
                                    )}
                                </Formik>
                            </div>
                        </Transition.Child>
                    </div>
                </div>
            </Dialog>
        </Transition.Root>
    );
}

export function DownloadClientUpdateForm({ client, isOpen, toggle }: any) {
    const [isTesting, setIsTesting] = useState(false);
    const [isSuccessfulTest, setIsSuccessfulTest] = useState(false);
    const [isErrorTest, setIsErrorTest] = useState(false);
    const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);

    const mutation = useMutation(
        (client: DownloadClient) => APIClient.download_clients.update(client),
        {
            onSuccess: () => {
                queryClient.invalidateQueries(["downloadClients"]);
                toast.custom((t) => <Toast type="success" body={`${client.name} was updated successfully`} t={t} />)
                toggle();
            },
        }
    );

    const deleteMutation = useMutation(
        (clientID: number) => APIClient.download_clients.delete(clientID),
        {
            onSuccess: () => {
                queryClient.invalidateQueries();
                toast.custom((t) => <Toast type="success" body={`${client.name} was deleted.`} t={t} />)
                toggleDeleteModal();
            },
        }
    );

    const testClientMutation = useMutation(
        (client: DownloadClient) => APIClient.download_clients.test(client),
        {
            onMutate: () => {
                setIsTesting(true);
                setIsErrorTest(false);
                setIsSuccessfulTest(false);
            },
            onSuccess: () => {
                sleep(1000)
                    .then(() => {
                        setIsTesting(false);
                        setIsSuccessfulTest(true);
                    })
                    .then(() => {
                        sleep(2500).then(() => {
                            setIsSuccessfulTest(false);
                        });
                    });
            },
            onError: () => {
                setIsTesting(false);
                setIsErrorTest(true);
                sleep(2500).then(() => {
                    setIsErrorTest(false);
                });
            },
        }
    );

    const onSubmit = (data: any) => {
        mutation.mutate(data);
    };

    const cancelButtonRef = useRef(null);
    const cancelModalButtonRef = useRef(null);

    const deleteAction = () => {
        deleteMutation.mutate(client.id);
    };

    const testClient = (data: any) => {
        testClientMutation.mutate(data);
    };

    const initialValues = {
        id: client.id,
        name: client.name,
        type: client.type,
        enabled: client.enabled,
        host: client.host,
        port: client.port,
        tls: client.tls,
        tls_skip_verify: client.tls_skip_verify,
        username: client.username,
        password: client.password,
        settings: client.settings,
    }

    return (
        <Transition.Root show={isOpen} as={Fragment}>
            <Dialog
                as="div"
                static
                className="fixed inset-0 overflow-hidden"
                open={isOpen}
                onClose={toggle}
                initialFocus={cancelButtonRef}
            >
                <DeleteModal
                    isOpen={deleteModalIsOpen}
                    toggle={toggleDeleteModal}
                    buttonRef={cancelModalButtonRef}
                    deleteAction={deleteAction}
                    title="Remove download client"
                    text="Are you sure you want to remove this download client? This action cannot be undone."
                />
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
                            <div className="w-screen max-w-2xl border-l dark:border-gray-700">
                                <Formik
                                    initialValues={initialValues}
                                    onSubmit={onSubmit}
                                >
                                    {({ handleSubmit, values }) => {
                                        return (
                                            <Form
                                                className="h-full flex flex-col bg-white dark:bg-gray-800 shadow-xl overflow-y-scroll"
                                                onSubmit={handleSubmit}
                                            >
                                                <div className="flex-1">
                                                    <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
                                                        <div className="flex items-start justify-between space-x-3">
                                                            <div className="space-y-1">
                                                                <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">
                                                                    Edit client
                                                                </Dialog.Title>
                                                                <p className="text-sm text-gray-500 dark:text-gray-400">
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
                                                                    <XIcon
                                                                        className="h-6 w-6"
                                                                        aria-hidden="true"
                                                                    />
                                                                </button>
                                                            </div>
                                                        </div>
                                                    </div>

                                                    <div className="py-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y dark:divide-gray-700">
                                                        <TextFieldWide name="name" label="Name" />

                                                        <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                                                            <SwitchGroupWide name="enabled" label="Enabled" />
                                                        </div>

                                                        <RadioFieldsetWide
                                                            name="type"
                                                            legend="Type"
                                                            options={DownloadClientTypeOptions}
                                                        />

                                                        <div>{componentMap[values.type]}</div>
                                                    </div>
                                                </div>

                                                {rulesComponentMap[values.type]}

                                                <DownloadClientFormButtons
                                                    type="UPDATE"
                                                    toggleDeleteModal={toggleDeleteModal}
                                                    isTesting={isTesting}
                                                    isSuccessfulTest={isSuccessfulTest}
                                                    isErrorTest={isErrorTest}
                                                    cancelFn={toggle}
                                                    testFn={testClient}
                                                    values={values}
                                                />

                                                <DEBUG values={values} />
                                            </Form>
                                        );
                                    }}
                                </Formik>
                            </div>
                        </Transition.Child>
                    </div>
                </div>
            </Dialog>
        </Transition.Root>
    );
}
