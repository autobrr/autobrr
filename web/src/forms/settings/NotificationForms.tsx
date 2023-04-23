import { Dialog, Transition } from "@headlessui/react";
import { Fragment } from "react";
import type { FieldProps } from "formik";
import { Field, Form, Formik, FormikErrors, FormikValues } from "formik";
import { XMarkIcon } from "@heroicons/react/24/solid";
import Select, { components, ControlProps, InputProps, MenuProps, OptionProps } from "react-select";
import { PasswordFieldWide, SwitchGroupWide, TextFieldWide } from "../../components/inputs";
import DEBUG from "../../components/debug";
import { EventOptions, NotificationTypeOptions, SelectOption } from "../../domain/constants";
import { useMutation, useQueryClient } from "react-query";
import { APIClient } from "../../api/APIClient";
import { toast } from "react-hot-toast";
import Toast from "../../components/notifications/Toast";
import { SlideOver } from "../../components/panels";
import { componentMapType } from "./DownloadClientForms";

const Input = (props: InputProps) => {
  return (
    <components.Input
      {...props}
      inputClassName="outline-none border-none shadow-none focus:ring-transparent"
      className="text-gray-400 dark:text-gray-100"
      children={props.children}
    />
  );
};

const Control = (props: ControlProps) => {
  return (
    <components.Control
      {...props}
      className="p-1 block w-full dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:text-gray-100 sm:text-sm"
      children={props.children}
    />
  );
};

const Menu = (props: MenuProps) => {
  return (
    <components.Menu
      {...props}
      className="dark:bg-gray-800 border border-gray-300 dark:border-gray-700 dark:text-gray-400 rounded-md shadow-sm"
      children={props.children}
    />
  );
};

const Option = (props: OptionProps) => {
  return (
    <components.Option
      {...props}
      className="dark:text-gray-400 dark:bg-gray-800 dark:hover:bg-gray-900 dark:focus:bg-gray-900"
      children={props.children}
    />
  );
};

function FormFieldsDiscord() {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4 space-y-1">
        <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">Settings</Dialog.Title>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Create a <a href="https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks" rel="noopener noreferrer" target="_blank" className="font-medium text-blue-500 underline underline-offset-1 hover:text-blue-400">webhook integration</a> in your server.
        </p>
      </div>

      <PasswordFieldWide
        name="webhook"
        label="Webhook URL"
        help="Discord channel webhook url"
        placeholder="https://discordapp.com/api/webhooks/xx/xx"
      />
    </div>
  );
}

function FormFieldsNotifiarr() {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4 space-y-1">
        <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">Settings</Dialog.Title>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Enable the autobrr integration and optionally create a new API Key.
        </p>
      </div>

      <PasswordFieldWide
        name="api_key"
        label="API Key"
        help="Notifiarr API Key"
      />
    </div>
  );
}

function FormFieldsTelegram() {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4 space-y-1">
        <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">Settings</Dialog.Title>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Read how to <a href="https://core.telegram.org/bots#3-how-do-i-create-a-bot" rel="noopener noreferrer" target="_blank" className="font-medium text-blue-500 underline underline-offset-1 hover:text-blue-400">create a bot</a>.
        </p>
      </div>

      <PasswordFieldWide
        name="token"
        label="Bot token"
        help="Bot token"
      />
      <PasswordFieldWide
        name="channel"
        label="Chat ID"
        help="Chat ID"
      />
    </div>
  );
}

function FormFieldsPushover() {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-4">
      <div className="px-4 space-y-1">
        <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">Settings</Dialog.Title>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Register a new <a href="https://support.pushover.net/i175-how-do-i-get-an-api-or-application-token" rel="noopener noreferrer" target="_blank" className="font-medium text-blue-500 underline underline-offset-1 hover:text-blue-400">application</a> and add its API Token here.
        </p>
      </div>

      <PasswordFieldWide
        name="api_key"
        label="API Token"
        help="API Token"
      />
      <PasswordFieldWide
        name="token"
        label="User Key"
        help="User Key"
      />
      <TextFieldWide
        name="priority"
        label="Priority"
        help="-2, -1, 0 (default), 1, or 2"
        required={true}
      />
    </div>
  );
}

const componentMap: componentMapType = {
  DISCORD: <FormFieldsDiscord />,
  NOTIFIARR: <FormFieldsNotifiarr />,
  TELEGRAM: <FormFieldsTelegram />,
  PUSHOVER: <FormFieldsPushover />
};

interface NotificationAddFormValues {
    name: string;
    enabled: boolean;
}

interface AddProps {
    isOpen: boolean;
    toggle: () => void;
}

