import { Fragment, useEffect } from "react";
import { useMutation } from "react-query";
import { Action, DownloadClient, Filter } from "../../domain/interfaces";
import { queryClient } from "../../App";
import { sleep } from "../../utils/utils";
import { XIcon } from "@heroicons/react/solid";
import { Dialog, Transition } from "@headlessui/react";
import { Form } from "react-final-form";
import DEBUG from "../../components/debug";
import APIClient from "../../api/APIClient";
import { ActionTypeOptions } from "../../domain/constants";
import { AlertWarning } from "../../components/alerts";
import { TextFieldWide } from "../../components/inputs";
import {
  NumberFieldWide,
  RadioFieldsetWide,
} from "../../components/inputs/wide";
import { DownloadClientSelect } from "./FilterActionAddForm";

interface props {
  filter: Filter;
  isOpen: boolean;
  toggle: any;
  clients: DownloadClient[];
  action: Action;
}

function FilterActionUpdateForm({
  filter,
  isOpen,
  toggle,
  clients,
  action,
}: props) {
  const mutation = useMutation(
    (action: Action) => APIClient.actions.update(action),
    {
      onSuccess: () => {
        // console.log("add action");
        queryClient.invalidateQueries(["filter", filter.id]);
        sleep(1500);

        toggle();
      },
    }
  );

  useEffect(() => {
    // console.log("render add action form", clients)
  }, [clients]);

  const onSubmit = (data: any) => {
    // TODO clear data depending on type

    console.log(data);
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
                                  Update action
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

export default FilterActionUpdateForm;
