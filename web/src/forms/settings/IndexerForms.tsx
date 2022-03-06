import {Fragment, useState} from "react";
import { toast } from "react-hot-toast";
import { useMutation, useQuery } from "react-query";
import Select, { components } from "react-select";
import { Field, Form, Formik } from "formik";
import type { FieldProps } from "formik";

import { XIcon } from "@heroicons/react/solid";
import { Dialog, Transition } from "@headlessui/react";

import { sleep } from "../../utils";
import { queryClient } from "../../App";
import DEBUG from "../../components/debug";
import { APIClient } from "../../api/APIClient";
import {
    TextFieldWide,
    PasswordFieldWide,
    SwitchGroupWide
} from "../../components/inputs";
import { SlideOver } from "../../components/panels";
import Toast from '../../components/notifications/Toast';

const Input = (props: any) => {
  return (
    <components.Input 
      {...props} 
      inputClassName="outline-none border-none shadow-none focus:ring-transparent"
      className="text-gray-400 dark:text-gray-100"
    />
  );
}

const Control = (props: any) => {
  return (
    <components.Control 
      {...props} 
      className="block w-full dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:text-gray-100 sm:text-sm"
    />
  );
}

const Menu = (props: any) => {
  return (
    <components.Menu 
      {...props}
      className="dark:bg-gray-800 border border-gray-300 dark:border-gray-700 dark:text-gray-400 rounded-md shadow-sm"
    />
  );
}

const Option = (props: any) => {
    return (
      <components.Option 
        {...props}
        className="dark:text-gray-400 dark:bg-gray-800 dark:hover:bg-gray-900 dark:focus:bg-gray-900"
      />
    );
}

