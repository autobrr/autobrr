/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useRef, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "react-hot-toast";

import { APIClient } from "@api/APIClient";
import { ReleaseKeys } from "@api/query_keys";
import Toast from "@components/notifications/Toast";
import { useToggle } from "@hooks/hooks";
import { DeleteModal } from "@components/modals";
import { Section } from "./_components";

const ReleaseSettings = () => (
  <Section
    title="Releases"
    description="Manage release history."
  >
    <div className="border border-red-500 rounded">
      <div className="py-6 px-4 sm:p-6">
        <div>
          <h2 className="text-lg leading-4 font-bold text-gray-900 dark:text-white">Danger zone</h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            This will clear release history in your database
          </p>
        </div>
      </div>

      <div className="py-6 px-4 sm:p-6">
        <DeleteReleases />
      </div>
    </div>
  </Section>
);


const getDurationLabel = (durationValue: number): string => {
  const durationOptions: Record<number, string> = {
    0: "all time",
    1: "1 hour",
    12: "12 hours",
    24: "1 day",
    168: "1 week",
    720: "1 month",
    2160: "3 months",
    4320: "6 months",
    8760: "1 year"
  };

  return durationOptions[durationValue] || "Invalid duration";
};

function DeleteReleases() {
  const queryClient = useQueryClient();
  const [duration, setDuration] = useState<string>("");
  const [parsedDuration, setParsedDuration] = useState<number>(0);
  const cancelModalButtonRef = useRef<HTMLInputElement | null>(null);
  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);

  const deleteOlderMutation = useMutation({
    mutationFn: (olderThan: number) => APIClient.release.delete(olderThan),
    onSuccess: () => {
      if (parsedDuration === 0) {
        toast.custom((t) => (
          <Toast type="success" body={"All releases were deleted."} t={t} />
        ));
      } else {
        toast.custom((t) => (
          <Toast type="success" body={`Releases older than ${getDurationLabel(parsedDuration)} were deleted.`} t={t} />
        ));
      }

      // Invalidate filters just in case, most likely not necessary but can't hurt.
      queryClient.invalidateQueries({ queryKey: ReleaseKeys.lists() });
    }
  });

  const deleteOlderReleases = () => {
    if (isNaN(parsedDuration) || parsedDuration < 0) {
      toast.custom((t) => <Toast type="error" body={"Please select a valid duration."} t={t} />);
      return;
    }

    deleteOlderMutation.mutate(parsedDuration);
  };

  return (
    <div className="flex flex-col sm:flex-row gap-2 justify-between items-center rounded-md">
      <DeleteModal
        isOpen={deleteModalIsOpen}
        isLoading={deleteOlderMutation.isPending}
        toggle={toggleDeleteModal}
        buttonRef={cancelModalButtonRef}
        deleteAction={deleteOlderReleases}
        title="Remove releases"
        text={`Are you sure you want to remove releases older than ${getDurationLabel(parsedDuration)}? This action cannot be undone.`}
      />

      <label htmlFor="duration" className="flex flex-col">
        <p className="text-sm font-medium text-gray-900 dark:text-white">Delete</p>
        <p className="text-sm text-gray-500 dark:text-gray-400">Delete releases older than select duration</p>
      </label>
      <div className="flex flex-wrap gap-2">
        <select
          name="duration"
          id="duration"
          className="focus:outline-none focus:ring-1 focus:ring-offset-0 focus:ring-blue-500 dark:focus:ring-blue-500 rounded-md sm:text-sm border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-white"
          value={duration}
          onChange={(e) => {
            const parsedDuration = parseInt(e.target.value, 10);
            setParsedDuration(parsedDuration);
            setDuration(e.target.value);
          }}
        >
          <option value="">Select duration</option>
          <option value="1">1 hour</option>
          <option value="12">12 hours</option>
          <option value="24">1 day</option>
          <option value="168">1 week</option>
          <option value="720">1 month</option>
          <option value="2160">3 months</option>
          <option value="4320">6 months</option>
          <option value="8760">1 year</option>
          <option value="0">Delete everything</option>
        </select>
        <button
          type="button"
          onClick={toggleDeleteModal}
          className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-red-700 hover:text-red-800 dark:text-white bg-red-200 dark:bg-red-700 hover:bg-red-300 dark:hover:bg-red-800 focus:outline-none focus:ring-1 focus:ring-inset focus:ring-red-600"
        >
          Delete
        </button>
      </div>
    </div>
  );
}

export default ReleaseSettings;
