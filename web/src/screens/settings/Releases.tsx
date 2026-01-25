/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment, useRef, useState } from "react";
import { useMutation, useQueryClient, useQuery, useSuspenseQuery } from "@tanstack/react-query";
import { MultiSelect as RMSC } from "react-multi-select-component";
import { Menu, MenuButton, MenuItem, MenuItems, Transition } from "@headlessui/react";
import { EllipsisHorizontalIcon, ForwardIcon, PencilSquareIcon, TrashIcon } from "@heroicons/react/24/outline";
import { PlusIcon } from "@heroicons/react/24/solid";
import { format } from "date-fns";

import { APIClient } from "@api/APIClient";
import { ReleaseKeys } from "@api/query_keys";
import { ReleaseProfileDuplicateList } from "@api/queries";
import { useToggle } from "@hooks/hooks";

import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { AgeSelect } from "@components/inputs"
import { DeleteModal } from "@components/modals";
import { EmptySimple } from "@components/emptystates";
import { Checkbox } from "@components/Checkbox";
import { Section } from "./_components";
import { ReleaseProfileDuplicateAddForm, ReleaseProfileDuplicateUpdateForm } from "@forms/settings/ReleaseForms";
import { CleanupJobAddForm, CleanupJobUpdateForm } from "@forms/settings/CleanupJobForms";
import { classNames } from "@utils";
import { PushStatusOptions } from "@domain/constants";

const ReleaseSettings = () => (
  <div className="lg:col-span-9">
    <ReleaseProfileDuplicates/>

    <div className="py-6 px-4 sm:p-6">
      <div className="border border-red-500 rounded-sm">
        {/* Warning about dangerous operations */}
        <div className="px-6 pt-6 pb-4">
          <span className="text-red-600 dark:text-red-500">
            <strong>Warning:</strong> This section manages release history deletion. Automated cleanup jobs run on schedules, while manual deletion allows immediate one-time cleanup. Both operations permanently delete data and cannot be undone.
          </span>
          <ul className="list-disc pl-5 mt-4 text-sm text-gray-500 dark:text-gray-400">
            <li>
              <strong className="text-gray-600 dark:text-gray-300">Older than</strong> - How old releases must be before deletion (Required for both automated and manual deletion)
            </li>
            <li><strong className="text-gray-600 dark:text-gray-300">Indexers</strong> - Optional filter (if none selected, applies to all indexers)</li>
            <li><strong className="text-gray-600 dark:text-gray-300">Release statuses</strong> - Optional filter (if none selected, applies to all release statuses)</li>
          </ul>
        </div>

        <ReleaseCleanupJobs/>

        <div className="py-6 px-4 sm:p-6">
          <DeleteReleases/>
        </div>
      </div>
    </div>
  </div>
);

interface ReleaseProfileProps {
  profile: ReleaseProfileDuplicate;
}

