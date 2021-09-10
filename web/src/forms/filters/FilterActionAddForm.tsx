import { Fragment, useEffect } from "react";
import { useMutation } from "react-query";
import { Action, DownloadClient, Filter } from "../../domain/interfaces";
import { queryClient } from "../../App";
import { sleep } from "../../utils/utils";
import { CheckIcon, SelectorIcon, XIcon } from "@heroicons/react/solid";
import { Dialog, Listbox, Transition } from "@headlessui/react";
import { classNames } from "../../styles/utils";
import { Field, Form } from "react-final-form";
import DEBUG from "../../components/debug";
import APIClient from "../../api/APIClient";
import { ActionTypeOptions } from "../../domain/constants";
import { SwitchGroup, TextFieldWide } from "../../components/inputs";
import { AlertWarning } from "../../components/alerts";
import {
  NumberFieldWide,
  RadioFieldsetWide,
} from "../../components/inputs/wide";

import { toast } from 'react-hot-toast'
import Toast from '../../components/notifications/Toast';

interface DownloadClientSelectProps {
  name: string;
  clients: DownloadClient[];
  values: any;
}

export function DownloadClientSelect({
  name,
  clients,
  values,
}: DownloadClientSelectProps) {
  return (
    <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
      <Field
        name={name}
        type="select"
        render={({ input }) => (
          <Listbox value={input.value} onChange={input.onChange}>
            {({ open }) => (
              <>
                <Listbox.Label className="block text-sm font-medium text-gray-700">
                  Client
                </Listbox.Label>
                <div className="mt-1 relative">
                  <Listbox.Button className="bg-white relative w-full border border-gray-300 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm">
                    <span className="block truncate">
                      {input.value
                        ? clients.find((c) => c.id === input.value)!.name
                        : "Choose a client"}
                    </span>
                    {/*<span className="block truncate">Choose a client</span>*/}
                    <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
                      <SelectorIcon
                        className="h-5 w-5 text-gray-400"
                        aria-hidden="true"
                      />
                    </span>
                  </Listbox.Button>

                  <Transition
                    show={open}
                    as={Fragment}
                    leave="transition ease-in duration-100"
                    leaveFrom="opacity-100"
                    leaveTo="opacity-0"
                  >
                    <Listbox.Options
                      static
                      className="absolute z-10 mt-1 w-full bg-white shadow-lg max-h-60 rounded-md py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm"
                    >
                      {clients
                        .filter((c) => c.type === values.type)
                        .map((client: any) => (
                          <Listbox.Option
                            key={client.id}
                            className={({ active }) =>
                              classNames(
                                active
                                  ? "text-white bg-indigo-600"
                                  : "text-gray-900",
                                "cursor-default select-none relative py-2 pl-3 pr-9"
                              )
                            }
                            value={client.id}
                          >
                            {({ selected, active }) => (
                              <>
                                <span
                                  className={classNames(
                                    selected ? "font-semibold" : "font-normal",
                                    "block truncate"
                                  )}
                                >
                                  {client.name}
                                </span>

                                {selected ? (
                                  <span
                                    className={classNames(
                                      active ? "text-white" : "text-indigo-600",
                                      "absolute inset-y-0 right-0 flex items-center pr-4"
                                    )}
                                  >
                                    <CheckIcon
                                      className="h-5 w-5"
                                      aria-hidden="true"
                                    />
                                  </span>
                                ) : null}
                              </>
                            )}
                          </Listbox.Option>
                        ))}
                    </Listbox.Options>
                  </Transition>
                </div>
              </>
            )}
          </Listbox>
        )}
      />
    </div>
  );
}

interface props {
  filter: Filter;
  isOpen: boolean;
  toggle: any;
  clients: DownloadClient[];
}

