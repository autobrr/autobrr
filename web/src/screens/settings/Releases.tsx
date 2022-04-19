import { useRef } from "react";
import { useMutation } from "react-query";
import { toast } from "react-hot-toast";

import { APIClient } from "../../api/APIClient";
import { Toast } from "../../components/notifications/Toast";
import { queryClient } from "../../App";
import { useToggle } from "../../hooks/hooks";
import { DeleteModal } from "../../components/modals";

function ReleaseSettings() {
  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);
  const deleteMutation = useMutation(() => APIClient.release.delete(), {
    onSuccess: () => {
      toast.custom((t) => (
        <Toast type="success" body={"All releases was deleted"} t={t}/>
      ));

      // Invalidate filters just in case, most likely not necessary but can't hurt.
      queryClient.invalidateQueries("releases");

      toggleDeleteModal();
    }
  });

  const deleteAction = () => {
    deleteMutation.mutate();
  };

  const cancelModalButtonRef = useRef(null);

  return (
    <form className="divide-y divide-gray-200 dark:divide-gray-700 lg:col-span-9" action="#" method="POST">
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
          <h2 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">Releases</h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                        Release settings. Reset state.
          </p>
        </div>
      </div>

      <div className="pb-6 divide-y divide-gray-200 dark:divide-gray-700">
        <div className="px-4 py-5 sm:p-0">
          <div className="px-4 py-5 sm:p-6">

            <div>
              <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">Danger Zone</h3>
            </div>

            <ul className="p-4 mt-6 divide-y divide-gray-200 dark:divide-gray-700 border-red-500 border rounded-lg">
              <div className="flex justify-between items-center py-2">
                <p className="text-sm text-gray-500 dark:text-gray-400">
                                    Delete all releases
                </p>
                <button
                  type="button"
                  onClick={toggleDeleteModal}
                  className="inline-flex items-center justify-center px-4 py-2 border border-transparent font-medium rounded-md text-red-700 dark:text-red-100 bg-red-100 dark:bg-red-500 hover:bg-red-200 dark:hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:text-sm"
                >
                                    Delete all releases
                </button>
              </div>
            </ul>
          </div>
        </div>
      </div>
    </form>
  );
}

export default ReleaseSettings;