function ReleaseProfileListItem({ profile }: ReleaseProfileProps) {
  const [updatePanelIsOpen, toggleUpdatePanel] = useToggle(false);

  return (
    <li>
      <div className="grid grid-cols-12 items-center py-2">
        <ReleaseProfileDuplicateUpdateForm isOpen={updatePanelIsOpen} toggle={toggleUpdatePanel} data={profile}/>
        <div
          className="col-span-2 sm:col-span-2 lg:col-span-2 pl-4 sm:pl-4 pr-6 py-3 block flex-col text-sm font-medium text-gray-900 dark:text-white truncate"
          title={profile.name}>
          {profile.name}
        </div>
        <div className="col-span-9 sm:col-span-9 lg:col-span-9 pl-4 sm:pl-4 pr-6 py-3 flex gap-x-0.5 flex-row text-sm font-medium text-gray-900 dark:text-white truncate">
          {profile.release_name && <EnabledPill value={profile.release_name} label="RLS" title="Release name" />}
          {profile.hash && <EnabledPill value={profile.hash} label="Hash" title="Normalized hash of the release name. Use with Release name for exact match" />}
          {profile.title && <EnabledPill value={profile.title} label="Title" title="Parsed title" />}
          {profile.sub_title && <EnabledPill value={profile.sub_title} label="Sub Title" title="Parsed sub title like Episode name" />}
          {profile.group && <EnabledPill value={profile.group} label="Group" title="Release group" />}
          {profile.year && <EnabledPill value={profile.year} label="Year" title="Year" />}
          {profile.month && <EnabledPill value={profile.month} label="Month" title="Month" />}
          {profile.day && <EnabledPill value={profile.day} label="Day" title="Day" />}
          {profile.source && <EnabledPill value={profile.source} label="Source" title="Source" />}
          {profile.resolution && <EnabledPill value={profile.resolution} label="Resolution" title="Resolution" />}
          {profile.codec && <EnabledPill value={profile.codec} label="Codec" title="Codec" />}
          {profile.container && <EnabledPill value={profile.container} label="Container" title="Container" />}
          {profile.dynamic_range && <EnabledPill value={profile.dynamic_range} label="Dynamic Range" title="Dynamic Range (HDR,DV)" />}
          {profile.audio && <EnabledPill value={profile.audio} label="Audio" title="Audio formats" />}
          {profile.season && <EnabledPill value={profile.season} label="Season" title="Season number" />}
          {profile.episode && <EnabledPill value={profile.episode} label="Episode" title="Episode number" />}
          {profile.website && <EnabledPill value={profile.website} label="Website" title="Website/Service" />}
          {profile.proper && <EnabledPill value={profile.proper} label="Proper" title="Scene proper" />}
          {profile.repack && <EnabledPill value={profile.repack} label="Repack" title="Scene repack" />}
          {profile.edition && <EnabledPill value={profile.edition} label="Edition" title="Edition (eg. Collectors Edition) and Cut (eg. Directors Cut)" />}
          {profile.language && <EnabledPill value={profile.language} label="Language" title="Language and Region" />}
        </div>
        <div className="col-span-1 pl-0.5 whitespace-nowrap text-center text-sm font-medium">
          <span className="text-blue-600 dark:text-gray-300 hover:text-blue-900 cursor-pointer"
            onClick={toggleUpdatePanel}
          >
            Edit
          </span>
        </div>
      </div>

    </li>
  )
}

interface PillProps {
 value: boolean;
 label: string;
 title: string;
}

const EnabledPill = ({ value, label, title }: PillProps) => (
  <span title={title} className={classNames("inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium ring-1 ring-inset", value ? "bg-blue-100 dark:bg-blue-400/10 text-blue-700 dark:text-blue-400 ring-blue-700/10 dark:ring-blue-400/30" : "bg-gray-100 dark:bg-gray-400/10 text-gray-600 dark:text-gray-400 ring-gray-500/10 dark:ring-gray-400/30")}>
    {label}
  </span>
);

function ReleaseProfileDuplicates() {
  const [addPanelIsOpen, toggleAdd] = useToggle(false);

  const releaseProfileQuery = useSuspenseQuery(ReleaseProfileDuplicateList())

  return (
    <Section
      title="Release Duplicate Profiles"
      description="Manage duplicate profiles."
      rightSide={
        <button
          type="button"
          className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-xs cursor-pointer text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
          onClick={toggleAdd}
        >
          <PlusIcon className="h-5 w-5 mr-1"/>
          Add new
        </button>
      }
    >
      <ReleaseProfileDuplicateAddForm isOpen={addPanelIsOpen} toggle={toggleAdd}/>

      <div className="flex flex-col">
        {releaseProfileQuery.data.length > 0 ? (
          <ul className="min-w-full relative">
            <li className="grid grid-cols-12 border-b border-gray-200 dark:border-gray-700">
              <div
                className="col-span-2 sm:col-span-1 pl-1 sm:pl-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Name
              </div>
              {/*<div*/}
              {/*  className="col-span-6 sm:col-span-4 lg:col-span-4 pl-10 sm:pl-12 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"*/}
              {/*  // onClick={() => sortedClients.requestSort("name")}*/}
              {/*>*/}
              {/*  Name*/}
              {/*</div>*/}

              {/*<div*/}
              {/*  className="hidden sm:flex col-span-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"*/}
              {/*  onClick={() => sortedClients.requestSort("host")}*/}
              {/*>*/}
              {/*  Host <span className="sort-indicator">{sortedClients.getSortIndicator("host")}</span>*/}
              {/*</div>*/}
              {/*<div className="hidden sm:flex col-span-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer"*/}
              {/*     onClick={() => sortedClients.requestSort("type")}*/}
              {/*>*/}
              {/*  Type <span className="sort-indicator">{sortedClients.getSortIndicator("type")}</span>*/}
              {/*</div>*/}
            </li>
            {releaseProfileQuery.data.map((profile) => (
              <ReleaseProfileListItem key={profile.id} profile={profile}/>
            ))}
          </ul>
        ) : (
          <EmptySimple title="No duplicate rlease profiles" subtitle="" buttonText="Add new profile"
                       buttonAction={toggleAdd}/>
        )}
      </div>
    </Section>
  )
}

