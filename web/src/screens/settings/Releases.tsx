/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "react-hot-toast";

import { APIClient } from "@api/APIClient";
import Toast from "@components/notifications/Toast";
import { releaseKeys } from "@screens/releases/ReleaseTable";

function ReleaseSettings() {

  const queryClient = useQueryClient();
  const [duration, setDuration] = useState<string>("");

  const getDurationLabel = (durationValue: number): string => {
    const durationOptions: Record<number, string> = {
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

  const deleteOlderMutation = useMutation({
    mutationFn: (duration: number) => APIClient.release.deleteOlder(duration),
    onSuccess: () => {
      const parsedDuration = parseInt(duration, 10);
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
      queryClient.invalidateQueries({ queryKey: releaseKeys.lists() });
    }
  });

  const deleteOlderReleases = () => {
    const parsedDuration = parseInt(duration, 10);

    if (isNaN(parsedDuration) || parsedDuration < 0) {
      toast.custom((t) => <Toast type="error" body={"Please select a valid duration."} t={t} />);
    } else {
      deleteOlderMutation.mutate(parsedDuration);
    }
  };

  return (
    <form
      className="lg:col-span-9"
      action="#"
      method="POST"
    >
      <div className="py-6 px-4 sm:p-6 lg:pb-8">
        <div>
          <h2 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">
            Releases
          </h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Manage release history.
          </p>
        </div>
      </div>

      <div className="pb-6 divide-y divide-gray-200 dark:divide-gray-700">
        <div className="px-4">
          <div className="px-4">
            <div>
              <h3 className="text-center sm:text-lg leading-6 font-medium text-gray-900 dark:text-white">
                Danger Zone
              </h3>
              <p className="text-center mt-1 text-sm text-gray-900 dark:text-white">
                This will clear release history in your database
              </p>
            </div>
            <div className="mt-8">
              <div className="max-w-sm mx-auto">
                <label htmlFor="duration" className="block text-sm text-gray-700 dark:text-white">
                  Delete releases older than:
                </label>
                <div className="flex items-center mt-2 rounded-md shadow-sm">
                  <select
                    name="duration"
                    id="duration"
                    className="focus:outline-none focus:ring-1 focus:ring-offset-0 focus:ring-blue-500 dark:focus:ring-blue-500 flex-1 block w-full rounded-md sm:text-sm border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-white"
                    value={duration}
                    onChange={(e) => setDuration(e.target.value)}
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
                    onClick={deleteOlderReleases}
                    className="ml-2 inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-red-700 hover:text-red-900 dark:text-white bg-red-100 dark:bg-red-800 hover:bg-red-200 dark:hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
                  >
                    Delete
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </form>
  );
}

export default ReleaseSettings;