/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useRef } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "react-hot-toast";

import { APIClient } from "@api/APIClient";
import Toast from "@components/notifications/Toast";
import { useToggle } from "@hooks/hooks";
import { DeleteModal } from "@components/modals";
import { releaseKeys } from "@screens/releases/ReleaseTable";

function ReleaseSettings() {
  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);
  const queryClient = useQueryClient();

  const deleteMutation = useMutation({
    mutationFn:  APIClient.release.delete,
    onSuccess: () => {
      toast.custom((t) => (
        <Toast type="success" body={"All releases were deleted"} t={t}/>
      ));

      // Invalidate filters just in case, most likely not necessary but can't hurt.
      queryClient.invalidateQueries({ queryKey: releaseKeys.lists() });
    }
  });

  const deleteAction = () => deleteMutation.mutate();

  const cancelModalButtonRef = useRef(null);

  return (
    <form
      className="lg:col-span-9"
      action="#"
      method="POST"
    >
      <DeleteModal
        isOpen={deleteModalIsOpen}
        toggle={toggleDeleteModal}
        buttonRef={cancelModalButtonRef}
        deleteAction={deleteAction}
        title={"Delete all releases"}
        text="Are you sure you want to delete all releases? This action cannot be undone."
      />

      <div className="py-6 px-4 sm:p-6 lg:pb-8">
        <div>
          <h2 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">
            Releases
          </h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Release settings. Reset state.
          </p>
        </div>
      </div>

      <div className="pb-6 divide-y divide-gray-200 dark:divide-gray-700">
        <div className="px-4 py-5 sm:p-0">
          <div className="px-4 py-5 sm:p-6">
            <div>
              <h3 style={{ textAlign: "center" }} className="text-lg leading-6 font-medium text-gray-900 dark:text-white">
                Danger Zone
              </h3>
              <p style={{ textAlign: "center" }} className="mt-1 text-sm text-gray-900 dark:text-white">This will clear all release history in your database.</p>
            </div>
            <div className="flex justify-between items-center p-2 mt-2 max-w-sm m-auto">
              <button
                type="button"
                onClick={toggleDeleteModal}
                className="w-full inline-flex items-center justify-center px-4 py-2 border border-transparent font-medium rounded-md text-red-700 hover:text-red-900 dark:text-white bg-red-100 dark:bg-red-800 hover:bg-red-200 dark:hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:text-sm"
              >
                Delete all releases
              </button>
            </div>
          </div>
        </div>
      </div>
    </form>
  );
}

export default ReleaseSettings;