function FilterActionAddForm({ filter, isOpen, toggle, clients }: props) {
  const mutation = useMutation(
    (action: Action) => APIClient.actions.create(action),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(["filter", filter.id]);
        toast.custom((t) => <Toast type="success" body="Action was added" t={t} />)

        sleep(500).then(() => toggle());
      },
    }
  );

  useEffect(() => {
    // console.log("render add action form", clients)
  }, []);

  const onSubmit = (data: any) => {
    // TODO clear data depending on type
    mutation.mutate(data);
  };

  const TypeForm = (values: any) => {
    switch (values.type) {
      case "TEST":
        return (
          <AlertWarning
            title="Notice"
            text="The test action does nothing except to show if the filter works."
          />
        );
      case "WATCH_FOLDER":
        return (
          <div>
            <TextFieldWide
              name="watch_folder"
              label="Watch dir"
              placeholder="Watch directory eg. /home/user/watch_folder"
            />
          </div>
        );
      case "EXEC":
        return (
          <div>
            <TextFieldWide
              name="exec_cmd"
              label="Program"
              placeholder="Path to program eg. /bin/test"
            />

            <TextFieldWide
              name="exec_args"
              label="Arguments"
              placeholder="Arguments eg. --test"
            />
          </div>
        );
      case "QBITTORRENT":
        return (
          <div>
            <DownloadClientSelect
              name="client_id"
              clients={clients}
              values={values}
            />

            <TextFieldWide name="category" label="Category" placeholder="" />
            <TextFieldWide
              name="tags"
              label="Tags"
              placeholder="Comma separated eg. 4k,remux"
            />
            <TextFieldWide name="save_path" label="Save path" />

            <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                <SwitchGroup name="paused" label="Add paused" />
            </div>

            <div className="divide-y divide-gray-200 pt-8 space-y-6 sm:pt-10 sm:space-y-5">
              <div className="px-4">
                <h3 className="text-lg leading-6 font-medium text-gray-900">
                  Limit speeds
                </h3>
                <p className="mt-1 max-w-2xl text-sm text-gray-500">
                  Limit download and upload speed for torrents in this filter.
                  In KB/s.
                </p>
              </div>
              <NumberFieldWide
                name="limit_download_speed"
                label="Limit download speed"
              />
              <NumberFieldWide
                name="limit_upload_speed"
                label="Limit upload speed"
              />
            </div>
          </div>
        );
      case "DELUGE_V1":
      case "DELUGE_V2":
        return (
          <div>
            <DownloadClientSelect
              name="client_id"
              clients={clients}
              values={values}
            />

            <TextFieldWide name="label" label="Label" />
            <TextFieldWide name="save_path" label="Save path" />

            <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                <SwitchGroup name="paused" label="Add paused" />
            </div>

            <div className="divide-y divide-gray-200 pt-8 space-y-6 sm:pt-10 sm:space-y-5">
              <div className="px-4">
                <h3 className="text-lg leading-6 font-medium text-gray-900">
                  Limit speeds
                </h3>
                <p className="mt-1 max-w-2xl text-sm text-gray-500">
                  Limit download and upload speed for torrents in this filter.
                  In KB/s.
                </p>
              </div>
              <NumberFieldWide
                name="limit_download_speed"
                label="Limit download speed"
              />
              <NumberFieldWide
                name="limit_upload_speed"
                label="Limit upload speed"
              />
            </div>
          </div>
        );
      case "RADARR":
      case "SONARR":
      case "LIDARR":
        return (
          <div>
            <DownloadClientSelect
              name="client_id"
              clients={clients}
              values={values}
            />
          </div>
        );
      default:
        return (
          <AlertWarning
            title="Notice"
            text="The test action does nothing except to show if the filter works."
          />
        );
    }
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
              <div className="w-screen max-w-2xl">
                <Form
                  initialValues={{
                    name: "",
                    enabled: false,
                    type: "TEST",
                    watch_folder: "",
                    exec_cmd: "",
                    exec_args: "",
                    category: "",
                    tags: "",
                    label: "",
                    save_path: "",
                    paused: false,
                    ignore_rules: false,
                    limit_upload_speed: 0,
                    limit_download_speed: 0,
                    filter_id: filter.id,
                    client_id: null,
                  }}
                  onSubmit={onSubmit}
                >
                  {({ handleSubmit, values }) => {
                    return (
                      <form
                        className="h-full flex flex-col bg-white shadow-xl overflow-y-scroll"
                        onSubmit={handleSubmit}
                      >
                        <div className="flex-1">
                          <div className="px-4 py-6 bg-gray-50 sm:px-6">
                            <div className="flex items-start justify-between space-x-3">
                              <div className="space-y-1">
                                <Dialog.Title className="text-lg font-medium text-gray-900">
                                  Add action
                                </Dialog.Title>
                                <p className="text-sm text-gray-500">
                                  Add filter action.
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

                          <div className="py-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                            <TextFieldWide name="name" label="Action name" />
                            <RadioFieldsetWide
                              name="type"
                              legend="Type"
                              options={ActionTypeOptions}
                            />

                            {TypeForm(values)}
                          </div>
                        </div>

                        <div className="flex-shrink-0 px-4 border-t border-gray-200 py-5 sm:px-6">
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
                    );
                  }}
                </Form>
              </div>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition.Root>
  );
}

export default FilterActionAddForm;
