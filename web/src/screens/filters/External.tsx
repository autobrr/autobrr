/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Field, FieldArray, FieldArrayRenderProps, FieldProps, useFormikContext } from "formik";
import { NumberField, Select, TextField } from "@components/inputs";
import { TextArea } from "@components/inputs/input";
import { Fragment, useRef } from "react";
import { EmptyListState } from "@components/emptystates";
import { useToggle } from "@hooks/hooks";
import { classNames } from "@utils";
import { Switch as SwitchBasic } from "@headlessui/react";
import {
  ExternalFilterTypeNameMap,
  ExternalFilterTypeOptions,
  ExternalFilterWebhookMethodOptions
} from "@domain/constants";
import { ChevronRightIcon } from "@heroicons/react/24/solid";
import { DeleteModal } from "@components/modals";
import { ArrowDownIcon, ArrowUpIcon } from "@heroicons/react/24/outline";
import { DocsLink } from "@components/ExternalLink";

export function External() {
  const { values } = useFormikContext<Filter>();

  const newItem: ExternalFilter = {
    id: values.external.length + 1,
    index: values.external.length,
    name: `External ${values.external.length + 1}`,
    enabled: false,
    type: "EXEC"
  };

  return (
    <div className="mt-10">
      <FieldArray name="external">
        {({ remove, push, move }: FieldArrayRenderProps) => (
          <Fragment>
            <div className="-ml-4 -mt-4 mb-6 flex justify-between items-center flex-wrap sm:flex-nowrap">
              <div className="ml-4 mt-4">
                <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-gray-200">External filters</h3>
                <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  Run external scripts or webhooks and check status as part of filtering.
                </p>
              </div>
              <div className="ml-4 mt-4 flex-shrink-0">
                <button
                  type="button"
                  className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                  onClick={() => push(newItem)}
                >
                  Add new
                </button>
              </div>
            </div>

            <div className="light:bg-white dark:bg-gray-800 light:shadow sm:rounded-md">
              {values.external.length > 0
                ? <ul className="divide-y divide-gray-200 dark:divide-gray-700">
                  {values.external.map((f, index: number) => (
                    <FilterExternalItem external={f} idx={index} key={index} remove={remove} move={move} initialEdit={true} />
                  ))}
                </ul>
                : <EmptyListState text="No external filters yet!" />
              }
            </div>
          </Fragment>
        )}
      </FieldArray>
    </div>
  );
}

interface FilterExternalItemProps {
  external: ExternalFilter;
  idx: number;
  initialEdit: boolean;
  remove: <T>(index: number) => T | undefined;
  move: (from: number, to: number) => void;
}

