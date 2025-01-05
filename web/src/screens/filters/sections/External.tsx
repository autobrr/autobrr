/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useRef } from "react";
import { ChevronRightIcon } from "@heroicons/react/24/solid";
import { ArrowDownIcon, ArrowUpIcon, SquaresPlusIcon } from "@heroicons/react/24/outline";
import { Field, FieldArray, FieldArrayRenderProps, FieldProps, useFormikContext } from "formik";

import { classNames } from "@utils";
import { useToggle } from "@hooks/hooks";
import { TextAreaAutoResize } from "@components/inputs/input";
import { EmptyListState } from "@components/emptystates";
import { NumberField, Select, TextField } from "@components/inputs";
import {
  ExternalFilterTypeNameMap,
  ExternalFilterTypeOptions,
  ExternalFilterWebhookMethodOptions
} from "@domain/constants";

import { DeleteModal } from "@components/modals";
import { DocsLink } from "@components/ExternalLink";
import { Checkbox } from "@components/Checkbox";
import { TitleSubtitle } from "@components/headings";
import { FilterHalfRow, FilterLayout, FilterPage, FilterSection } from "@screens/filters/sections/_components.tsx";

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
    <div className="mt-5">
      <FieldArray name="external">
        {({ remove, push, move }: FieldArrayRenderProps) => (
          <>
            <div className="-ml-4 -mt-4 mb-6 flex justify-between items-center flex-wrap sm:flex-nowrap">
              <TitleSubtitle
                className="ml-4 mt-4"
                title="External filters"
                subtitle="Run external scripts or webhooks and check status as part of filtering."
              />
              <div className="ml-4 mt-4 flex-shrink-0">
                <button
                  type="button"
                  className="relative inline-flex items-center px-4 py-2 transition border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                  onClick={() => push(newItem)}
                >
                  <SquaresPlusIcon
                    className="w-5 h-5 mr-1"
                    aria-hidden="true"
                  />
                  Add new
                </button>
              </div>
            </div>

            {values.external.length > 0 ? (
              <ul className="rounded-md">
                {values.external.map((external, index: number) => (
                  <FilterExternalItem
                    key={external.id}
                    initialEdit
                    external={external}
                    idx={index}
                    remove={remove}
                    move={move}
                  />
                ))}
              </ul>
            ) : (
              <EmptyListState text="No external filters yet!" />
            )}
          </>
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
          idx % 2 === 0
            ? "bg-white dark:bg-gray-775"
            : "bg-gray-100 dark:bg-gray-815",
          "flex items-center transition px-2 sm:px-6 rounded-md my-1 border border-gray-150 dark:border-gray-750 hover:bg-gray-200 dark:hover:bg-gray-850"
        )}
      >
        {((idx > 0) || (idx < values.external.length - 1)) ? (
          <div className="flex flex-col pr-2 justify-between">
            {idx > 0 && (
              <button type="button" onClick={moveUp}>
                <ArrowUpIcon
                  className="p-0.5 h-4 w-4 text-gray-700 dark:text-gray-400"
                  aria-hidden="true"
                />
              </button>
            )}

            {idx < values.external.length - 1 && (
              <button type="button" onClick={moveDown}>
                <ArrowDownIcon
                  className="p-0.5 h-4 w-4 text-gray-700 dark:text-gray-400"
                  aria-hidden="true"
                />
              </button>
            )}
          </div>
        ) : null}

        <Field name={`external.${idx}.enabled`} type="checkbox">
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
            <div className="truncate">
              <div className="flex text-sm">
                <p className="font-medium text-dark-600 dark:text-gray-100 truncate">
                  {external.name}
                </p>
              </div>
            </div>
            <div className="flex-shrink-0 sm:mt-0 sm:ml-5">
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
                <FilterHalfRow>
                  <Select
                    name={`external.${idx}.type`}
                    label="Type"
                    optionDefaultText="Select type"
                    options={ExternalFilterTypeOptions}
                    tooltip={<div><p>Select the type for this external filter.</p></div>}
                  />
                </FilterHalfRow>

                <FilterHalfRow>
                  <TextField name={`external.${idx}.name`} label="Name" />
                </FilterHalfRow>
              </FilterLayout>
            </FilterSection>

            <TypeForm external={external} idx={idx} />

            <div className="pt-6 pb-4 space-x-2 flex justify-between">
              <button
                type="button"
                className="inline-flex items-center justify-center px-4 py-2 rounded-md sm:text-sm bg-red-700 dark:bg-red-900 hover:dark:bg-red-700 hover:bg-red-800 text-white focus:outline-none"
                onClick={toggleDeleteModal}
              >
                Remove External
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
          <TextAreaAutoResize
            name={`external.${idx}.exec_cmd`}
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
          <TextAreaAutoResize
            name={`external.${idx}.exec_args`}
            label="Exec Arguments"
            columns={5}
            placeholder={"Arguments eg. --test \"{{ .TorrentName }}\""}
          />
          <div className="col-span-12 sm:col-span-2">
            <NumberField
              name={`external.${idx}.exec_expect_status`}
              label="Expected exit status"
              placeholder="0"
            />
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
            <TextField
              name={`external.${idx}.webhook_host`}
              label="Endpoint"
              columns={6}
              placeholder="Host eg. http://localhost/webhook"
              tooltip={<p>URL or IP to your API. Pass params and set API tokens etc.</p>}
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
              label="HTTP Request Headers"
              columns={6}
              placeholder="HEADER=custom1,HEADER2=custom2"
            />
            <NumberField
              name={`external.${idx}.webhook_expect_status`}
              label="Expected HTTP status code"
              placeholder="200"
            />
          </FilterLayout>
        </FilterSection>
        <FilterSection
          title="Retry"
          subtitle="Retry behavior on request failure"
        >
          <FilterLayout>
            <TextField
              name={`external.${idx}.webhook_retry_status`}
              label="Retry http status code(s)"
              placeholder="Retry on status eg. 202, 204"
              columns={6}
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
          </FilterLayout>
        </FilterSection>
        <FilterSection
          title="Payload"
          subtitle="Specify your JSON payload"
        >
          <FilterLayout>
            <TextAreaAutoResize
              name={`external.${idx}.webhook_data`}
              label="Data (json)"
              placeholder={"Request data: { \"key\": \"value\" }"}
            />
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
