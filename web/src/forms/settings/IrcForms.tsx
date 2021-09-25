import { useMutation } from "react-query";
import { Network } from "../../domain/interfaces";
import { XIcon } from "@heroicons/react/solid";
import { Field } from "react-final-form";
import { SwitchGroup, TextFieldWide } from "../../components/inputs";
import { queryClient } from "../../App";

import arrayMutators from "final-form-arrays";
import { FieldArray } from "react-final-form-arrays";
import APIClient from "../../api/APIClient";
import { NumberFieldWide, PasswordFieldWide } from "../../components/inputs/wide";

import { toast } from 'react-hot-toast';
import Toast from '../../components/notifications/Toast';
import { SlideOver } from "../../components/panels";

export function IrcNetworkAddForm({ isOpen, toggle }: any) {
    const mutation = useMutation((network: Network) => APIClient.irc.createNetwork(network), {
        onSuccess: (data) => {
            queryClient.invalidateQueries(['networks']);
            toast.custom((t) => <Toast type="success" body="IRC Network added" t={t} />)
            toggle()
        },
        onError: () => {
            toast.custom((t) => <Toast type="error" body="IRC Network could not be added" t={t} />)
        },
    })

    const onSubmit = (data: any) => {
        // easy way to split textarea lines into array of strings for each newline.
        // parse on the field didn't really work.
        let cmds = data.connect_commands && data.connect_commands.length > 0 ? data.connect_commands.replace(/\r\n/g, "\n").split("\n") : [];
        data.connect_commands = cmds
        console.log("formated", data)

        mutation.mutate(data)
    };

    const validate = (values: any) => {
        const errors = {
            nickserv: {
                account: null,
            }
        } as any;

        if (!values.name) {
            errors.name = "Required";
        }

        if (!values.port) {
            errors.port = "Required";
        }

        if (!values.server) {
            errors.server = "Required";
        }

        if (!values.nickserv?.account) {
            errors.nickserv.account = "Required";
        }

        return errors;
    }

    const initialValues = {
        name: "",
        enabled: true,
        server: "",
        tls: false,
        pass: "",
        nickserv: {
            account: ""
        }
    }

    const mutators = {
        ...arrayMutators
    }

    return (
        <SlideOver
            type="CREATE"
            title="Network"
            isOpen={isOpen}
            toggle={toggle}
            onSubmit={onSubmit}
            initialValues={initialValues}
            mutators={mutators}
            validate={validate}
        >
            {() => (
                <>


                    <TextFieldWide name="name" label="Name" placeholder="Name" required={true} />

                    <div className="py-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y dark:divide-gray-700">

                        <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200 dark:sm:divide-gray-700">
                            <SwitchGroup name="enabled" label="Enabled" />
                        </div>

                        <div>
                            <TextFieldWide name="server" label="Server" placeholder="Address: Eg irc.server.net" required={true} />
                            <NumberFieldWide name="port" label="Port" placeholder="Eg 6667" required={true} />

                            <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                                <SwitchGroup name="tls" label="TLS" />
                            </div>

                            <PasswordFieldWide name="pass" label="Password" help="Network password" />

                            <TextFieldWide name="nickserv.account" label="NickServ Account" placeholder="NickServ Account" required={true} />
                            <PasswordFieldWide name="nickserv.password" label="NickServ Password" />

                            <PasswordFieldWide name="invite_command" label="Invite command" />
                        </div>
                    </div>

                    <div className="p-6">

                        <FieldArray name="channels">
                            {({ fields }) => (
                                <div className="flex flex-col border-2 border-dashed dark:border-gray-700 p-4">
                                    {fields && (fields.length as any) > 0 ? (
                                        fields.map((name, index) => (
                                            <div key={name} className="flex justify-between">
                                                <div className="flex">
                                                    <Field
                                                        name={`${name}.name`}
                                                        component="input"
                                                        type="text"
                                                        placeholder="#Channel"
                                                        className="mr-4 dark:bg-gray-700 focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-600 block w-full shadow-sm sm:text-sm dark:text-white rounded-md"
                                                    />
                                                    <Field
                                                        name={`${name}.password`}
                                                        component="input"
                                                        type="text"
                                                        placeholder="Password"
                                                        className="dark:bg-gray-700 focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-600 block w-full shadow-sm sm:text-sm dark:text-white rounded-md"
                                                    />
                                                </div>

                                                <button
                                                    type="button"
                                                    className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                                                    onClick={() => fields.remove(index)}
                                                >
                                                    <span className="sr-only">Remove</span>
                                                    <XIcon className="h-6 w-6" aria-hidden="true" />
                                                </button>
                                            </div>
                                        ))
                                    ) : (
                                        <span className="text-center text-sm text-grey-darker dark:text-white">
                                            No channels!
                                        </span>
                                    )}
                                    <button
                                        type="button"
                                        className="border dark:border-gray-600 dark:bg-gray-700 my-4 px-4 py-2 text-sm text-gray-700 dark:text-white hover:bg-gray-50 dark:hover:bg-gray-600 rounded self-center text-center"
                                        onClick={() => fields.push({ name: "", password: "" })}
                                    >
                                        Add Channel
                                    </button>
                                </div>
                            )}
                        </FieldArray>
                    </div>
                </>
            )}
        </SlideOver>
    )
}

