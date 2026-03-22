/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Field, FieldArray, useFormikContext } from "formik";
import type { FieldProps, FieldArrayRenderProps } from "formik";
import { ChevronRightIcon, BoltIcon } from "@heroicons/react/24/solid";
import { useTranslation } from "react-i18next";

import { classNames } from "@utils";
import { useToggle } from "@hooks/hooks";
import { APIClient } from "@api/APIClient";
import { DOWNLOAD_CLIENTS, getActionTypeNameMap, getActionTypeOptions } from "@domain/constants";

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
  NZBGet,
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
  const { t } = useTranslation(["options", "filters"]);
  const { values } = useFormikContext<Filter>();
  const actionTypeOptions = getActionTypeOptions(t);

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
                title={t("filters:actionsSection.title")}
                subtitle={t("filters:actionsSection.subtitle")}
              />
              <div className="ml-4 mt-4 shrink-0">
                <button
                  type="button"
                  className="relative inline-flex items-center px-4 py-2 border border-transparent transition shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                  onClick={() => push(newAction)}
                >
                  <BoltIcon
                    className="w-5 h-5 mr-1"
                    aria-hidden="true"
                  />
                  {t("filters:actionsSection.addNew")}
                </button>
              </div>
            </div>

            {values.actions.length > 0 ? (
              <ul className="rounded-md">
                {values.actions.map((action: Action, index: number) => (
                  <FilterActionsItem
                    key={action.id}
                    action={action}
                    actionTypeOptions={actionTypeOptions}
                    clients={data ?? []}
                    idx={index}
                    initialEdit={values.actions.length === 1}
                    remove={remove}
                  />
                ))}
              </ul>
            ) : (
              <EmptyListState text={t("filters:actionsSection.empty")} />
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
  case "NZBGET":
    return <NZBGet {...props} />;
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
  actionTypeOptions: ReturnType<typeof getActionTypeOptions>;
  clients: DownloadClient[];
  idx: number;
  initialEdit: boolean;
  remove: <T>(index: number) => T | undefined;
}

function FilterActionsItem({ action, actionTypeOptions, clients, idx, initialEdit, remove }: FilterActionsItemProps) {
  const { t } = useTranslation(["options", "filters"]);
  const cancelButtonRef = useRef(null);
  const actionTypeNameMap = getActionTypeNameMap(t);

  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);
  const [edit, toggleEdit] = useToggle(initialEdit);

  const removeMutation = useMutation({
    mutationFn: (id: number) => APIClient.actions.delete(id),
    onSuccess: () => {
      remove(idx);
      // Invalidate filters just in case, most likely not necessary but can't hurt.
      // queryClient.invalidateQueries({ queryKey: filterKeys.detail(id) });

      toast.custom((t) => (
        <Toast type="success" body={t("filters:actionsSection.deleted", { name: action?.name })} t={t} />
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
            <div className="shrink-0 sm:mt-0 sm:ml-5">
              <div className="flex overflow-hidden -space-x-1">
                <span className="text-sm font-normal text-gray-500 dark:text-gray-400">
                  {actionTypeNameMap[action.type]}
                </span>
              </div>
            </div>
          </div>
          <div className="ml-5 shrink-0">
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
            title={t("filters:actionsSection.removeTitle")}
            text={t("filters:actionsSection.removeText")}
          />

          <FilterPage gap="sm:gap-y-6">
            <FilterSection
              title={t("filters:actionsSection.actionTitle")}
              subtitle={t("filters:actionsSection.actionSubtitle")}
            >
              <FilterLayout>
                <FilterHalfRow>
                  <Select
                    name={`actions.${idx}.type`}
                    label={t("filters:actionsSection.actionType")}
                    optionDefaultText={t("filters:actionsSection.selectType")}
                    options={actionTypeOptions}
                    tooltip={<div><p>{t("filters:actionsSection.actionTypeTooltip")}</p></div>}
                  />
                </FilterHalfRow>

                <FilterHalfRow>
                  <TextField name={`actions.${idx}.name`} label={t("filters:actionsSection.name")} />
                </FilterHalfRow>
              </FilterLayout>
            </FilterSection>

            <TypeForm action={action} clients={clients} idx={idx} />

            <div className="pt-6 pb-4 flex space-x-2 justify-between">
              <button
                type="button"
                className="inline-flex items-center justify-center px-4 py-2 rounded-md sm:text-sm bg-red-700 dark:bg-red-900 dark:hover:bg-red-700 hover:bg-red-800 text-white focus:outline-hidden"
                onClick={toggleDeleteModal}
              >
                {t("filters:actionsSection.removeAction")}
              </button>

              <button
                type="button"
                className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden"
                onClick={toggleEdit}
              >
                {t("filters:actionsSection.close")}
              </button>
            </div>
          </FilterPage>
        </div>
      )}
    </li>
  );
}