const IrcSettingFields = (ind: IndexerDefinition, indexer: string) => {
    if (indexer !== "") {
        return (
            <Fragment>
                {ind && ind.irc && ind.irc.settings && (
                    <div className="border-t border-gray-200 dark:border-gray-700 py-5">
                        <div className="px-6 space-y-1">
                            <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">IRC</Dialog.Title>
                            <p className="text-sm text-gray-500 dark:text-gray-200">
                                Networks, channels and invite commands are configured automatically.
                            </p>
                        </div>
                        {ind.irc.settings.map((f: IndexerSetting, idx: number) => {
                            switch (f.type) {
                                case "text":
                                    return <TextFieldWide name={`irc.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} />
                                case "secret":
                                    if (f.name === "invite_command") {
                                        return <PasswordFieldWide name={`irc.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} defaultVisible={true} defaultValue={f.default} />
                                    }
                                    return <PasswordFieldWide name={`irc.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} defaultValue={f.default} />
                            }
                            return null
                        })}

                        {/* <div hidden={false}>
                                <TextFieldWide name="irc.server" label="Server" defaultValue={ind.irc.server} />
                                <NumberFieldWide name="irc.port" label="Port" defaultValue={ind.irc.port} />
                                <SwitchGroupWide name="irc.tls" label="TLS" defaultValue={ind.irc.tls} />
                            </div> */}
                    </div>
                )}
            </Fragment>
        )
    }
}

const SettingFields = (ind: IndexerDefinition, indexer: string) => {
    if (indexer !== "") {
        return (
            <div key="opt">
                {ind && ind.settings && ind.settings.map((f: any, idx: number) => {
                    switch (f.type) {
                        case "text":
                            return (
                                <TextFieldWide name={`settings.${f.name}`} label={f.label} key={idx} help={f.help} defaultValue="" />
                            )
                        case "secret":
                            return (
                                <PasswordFieldWide name={`settings.${f.name}`} label={f.label} key={idx} help={f.help} defaultValue="" />
                            )
                    }
                    return null
                })}
                <div hidden={true}>
                    <TextFieldWide name="name" label="Name" defaultValue={ind?.name} />
                </div>
            </div>
        )
    }
}

interface AddProps {
    isOpen: boolean;
    toggle: any;
}

export function IndexerAddForm({ isOpen, toggle }: AddProps) {
    const [indexer, setIndexer] = useState<IndexerDefinition>({} as IndexerDefinition)

    const { data } = useQuery('indexerDefinition', APIClient.indexers.getSchema,
        {
            enabled: isOpen,
            refetchOnWindowFocus: false
        }
    )

    const mutation = useMutation(
      (indexer: Indexer) => APIClient.indexers.create(indexer), {
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

    const ircMutation = useMutation(
        (network: IrcNetwork) => APIClient.irc.createNetwork(network)
    );

    const onSubmit = (formData: any) => {
        const ind = data && data.find(i => i.identifier === formData.identifier);
        if (!ind)
            return;

        const channels: IrcChannel[] = [];
        if (ind.irc.channels.length) {
            ind.irc.channels.forEach(element => {
                channels.push({
                    id: 0,
                    enabled: true,
                    name: element,
                    password: "",
                    detached: false,
                    monitoring: false
                });
            });
        }

        const network: IrcNetwork = {
            id: 0,
            name: ind.irc.network,
            pass: "",
            enabled: false,
            connected: false,
            connected_since: 0,
            server: ind.irc.server,
            port: ind.irc.port,
            tls: ind.irc.tls,
            nickserv: formData.irc.nickserv,
            invite_command: formData.irc.invite_command,
            channels: channels,
        }

        mutation.mutate(formData, {
            onSuccess: () => ircMutation.mutate(network)
        });
    };

    const renderSettingFields = (indexer: string) => {
        if (indexer !== "") {
            const ind = data && data.find(i => i.identifier === indexer);
            return (
                <div key="opt">
                    {ind && ind.settings && ind.settings.map((f: any, idx: number) => {
                        switch (f.type) {
                            case "text":
                                return (
                                    <TextFieldWide name={`settings.${f.name}`} label={f.label} key={idx} help={f.help} defaultValue="" />
                                )
                            case "secret":
                                return (
                                    <PasswordFieldWide name={`settings.${f.name}`} label={f.label} key={idx} help={f.help} defaultValue="" />
                                )
                        }
                        return null
                    })}
                    <div hidden={true}>
                        <TextFieldWide name="name" label="Name" defaultValue={ind?.name} />
                    </div>
                </div>
            )
        }
    }

    const renderIrcSettingFields = (indexer: string) => {
        if (indexer !== "") {
            const ind = data && data.find(i => i.identifier === indexer);
            return (
                <Fragment>
                    {ind && ind.irc && ind.irc.settings && (
                        <div className="border-t border-gray-200 dark:border-gray-700 py-5">
                            <div className="px-6 space-y-1">
                                <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">IRC</Dialog.Title>
                                <p className="text-sm text-gray-500 dark:text-gray-200">
                                    Networks, channels and invite commands are configured automatically.
                                </p>
                            </div>
                            {ind.irc.settings.map((f: IndexerSetting, idx: number) => {
                                switch (f.type) {
                                    case "text":
                                        return <TextFieldWide name={`irc.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} />
                                    case "secret":
                                        return <PasswordFieldWide name={`irc.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} defaultValue={f.default} />
                                }
                                return null
                            })}

                            {/* <div hidden={false}>
                                <TextFieldWide name="irc.server" label="Server" defaultValue={ind.irc.server} />
                                <NumberFieldWide name="irc.port" label="Port" defaultValue={ind.irc.port} />
                                <SwitchGroupWide name="irc.tls" label="TLS" defaultValue={ind.irc.tls} />
                            </div> */}
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
                            <div className="w-screen max-w-2xl dark:border-gray-700 border-l">
                                <Formik
                                    enableReinitialize={true}
                                    initialValues={{
                                        enabled: true,
                                        identifier: "",
                                        name: "",
                                        irc: {
                                            invite_command: "",
                                        },
                                        settings: {},
                                    }}
                                    onSubmit={onSubmit}
                                >
                                    {({ values }) => (
                                            <Form className="h-full flex flex-col bg-white dark:bg-gray-800 shadow-xl overflow-y-scroll">
                                                <div className="flex-1">
                                                    <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
                                                        <div className="flex items-start justify-between space-x-3">
                                                            <div className="space-y-1">
                                                                <Dialog.Title
                                                                    className="text-lg font-medium text-gray-900 dark:text-white">Add
                                                                    indexer</Dialog.Title>
                                                                <p className="text-sm text-gray-500 dark:text-gray-200">
                                                                    Add indexer.
                                                                </p>
                                                            </div>
                                                            <div className="h-7 flex items-center">
                                                                <button
                                                                    type="button"
                                                                    className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                                                                    onClick={toggle}
                                                                >
                                                                    <span className="sr-only">Close panel</span>
                                                                    <XIcon className="h-6 w-6" aria-hidden="true" />
                                                                </button>
                                                            </div>
                                                        </div>
                                                    </div>

                                                    <div className="py-6 space-y-4 divide-y divide-gray-200 dark:divide-gray-700">
                                                        <div className="py-4 flex items-center justify-between space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                                                            <div>
                                                                <label
                                                                    htmlFor="identifier"
                                                                    className="block text-sm font-medium text-gray-900 dark:text-white"
                                                                >
                                                                    Indexer
                                                                </label>
                                                            </div>
                                                            <div className="sm:col-span-2">
                                                                <Field name="identifier" type="select">
                                                                    {({ field, form: { setFieldValue, resetForm } }: FieldProps) => (
                                                                        <Select {...field}
                                                                            isClearable={true}
                                                                            isSearchable={true}
                                                                            components={{ Input, Control, Menu, Option }}
                                                                            placeholder="Choose an indexer"
                                                                            styles={{
                                                                                singleValue: (base) => ({
                                                                                    ...base,
                                                                                    color: "unset"
                                                                                })
                                                                            }}
                                                                            theme={(theme) => ({
                                                                                ...theme,
                                                                                spacing: {
                                                                                  ...theme.spacing,
                                                                                  controlHeight: 30,
                                                                                  baseUnit: 2,
                                                                                }
                                                                            })}
                                                                            value={field?.value && field.value.value}
                                                                            onChange={(option: any) => {
                                                                                resetForm()
                                                                                setFieldValue("name", option?.label ?? "")
                                                                                setFieldValue(field.name, option?.value ?? "")

                                                                                const ind = data!.find(i => i.identifier === option.value);
                                                                                setIndexer(ind!)
                                                                                if (ind!.irc.settings) {
                                                                                    ind!.irc.settings.forEach((s) => {
                                                                                        setFieldValue(`irc.${s.name}`, s.default ?? "")
                                                                                    })
                                                                                }
                                                                            }}
                                                                            options={data && data.sort((a, b): any => a.name.localeCompare(b.name)).map(v => ({
                                                                                label: v.name,
                                                                                value: v.identifier
                                                                            }))} 
                                                                        />
                                                                    )}
                                                                </Field>

                                                            </div>
                                                        </div>

                                                        <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                                                            <SwitchGroupWide name="enabled" label="Enabled" />
                                                        </div>

                                                        {SettingFields(indexer, values.identifier)}

                                                    </div>

                                                    {IrcSettingFields(indexer, values.identifier)}
                                                </div>

                                                <div
                                                    className="flex-shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-5 sm:px-6">
                                                    <div className="space-x-3 flex justify-end">
                                                        <button
                                                            type="button"
                                                            className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                                                            onClick={toggle}
                                                        >
                                                            Cancel
                                                        </button>
                                                        <button
                                                            type="submit"
                                                            className="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 dark:bg-blue-600 hover:bg-indigo-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                                                        >
                                                            Save
                                                        </button>
                                                    </div>
                                                </div>

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
    )
}

interface UpdateProps {
    isOpen: boolean;
    toggle: any;
    indexer: Indexer;
}

export function IndexerUpdateForm({ isOpen, toggle, indexer }: UpdateProps) {
    const mutation = useMutation((indexer: Indexer) => APIClient.indexers.update(indexer), {
        onSuccess: () => {
            queryClient.invalidateQueries(['indexer']);
            toast.custom((t) => <Toast type="success" body={`${indexer.name} was updated successfully`} t={t} />)
            sleep(1500)

            toggle()
        }
    })

    const deleteMutation = useMutation((id: number) => APIClient.indexers.delete(id), {
        onSuccess: () => {
            queryClient.invalidateQueries(['indexer']);
            toast.custom((t) => <Toast type="success" body={`${indexer.name} was deleted.`} t={t} />)
        }
    })

    const onSubmit = (data: any) => {
        // TODO clear data depending on type
        mutation.mutate(data)
    };

    const deleteAction = () => {
        deleteMutation.mutate(indexer.id)
    }

    const renderSettingFields = (settings: IndexerSetting[]) => {
        if (settings === undefined) {
            return null
        }

        return (
            <div key="opt">
                {settings.map((f: IndexerSetting, idx: number) => {
                    switch (f.type) {
                        case "text":
                            return (
                                <TextFieldWide name={`settings.${f.name}`} label={f.label} key={idx} help={f.help} />
                            )
                        case "secret":
                            return (
                                <PasswordFieldWide name={`settings.${f.name}`} label={f.label} key={idx} help={f.help} />
                            )
                    }
                    return null
                })}
            </div>
        )
    }

    const initialValues = {
        id: indexer.id,
        name: indexer.name,
        enabled: indexer.enabled,
        identifier: indexer.identifier,
        settings: indexer.settings?.reduce(
            (o: Record<string, string>, obj: IndexerSetting) => ({
                ...o,
                [obj.name]: obj.value
            } as Record<string, string>),
            {} as Record<string, string>
        ),
    }

    return (
        <SlideOver
            type="UPDATE"
            title="Indexer"
            isOpen={isOpen}
            toggle={toggle}
            deleteAction={deleteAction}
            onSubmit={onSubmit}
            initialValues={initialValues}
        >
            {() => (
                <div className="py-6 space-y-6 sm:py-0 sm:space-y-0 divide-y divide-gray-200 dark:divide-gray-700">
                    <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                        <div>
                            <label
                                htmlFor="name"
                                className="block text-sm font-medium text-gray-900 dark:text-white sm:mt-px sm:pt-2"
                            >
                                Name
                            </label>
                        </div>
                        <Field name="name">
                            {({ field, meta }: FieldProps) => (
                                <div className="sm:col-span-2">
                                    <input
                                        type="text"
                                        {...field}
                                        className="block w-full shadow-sm dark:bg-gray-800 sm:text-sm dark:text-white focus:ring-indigo-500 focus:border-indigo-500 border-gray-300 dark:border-gray-700 rounded-md"
                                    />
                                    {meta.touched && meta.error && <span>{meta.error}</span>}
                                </div>
                            )}
                        </Field>
                    </div>

                    <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200 dark:sm:divide-gray-700">
                        <SwitchGroupWide name="enabled" label="Enabled" />
                    </div>

                    {renderSettingFields(indexer.settings)}
                </div>
            )}
        </SlideOver>
    )
}