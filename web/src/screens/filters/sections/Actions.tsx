/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Field, FieldArray, useFormikContext } from "formik";
import type { FieldProps, FieldArrayRenderProps } from "formik";
import { ChevronRightIcon, BoltIcon } from "@heroicons/react/24/solid";

import { classNames } from "@utils";
import { useToggle } from "@hooks/hooks";
import { APIClient } from "@api/APIClient";
import { ActionTypeNameMap, ActionTypeOptions, DOWNLOAD_CLIENTS } from "@domain/constants";

import { Select, TextField } from "@components/inputs";
import { DeleteModal } from "@components/modals";
import { EmptyListState } from "@components/emptystates";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";

import { Checkbox } from "@components/Checkbox";
import { TitleSubtitle } from "@components/headings";

import { DownloadClientsQueryOptions } from "@api/queries";
import { FilterHalfRow, FilterLayout, FilterPage, FilterSection } from "@screens/filters/sections/_components.tsx";
import {
  Arr,
  Deluge, Exec,
  Porla,
  QBittorrent,
  RTorrent,
  SABnzbd, Test,
  Transmission, WatchFolder, WebHook
} from "@screens/filters/sections/action_components";

// interface FilterActionsProps {
//   filter: Filter;
//   values: FormikValues;
// }

export function Actions() {
  const { values } = useFormikContext<Filter>();

  const { data } = useQuery(DownloadClientsQueryOptions());

  const newAction: Action = {
    id: 0,
    name: "new action",
    enabled: true,
    type: "TEST",
    watch_folder: "",
    exec_cmd: "",
    exec_args: "",
    category: "",
    tags: "",
    label: "",
    save_path: "",
    paused: false,
    ignore_rules: false,
    first_last_piece_prio: false,
    skip_hash_check: false,
    content_layout: "",
    priority: "",
    limit_upload_speed: 0,
    limit_download_speed: 0,
    limit_ratio: 0,
    limit_seed_time: 0,
    reannounce_skip: false,
    reannounce_delete: false,
    reannounce_interval: 7,
    reannounce_max_attempts: 25,
    filter_id: values.id,
    webhook_host: "",
    webhook_type: "",
    webhook_method: "",
    webhook_data: "",
    webhook_headers: [],
    external_download_client_id: 0,
    external_download_client: "",
    client_id: 0
  };

  return (
    <div className="mt-5">
      <FieldArray name="actions">
        {({ remove, push }: FieldArrayRenderProps) => (
          <>
            <div className="-ml-4 -mt-4 mb-6 flex justify-between items-center flex-wrap sm:flex-nowrap">
              <TitleSubtitle
                className="ml-4 mt-4"
                title="Actions"
                subtitle="Add to download clients or run custom commands."
              />
              <div className="ml-4 mt-4 flex-shrink-0">
                <button
                  type="button"
                  className="relative inline-flex items-center px-4 py-2 border border-transparent transition shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                  onClick={() => push(newAction)}
                >
                  <BoltIcon
                    className="w-5 h-5 mr-1"
                    aria-hidden="true"
                  />
                  Add new
                </button>
              </div>
            </div>

            {values.actions.length > 0 ? (
              <ul className="rounded-md">
                {values.actions.map((action: Action, index: number) => (
                  <FilterActionsItem
                    key={action.id}
                    action={action}
                    clients={data ?? []}
                    idx={index}
                    initialEdit={values.actions.length === 1}
                    remove={remove}
                  />
                ))}
              </ul>
            ) : (
              <EmptyListState text="No actions yet!" />
            )}
          </>
        )}
      </FieldArray>
    </div>
  );
}

const TypeForm = (props: ClientActionProps) => {
  const { setFieldValue } = useFormikContext();
  const [prevActionType, setPrevActionType] = useState<string | null>(null);

  const { action, idx } = props;

  useEffect(() => {
    if (prevActionType !== null && prevActionType !== action.type && DOWNLOAD_CLIENTS.includes(action.type)) {
      // Reset the client_id field value
      setFieldValue(`actions.${idx}.client_id`, 0);
    }

    setPrevActionType(action.type);
  }, [action.type, idx, prevActionType, setFieldValue]);

  switch (action.type) {
  // torrent clients
  case "QBITTORRENT":
    return <QBittorrent {...props} />;
  case "DELUGE_V1":
  case "DELUGE_V2":
    return <Deluge {...props} />;
  case "RTORRENT":
    return <RTorrent {...props} />;
  case "TRANSMISSION":
    return <Transmission {...props} />;
  case "PORLA":
    return <Porla {...props} />;
  // arrs
  case "RADARR":
  case "SONARR":
  case "LIDARR":
  case "WHISPARR":
  case "READARR":
    return <Arr {...props} />;
  // nzb
  case "SABNZBD":
    return <SABnzbd {...props} />;
  // autobrr actions
  case "TEST":
    return <Test />;
  case "EXEC":
    return <Exec {...props} />;
  case "WATCH_FOLDER":
    return <WatchFolder {...props} />;
  case "WEBHOOK":
    return <WebHook {...props} />;
  default:
    // TODO(stacksmash76): Indicate error
    return null;
  }
};

