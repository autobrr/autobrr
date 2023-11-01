/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment, useEffect, useRef, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Field, FieldArray, FieldProps, FormikValues, useFormikContext } from "formik";
import type { FieldArrayRenderProps } from "formik";
import { Dialog, Switch as SwitchBasic, Transition } from "@headlessui/react";
import { ChevronRightIcon } from "@heroicons/react/24/solid";
import { Link } from "react-router-dom";
import { toast } from "react-hot-toast";

import {
  ActionContentLayoutOptions,
  ActionRtorrentRenameOptions,
  ActionTypeNameMap,
  ActionTypeOptions
} from "@domain/constants";
import { AlertWarning } from "@components/alerts";
import { DownloadClientSelect, NumberField, Select, SwitchGroup, TextField } from "@components/inputs";
import { APIClient } from "@api/APIClient";
import { EmptyListState } from "@components/emptystates";
import { useToggle } from "@hooks/hooks";
import { classNames } from "@utils";
import { DeleteModal } from "@components/modals";
import { CollapsableSection } from "./Details";
import { TextArea } from "@components/inputs/input";
import Toast from "@components/notifications/Toast";
import { DocsLink } from "@components/ExternalLink";

interface FilterActionsProps {
  filter: Filter;
  values: FormikValues;
}

export function FilterActions({ filter, values }: FilterActionsProps) {
  const { data } = useQuery(
    ["filters", "download_clients"],
    () => APIClient.download_clients.getAll(),
    { refetchOnWindowFocus: false }
  );

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
    skip_hash_check: false,
    content_layout: "" || undefined,
    limit_upload_speed: 0,
    limit_download_speed: 0,
    limit_ratio: 0,
    limit_seed_time: 0,
    reannounce_skip: false,
    reannounce_delete: false,
    reannounce_interval: 7,
    reannounce_max_attempts: 25,
    filter_id: filter.id,
    webhook_host: "",
    webhook_type: "",
    webhook_method: "",
    webhook_data: "",
    webhook_headers: [],
    external_download_client_id: 0,
    client_id: 0
  };

  return (
    <div className="mt-10">
      <FieldArray name="actions">
        {({ remove, push }: FieldArrayRenderProps) => (
          <Fragment>
            <div className="-ml-4 -mt-4 mb-6 flex justify-between items-center flex-wrap sm:flex-nowrap">
              <div className="ml-4 mt-4">
                <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-gray-200">Actions</h3>
                <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  Add to download clients or run custom commands.
                </p>
              </div>
              <div className="ml-4 mt-4 flex-shrink-0">
                <button
                  type="button"
                  className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                  onClick={() => push(newAction)}
                >
                  Add new
                </button>
              </div>
            </div>

            <div className="light:bg-white dark:bg-gray-800 light:shadow sm:rounded-md">
              {values.actions.length > 0 ?
                <ul className="divide-y divide-gray-200 dark:divide-gray-700">
                  {values.actions.map((action: Action, index: number) => (
                    <FilterActionsItem action={action} clients={data ?? []} idx={index} initialEdit={values.actions.length === 1} remove={remove} key={index} />
                  ))}
                </ul>
                : <EmptyListState text="No actions yet!" />
              }
            </div>
          </Fragment>
        )}
      </FieldArray>
    </div>
  );
}

interface TypeFormProps {
  action: Action;
  idx: number;
  clients: Array<DownloadClient>;
}

