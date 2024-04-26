/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useRef, useState } from "react";
import { useMutation, useQueryClient, useQuery } from "@tanstack/react-query";
import { toast } from "react-hot-toast";
import { MultiSelect as RMSC } from "react-multi-select-component";
import { AgeSelect } from "@components/inputs"

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
          <h2 className="text-lg font-medium text-gray-900 dark:text-white">Danger zone</h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            This action will permanently delete release history from your database.
          </p>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            This change cannot be undone.
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

interface Indexer {
  label: string;
  value: string;
}

interface ReleaseStatus {
  label: string;
  value: string;
}

function DeleteReleases() {
  const queryClient = useQueryClient();
  const [duration, setDuration] = useState<string>("");
  const [parsedDuration, setParsedDuration] = useState<number>();
  const [indexers, setIndexers] = useState<Indexer[]>([]);
  const [releaseStatuses, setReleaseStatuses] = useState<ReleaseStatus[]>([]);
  const cancelModalButtonRef = useRef<HTMLInputElement | null>(null);
  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);

  const { data: indexerOptions } = useQuery<IndexerDefinition[], Error, { identifier: string; name: string; }[]>({
    queryKey: ['indexers'],
    queryFn: () => APIClient.indexers.getAll(),
    select: data => data.map(indexer => ({
      identifier: indexer.identifier,
      name: indexer.name
    })),
  });

  const releaseStatusOptions = [
    { label: "Approved", value: "PUSH_APPROVED" },
    { label: "Rejected", value: "PUSH_REJECTED" },
    { label: "Errored", value: "PUSH_ERROR" }
  ];

  const deleteOlderMutation = useMutation({
    mutationFn: (params: { olderThan: number, indexers: string[], releaseStatuses: string[] }) =>
      APIClient.release.delete(params),
    onSuccess: () => {
      if (parsedDuration === 0) {
        toast.custom((t) => (
          <Toast type="success" body={"All releases based on criteria were deleted."} t={t} />
        ));
      } else {
        toast.custom((t) => (
          <Toast type="success" body={`Releases older than ${getDurationLabel(parsedDuration ?? 0)} were deleted.`} t={t} />
        ));
      }

      queryClient.invalidateQueries({ queryKey: ReleaseKeys.lists() });
    }
  });

  const deleteOlderReleases = () => {
    if (parsedDuration === undefined || isNaN(parsedDuration) || parsedDuration < 0) {
      toast.custom((t) => <Toast type="error" body={"Please select a valid duration."} t={t} />);
      return;
    }

    deleteOlderMutation.mutate({ olderThan: parsedDuration, indexers: indexers.map(i => i.value), releaseStatuses: releaseStatuses.map(rs => rs.value) });
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
        text={`Are you sure you want to remove releases matching the selected criteria? This action cannot be undone.`}
      />
      <div className="flex flex-col gap-2 w-full">
        <div>
          <h2 className="text-sm font-medium text-gray-900 dark:text-white">Manage release history</h2>
          <p className="text-sm mt-1 text-gray-500 dark:text-gray-400">Provide options to delete release history by indexers, statuses, and age.</p>
          <p className="text-sm text-gray-500 dark:text-gray-400">Choosing age is mandatory.</p>
        </div>
        <div className="flex flex-row gap-2 py-4 items-center">
          <div className="flex w-full flex-col">
            <p className="text-sm text-gray-900 dark:text-gray-400 p-1">Age:</p>
            <AgeSelect
              duration={duration}
              setDuration={setDuration}
              setParsedDuration={setParsedDuration}
            />
          </div>

          <div className="flex w-full flex-col">
            <p className="text-sm text-gray-900 dark:text-gray-400 p-1">Indexers:</p>
            <RMSC
              options={indexerOptions?.map(option => ({ value: option.identifier, label: option.name })) || []}
              value={indexers}
              onChange={setIndexers}
              labelledBy="Select indexers"
              className="w-full focus:outline-none focus:ring-1 focus:ring-offset-0 focus:ring-blue-500 dark:focus:ring-blue-500 rounded-md sm:text-sm border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-white"
            />
          </div>

          <div className="flex w-full flex-col">
            <p className="text-sm text-gray-900 dark:text-gray-400 p-1">Release status:</p>
            <RMSC
              options={releaseStatusOptions}
              value={releaseStatuses}
              onChange={setReleaseStatuses}
              labelledBy="Select release statuses"
              className="w-full focus:outline-none focus:ring-1 focus:ring-offset-0 focus:ring-blue-500 dark:focus:ring-blue-500 rounded-md sm:text-sm border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-white"
            />
          </div>

          <button
            type="button"
            onClick={toggleDeleteModal}
            className="inline-flex items-center px-4 py-2 mt-7 border border-transparent text-sm font-medium rounded-md text-red-700 hover:text-red-800 dark:text-white bg-red-200 dark:bg-red-700 hover:bg-red-300 dark:hover:bg-red-800 focus:outline-none focus:ring-1 focus:ring-inset focus:ring-red-600"
          >
            Delete
          </button>

        </div>
      </div>
    </div>
  );

}

export default ReleaseSettings;
