import { Dialog, Transition } from "@headlessui/react";
import { Fragment } from "react";
import {Field, Form, Formik} from "formik";
import type {FieldProps} from "formik";
import {XIcon} from "@heroicons/react/solid";
import Select, {components} from "react-select";
import {
    SwitchGroupWide,
    TextFieldWide
} from "../../components/inputs";
import DEBUG from "../../components/debug";
import {EventOptions, NotificationTypeOptions} from "../../domain/constants";
import {useMutation} from "react-query";
import {APIClient} from "../../api/APIClient";
import {queryClient} from "../../App";
import {toast} from "react-hot-toast";
import Toast from "../../components/notifications/Toast";
import {SlideOver} from "../../components/panels";

const Input = (props: any) => {
    return (
        <components.Input
            {...props}
            inputClassName="outline-none border-none shadow-none focus:ring-transparent"
            className="text-gray-400 dark:text-gray-100"
        />
    );
};

const Control = (props: any) => {
    return (
        <components.Control
            {...props}
            className="p-1 block w-full dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:text-gray-100 sm:text-sm"
        />
    );
};

const Menu = (props: any) => {
    return (
        <components.Menu
            {...props}
            className="dark:bg-gray-800 border border-gray-300 dark:border-gray-700 dark:text-gray-400 rounded-md shadow-sm"
        />
    );
};

const Option = (props: any) => {
    return (
        <components.Option
            {...props}
            className="dark:text-gray-400 dark:bg-gray-800 dark:hover:bg-gray-900 dark:focus:bg-gray-900"
        />
    );
};


function FormFieldsDiscord() {
    return (
        <div className="border-t border-gray-200 dark:border-gray-700 py-5">
            {/*<div className="px-6 space-y-1">*/}
            {/*    <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">Credentials</Dialog.Title>*/}
            {/*    <p className="text-sm text-gray-500 dark:text-gray-400">*/}
            {/*        Api keys etc*/}
            {/*    </p>*/}
            {/*</div>*/}

            <TextFieldWide
                name="webhook"
                label="Webhook URL"
                help="Discord channel webhook url"
                placeholder="https://discordapp.com/api/webhooks/xx/xx"
            />
        </div>
    );
}

const componentMap: any = {
    DISCORD: <FormFieldsDiscord/>
};

interface NotificationAddFormValues {
    name: string;
    enabled: boolean;
}

interface AddProps {
    isOpen: boolean;
    toggle: any;
}

export function NotificationAddForm({isOpen, toggle}: AddProps) {
    const mutation = useMutation(
        (notification: Notification) => APIClient.notifications.create(notification),
        {
            onSuccess: () => {
                queryClient.invalidateQueries(["notifications"]);
                toast.custom((t) => <Toast type="success" body="Notification added!" t={t} />);
                toggle();
            },
            onError: () => {
                toast.custom((t) => <Toast type="error" body="Notification could not be added" t={t} />);
            }
        }
    );

    const onSubmit = (formData: any) => {
        mutation.mutate(formData);
    };

    const validate = (values: NotificationAddFormValues) => {
        const errors = {} as any;
        if (!values.name)
            errors.name = "Required";

        return errors;
    };

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
                            <div className="w-screen max-w-2xl dark:border-gray-700 border-l">
                                <Formik
                                    enableReinitialize={true}
                                    initialValues={{
                                        enabled: true,
                                        type: "",
                                        name: "",
                                        webhook: "",
                                        events: []
                                    }}
                                    onSubmit={onSubmit}
                                    validate={validate}
                                >
                                    {({values}) => (
                                        <Form className="h-full flex flex-col bg-white dark:bg-gray-800 shadow-xl overflow-y-scroll">
                                            <div className="flex-1">
                                                <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
                                                    <div className="flex items-start justify-between space-x-3">
                                                        <div className="space-y-1">
                                                            <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">Add
                                                                Notifications</Dialog.Title>
                                                            <p className="text-sm text-gray-500 dark:text-gray-200">
                                                                Trigger notifications on different events.
                                                            </p>
                                                        </div>
                                                        <div className="h-7 flex items-center">
                                                            <button
                                                                type="button"
                                                                className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                                                                onClick={toggle}
                                                            >
                                                                <span className="sr-only">Close panel</span>
                                                                <XIcon className="h-6 w-6" aria-hidden="true"/>
                                                            </button>
                                                        </div>
                                                    </div>
                                                </div>

                                                <TextFieldWide name="name" label="Name" required={true}/>

                                                <div className="space-y-4 divide-y divide-gray-200 dark:divide-gray-700">
                                                    <div className="py-4 flex items-center justify-between space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                                                        <div>
                                                            <label
                                                                htmlFor="type"
                                                                className="block text-sm font-medium text-gray-900 dark:text-white"
                                                            >
                                                                Type
                                                            </label>
                                                        </div>
                                                        <div className="sm:col-span-2">
                                                            <Field name="type" type="select">
                                                                {({
                                                                      field,
                                                                      form: {setFieldValue, resetForm}
                                                                  }: FieldProps) => (
                                                                    <Select {...field}
                                                                            isClearable={true}
                                                                            isSearchable={true}
                                                                            components={{Input, Control, Menu, Option}}
                                                                            placeholder="Choose a type"
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
                                                                                    baseUnit: 2
                                                                                }
                                                                            })}
                                                                            value={field?.value && field.value.value}
                                                                            onChange={(option: any) => {
                                                                                resetForm();
                                                                                // setFieldValue("name", option?.label ?? "")
                                                                                setFieldValue(field.name, option?.value ?? "");
                                                                            }}
                                                                            options={NotificationTypeOptions}
                                                                    />
                                                                )}
                                                            </Field>

                                                        </div>
                                                    </div>

                                                    <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                                                        <SwitchGroupWide name="enabled" label="Enabled"/>
                                                    </div>

                                                    <div className="border-t border-gray-200 dark:border-gray-700 py-5">
                                                        <div className="px-6 space-y-1">
                                                            <Dialog.Title
                                                                className="text-lg font-medium text-gray-900 dark:text-white">Events</Dialog.Title>
                                                            <p className="text-sm text-gray-500 dark:text-gray-400">
                                                                Select what events to trigger on
                                                            </p>
                                                        </div>

                                                        <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:gap-4 sm:px-6 sm:py-5">
                                                            <EventCheckBoxes />
                                                        </div>
                                                    </div>

                                                </div>
                                                {componentMap[values.type]}
                                            </div>

                                            <div className="flex-shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-5 sm:px-6">
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

                                            <DEBUG values={values}/>
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

