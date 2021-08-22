import { Fragment, useRef, useState } from "react";
import { useToggle } from "../../../hooks/hooks";
import { useMutation } from "react-query";
import { DownloadClient } from "../../../domain/interfaces";
import { queryClient } from "../../../App";
import { Dialog, Transition } from "@headlessui/react";
import { XIcon } from "@heroicons/react/solid";
import { classNames } from "../../../styles/utils";
import { Form } from "react-final-form";
import DEBUG from "../../../components/debug";
import { SwitchGroup, TextFieldWide } from "../../../components/inputs";
import { DownloadClientTypeOptions } from "../../../domain/constants";
import APIClient from "../../../api/APIClient";
import { sleep } from "../../../utils/utils";
import { componentMap, rulesComponentMap } from "./shared";
import { RadioFieldsetWide } from "../../../components/inputs/wide";
import { DeleteModal } from "../../../components/modals";

function DownloadClientUpdateForm({ client, isOpen, toggle }: any) {
  const [isTesting, setIsTesting] = useState(false);
  const [isSuccessfulTest, setIsSuccessfulTest] = useState(false);
  const [isErrorTest, setIsErrorTest] = useState(false);
  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);

  const mutation = useMutation(
    (client: DownloadClient) => APIClient.download_clients.update(client),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(["downloadClients"]);

        toggle();
      },
    }
  );

  const deleteMutation = useMutation(
    (clientID: number) => APIClient.download_clients.delete(clientID),
    {
      onSuccess: () => {
        queryClient.invalidateQueries();
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
      onError: (error) => {
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
                    password: client.password,
                    settings: client.settings,
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
                                  Edit client
                                </Dialog.Title>
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
                                  <XIcon
                                    className="h-6 w-6"
                                    aria-hidden="true"
                                  />
                                </button>
                              </div>
                            </div>
                          </div>

                          <div className="py-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                            <TextFieldWide name="name" label="Name" />

                            <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                              <SwitchGroup name="enabled" label="Enabled" />
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

                        <div className="flex-shrink-0 px-4 border-t border-gray-200 py-5 sm:px-6">
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
                                className={classNames(
                                  isSuccessfulTest
                                    ? "text-green-500 border-green-500 bg-green-50"
                                    : isErrorTest
                                    ? "text-red-500 border-red-500 bg-red-50"
                                    : "border-gray-300 text-gray-700 bg-white hover:bg-gray-50 focus:border-rose-700 active:bg-rose-700",
                                  isTesting ? "cursor-not-allowed" : "",
                                  "mr-2 inline-flex items-center px-4 py-2 border font-medium rounded-md shadow-sm text-sm transition ease-in-out duration-150"
                                )}
                                disabled={isTesting}
                                onClick={() => testClient(values)}
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

export default DownloadClientUpdateForm;
