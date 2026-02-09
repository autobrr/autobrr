/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useRef } from "react";
import { ChevronDownIcon, ChevronRightIcon } from "@heroicons/react/24/solid";
import { ArrowDownIcon, ArrowUpIcon, SquaresPlusIcon } from "@heroicons/react/24/outline";

import { classNames } from "@utils";
import { useToggle } from "@hooks/hooks";
import { EmptyListState } from "@components/emptystates";
import {
  ExternalFilterOnErrorOptions,
  ExternalFilterTypeNameMap,
  ExternalFilterTypeOptions,
  ExternalFilterWebhookMethodOptions
} from "@domain/constants";

import { useFormContext, useStore, ContextField } from "@app/lib/form";
import { NumberField, Select, TextField, TextAreaAutoResize } from "@components/inputs/tanstack";
import { DeleteModal } from "@components/modals";
import { DocsLink } from "@components/ExternalLink";
import { Checkbox } from "@components/Checkbox";
import { TitleSubtitle } from "@components/headings";
import { FilterLayout, FilterPage, FilterSection } from "@screens/filters/sections/_components.tsx";

export function External() {
  const form = useFormContext();

  const external = useStore(form.store, (s: any) => s.values.external) as ExternalFilter[];

  const newItem: ExternalFilter = {
    id: external.length + 1,
    index: external.length,
    name: `External ${external.length + 1}`,
    enabled: false,
    type: "EXEC",
    on_error: "REJECT",
  };

  const pushItem = () => {
    (form as any).pushFieldValue("external", newItem);
  };

  const removeItem = (index: number) => {
    (form as any).removeFieldValue("external", index);
  };

  const moveItem = (from: number, to: number) => {
    (form as any).swapFieldValues("external", from, to);
  };

  return (
    <div className="mt-5">
      <>
        <div className="-ml-4 -mt-4 mb-6 flex justify-between items-center flex-wrap sm:flex-nowrap">
          <TitleSubtitle
            className="ml-4 mt-4"
            title="External filters"
            subtitle="Run external scripts or webhooks and check status as part of filtering."
          />
          <div className="ml-4 mt-4 shrink-0">
            <button
              type="button"
              className="relative inline-flex items-center px-4 py-2 transition border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
              onClick={pushItem}
            >
              <SquaresPlusIcon
                className="w-5 h-5 mr-1"
                aria-hidden="true"
              />
              Add new
            </button>
          </div>
        </div>

        {external.length > 0 ? (
          <ul className="rounded-md">
            {external.map((ext, index: number) => (
              <FilterExternalItem
                key={ext.id}
                initialEdit
                external={ext}
                idx={index}
                totalCount={external.length}
                remove={removeItem}
                move={moveItem}
              />
            ))}
          </ul>
        ) : (
          <EmptyListState text="No external filters yet!" />
        )}
      </>
    </div>
  );
}

interface FilterExternalItemProps {
  external: ExternalFilter;
  idx: number;
  totalCount: number;
  initialEdit: boolean;
  remove: (index: number) => void;
  move: (from: number, to: number) => void;
}