function FilterExternalItem({ idx, external, initialEdit, remove, move }: FilterExternalItemProps) {
  const { values, setFieldValue } = useFormikContext<Filter>();
  const cancelButtonRef = useRef(null);

  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);
  const [edit, toggleEdit] = useToggle(initialEdit);

  const removeAction = () => {
    remove(idx);
  };

  const moveUp = () => {
    move(idx, idx - 1);
    setFieldValue(`external.${idx}.index`, idx - 1);
  };

  const moveDown = () => {
    move(idx, idx + 1);
    setFieldValue(`external.${idx}.index`, idx + 1);
  };

  return (
    <li>
      <div
        className={classNames(
          idx % 2 === 0 ? "bg-white dark:bg-gray-800" : "bg-gray-50 dark:bg-gray-700",
          "flex items-center sm:px-6 hover:bg-gray-50 dark:hover:bg-gray-600"
        )}
      >
        <div className="flex flex-col pr-2 justify-between">
          {idx > 0 && (
            <button type="button" className="bg-gray-600 hover:bg-gray-700" onClick={moveUp}>
              <ArrowUpIcon
                className="p-0.5 h-4 w-4 text-gray-400"
                aria-hidden="true"
              />
            </button>
          )}

          {idx < values.external.length - 1 && (
            <button type="button" className="bg-gray-600 hover:bg-gray-700" onClick={moveDown}>
              <ArrowDownIcon
                className="p-0.5 h-4 w-4 text-gray-400"
                aria-hidden="true"
              />
            </button>
          )}
        </div>

        <Field name={`external.${idx}.enabled`} type="checkbox">
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
                  {external.name}
                </p>
              </div>
            </div>
            <div className="mt-4 flex-shrink-0 sm:mt-0 sm:ml-5">
              <div className="flex overflow-hidden -space-x-1">
                <span className="text-sm font-normal text-gray-500 dark:text-gray-400">
                  {ExternalFilterTypeNameMap[external.type]}
                </span>
              </div>
            </div>
          </div>
          <div className="ml-5 flex-shrink-0">
            <ChevronRightIcon className="h-5 w-5 text-gray-400" aria-hidden="true" />
          </div>
        </button>

      </div>
      {edit && (
        <div className="px-4 py-4 flex items-center sm:px-6 border dark:border-gray-600">
          <DeleteModal
            isOpen={deleteModalIsOpen}
            isLoading={false}
            buttonRef={cancelButtonRef}
            toggle={toggleDeleteModal}
            deleteAction={removeAction}
            title="Remove external filter"
            text="Are you sure you want to remove this external filter? This action cannot be undone."
          />

          <div className="w-full">
            <div className="mt-6 grid grid-cols-12 gap-6">
              <Select
                name={`external.${idx}.type`}
                label="Type"
                optionDefaultText="Select type"
                options={ExternalFilterTypeOptions}
                tooltip={<div><p>Select the type for this external filter.</p></div>}
              />

              <TextField name={`external.${idx}.name`} label="Name" columns={6} />
            </div>

            <TypeForm external={external} idx={idx} />

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
                    className={
                      "bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none"
                    }
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


interface TypeFormProps {
  external: ExternalFilter;
  idx: number;
}

const TypeForm = ({ external, idx }: TypeFormProps) => {
  switch (external.type) {
  case "EXEC":
    return (
      <div>
        <div className="mt-6 grid grid-cols-12 gap-6">
          <TextField
            name={`external.${idx}.exec_cmd`}
            label="Command"
            columns={6}
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
          <TextField
            name={`external.${idx}.exec_args`}
            label="Arguments"
            columns={6}
            placeholder={"Arguments eg. --test \"{{ .TorrentName }}\""}
          />
        </div>
        <div className="mt-6 grid grid-cols-12 gap-6">
          <NumberField
            name={`external.${idx}.exec_expect_status`}
            label="Expected exit status"
            placeholder="0"
          />
        </div>
      </div>
    );
  case "WEBHOOK":
    return (
      <div className="mt-6 grid grid-cols-12 gap-6">
        <TextField
          name={`external.${idx}.webhook_host`}
          label="Host"
          columns={6}
          placeholder="Host eg. http://localhost/webhook"
          tooltip={<p>URL or IP to api. Pass params and set api tokens etc.</p>}
        />
        <Select
          name={`external.${idx}.webhook_method`}
          label="HTTP method"
          optionDefaultText="Select http method"
          options={ExternalFilterWebhookMethodOptions}
          tooltip={<div><p>Select the HTTP method for this webhook. Defaults to POST</p></div>}
        />
        <TextField
          name={`external.${idx}.webhook_headers`}
          label="Headers"
          columns={6}
          placeholder="HEADER=custom1,HEADER2=custom2"
        />
        <TextArea
          name={`external.${idx}.webhook_data`}
          label="Data (json)"
          columns={6}
          rows={5}
          placeholder={"Request data: { \"key\": \"value\" }"}
        />
        <NumberField
          name={`external.${idx}.webhook_expect_status`}
          label="Expected http status code"
          placeholder="200"
        />
        <TextField
          name={`external.${idx}.webhook_retry_status`}
          label="Retry http status code(s)"
          placeholder="Retry on status eg. 202, 204"
        />
        <NumberField
          name={`external.${idx}.webhook_retry_attempts`}
          label="Maximum retry attempts"
          placeholder="10"
        />
        <NumberField
          name={`external.${idx}.webhook_retry_delay_seconds`}
          label="Retry delay in seconds"
          placeholder="1"
        />
        <NumberField
          name={`external.${idx}.webhook_retry_max_jitter_seconds`}
          label="Max jitter in seconds"
          placeholder="1"
        />
      </div>
    );

  default:
    return null;
  }
};
