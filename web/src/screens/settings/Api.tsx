import { useRef } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "react-hot-toast";
import { TrashIcon } from "@heroicons/react/24/outline";

import { KeyField } from "@components/fields/text";
import { DeleteModal } from "@components/modals";
import APIKeyAddForm from "@forms/settings/APIKeyAddForm";
import Toast from "@components/notifications/Toast";
import { APIClient } from "@api/APIClient";
import { useToggle } from "@hooks/hooks";
import { classNames } from "@utils";
import { EmptySimple } from "@components/emptystates";

export const apiKeys = {
  all: ["api_keys"] as const,
  lists: () => [...apiKeys.all, "list"] as const,
  details: () => [...apiKeys.all, "detail"] as const,
  // detail: (id: number) => [...apiKeys.details(), id] as const
  detail: (id: string) => [...apiKeys.details(), id] as const
};

function APISettings() {
  const [addFormIsOpen, toggleAddForm] = useToggle(false);

  const { data } = useQuery({
    queryKey: apiKeys.lists(),
    queryFn: APIClient.apikeys.getAll,
    retry: false,
    refetchOnWindowFocus: false,
    onError: (err) => console.log(err)
  });

  return (
    <div className="divide-y divide-gray-200 dark:divide-gray-700 lg:col-span-9">
      <div className="pb-6 py-6 px-4 sm:p-6 lg:pb-8">
        <APIKeyAddForm isOpen={addFormIsOpen} toggle={toggleAddForm} />

        <div className="-ml-4 -mt-4 flex justify-between items-center flex-wrap sm:flex-nowrap">
          <div className="ml-4 mt-4">
            <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">
              API keys
            </h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Manage API keys.
            </p>
          </div>
          <div className="ml-4 mt-4 flex-shrink-0">
            <button
              type="button"
              className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              onClick={toggleAddForm}
            >
              Add new
            </button>
          </div>
        </div>

        {data && data.length > 0 ? (
          <section className="mt-6 light:bg-white dark:bg-gray-800 light:shadow sm:rounded-md">
            <ol className="min-w-full relative">
              <li className="hidden sm:grid grid-cols-12 gap-4 mb-2 border-b border-gray-200 dark:border-gray-700">
                <div className="col-span-5 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Name
                </div>
                <div className="col-span-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Key
                </div>
              </li>

              {data && data.map((k, idx) => <APIListItem key={idx} apikey={k} />)}
            </ol>
          </section>
        ) : (
          <EmptySimple
            title="No API keys"
            subtitle=""
            buttonAction={toggleAddForm}
            buttonText="Create API key"
          />
        )}
      </div>
    </div>
  );
}

interface ApiKeyItemProps {
  apikey: APIKey;
}

function APIListItem({ apikey }: ApiKeyItemProps) {
  const cancelModalButtonRef = useRef(null);
  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);

  const queryClient = useQueryClient();

  const deleteMutation = useMutation({
    mutationFn: (key: string) => APIClient.apikeys.delete(key),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: apiKeys.lists() });
      queryClient.invalidateQueries({ queryKey: apiKeys.detail(apikey.key) });

      toast.custom((t) => (
        <Toast
          type="success"
          body={`API key ${apikey?.name} was deleted`}
          t={t}
        />
      ));
    }
  });

  return (
    <li className="text-gray-500 dark:text-gray-400">
      <DeleteModal
        isOpen={deleteModalIsOpen}
        toggle={toggleDeleteModal}
        buttonRef={cancelModalButtonRef}
        deleteAction={() => {
          deleteMutation.mutate(apikey.key);
          toggleDeleteModal();
        }}
        title={`Remove API key: ${apikey.name}`}
        text="Are you sure you want to remove this API key? This action cannot be undone."
      />

      <div className="sm:grid grid-cols-12 gap-4 items-center py-2">
        <div className="col-span-5 px-2 sm:px-6 py-2 sm:py-0 truncate block sm:text-sm text-md font-medium text-gray-900 dark:text-white">
          <div className="flex justify-between">
            <div className="pl-1 py-2">{apikey.name}</div>
            <div>
              <button
                className={classNames(
                  "text-gray-900 dark:text-gray-300",
                  "sm:hidden font-medium group rounded-md items-center px-2 py-2 text-sm"
                )}
                onClick={toggleDeleteModal}
                title="Delete key"
              >
                <TrashIcon
                  className="text-red-500 w-5 h-5"
                  aria-hidden="true"
                />
              </button>
            </div>
          </div>
        </div>
        <div className="col-span-6 flex items-center text-sm font-medium text-gray-900 dark:text-white">
          <KeyField value={apikey.key} />
        </div>

        <div className="col-span-1 hidden sm:flex items-center text-sm font-medium text-gray-900 dark:text-white">
          <button
            className={classNames(
              "text-gray-900 dark:text-gray-300",
              "font-medium group flex rounded-md items-center px-2 py-2 text-sm"
            )}
            onClick={toggleDeleteModal}
            title="Delete key"
          >
            <TrashIcon className="text-red-500 w-5 h-5" aria-hidden="true" />
          </button>
        </div>
      </div>
    </li>
  );
}

export default APISettings;
