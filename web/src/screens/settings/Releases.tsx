/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useRef, useState } from "react";
import { useMutation, useQueryClient, useQuery } from "@tanstack/react-query";
import { MultiSelect as RMSC } from "react-multi-select-component";
import { AgeSelect } from "@components/inputs"

import { APIClient } from "@api/APIClient";
import { ReleaseKeys } from "@api/query_keys";
import { toast } from "@components/hot-toast";
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
      toast.custom((t) => <Toast type="error" body={"Please select a valid age."} t={t} />);
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
        text={`You are about to ${parsedDuration ? `permanently delete all release history records older than ${getDurationLabel(parsedDuration)} for ` : 'delete all release history records for '}${indexers.length ? 'the chosen indexers' : 'all indexers'}${releaseStatuses.length ? ` and with the following release statuses: ${releaseStatuses.map(status => status.label).join(', ')}` : ''}.`}
      />
      <div className="flex flex-col gap-2 w-full">
        <div>
          <h2 className="text-lg leading-4 font-bold text-gray-900 dark:text-white">Delete release history</h2>
          <p className="text-sm mt-1 text-gray-500 dark:text-gray-400">
            Select the criteria below to permanently delete release history records that are older than the chosen age and optionally match the selected indexers and release statuses:
            <ul className="list-disc pl-5 mt-2">
              <li>
                Older than (e.g., 6 months - all records older than 6 months will be deleted) - <strong className="text-gray-600 dark:text-gray-300">Required</strong>
              </li>
              <li>Indexers - Optional (if none selected, applies to all indexers)</li>
              <li>Release statuses - Optional (if none selected, applies to all release statuses)</li>
            </ul>
            <p className="mt-2 text-red-600 dark:text-red-500">
              <strong>Warning:</strong> If no indexers or release statuses are selected, all release history records older than the selected age will be permanently deleted, regardless of indexer or status.
            </p>
          </p>
        </div>

        <div className="flex flex-col sm:flex-row gap-2 pt-4 items-center text-sm">
          {[
            {
              label: (
                <>
                  Older than:
                  <span className="text-red-600 dark:text-red-500"> *</span>
                </>
              ),
              content: <AgeSelect duration={duration} setDuration={setDuration} setParsedDuration={setParsedDuration} />
            },
            {
              label: 'Indexers:',
              content: <RMSC options={indexerOptions?.map(option => ({ value: option.identifier, label: option.name })) || []} value={indexers} onChange={setIndexers} labelledBy="Select indexers" />
            },
            {
              label: 'Release statuses:',
              content: <RMSC options={releaseStatusOptions} value={releaseStatuses} onChange={setReleaseStatuses} labelledBy="Select release statuses" />
            }
          ].map((item, index) => (
            <div key={index} className="flex flex-col w-full">
              <p className="text-xs font-bold text-gray-800 dark:text-gray-100 uppercase p-1 cursor-default">{item.label}</p>
              {item.content}
            </div>
          ))}
          <button
            type="button"
            onClick={() => {
              if (parsedDuration === undefined || isNaN(parsedDuration)) {
                toast.custom((t) => (
                  <Toast
                    type="error"
                    body={
                      "Please enter a valid age. For example, 6 months or 1 year."
                    }
                    t={t}
                  />
                ));
              } else {
                toggleDeleteModal();
              }
            }}
            className="inline-flex justify-center sm:w-1/5 md:w-1/5 w-full px-4 py-2 sm:mt-6 border border-transparent text-sm font-medium rounded-md text-red-700 hover:text-red-800 dark:text-white bg-red-200 dark:bg-red-700 hover:bg-red-300 dark:hover:bg-red-800 focus:outline-none focus:ring-1 focus:ring-inset focus:ring-red-600"
          >
            Delete
          </button>

        </div>
      </div>
    </div>

  );

}

export default ReleaseSettings;