export function NotificationAddForm({ isOpen, toggle }: AddProps) {
  const queryClient = useQueryClient();

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

  const onSubmit = (formData: unknown) => {
    mutation.mutate(formData as Notification);
  };

  const testMutation = useMutation(
    (n: Notification) => APIClient.notifications.test(n),
    {
      onError: (err) => {
        console.error(err);
      }
    }
  );

  const testNotification = (data: unknown) => {
    testMutation.mutate(data as Notification);
  };

  const validate = (values: NotificationAddFormValues) => {
    const errors = {} as FormikErrors<FormikValues>;
    if (!values.name)
      errors.name = "Required";

    return errors;
  };

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
                  {({ values }) => (
                    <Form className="h-full flex flex-col bg-white dark:bg-gray-800 shadow-xl overflow-y-scroll">
                      <div className="flex-1">
                        <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
                          <div className="flex items-start justify-between space-x-3">
                            <div className="space-y-1">
                              <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">
                                Add Notifications
                              </Dialog.Title>
                              <p className="text-sm text-gray-500 dark:text-gray-200">
                                Trigger notifications on different events.
                              </p>
                            </div>
                            <div className="h-7 flex items-center">
                              <button
                                type="button"
                                className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                                onClick={toggle}
                              >
                                <span className="sr-only">Close panel</span>
                                <XMarkIcon className="h-6 w-6" aria-hidden="true" />
                              </button>
                            </div>
                          </div>
                        </div>

                        <div className="flex flex-col space-y-4 px-1 py-6 sm:py-0 sm:space-y-0">
                          <TextFieldWide
                            name="name"
                            label="Name"
                            required={true}
                          />

                          <div className="flex items-center justify-between space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
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
                                  form: { setFieldValue, resetForm }
                                }: FieldProps) => (
                                  <Select
                                    {...field}
                                    isClearable={true}
                                    isSearchable={true}
                                    components={{
                                      Input,
                                      Control,
                                      Menu,
                                      Option
                                    }}
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
                                    onChange={(option: unknown) => {
                                      resetForm();

                                      const opt = option as SelectOption;
                                      // setFieldValue("name", option?.label ?? "")
                                      setFieldValue(
                                        field.name,
                                        opt.value ?? ""
                                      );
                                    }}
                                    options={NotificationTypeOptions}
                                  />
                                )}
                              </Field>
                            </div>
                          </div>

                          <SwitchGroupWide name="enabled" label="Enabled" />

                          <div className="border-t mt-2 border-gray-200 dark:border-gray-700 py-4">
                            <div className="px-4 space-y-1">
                              <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">
                                Events
                              </Dialog.Title>
                              <p className="text-sm text-gray-500 dark:text-gray-400">
                                Select what events to trigger on
                              </p>
                            </div>

                            <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:gap-4 sm:py-4">
                              <EventCheckBoxes />
                            </div>
                          </div>
                        </div>
                        {componentMap[values.type]}
                      </div>

                      <div className="flex-shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-4 sm:px-6">
                        <div className="space-x-3 flex justify-end">
                          <button
                            type="button"
                            className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                            onClick={() => testNotification(values)}
                          >
                            Test
                          </button>
                          <button
                            type="button"
                            className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                            onClick={toggle}
                          >
                            Cancel
                          </button>
                          <button
                            type="submit"
                            className="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
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
    toggle: () => void;
    notification: Notification;
}

interface InitialValues {
  id: number;
  enabled: boolean;
  type: NotificationType;
  name: string;
  webhook?: string;
  token?: string;
  api_key?: string;
  priority?: number;
  channel?: string;
  events: NotificationEvent[];
}

export function NotificationUpdateForm({ isOpen, toggle, notification }: UpdateProps) {
  const queryClient = useQueryClient();

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

  const onSubmit = (formData: unknown) => {
    mutation.mutate(formData as Notification);
  };

  const deleteAction = () => {
    deleteMutation.mutate(notification.id);
  };

  const testMutation = useMutation(
    (n: Notification) => APIClient.notifications.test(n),
    {
      onError: (err) => {
        console.error(err);
      }
    }
  );

  const testNotification = (data: unknown) => {
    testMutation.mutate(data as Notification);
  };

  const initialValues: InitialValues = {
    id: notification.id,
    enabled: notification.enabled,
    type: notification.type,
    name: notification.name,
    webhook: notification.webhook,
    token: notification.token,
    api_key: notification.api_key,
    priority: notification.priority,
    channel: notification.channel,
    events: notification.events || []
  };

  return (
    <SlideOver<InitialValues>
      type="UPDATE"
      title="Notification"
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      deleteAction={deleteAction}
      initialValues={initialValues}
      testFn={testNotification}
    >
      {(values) => (
        <div>
          <TextFieldWide name="name" label="Name" required={true}/>

          <div className="space-y-2 divide-y divide-gray-200 dark:divide-gray-700">
            <div className="py-4 flex items-center justify-between space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:py-4">
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
                  {({ field, form: { setFieldValue, resetForm } }: FieldProps) => (
                    <Select {...field}
                      isClearable={true}
                      isSearchable={true}
                      components={{ Input, Control, Menu, Option }}

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
                      onChange={(option: unknown) => {
                        resetForm();
                        const opt = option as SelectOption;
                        setFieldValue(field.name, opt.value ?? "");
                      }}
                      options={NotificationTypeOptions}
                    />
                  )}
                </Field>
              </div>
            </div>
            <SwitchGroupWide name="enabled" label="Enabled"/>
            <div className="border-t border-gray-200 dark:border-gray-700 py-4">
              <div className="px-4 space-y-1">
                <Dialog.Title
                  className="text-lg font-medium text-gray-900 dark:text-white">Events</Dialog.Title>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Select what events to trigger on
                </p>
              </div>

              <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:gap-4 sm:py-2">
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