export function IrcNetworkUpdateForm({ isOpen, toggle, network }: any) {
    const mutation = useMutation((network: Network) => APIClient.irc.updateNetwork(network), {
        onSuccess: () => {
            queryClient.invalidateQueries(['networks']);
            toast.custom((t) => <Toast type="success" body={`${network.name} was updated successfully`} t={t} />)
            toggle()
        }
    })

    const deleteMutation = useMutation((id: number) => APIClient.irc.deleteNetwork(id), {
        onSuccess: () => {
            queryClient.invalidateQueries(['networks']);
            toast.custom((t) => <Toast type="success" body={`${network.name} was deleted.`} t={t} />)

            toggle()
        }
    })

    const onSubmit = (data: any) => {
        console.log(data)

        // easy way to split textarea lines into array of strings for each newline.
        // parse on the field didn't really work.
        // TODO fix connect_commands on network update
        // let cmds = data.connect_commands && data.connect_commands.length > 0 ? data.connect_commands.replace(/\r\n/g,"\n").split("\n") : [];
        // data.connect_commands = cmds
        // console.log("formatted", data)

        mutation.mutate(data)
    };

    const validate = (values: any) => {
        const errors = {} as any;

        if (!values.name) {
            errors.name = "Required";
        }

        if (!values.server) {
            errors.server = "Required";
        }

        if (!values.port) {
            errors.port = "Required";
        }

        if (!values.nickserv?.account) {
            errors.nickserv.account = "Required";
        }

        return errors;
    }

    const deleteAction = () => {
        deleteMutation.mutate(network.id)
    }

    const initialValues = {
        id: network.id,
        name: network.name,
        enabled: network.enabled,
        server: network.server,
        port: network.port,
        tls: network.tls,
        nickserv: network.nickserv,
        pass: network.pass,
        invite_command: network.invite_command,
        // connect_commands: network.connect_commands,
        channels: network.channels
    }

    const mutators = {
        ...arrayMutators
    }

    return (
        <SlideOver
            type="UPDATE"
            title="Network"
            isOpen={isOpen}
            toggle={toggle}
            onSubmit={onSubmit}
            deleteAction={deleteAction}
            initialValues={initialValues}
            mutators={mutators}
            validate={validate}
        >
            {() => (
                <>

                    <TextFieldWide name="name" label="Name" placeholder="Name" required={true} />

                    <div className="py-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y dark:divide-gray-700">

                        <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0">
                            <SwitchGroup name="enabled" label="Enabled" />
                        </div>

                        <div>
                            <TextFieldWide name="server" label="Server" placeholder="Address: Eg irc.server.net" required={true} />
                            <NumberFieldWide name="port" label="Port" placeholder="Eg 6667" required={true} />

                            <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                                <SwitchGroup name="tls" label="TLS" />
                            </div>

                            <PasswordFieldWide name="pass" label="Password" help="Network password" />

                            <TextFieldWide name="nickserv.account" label="NickServ Account" placeholder="NickServ Account" required={true} />
                            <PasswordFieldWide name="nickserv.password" label="NickServ Password" />

                            <PasswordFieldWide name="invite_command" label="Invite command" />
                        </div>
                    </div>

                    <div className="p-6">

                        <FieldArray name="channels">
                            {({ fields }) => (
                                <div className="flex flex-col border-2 border-dashed dark:border-gray-700 p-4">
                                    {fields && (fields.length as any) > 0 ? (
                                        fields.map((name, index) => (
                                            <div key={name} className="flex justify-between">
                                                <div className="flex">
                                                    <Field
                                                        name={`${name}.name`}
                                                        component="input"
                                                        type="text"
                                                        placeholder="#Channel"
                                                        className="mr-4 dark:bg-gray-700 focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-600 block w-full shadow-sm sm:text-sm dark:text-white rounded-md"
                                                    />
                                                    <Field
                                                        name={`${name}.password`}
                                                        component="input"
                                                        type="text"
                                                        placeholder="Password"
                                                        className="dark:bg-gray-700 focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-600 block w-full shadow-sm sm:text-sm dark:text-white rounded-md"
                                                    />
                                                </div>

                                                <button
                                                    type="button"
                                                    className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                                                    onClick={() => fields.remove(index)}
                                                >
                                                    <span className="sr-only">Remove</span>
                                                    <XIcon className="h-6 w-6" aria-hidden="true" />
                                                </button>
                                            </div>
                                        ))
                                    ) : (
                                        <span className="text-center text-sm text-grey-darker dark:text-white">
                                            No channels!
                                        </span>
                                    )}
                                    <button
                                        type="button"
                                        className="border dark:border-gray-600 dark:bg-gray-700 my-4 px-4 py-2 text-sm text-gray-700 dark:text-white hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:focus:ring-blue-500 rounded self-center text-center"
                                        onClick={() => fields.push({ name: "", password: "" })}
                                    >
                                        Add Channel
                                    </button>
                                </div>
                            )}
                        </FieldArray>
                    </div>
                </>
            )}
        </SlideOver>
    )
}