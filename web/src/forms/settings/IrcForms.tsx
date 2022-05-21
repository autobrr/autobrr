import { useMutation } from "react-query";
import { toast } from "react-hot-toast";
import { XIcon } from "@heroicons/react/solid";
import { Field, FieldArray, FormikErrors, FormikValues } from "formik";
import type { FieldProps } from "formik";

import { queryClient } from "../../App";
import { APIClient } from "../../api/APIClient";

import {
  TextFieldWide,
  PasswordFieldWide,
  SwitchGroupWide,
  NumberFieldWide
} from "../../components/inputs";
import { SlideOver } from "../../components/panels";
import Toast from "../../components/notifications/Toast";

interface ChannelsFieldArrayProps {
  channels: IrcChannel[];
}

const ChannelsFieldArray = ({ channels }: ChannelsFieldArrayProps) => (
  <div className="p-6">
    <FieldArray name="channels">
      {({ remove, push }) => (
        <div className="flex flex-col border-2 border-dashed dark:border-gray-700 p-4">
          {channels && channels.length > 0 ? (
            channels.map((_channel: IrcChannel, index: number) => (
              <div key={index} className="flex justify-between">
                <div className="flex">
                  <Field name={`channels.${index}.name`}>
                    {({ field }: FieldProps) => (
                      <input
                        {...field}
                        type="text"
                        value={field.value ?? ""}
                        onChange={field.onChange}
                        placeholder="#Channel"
                        className="mr-4 dark:bg-gray-700 focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-600 block w-full shadow-sm sm:text-sm dark:text-white rounded-md"
                      />
                    )}
                  </Field>

                  <Field name={`channels.${index}.password`}>
                    {({ field }: FieldProps) => (
                      <input
                        {...field}
                        type="text"
                        value={field.value ?? ""}
                        onChange={field.onChange}
                        placeholder="Password"
                        className="mr-4 dark:bg-gray-700 focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-600 block w-full shadow-sm sm:text-sm dark:text-white rounded-md"
                      />
                    )}
                  </Field>
                </div>

                <button
                  type="button"
                  className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                  onClick={() => remove(index)}
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
            onClick={() => push({ name: "", password: "" })}
          >
                        Add Channel
          </button>
        </div>
      )}
    </FieldArray>
  </div>
);

interface IrcNetworkAddFormValues {
    name: string;
    enabled: boolean;
    server : string;
    port: number;
    tls: boolean;
    pass: string;
    nickserv: NickServ;
    channels: IrcChannel[];
}

interface AddFormProps {
  isOpen: boolean;
  toggle: () => void;
}

export function IrcNetworkAddForm({ isOpen, toggle }: AddFormProps) {
  const mutation = useMutation(
    (network: IrcNetwork) => APIClient.irc.createNetwork(network),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(["networks"]);
        toast.custom((t) => <Toast type="success" body="IRC Network added. Please allow up to 30 seconds for the network to come online." t={t} />);
        toggle();
      },
      onError: () => {
        toast.custom((t) => <Toast type="error" body="IRC Network could not be added" t={t} />);
      }
    }
  );

  const onSubmit = (data: unknown) => {
    mutation.mutate(data as IrcNetwork);
  };
  const validate = (values: FormikValues) => {
    const errors = {} as FormikErrors<FormikValues>;
    if (!values.name)
      errors.name = "Required";

    if (!values.port)
      errors.port = "Required";

    if (!values.server)
      errors.server = "Required";

    if (!values.nickserv || !values.nickserv.account)
      errors.nickserv = { account: "Required" };

    return errors;
  };

  const initialValues: IrcNetworkAddFormValues = {
    name: "",
    enabled: true,
    server: "",
    port: 6667,
    tls: false,
    pass: "",
    nickserv: {
      account: ""
    },
    channels: []
  };

  return (
    <SlideOver
      type="CREATE"
      title="Network"
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      initialValues={initialValues}
      validate={validate}
    >
      {(values) => (
        <>
          <TextFieldWide name="name" label="Name" placeholder="Name" required={true} />

          <div className="py-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y dark:divide-gray-700">

            <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200 dark:sm:divide-gray-700">
              <SwitchGroupWide name="enabled" label="Enabled" />
            </div>

            <div>
              <TextFieldWide name="server" label="Server" placeholder="Address: Eg irc.server.net" required={true} />
              <NumberFieldWide name="port" label="Port" placeholder="Eg 6667" required={true} />

              <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                <SwitchGroupWide name="tls" label="TLS" />
              </div>

              <PasswordFieldWide name="pass" label="Password" help="Network password" />

              <TextFieldWide name="nickserv.account" label="NickServ Account" placeholder="NickServ Account" required={true} />
              <PasswordFieldWide name="nickserv.password" label="NickServ Password" />

              <PasswordFieldWide name="invite_command" label="Invite command" />
            </div>
          </div>

          <ChannelsFieldArray channels={values.channels} />
        </>
      )}
    </SlideOver>
  );
}

interface IrcNetworkUpdateFormValues {
    id: number;
    name: string;
    enabled: boolean;
    server: string;
    port: number;
    tls: boolean;
    nickserv?: NickServ;
    pass: string;
    invite_command: string;
    channels: Array<IrcChannel>;
}

interface IrcNetworkUpdateFormProps {
    isOpen: boolean;
    toggle: () => void;
    network: IrcNetwork;
}

export function IrcNetworkUpdateForm({
  isOpen,
  toggle,
  network
}: IrcNetworkUpdateFormProps) {
  const mutation = useMutation((network: IrcNetwork) => APIClient.irc.updateNetwork(network), {
    onSuccess: () => {
      queryClient.invalidateQueries(["networks"]);
      toast.custom((t) => <Toast type="success" body={`${network.name} was updated successfully`} t={t} />);
      toggle();
    }
  });

  const deleteMutation = useMutation((id: number) => APIClient.irc.deleteNetwork(id), {
    onSuccess: () => {
      queryClient.invalidateQueries(["networks"]);
      toast.custom((t) => <Toast type="success" body={`${network.name} was deleted.`} t={t} />);

      toggle();
    }
  });

  const onSubmit = (data: unknown) => {
    mutation.mutate(data as IrcNetwork);
  };

  const validate = (values: FormikValues) => {
    const errors = {} as FormikErrors<FormikValues>;

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
      errors.nickserv = {
        account: "Required"
      };
    }

    return errors;
  };

  const deleteAction = () => {
    deleteMutation.mutate(network.id);
  };

  const initialValues: IrcNetworkUpdateFormValues = {
    id: network.id,
    name: network.name,
    enabled: network.enabled,
    server: network.server,
    port: network.port,
    tls: network.tls,
    nickserv: network.nickserv,
    pass: network.pass,
    channels: network.channels,
    invite_command: network.invite_command
  };

  return (
    <SlideOver
      type="UPDATE"
      title="Network"
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      deleteAction={deleteAction}
      initialValues={initialValues}
      validate={validate}
    >
      {(values) => (
        <>
          <TextFieldWide name="name" label="Name" placeholder="Name" required={true} />

          <div className="py-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y dark:divide-gray-700">

            <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0">
              <SwitchGroupWide name="enabled" label="Enabled" />
            </div>

            <div>
              <TextFieldWide name="server" label="Server" placeholder="Address: Eg irc.server.net" required={true} />
              <NumberFieldWide name="port" label="Port" placeholder="Eg 6667" required={true} />

              <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                <SwitchGroupWide name="tls" label="TLS" />
              </div>

              <PasswordFieldWide name="pass" label="Password" help="Network password" />

              <TextFieldWide name="nickserv.account" label="NickServ Account" placeholder="NickServ Account" required={true} />
              <PasswordFieldWide name="nickserv.password" label="NickServ Password" />

              <PasswordFieldWide name="invite_command" label="Invite command" />
            </div>
          </div>

          <ChannelsFieldArray channels={values.channels} />
        </>
      )}
    </SlideOver>
  );
}