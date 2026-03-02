/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { MultiSelect as RMSC } from "react-multi-select-component";
import { format } from "date-fns";

import { APIClient } from "@api/APIClient";
import { ReleaseKeys } from "@api/query_keys";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { DurationFieldWide, SwitchGroupWide, TextFieldWide } from "@components/inputs/tanstack";
import { SlideOver } from "@components/panels";
import { AddFormProps, UpdateFormProps } from "@forms/_shared";
import { classNames } from "@utils";
import { PushStatusOptions } from "@domain/constants";
import { ContextField, useFormContext, useStore } from "@app/lib/form";

function IndexerMultiSelectField({ indexerOptions, labelledBy }: { indexerOptions?: { identifier: string; name: string }[]; labelledBy: string }) {
  const form = useFormContext();
  const value = useStore(form.store, (s: any) => s.values.indexers) as string;

  const computedValue = !value || value === '' || !indexerOptions
    ? []
    : value.split(',').filter(Boolean).map((v: string) => {
        const option = indexerOptions.find(opt => opt.identifier === v);
        return option ? { value: option.identifier, label: option.name } : null;
      }).filter((item): item is { value: string; label: string } => item !== null);

  return (
    <div onClick={(e) => { e.stopPropagation(); e.nativeEvent.stopImmediatePropagation(); }}>
      <RMSC
        options={indexerOptions?.map(opt => ({ value: opt.identifier, label: opt.name })) || []}
        value={computedValue}
        onChange={(selected: { value: string; label: string }[]) => {
          const indexerString = selected.map(s => s.value).join(',');
          (form as any).setFieldValue("indexers", indexerString);
        }}
        labelledBy={labelledBy}
      />
    </div>
  );
}

function StatusMultiSelectField({ labelledBy }: { labelledBy: string }) {
  const form = useFormContext();
  const value = useStore(form.store, (s: any) => s.values.statuses) as string;

  const computedValue = !value || value === ''
    ? []
    : value.split(',').filter(Boolean).map((v: string) => {
        const option = PushStatusOptions.find(opt => opt.value === v);
        return option ? { value: option.value, label: option.label } : null;
      }).filter((item): item is { value: string; label: string } => item !== null);

  return (
    <div onClick={(e) => { e.stopPropagation(); e.nativeEvent.stopImmediatePropagation(); }}>
      <RMSC
        options={PushStatusOptions}
        value={computedValue}
        onChange={(selected: { value: string; label: string }[]) => {
          const statusString = selected.map(s => s.value).join(',');
          (form as any).setFieldValue("statuses", statusString);
        }}
        labelledBy={labelledBy}
      />
    </div>
  );
}

export function CleanupJobAddForm({isOpen, toggle}: AddFormProps) {
  const queryClient = useQueryClient();

  const addMutation = useMutation({
    mutationFn: (job: ReleaseCleanupJob) => APIClient.release.cleanupJobs.store(job),
    onSuccess: () => {
      queryClient.invalidateQueries({queryKey: ReleaseKeys.cleanupJobs.lists()});
      toast.custom((t) => <Toast type="success" body="Cleanup job created" t={t}/>);
      toggle();
    },
    onError: () => {
      toast.custom((t) => <Toast type="error" body="Job could not be created" t={t}/>);
    }
  });

  const onSubmit = (data: unknown) => addMutation.mutate(data as ReleaseCleanupJob);

  const initialValues: Partial<ReleaseCleanupJob> = {
    name: "",
    enabled: true,
    schedule: "0 3 * * *",    // Default: Daily at 3 AM
    older_than: 720,           // Default: 30 days in hours
    indexers: "",
    statuses: ""
  };

  const {data: indexerOptions} = useQuery<IndexerDefinition[], Error, { identifier: string; name: string; }[]>({
    queryKey: ['indexers'],
    queryFn: () => APIClient.indexers.getAll(),
    select: data => data.map(indexer => ({
      identifier: indexer.identifier,
      name: indexer.name
    })),
  });

  return (
    <SlideOver
      type="CREATE"
      title="Cleanup Job"
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      initialValues={initialValues}
    >
      {() => (
        <div className="py-2 space-y-6 sm:py-0 sm:space-y-0 divide-y divide-gray-200 dark:divide-gray-700">
          <ContextField name="name">
            <TextFieldWide required label="Name" placeholder="Weekly Cleanup"/>
          </ContextField>

          <ContextField name="enabled">
            <SwitchGroupWide label="Enabled" description="Enable this cleanup job"/>
          </ContextField>

          <ContextField name="schedule">
            <TextFieldWide
              required
              label="Schedule (Cron)"
              placeholder="0 3 * * *"
              help="Cron expression. Examples: '0 3 * * *' = daily 3 AM, '0 */6 * * *' = every 6 hours"
            />
          </ContextField>

          <ContextField name="older_than">
            <DurationFieldWide
              required
              label="Older than"
              placeholder="30"
              help="Delete releases older than this duration"
              defaultValue={30}
              defaultUnit="days"
              units={["hours", "days", "weeks", "months", "years"]}
              storeAsHours={true}
            />
          </ContextField>

          <div className="py-4 sm:py-5 sm:grid sm:grid-cols-3 sm:gap-4 px-4 sm:px-6">
            <label className="text-sm font-medium text-gray-900 dark:text-white">
              Indexers (Optional)
            </label>
            <div className="col-span-2">
              <IndexerMultiSelectField indexerOptions={indexerOptions} labelledBy="cleanup-job-add-indexers" />
              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Leave empty to apply to all indexers
              </p>
            </div>
          </div>

          <div className="py-4 sm:py-5 sm:grid sm:grid-cols-3 sm:gap-4 px-4 sm:px-6">
            <label className="text-sm font-medium text-gray-900 dark:text-white">
              Statuses (Optional)
            </label>
            <div className="col-span-2">
              <StatusMultiSelectField labelledBy="cleanup-job-add-statuses" />
              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Leave empty to apply to all statuses
              </p>
            </div>
          </div>
        </div>
      )}
    </SlideOver>
  );
}