interface FilterActionsItemProps {
  action: Action;
  clients: DownloadClient[];
  idx: number;
  initialEdit: boolean;
  remove: <T>(index: number) => T | undefined;
}

function FilterActionsItem({ action, clients, idx, initialEdit, remove }: FilterActionsItemProps) {
  const cancelButtonRef = useRef(null);

  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);
  const [edit, toggleEdit] = useToggle(initialEdit);

  const removeMutation = useMutation({
    mutationFn: (id: number) => APIClient.actions.delete(id),
    onSuccess: () => {
      remove(idx);
      // Invalidate filters just in case, most likely not necessary but can't hurt.
      // queryClient.invalidateQueries({ queryKey: filterKeys.detail(id) });

      toast.custom((t) => (
        <Toast type="success" body={`Action ${action?.name} was deleted`} t={t} />
      ));
    }
  });

  const removeAction = (id: number) => {
    removeMutation.mutate(id);
  };

  return (
    <li>
      <div
        className={classNames(
          idx % 2 === 0
            ? "bg-white dark:bg-gray-775"
            : "bg-gray-100 dark:bg-gray-815",
          "flex items-center transition px-2 sm:px-6 rounded-md my-1 border border-gray-150 dark:border-gray-750 hover:bg-gray-200 dark:hover:bg-gray-850"
        )}
      >
        <Field name={`actions.${idx}.enabled`} type="checkbox">
          {({
            field,
            form: { setFieldValue }
          }: FieldProps) => (
            <Checkbox
              {...field}
              value={!!field.checked}
              setValue={(value: boolean) => {
                setFieldValue(field.name, value);
              }}
            />
          )}
        </Field>

        <button className="pl-2 pr-0 sm:px-4 py-4 w-full flex items-center" type="button" onClick={toggleEdit}>
          <div className="min-w-0 flex-1 sm:flex sm:items-center sm:justify-between">
            <div className="flex text-sm truncate">
              <p className="font-medium text-dark-600 dark:text-gray-100 truncate">
                {action.name}
              </p>
            </div>
            <div className="flex-shrink-0 sm:mt-0 sm:ml-5">
              <div className="flex overflow-hidden -space-x-1">
                <span className="text-sm font-normal text-gray-500 dark:text-gray-400">
                  {ActionTypeNameMap[action.type]}
                </span>
              </div>
            </div>
          </div>
          <div className="ml-5 flex-shrink-0">
            <ChevronRightIcon
              className="h-5 w-5 text-gray-400"
              aria-hidden="true"
            />
          </div>
        </button>

      </div>
      {edit && (
        <div className="flex items-center mt-1 px-3 sm:px-5 rounded-md border border-gray-150 dark:border-gray-750">
          <DeleteModal
            isOpen={deleteModalIsOpen}
            isLoading={removeMutation.isPending}
            buttonRef={cancelButtonRef}
            toggle={toggleDeleteModal}
            deleteAction={() => removeAction(action.id)}
            title="Remove filter action"
            text="Are you sure you want to remove this action? This action cannot be undone."
          />

          <FilterPage gap="sm:gap-y-6">
            <FilterSection
              title="Action"
              subtitle="Define the download client for your action and its name"
            >
              <FilterLayout>
                <FilterHalfRow>
                  <Select
                    name={`actions.${idx}.type`}
                    label="Action type"
                    optionDefaultText="Select type"
                    options={ActionTypeOptions}
                    tooltip={<div><p>Select the action type for this action.</p></div>}
                  />
                </FilterHalfRow>

                <FilterHalfRow>
                  <TextField name={`actions.${idx}.name`} label="Name" />
                </FilterHalfRow>
              </FilterLayout>
            </FilterSection>

            <TypeForm action={action} clients={clients} idx={idx} />

            <div className="pt-6 pb-4 flex space-x-2 justify-between">
              <button
                type="button"
                className="inline-flex items-center justify-center px-4 py-2 rounded-md sm:text-sm bg-red-700 dark:bg-red-900 hover:dark:bg-red-700 hover:bg-red-800 text-white focus:outline-none"
                onClick={toggleDeleteModal}
              >
                Remove Action
              </button>

              <button
                type="button"
                className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none"
                onClick={toggleEdit}
              >
                Close
              </button>
            </div>
          </FilterPage>
        </div>
      )}
    </li>
  );
}