function ReleaseCleanupJobs() {
  const [addPanelIsOpen, toggleAdd] = useToggle(false);

  const cleanupJobsQuery = useSuspenseQuery({
    queryKey: ReleaseKeys.cleanupJobs.lists(),
    queryFn: () => APIClient.release.cleanupJobs.list()
  });

  return (
    <Section
      title="Release Cleanup Jobs"
      description="Schedule automatic cleanup of old releases with custom filters."
      rightSide={
        <button
          type="button"
          className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-xs cursor-pointer text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
          onClick={toggleAdd}
        >
          <PlusIcon className="h-5 w-5 mr-1"/>
          Add new
        </button>
      }
    >
      <CleanupJobAddForm isOpen={addPanelIsOpen} toggle={toggleAdd}/>

      <div className="flex flex-col">
        {cleanupJobsQuery.data.length > 0 ? (
          <ul className="min-w-full relative">
            <li className="grid grid-cols-12 border-b border-gray-200 dark:border-gray-700">
              <div className="col-span-1 pl-1 sm:pl-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Enabled
              </div>
              <div className="col-span-6 pl-12 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Name
              </div>
              <div className="col-span-2 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Last Run
              </div>
              <div className="col-span-2 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Next Run
              </div>
            </li>
            {cleanupJobsQuery.data.map((job) => (
              <CleanupJobListItem key={job.id} job={job}/>
            ))}
          </ul>
        ) : (
          <EmptySimple
            title="No cleanup jobs"
            subtitle="Create automated cleanup schedules"
            buttonText="Add new job"
            buttonAction={toggleAdd}
          />
        )}
      </div>
    </Section>
  );
}

interface CleanupJobListItemProps {
  job: ReleaseCleanupJob;
}

function CleanupJobListItem({ job }: CleanupJobListItemProps) {
  const [updatePanelIsOpen, toggleUpdatePanel] = useToggle(false);
  const queryClient = useQueryClient();

  const toggleMutation = useMutation({
    mutationFn: (enabled: boolean) => APIClient.release.cleanupJobs.toggleEnabled(job.id, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ReleaseKeys.cleanupJobs.lists() });
      toast.custom(t => <Toast type="success" body={`${job.name} ${job.enabled ? "disabled" : "enabled"}`} t={t} />);
    }
  });

  const forceRunMutation = useMutation({
    mutationFn: () => APIClient.release.cleanupJobs.forceRun(job.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ReleaseKeys.cleanupJobs.lists() });
      toast.custom(t => <Toast type="success" body={`${job.name} triggered`} t={t} />);
    }
  });

  // Format next_run timestamp (or "Not scheduled" if disabled)
  const nextRunDisplay = job.enabled && job.next_run !== "0001-01-01T00:00:00Z"
    ? format(new Date(job.next_run), "MMM d, HH:mm")
    : "—";

  // Format last_run status
  const lastRunDisplay = job.last_run !== "0001-01-01T00:00:00Z"
    ? job.last_run_status
    : "Never";

  return (
    <li>
      <CleanupJobUpdateForm isOpen={updatePanelIsOpen} toggle={toggleUpdatePanel} data={job}/>

      <div className="grid grid-cols-12 items-center py-1">
        <div className="col-span-1 flex pl-1 sm:pl-4 items-center">
          <Checkbox
            value={job.enabled}
            setValue={(newValue) => toggleMutation.mutate(newValue)}
          />
        </div>

        <div className="col-span-6 pl-12 pr-6 py-3 text-sm font-medium text-gray-900 dark:text-white truncate" title={job.name}>
          {job.name}
        </div>

        <div className="col-span-2 py-3 text-sm">
          <span className={classNames(
            "inline-flex items-center rounded-md px-2 py-1 text-xs font-medium ring-1 ring-inset",
            job.last_run_status === "SUCCESS"
              ? "bg-green-100 dark:bg-green-400/10 text-green-700 dark:text-green-400 ring-green-700/10 dark:ring-green-400/30"
              : job.last_run_status === "ERROR"
              ? "bg-red-100 dark:bg-red-400/10 text-red-700 dark:text-red-400 ring-red-700/10 dark:ring-red-400/30"
              : "bg-gray-100 dark:bg-gray-400/10 text-gray-600 dark:text-gray-400 ring-gray-500/10 dark:ring-gray-400/30"
          )}>
            {lastRunDisplay}
          </span>
        </div>

        <div className="col-span-2 py-3 text-sm text-gray-500 dark:text-gray-400">
          {nextRunDisplay}
        </div>

        <div className="col-span-1 md:col-span-1 sm:col-span-2 flex justify-center items-center md:px-6">
          <CleanupItemDropdown job={job} toggleUpdate={toggleUpdatePanel} forceRun={forceRunMutation.mutate} />
        </div>
      </div>
    </li>
  );
}