const TypeForm = ({ action, idx, clients }: TypeFormProps) => {
  const { setFieldValue } = useFormikContext();

  const resetClientField = (action: Action, idx: number, prevActionType: string): void => {
    const fieldName = `actions.${idx}.client_id`;

    if (prevActionType !== action.type && (
      action.type === "QBITTORRENT" ||
      action.type === "DELUGE_V1" ||
      action.type === "DELUGE_V2" ||
      action.type === "RTORRENT" ||
      action.type === "TRANSMISSION" ||
      action.type === "PORLA" ||
      action.type === "RADARR" ||
      action.type === "SONARR" ||
      action.type === "LIDARR" ||
      action.type === "WHISPARR" ||
      action.type === "READARR" ||
      action.type === "SABNZBD"
    )) {
      setFieldValue(fieldName, 0); // Reset the client_id field value
    }
  };

  const [prevActionType, setPrevActionType] = useState<string | null>(null);
  useEffect(() => {
    if (prevActionType !== null) {
      resetClientField(action, idx, prevActionType);
    }
    setPrevActionType(action.type);
  }, [action.type, idx, setFieldValue]);

  switch (action.type) {
  case "TEST":
    return (
      <AlertWarning
        text="The test action does nothing except to show if the filter works."
      />
    );
  case "EXEC":
    return (
      <div>
        <div className="mt-6 grid grid-cols-12 gap-6">
          <TextField
            name={`actions.${idx}.exec_cmd`}
            label="Command"
            columns={6}
            placeholder="Path to program eg. /bin/test"
          />
          <TextField
            name={`actions.${idx}.exec_args`}
            label="Arguments"
            columns={6}
            placeholder="Arguments eg. --test"
          />
        </div>
      </div>
    );
  case "WATCH_FOLDER":
    return (
      <div className="mt-6 grid grid-cols-12 gap-6">
        <TextField
          name={`actions.${idx}.watch_folder`}
          label="Watch folder"
          columns={6}
          placeholder="Watch directory eg. /home/user/rwatch"
        />
      </div>
    );
  case "WEBHOOK":
    return (
      <div className="mt-6 grid grid-cols-12 gap-6">
        <TextField
          name={`actions.${idx}.webhook_host`}
          label="Host"
          columns={6}
          placeholder="Host eg. http://localhost/webhook"
        />
        <TextArea
          name={`actions.${idx}.webhook_data`}
          label="Data (json)"
          columns={6}
          rows={5}
          placeholder={"Request data: { \"key\": \"value\" }"}
        />
      </div>
    );
  case "QBITTORRENT":
    return (
      <div className="w-full">
        <div className="mt-6 grid grid-cols-12 gap-6">
          <DownloadClientSelect
            name={`actions.${idx}.client_id`}
            action={action}
            clients={clients}
          />

          <div className="col-span-6 sm:col-span-6">
            <TextField
              name={`actions.${idx}.save_path`}
              label="Save path"
              columns={6}
              placeholder="eg. /full/path/to/download_folder"
              tooltip={
                <div>
                  <p>Set a custom save path for this action. Automatic Torrent Management will take care of this if using qBittorrent with categories.</p>
                  <br />
                  <p>The field can use macros to transform/add values from metadata:</p>
                  <DocsLink href="https://autobrr.com/filters/macros" />
                </div>
              }
            />
          </div>
        </div>

        <div className="mt-6 grid grid-cols-12 gap-6">
          <TextField
            name={`actions.${idx}.category`}
            label="Category"
            columns={6}
            placeholder="eg. category"
            tooltip={
              <div>
                <p>The field can use macros to transform/add values from metadata:</p>
                <DocsLink href="https://autobrr.com/filters/macros" />
              </div>
            }
          />
          <TextField
            name={`actions.${idx}.tags`}
            label="Tags"
            columns={6}
            placeholder="eg. tag1,tag2"
            tooltip={
              <div>
                <p>The field can use macros to transform/add values from metadata:</p>
                <DocsLink href="https://autobrr.com/filters/macros" />
              </div>
            }
          />
        </div>

        <CollapsableSection title="Rules" subtitle="client options">
          <div className="col-span-12">
            <div className="mt-6 grid grid-cols-12 gap-6">
              <NumberField
                name={`actions.${idx}.limit_download_speed`}
                label="Limit download speed (KiB/s)"
                placeholder="Takes any number (0 is no limit)"
              />
              <NumberField
                name={`actions.${idx}.limit_upload_speed`}
                label="Limit upload speed (KiB/s)"
                placeholder="Takes any number (0 is no limit)"
              />
            </div>

            <div className="mt-6 grid grid-cols-12 gap-6">
              <NumberField
                name={`actions.${idx}.limit_ratio`}
                label="Ratio limit"
                placeholder="Takes any number (0 is no limit)"
                step={0.25}
                isDecimal
              />
              <NumberField
                name={`actions.${idx}.limit_seed_time`}
                label="Seed time limit (minutes)"
                placeholder="Takes any number (0 is no limit)"
              />
            </div>
          </div>
          <div className="col-span-6">
            <SwitchGroup
              name={`actions.${idx}.paused`}
              label="Add paused"
              description="Add torrent as paused"
            />
            <SwitchGroup
              name={`actions.${idx}.ignore_rules`}
              label="Ignore client rules"
              tooltip={
                <div>
                  <p>
                    Choose to ignore rules set in <Link className="text-blue-400 visited:text-blue-400" to="/settings/clients">Client Settings</Link>.
                  </p>
                </div>
              }
            />
          </div>
          <div className="col-span-6">
            <Select
              name={`actions.${idx}.content_layout`}
              label="Content Layout"
              optionDefaultText="Select content layout"
              options={ActionContentLayoutOptions}></Select>

            <div className="mt-2">
              <SwitchGroup
                name={`actions.${idx}.skip_hash_check`}
                label="Skip hash check"
                description="Add torrent and skip hash check"
              />
            </div>
          </div>
        </CollapsableSection>

        <CollapsableSection title="Advanced" subtitle="Advanced options">
          <div className="col-span-12">
            <div className="mt-6 grid grid-cols-12 gap-6">
              <NumberField
                name={`actions.${idx}.reannounce_interval`}
                label="Reannounce interval. Run every X seconds"
                placeholder="7 is default and recommended"
              />
              <NumberField
                name={`actions.${idx}.reannounce_max_attempts`}
                label="Run reannounce Y times"
              />
            </div>
          </div>
          <div className="col-span-6">
            <SwitchGroup
              name={`actions.${idx}.reannounce_skip`}
              label="Disable reannounce"
              description="Reannounce is enabled by default. Disable if not needed."
            />
            <SwitchGroup
              name={`actions.${idx}.reannounce_delete`}
              label="Delete stalled"
              description="Delete stalled torrents after X attempts"
            />
          </div>
        </CollapsableSection>
      </div>
    );
  case "DELUGE_V1":
  case "DELUGE_V2":
    return (
      <div>
        <div className="mt-6 grid grid-cols-12 gap-6">
          <DownloadClientSelect
            name={`actions.${idx}.client_id`}
            action={action}
            clients={clients}
          />

          <div className="col-span-12 sm:col-span-6">
            <TextField
              name={`actions.${idx}.save_path`}
              label="Save path"
              columns={6}
              placeholder="eg. /full/path/to/download_folder"
            />
          </div>
        </div>

        <div className="mt-6 col-span-12 sm:col-span-6">
          <TextField
            name={`actions.${idx}.label`}
            label="Label"
            columns={6}
            placeholder="eg. label1 (must exist in Deluge to work)"
          />
        </div>

        <div className="mt-6 grid grid-cols-12 gap-6">
          <NumberField
            name={`actions.${idx}.limit_download_speed`}
            label="Limit download speed (KB/s)"
          />
          <NumberField
            name={`actions.${idx}.limit_upload_speed`}
            label="Limit upload speed (KB/s)"
          />
        </div>

        <div className="mt-6 grid grid-cols-12 gap-6">
          <div className="col-span-6">
            <SwitchGroup
              name={`actions.${idx}.paused`}
              label="Add paused"
            />
          </div>
        </div>
      </div>
    );
  case "RTORRENT":
    return (
      <div>
        <div className="mt-6 grid grid-cols-12 gap-6">
          <DownloadClientSelect
            name={`actions.${idx}.client_id`}
            action={action}
            clients={clients}
          />

          <div className="col-span-12 sm:col-span-6">
            <TextField
              name={`actions.${idx}.label`}
              label="Label"
              columns={6}
              placeholder="eg. label1,label2"
            />
          </div>

          <div className="col-span-12 sm:col-span-6">
            <TextField
              name={`actions.${idx}.save_path`}
              label="Save path"
              columns={6}
              placeholder="eg. /full/path/to/download_folder"
            />
          </div>
          <div className="col-span-12 sm:col-span-6">
            <div className="col-span-6">
              <Select
                name={`actions.${idx}.content_layout`}
                label="Don't add torrent's name to path"
                optionDefaultText="No"
                options={ActionRtorrentRenameOptions}
              />
            </div>
          </div>
          <div className="col-span-12 sm:col-span-6">
            <div className="col-span-6">
              <SwitchGroup
                name={`actions.${idx}.paused`}
                label="Don't start download automatically"
              />
            </div>
          </div>
        </div>
      </div>
    );
  case "TRANSMISSION":
    return (
      <div>
        <div className="mt-6 grid grid-cols-12 gap-6">
          <DownloadClientSelect
            name={`actions.${idx}.client_id`}
            action={action}
            clients={clients}
          />

          <div className="col-span-12 sm:col-span-6">
            <TextField
              name={`actions.${idx}.save_path`}
              label="Save path"
              columns={6}
              placeholder="eg. /full/path/to/download_folder"
            />
          </div>
        </div>

        <div className="mt-6 grid grid-cols-12 gap-6">
          <div className="col-span-6">
            <SwitchGroup
              name={`actions.${idx}.paused`}
              label="Add paused"
            />
          </div>

          <TextField
              name={`actions.${idx}.label`}
              label="Label"
              columns={6}
              placeholder="eg. label1"
          />
        </div>

        <CollapsableSection title="Rules" subtitle="client options">
          <div className="col-span-12">
            <div className="mt-6 grid grid-cols-12 gap-6">
              <NumberField
                  name={`actions.${idx}.limit_download_speed`}
                  label="Limit download speed (KiB/s)"
                  placeholder="Takes any number (0 is no limit)"
              />
              <NumberField
                  name={`actions.${idx}.limit_upload_speed`}
                  label="Limit upload speed (KiB/s)"
                  placeholder="Takes any number (0 is no limit)"
              />
            </div>

            <div className="mt-6 grid grid-cols-12 gap-6">
              <NumberField
                  name={`actions.${idx}.limit_ratio`}
                  label="Ratio limit"
                  placeholder="Takes any number (0 is no limit)"
                  step={0.25}
                  isDecimal
              />
              <NumberField
                  name={`actions.${idx}.limit_seed_time`}
                  label="Seed time limit (minutes)"
                  placeholder="Takes any number (0 is no limit)"
              />
            </div>
          </div>
        </CollapsableSection>

        <CollapsableSection title="Re-announce" subtitle="Re-announce options">
          <div className="col-span-12">
            <div className="mt-6 grid grid-cols-12 gap-6">
              <NumberField
                name={`actions.${idx}.reannounce_interval`}
                label="Reannounce interval. Run every X seconds"
                placeholder="7 is default and recommended"
              />
              <NumberField
                name={`actions.${idx}.reannounce_max_attempts`}
                label="Run reannounce Y times"
              />
            </div>
          </div>
          <div className="col-span-6">
            <SwitchGroup
              name={`actions.${idx}.reannounce_skip`}
              label="Disable reannounce"
              description="Reannounce is enabled by default. Disable if not needed."
            />
            <SwitchGroup
              name={`actions.${idx}.reannounce_delete`}
              label="Delete stalled"
              description="Delete stalled torrents after X attempts"
            />
          </div>
        </CollapsableSection>
      </div>
    );
  case "PORLA":
    return (
      <div className="w-full">
        <div className="mt-6 grid grid-cols-12 gap-6">
          <DownloadClientSelect
            name={`actions.${idx}.client_id`}
            action={action}
            clients={clients}
          />

          <div className="col-span-6 sm:col-span-6">
            <TextField
              name={`actions.${idx}.save_path`}
              label="Save path"
              columns={6}
              placeholder="eg. /full/path/to/torrent/data"
            />
          </div>
        </div>

        <div className="mt-6 grid grid-cols-12 gap-6">
          <TextField
            name={`actions.${idx}.label`}
            label="Preset"
            columns={6}
            placeholder="eg. default"
            tooltip={<div>A case-sensitive preset name as configured in Porla.</div>} />
        </div>

        <CollapsableSection title="Rules" subtitle="client options">
          <div className="col-span-12">
            <div className="mt-6 grid grid-cols-12 gap-6">
              <NumberField
                name={`actions.${idx}.limit_download_speed`}
                label="Limit download speed (KiB/s)"
              />
              <NumberField
                name={`actions.${idx}.limit_upload_speed`}
                label="Limit upload speed (KiB/s)"
              />
            </div>
          </div>
        </CollapsableSection>
      </div>
    );
  case "RADARR":
  case "SONARR":
  case "LIDARR":
  case "WHISPARR":
  case "READARR":
    return (
      <div className="mt-6 grid grid-cols-12 gap-6">
        <DownloadClientSelect
          name={`actions.${idx}.client_id`}
          action={action}
          clients={clients}
        />
        <NumberField
          name={`actions.${idx}.external_download_client_id`}
          label="Override download client id for arr"
          tooltip={<p>Override Download client Id from the one set in Clients. Useful if you have multiple clients inside the arr.</p>}
        />
      </div>
    );
  case "SABNZBD":
    return (
      <div>
        <div className="mt-6 grid grid-cols-12 gap-6">
          <DownloadClientSelect
            name={`actions.${idx}.client_id`}
            action={action}
            clients={clients}
          />

          <TextField
            name={`actions.${idx}.category`}
            label="Category"
            columns={6}
            placeholder="eg. category"
            tooltip={<p>Category must exist already.</p>} />
        </div>
      </div>
    );

  default:
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
          idx % 2 === 0 ? "bg-white dark:bg-gray-800" : "bg-gray-50 dark:bg-gray-700",
          "flex items-center sm:px-6 hover:bg-gray-50 dark:hover:bg-gray-600"
        )}
      >
        <Field name={`actions.${idx}.enabled`} type="checkbox">
          {({
            field,
            form: { setFieldValue }
          }: FieldProps) => (
            <SwitchBasic
              {...field}
              type="button"
              value={field.value}
              checked={field.checked ?? false}
              onChange={(value: boolean) => {
                setFieldValue(field?.name ?? "", value);
              }}
              className={classNames(
                field.value ? "bg-blue-500" : "bg-gray-200 dark:bg-gray-600",
                "relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              )}
            >
              <span className="sr-only">toggle enabled</span>
              <span
                aria-hidden="true"
                className={classNames(
                  field.value ? "translate-x-5" : "translate-x-0",
                  "inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200"
                )}
              />
            </SwitchBasic>
          )}
        </Field>

        <button className="px-4 py-4 w-full flex" type="button" onClick={toggleEdit}>
          <div className="min-w-0 flex-1 sm:flex sm:items-center sm:justify-between">
            <div className="truncate">
              <div className="flex text-sm">
                <p className="ml-4 font-medium text-dark-600 dark:text-gray-100 truncate">
                  {action.name}
                </p>
              </div>
            </div>
            <div className="mt-4 flex-shrink-0 sm:mt-0 sm:ml-5">
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
        <div className="px-4 py-4 flex items-center sm:px-6 border dark:border-gray-600">
          <Transition.Root show={deleteModalIsOpen} as={Fragment}>
            <Dialog
              as="div"
              static
              className="fixed inset-0 overflow-y-auto"
              initialFocus={cancelButtonRef}
              open={deleteModalIsOpen}
              onClose={toggleDeleteModal}
            >
              <DeleteModal
                isOpen={deleteModalIsOpen}
                isLoading={removeMutation.isLoading}
                buttonRef={cancelButtonRef}
                toggle={toggleDeleteModal}
                deleteAction={() => removeAction(action.id)}
                title="Remove filter action"
                text="Are you sure you want to remove this action? This action cannot be undone."
              />
            </Dialog>
          </Transition.Root>

          <div className="w-full">

            <div className="mt-6 grid grid-cols-12 gap-6">
              <Select
                name={`actions.${idx}.type`}
                label="Type"
                optionDefaultText="Select type"
                options={ActionTypeOptions}
                tooltip={<div><p>Select the download client type for this action.</p></div>}
              />

              <TextField name={`actions.${idx}.name`} label="Name" columns={6} />
            </div>

            <TypeForm action={action} clients={clients} idx={idx} />

            <div className="pt-6 divide-y divide-gray-200">
              <div className="mt-4 pt-4 flex justify-between">
                <button
                  type="button"
                  className="inline-flex items-center justify-center px-4 py-2 rounded-md sm:text-sm bg-red-700 dark:bg-red-900 hover:dark:bg-red-700 hover:bg-red-800 text-white focus:outline-none"
                  onClick={toggleDeleteModal}
                >
                  Remove
                </button>

                <div>
                  <button
                    type="button"
                    className="light:bg-white light:border light:border-gray-300 rounded-md shadow-sm py-2 px-4 inline-flex justify-center text-sm font-medium text-gray-700 dark:text-gray-500 light:hover:bg-gray-50 dark:hover:text-gray-300 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                    onClick={toggleEdit}
                  >
                    Close
                  </button>
                </div>
              </div>
            </div>

          </div>
        </div>
      )}
    </li>
  );
}