const EventCheckBoxes = () => (
        <fieldset className="space-y-5">
            <legend className="sr-only">Notifications</legend>
            {EventOptions.map((e, idx) => (
                <div key={idx} className="relative flex items-start">
                    <div className="flex items-center h-5">
                        <Field
                            id={`events-${e.value}`}
                            aria-describedby={`events-${e.value}-description`}
                            name="events"
                            type="checkbox"
                            value={e.value}
                            className="focus:ring-blue-500 h-4 w-4 text-blue-600 border-gray-300 rounded"
                        />
                    </div>
                    <div className="ml-3 text-sm">
                        <label htmlFor={`events-${e.value}`}
                               className="font-medium text-gray-900 dark:text-gray-100">
                            {e.label}
                        </label>
                        {e.description && (
                            <p className="text-gray-500">{e.description}</p>
                        )}
                    </div>
                </div>
            ))}
        </fieldset>
);

interface UpdateProps {
    isOpen: boolean;
    toggle: any;
    notification: Notification;
}

export function NotificationUpdateForm({isOpen, toggle, notification}: UpdateProps) {
    const mutation = useMutation(
        (notification: Notification) => APIClient.notifications.update(notification),
        {
            onSuccess: () => {
                queryClient.invalidateQueries(["notifications"]);
                toast.custom((t) => <Toast type="success" body={`${notification.name} was updated successfully`} t={t}/>);
                toggle();
            }
        }
    );

    const deleteMutation = useMutation(
        (notificationID: number) => APIClient.notifications.delete(notificationID),
        {
            onSuccess: () => {
                queryClient.invalidateQueries(["notifications"]);
                toast.custom((t) => <Toast type="success" body={`${notification.name} was deleted.`} t={t}/>);
            }
        }
    );

    const onSubmit = (formData: any) => {
        mutation.mutate(formData);
    };

    const deleteAction = () => {
        deleteMutation.mutate(notification.id);
    };

    const initialValues = {
        id: notification.id,
        enabled: notification.enabled,
        type: notification.type,
        name: notification.name,
        webhook: notification.webhook,
        events: notification.events || []
    };

    return (
        <SlideOver
            type="UPDATE"
            title="Notification"
            isOpen={isOpen}
            toggle={toggle}
            onSubmit={onSubmit}
            deleteAction={deleteAction}
            initialValues={initialValues}
        >
            {(values) => (
                <div>
                    <TextFieldWide name="name" label="Name" required={true}/>

                    <div className="space-y-4 divide-y divide-gray-200 dark:divide-gray-700">
                        <div className="py-4 flex items-center justify-between space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
                            <div>
                                <label
                                    htmlFor="type"
                                    className="block text-sm font-medium text-gray-900 dark:text-white"
                                >
                                    Type
                                </label>
                            </div>
                            <div className="sm:col-span-2">
                                <Field name="type" type="select">
                                    {({field, form: {setFieldValue, resetForm}}: FieldProps) => (
                                        <Select {...field}
                                                isClearable={true}
                                                isSearchable={true}
                                                components={{Input, Control, Menu, Option}}

                                                placeholder="Choose a type"
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
                                                        baseUnit: 2
                                                    }
                                                })}
                                                value={field?.value && NotificationTypeOptions.find(o => o.value == field?.value)}
                                                onChange={(option: any) => {
                                                    resetForm();
                                                    setFieldValue(field.name, option?.value ?? "");
                                                }}
                                                options={NotificationTypeOptions}
                                        />
                                    )}
                                </Field>
                            </div>
                        </div>

                        <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                            <SwitchGroupWide name="enabled" label="Enabled"/>
                        </div>

                        <div className="border-t border-gray-200 dark:border-gray-700 py-5">
                            <div className="px-6 space-y-1">
                                <Dialog.Title
                                    className="text-lg font-medium text-gray-900 dark:text-white">Events</Dialog.Title>
                                <p className="text-sm text-gray-500 dark:text-gray-400">
                                    Select what events to trigger on
                                </p>
                            </div>

                            <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:gap-4 sm:px-6 sm:py-5">
                                <EventCheckBoxes />
                            </div>
                        </div>
                    </div>
                    {componentMap[values.type]}
                </div>
            )}
        </SlideOver>
    );
}