export function CleanupJobUpdateForm({isOpen, toggle, data: job}: UpdateFormProps<ReleaseCleanupJob>) {
  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: (job: ReleaseCleanupJob) => APIClient.release.cleanupJobs.update(job),
    onSuccess: () => {
      queryClient.invalidateQueries({queryKey: ReleaseKeys.cleanupJobs.lists()});
      toast.custom((t) => <Toast type="success" body="Job updated" t={t}/>);
      toggle();
    },
    onError: () => {
      toast.custom((t) => <Toast type="error" body="Job could not be updated" t={t}/>);
    }
  });

  const onSubmit = (data: unknown) => updateMutation.mutate(data as ReleaseCleanupJob);

  const deleteMutation = useMutation({
    mutationFn: (jobId: number) => APIClient.release.cleanupJobs.delete(jobId),
    onSuccess: () => {
      queryClient.invalidateQueries({queryKey: ReleaseKeys.cleanupJobs.lists()});
      queryClient.invalidateQueries({queryKey: ReleaseKeys.cleanupJobs.detail(job.id)});
      toast.custom((t) => <Toast type="success" body={`${job.name} deleted`} t={t}/>);
      toggle();
    },
  });

  const onDelete = () => deleteMutation.mutate(job.id);

  const initialValues: ReleaseCleanupJob = {
    id: job.id,
    name: job.name,
    enabled: job.enabled,
    schedule: job.schedule,
    older_than: job.older_than,
    indexers: job.indexers,
    statuses: job.statuses,
    // Read-only fields
    last_run: job.last_run,
    last_run_status: job.last_run_status,
    last_run_data: job.last_run_data,
    next_run: job.next_run,
    created_at: job.created_at,
    updated_at: job.updated_at
  };

  // Get indexer options for multi-select
  const {data: indexerOptions} = useQuery<IndexerDefinition[], Error, { identifier: string; name: string; }[]>({
    queryKey: ['indexers'],
    queryFn: () => APIClient.indexers.getAll(),
    select: data => data.map(indexer => ({
      identifier: indexer.identifier,
      name: indexer.name
    })),
  });

  return (
    <SlideOver
      type="UPDATE"
      title="Cleanup Job"
      isOpen={isOpen}
      toggle={toggle}
      deleteAction={onDelete}
      onSubmit={onSubmit}
      initialValues={initialValues}
    >
      {() => (
        <div className="py-2 space-y-6 sm:py-0 sm:space-y-0 divide-y divide-gray-200 dark:divide-gray-700">
          {/* Same fields as AddForm */}
          <ContextField name="name">
            <TextFieldWide required label="Name" placeholder="Weekly Cleanup"/>
          </ContextField>

          <ContextField name="enabled">
            <SwitchGroupWide label="Enabled" description="Enable this cleanup job"/>
          </ContextField>

          <ContextField name="schedule">
            <TextFieldWide
              required
              label="Schedule (Cron)"
              placeholder="0 3 * * *"
              help="Cron expression. Examples: '0 3 * * *' = daily 3 AM, '0 */6 * * *' = every 6 hours"
            />
          </ContextField>

          <ContextField name="older_than">
            <DurationFieldWide
              required
              label="Older than"
              placeholder="30"
              help="Delete releases older than this duration"
              defaultValue={30}
              defaultUnit="days"
              units={["hours", "days", "weeks", "months", "years"]}
              storeAsHours={true}
            />
          </ContextField>

          <div className="py-4 sm:py-5 sm:grid sm:grid-cols-3 sm:gap-4 px-4 sm:px-6">
            <label className="text-sm font-medium text-gray-900 dark:text-white">
              Indexers (Optional)
            </label>
            <div className="col-span-2">
              <IndexerMultiSelectField indexerOptions={indexerOptions} labelledBy="cleanup-job-edit-indexers" />
              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Leave empty to apply to all indexers
              </p>
            </div>
          </div>

          <div className="py-4 sm:py-5 sm:grid sm:grid-cols-3 sm:gap-4 px-4 sm:px-6">
            <label className="text-sm font-medium text-gray-900 dark:text-white">
              Statuses (Optional)
            </label>
            <div className="col-span-2">
              <StatusMultiSelectField labelledBy="cleanup-job-edit-statuses" />
              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Leave empty to apply to all statuses
              </p>
            </div>
          </div>

          {/* Execution History Section */}
          {job.last_run !== "0001-01-01T00:00:00Z" && (
            <div className="py-4 sm:py-5 px-4 sm:px-6">
              <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">
                Execution History
              </h3>
              <dl className="space-y-2">
                <div className="flex justify-between">
                  <dt className="text-sm text-gray-500 dark:text-gray-400">Last Run:</dt>
                  <dd className="text-sm text-gray-900 dark:text-white">
                    {format(new Date(job.last_run), "MMM d, yyyy HH:mm:ss")}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-sm text-gray-500 dark:text-gray-400">Status:</dt>
                  <dd>
                    <span className={classNames(
                      "inline-flex items-center rounded-md px-2 py-1 text-xs font-medium",
                      job.last_run_status === "SUCCESS"
                        ? "bg-green-100 text-green-700 dark:bg-green-400/10 dark:text-green-400"
                        : "bg-red-100 text-red-700 dark:bg-red-400/10 dark:text-red-400"
                    )}>
                      {job.last_run_status}
                    </span>
                  </dd>
                </div>
              </dl>
            </div>
          )}
        </div>
      )}
    </SlideOver>
  );
}
