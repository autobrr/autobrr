/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useRef } from "react";
import { useMutation, useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { TrashIcon } from "@heroicons/react/24/outline";

import { KeyField } from "@components/fields/text";
import { DeleteModal } from "@components/modals";
import { APIKeyAddForm } from "@forms/settings/APIKeyAddForm";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { APIClient } from "@api/APIClient";
import { ApikeysQueryOptions } from "@api/queries";
import { ApiKeys } from "@api/query_keys";
import { useToggle } from "@hooks/hooks";
import { classNames } from "@utils";
import { EmptySimple } from "@components/emptystates";
import { Section } from "./_components";
import { PlusIcon } from "@heroicons/react/24/solid";


function APISettings() {
  const [addFormIsOpen, toggleAddForm] = useToggle(false);

  const apikeysQuery = useSuspenseQuery(ApikeysQueryOptions())

  return (
    <Section
      title="API keys"
      description="Manage your autobrr API keys here."
      rightSide={
        <button
          type="button"
          className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          onClick={toggleAddForm}
        >
          <PlusIcon className="h-5 w-5 mr-1" />
          Add new
        </button>
      }
    >
      <APIKeyAddForm isOpen={addFormIsOpen} toggle={toggleAddForm} />

      {apikeysQuery.data && apikeysQuery.data.length > 0 ? (
        <ul className="min-w-full relative">
          <li className="hidden sm:grid grid-cols-12 gap-4 mb-2 border-b border-gray-200 dark:border-gray-700">
            <div className="col-span-3 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Name
            </div>
            <div className="col-span-8 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Key
            </div>
          </li>

          {apikeysQuery.data.map((k, idx) => <APIListItem key={idx} apikey={k} />)}
        </ul>
      ) : (
        <EmptySimple
          title="No API keys"
          subtitle=""
          buttonAction={toggleAddForm}
          buttonText="Create API key"
        />
      )}
    </Section>
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
      queryClient.invalidateQueries({ queryKey: ApiKeys.lists() });
      queryClient.invalidateQueries({ queryKey: ApiKeys.detail(apikey.key) });

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
        isLoading={deleteMutation.isPending}
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
        <div className="col-span-3 px-2 sm:px-6 py-2 sm:py-0 truncate block sm:text-sm text-md font-medium text-gray-900 dark:text-white">
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
        <div className="col-span-8 flex items-center text-sm font-medium text-gray-900 dark:text-white">
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