interface CleanupItemDropdownProps {
  job: ReleaseCleanupJob;
  toggleUpdate: () => void;
  forceRun: () => void;
}

function CleanupItemDropdown({ job, toggleUpdate, forceRun}: CleanupItemDropdownProps) {
  const cancelModalButtonRef = useRef(null);
  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);

  const queryClient = useQueryClient();

  const deleteMutation = useMutation({
    mutationFn: (jobId: number) => APIClient.release.cleanupJobs.delete(jobId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ReleaseKeys.cleanupJobs.lists() });
      queryClient.invalidateQueries({ queryKey: ReleaseKeys.cleanupJobs.detail(job.id) });
      toast.custom((t) => <Toast type="success" body={`${job.name} deleted`} t={t} />);
      toggleDeleteModal();
    },
  });

  return (
    <Menu as="div">
      <DeleteModal
        isOpen={deleteModalIsOpen}
        isLoading={deleteMutation.isPending}
        toggle={toggleDeleteModal}
        buttonRef={cancelModalButtonRef}
        deleteAction={() => {
          deleteMutation.mutate(job.id);
          toggleDeleteModal();
        }}
        title={`Remove cleanup job: ${job.name}`}
        text="Are you sure you want to remove this cleanup job? This action cannot be undone."
      />

      <MenuButton className="px-4 py-2">
        <EllipsisHorizontalIcon
          className="cursor-pointer w-5 h-5 text-gray-700 hover:text-gray-900 dark:text-gray-100 dark:hover:text-gray-400"
          aria-hidden="true"
        />
      </MenuButton>
      <Transition
        as={Fragment}
        enter="transition ease-out duration-100"
        enterFrom="transform opacity-0 scale-95"
        enterTo="transform opacity-100 scale-100"
        leave="transition ease-in duration-75"
        leaveFrom="transform opacity-100 scale-100"
        leaveTo="transform opacity-0 scale-95"
      >
        <MenuItems
          anchor={{ to: 'bottom end', padding: '8px' }} // padding: '8px' === m-2
          className="absolute w-56 bg-white dark:bg-gray-825 divide-y divide-gray-200 dark:divide-gray-750 rounded-md shadow-lg border border-gray-250 dark:border-gray-750 focus:outline-hidden z-10"
        >
          <div className="px-1 py-1">
            <MenuItem>
              {({ focus }) => (
                <button
                  className={classNames(
                    focus ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                    "cursor-pointer font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                  )}
                  onClick={() => toggleUpdate()}
                >
                  <PencilSquareIcon
                    className={classNames(
                      focus ? "text-white" : "text-blue-500",
                      "w-5 h-5 mr-2"
                    )}
                    aria-hidden="true"
                  />
                  Edit
                </button>
              )}
            </MenuItem>
          </div>
          <div className="px-1 py-1">
            <MenuItem>
              {({ focus }) => (
                <button
                  onClick={() => forceRun()}
                  className={classNames(
                    focus ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                    "cursor-pointer font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                  )}
                >
                  <ForwardIcon
                    className={classNames(
                      focus ? "text-white" : "text-blue-500",
                      "w-5 h-5 mr-2"
                    )}
                    aria-hidden="true"
                  />
                  Run now
                </button>
              )}
            </MenuItem>
          </div>
          <div className="px-1 py-1">
            <MenuItem>
              {({ focus }) => (
                <button
                  className={classNames(
                    focus ? "bg-red-600 text-white" : "text-gray-900 dark:text-gray-300",
                    "cursor-pointer font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                  )}
                  onClick={() => toggleDeleteModal()}
                >
                  <TrashIcon
                    className={classNames(
                      focus ? "text-white" : "text-red-500",
                      "w-5 h-5 mr-2"
                    )}
                    aria-hidden="true"
                  />
                  Delete
                </button>
              )}
            </MenuItem>
          </div>
        </MenuItems>
      </Transition>
    </Menu>
  )
}

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

  const deleteOlderMutation = useMutation({
    mutationFn: (params: { olderThan: number, indexers: string[], releaseStatuses: string[] }) =>
      APIClient.release.delete(params),
    onSuccess: () => {
      if (parsedDuration === 0) {
        toast.custom((t) => (
          <Toast type="success" body={"All releases based on criteria were deleted."} t={t}/>
        ));
      } else {
        toast.custom((t) => (
          <Toast type="success" body={`Releases older than ${getDurationLabel(parsedDuration ?? 0)} were deleted.`}
                 t={t}/>
        ));
      }

      queryClient.invalidateQueries({ queryKey: ReleaseKeys.lists() });
    }
  });

  const deleteOlderReleases = () => {
    if (parsedDuration === undefined || isNaN(parsedDuration) || parsedDuration < 0) {
      toast.custom((t) => <Toast type="error" body={"Please select a valid age."} t={t}/>);
      return;
    }

    deleteOlderMutation.mutate({
      olderThan: parsedDuration,
      indexers: indexers.map(i => i.value),
      releaseStatuses: releaseStatuses.map(rs => rs.value)
    });
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
          <p className="text-sm mt-2 text-gray-500 dark:text-gray-400">
            Select the criteria below to permanently delete release history records that are older than the chosen age
            and optionally match the selected indexers and release statuses:
          </p>
        </div>

        <div className="flex flex-col sm:flex-row gap-2 pt-4 items-center text-sm">
          {[
            {
              label: (
                <span>
                  Older than:
                  <span className="text-red-600 dark:text-red-500"> *</span>
                </span>
              ),
              content: <AgeSelect duration={duration} setDuration={setDuration} setParsedDuration={setParsedDuration}/>
            },
            {
              label: 'Indexers:',
              content: <RMSC
                options={indexerOptions?.map(option => ({ value: option.identifier, label: option.name })) || []}
                value={indexers} onChange={setIndexers} labelledBy="Select indexers"/>
            },
            {
              label: 'Release statuses:',
              content: <RMSC options={PushStatusOptions} value={releaseStatuses} onChange={setReleaseStatuses}
                             labelledBy="Select release statuses"/>
            }
          ].map((item, index) => (
            <div key={index} className="flex flex-col w-full">
              <p
                className="text-xs font-bold text-gray-800 dark:text-gray-100 uppercase p-1 cursor-default">{item.label}</p>
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
            className="inline-flex justify-center sm:w-1/5 md:w-1/5 w-full px-4 py-2 sm:mt-6 border border-transparent cursor-pointer text-sm font-medium rounded-md text-red-700 hover:text-red-800 dark:text-white bg-red-200 dark:bg-red-700 hover:bg-red-300 dark:hover:bg-red-800 focus:outline-hidden focus:ring-1 focus:ring-inset focus:ring-red-600"
          >
            Delete
          </button>

        </div>
      </div>
    </div>
  );
}

export default ReleaseSettings;