function FilterExternalItem({ idx, external, totalCount, initialEdit, remove, move }: FilterExternalItemProps) {
  const form = useFormContext();
  const cancelButtonRef = useRef(null);

  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);
  const [edit, toggleEdit] = useToggle(initialEdit);

  const externalEnabled = useStore(form.store, (s: any) => s.values.external?.[idx]?.enabled);

  const removeAction = () => {
    remove(idx);
  };

  const moveUp = () => {
    move(idx, idx - 1);
    (form as any).setFieldValue(`external[${idx}].index`, idx - 1);
  };

  const moveDown = () => {
    move(idx, idx + 1);
    (form as any).setFieldValue(`external[${idx}].index`, idx + 1);
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
        {((idx > 0) || (idx < totalCount - 1)) ? (
          <div className="flex flex-col pr-3 justify-between">
            {idx > 0 && (
              <button type="button" className="cursor-pointer" onClick={moveUp}>
                <ArrowUpIcon
                  className="p-0.5 h-4 w-4 text-gray-700 dark:text-gray-400"
                  aria-hidden="true"
                />
              </button>
            )}

            {idx < totalCount - 1 && (
              <button type="button" className="cursor-pointer" onClick={moveDown}>
                <ArrowDownIcon
                  className="p-0.5 h-4 w-4 text-gray-700 dark:text-gray-400"
                  aria-hidden="true"
                />
              </button>
            )}
          </div>
        ) : null}

        <Checkbox
          value={!!externalEnabled}
          setValue={(value: boolean) => {
            (form as any).setFieldValue(`external[${idx}].enabled`, value);
          }}
        />

        <button className="pl-2 pr-0 sm:px-4 py-4 w-full flex items-center cursor-pointer" type="button" onClick={toggleEdit}>
          <div className="min-w-0 flex-1 sm:flex sm:items-center sm:justify-between">
            <div className="truncate">
              <div className="flex text-sm">
                <p className="font-medium text-dark-600 dark:text-gray-100 truncate">
                  {external.name}
                </p>
              </div>
            </div>
            <div className="shrink-0 sm:mt-0 sm:ml-5">
              <div className="flex overflow-hidden -space-x-1">
                <span className="text-sm font-normal text-gray-500 dark:text-gray-400">
                  {ExternalFilterTypeNameMap[external.type]}
                </span>
              </div>
            </div>
          </div>
          <div className="ml-5 shrink-0">
            {edit ? <ChevronDownIcon className="h-5 w-5 text-gray-400" aria-hidden="true" /> : <ChevronRightIcon className="h-5 w-5 text-gray-400" aria-hidden="true" />}
          </div>
        </button>

      </div>
      {edit && (
        <div className="flex items-center mt-1 px-3 sm:px-5 rounded-md border border-gray-150 dark:border-gray-750">
          <DeleteModal
            isOpen={deleteModalIsOpen}
            isLoading={false}
            buttonRef={cancelButtonRef}
            toggle={toggleDeleteModal}
            deleteAction={removeAction}
            title="Remove external filter"
            text="Are you sure you want to remove this external filter? This action cannot be undone."
          />

          <FilterPage gap="sm:gap-y-6">
            <FilterSection
              title="External Filter"
              subtitle="Define the type of your filter and its name"
            >
              <FilterLayout>
                <ContextField name={`external[${idx}].type`}>
                  <Select
                    label="Type"
                    optionDefaultText="Select type"
                    options={ExternalFilterTypeOptions}
                    tooltip={<div><p>Select the type for this external filter.</p></div>}
                    columns={4}
                  />
                </ContextField>

                <ContextField name={`external[${idx}].name`}>
                  <TextField
                    label="Name" columns={4}
                  />
                </ContextField>

                <ContextField name={`external[${idx}].on_error`}>
                  <Select
                    label="On Error"
                    optionDefaultText="Select type"
                    options={ExternalFilterOnErrorOptions}
                    tooltip={<div><p>Select what to do on error for this external filter.</p></div>}
                    columns={4}
                  />
                </ContextField>
              </FilterLayout>
            </FilterSection>

            <TypeForm external={external} idx={idx} />

            <div className="pt-6 pb-4 space-x-2 flex justify-between">
              <button
                type="button"
                className="inline-flex items-center justify-center px-4 py-2 rounded-md sm:text-sm bg-red-700 dark:bg-red-900 dark:hover:bg-red-700 hover:bg-red-800 text-white focus:outline-hidden"
                onClick={toggleDeleteModal}
              >
                Remove External
              </button>

              <button
                type="button"
                className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden"
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


interface TypeFormProps {
  external: ExternalFilter;
  idx: number;
}

const TypeForm = ({ external, idx }: TypeFormProps) => {
  switch (external.type) {
  case "EXEC": {
    return (
      <FilterSection
        title="Execute"
        subtitle="Specify the executable, the argument and the expected exit status to run as a pre-filter"
      >
        <FilterLayout>
          <ContextField name={`external[${idx}].exec_cmd`}>
            <TextAreaAutoResize
              label="Path to Executable"
              columns={5}
              placeholder="Absolute path to executable eg. /bin/test"
              tooltip={
                <div>
                  <p>
                    For custom commands you should specify the full path to the binary/program
                    you want to run. And you can include your own static variables:
                  </p>
                  <DocsLink href="https://autobrr.com/filters/actions#custom-commands--exec" />
                </div>
              }
            />
          </ContextField>
          <ContextField name={`external[${idx}].exec_args`}>
            <TextAreaAutoResize
              label="Exec Arguments"
              columns={5}
              placeholder={"Arguments eg. --test \"{{ .TorrentName }}\""}
            />
          </ContextField>
          <div className="col-span-12 sm:col-span-2">
            <ContextField name={`external[${idx}].exec_expect_status`}>
              <NumberField
                label="Expected exit status"
                placeholder="0"
              />
            </ContextField>
          </div>
        </FilterLayout>
      </FilterSection>
    );
  }
  case "WEBHOOK": {
    return (
      <>
        <FilterSection
          title="Request"
          subtitle="Specify your request destination endpoint, headers and expected return status"
        >
          <FilterLayout>
            <ContextField name={`external[${idx}].webhook_host`}>
              <TextField
                label="Endpoint"
                columns={6}
                placeholder="Host eg. http://localhost/webhook"
                tooltip={<p>URL or IP to your API. Pass params and set API tokens etc.</p>}
              />
            </ContextField>
            <ContextField name={`external[${idx}].webhook_method`}>
              <Select
                label="HTTP method"
                optionDefaultText="Select http method"
                options={ExternalFilterWebhookMethodOptions}
                tooltip={<div><p>Select the HTTP method for this webhook. Defaults to POST</p></div>}
              />
            </ContextField>
            <ContextField name={`external[${idx}].webhook_headers`}>
              <TextField
                label="HTTP Request Headers"
                columns={6}
                placeholder="HEADER=custom1,HEADER2=custom2"
              />
            </ContextField>
            <ContextField name={`external[${idx}].webhook_expect_status`}>
              <NumberField
                label="Expected HTTP status code"
                placeholder="200"
              />
            </ContextField>
          </FilterLayout>
        </FilterSection>
        <FilterSection
          title="Retry"
          subtitle="Retry behavior on request failure"
        >
          <FilterLayout>
            <ContextField name={`external[${idx}].webhook_retry_status`}>
              <TextField
                label="Retry http status code(s)"
                placeholder="Retry on status eg. 202, 204"
                columns={6}
              />
            </ContextField>
            <ContextField name={`external[${idx}].webhook_retry_attempts`}>
              <NumberField
                label="Maximum retry attempts"
                placeholder="10"
              />
            </ContextField>
            <ContextField name={`external[${idx}].webhook_retry_delay_seconds`}>
              <NumberField
                label="Retry delay in seconds"
                placeholder="1"
              />
            </ContextField>
          </FilterLayout>
        </FilterSection>
        <FilterSection
          title="Payload"
          subtitle="Specify your JSON payload"
        >
          <FilterLayout>
            <ContextField name={`external[${idx}].webhook_data`}>
              <TextAreaAutoResize
                label="Data (json)"
                placeholder={"Request data: { \"key\": \"value\" }"}
              />
            </ContextField>
          </FilterLayout>
        </FilterSection>
      </>
    );
  }

  default: {
    return null;
  }
  }
};
