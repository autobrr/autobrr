/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import {Field, FieldArray, FieldProps, useFormikContext} from "formik";
import {NumberField, Select, SwitchGroup, TextField} from "@components/inputs";
import {TextArea} from "@components/inputs/input";
import {Fragment, useEffect, useRef, useState} from "react";
import {EmptyListState} from "@components/emptystates";
import {useToggle} from "@hooks/hooks";
import {useMutation} from "@tanstack/react-query";
import {APIClient} from "@api/APIClient";
import {toast} from "react-hot-toast";
import Toast from "@components/notifications/Toast";
import {classNames} from "@utils";
import { Dialog, Switch as SwitchBasic, Transition } from "@headlessui/react";
import {
  ExternalFilterTypeNameMap,
  ExternalFilterTypeOptions, ExternalFilterWebhookMethodOptions
} from "@domain/constants";
import {ChevronRightIcon} from "@heroicons/react/24/solid";
import {DeleteModal} from "@components/modals";
import {ArrowDownIcon, ArrowUpIcon} from "@heroicons/react/24/outline";

export function External() {
    const {values} = useFormikContext<Filter>();

  const newItem: ExternalFilter = {
      id: values.external.length+1,
    index: values.external.length+1,
    name: `External ${values.external.length+1}`,
    enabled: false,
    type: "EXEC",
  }

    return (
        <div className="mt-10">
          <FieldArray name="external">
            {({ remove, push, move }) => (
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
                    {/*{externalFilters.length > 0 ?*/}
                        <ul className="divide-y divide-gray-200 dark:divide-gray-700">
                          {values.external.map((f, index: number) => (
                              <FilterExternalItem external={f} idx={index} key={index} remove={remove} move={move} initialEdit={true} />
                          ))}
                        </ul>
                    {/*    : <EmptyListState text="No external filters yet!"/>*/}
                    {/*}*/}
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

function FilterExternalItem({ external, idx, initialEdit, remove, move }: FilterExternalItemProps) {
  const {values} = useFormikContext<Filter>();
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
          <Toast type="success" body={`Action ${external?.name} was deleted`} t={t} />
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
          <div className="flex flex-col pr-2 justify-between">
          {idx > 0 && (
              <button type="button" className="bg-gray-600 hover:bg-gray-700" onClick={() => move(idx, idx - 1)}>
                <ArrowUpIcon
                    className="p-0.5 h-4 w-4 text-gray-400"
                    aria-hidden="true"
                />
              </button>
          )}

          {idx < values.external.length - 1 && (
              <button type="button" className="bg-gray-600 hover:bg-gray-700" onClick={() => move(idx, idx + 1)}>
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
                  <p className="ml-4 font-medium text-blue-600 dark:text-gray-100 truncate">
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
                      buttonRef={cancelButtonRef}
                      toggle={toggleDeleteModal}
                      deleteAction={() => removeAction(external.id)}
                      title="Remove external filter"
                      text="Are you sure you want to remove this external filter? This action cannot be undone."
                  />
                </Dialog>
              </Transition.Root>

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

                <TypeForm external={external} idx={idx}/>

                <div className="pt-6 divide-y divide-gray-200">
                  <div className="mt-4 pt-4 flex justify-between">
                    <button
                        type="button"
                        className="inline-flex items-center justify-center py-2 border border-transparent font-medium rounded-md text-red-700 dark:text-red-500 hover:text-red-500 dark:hover:text-red-400 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:text-sm"
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


interface TypeFormProps {
  external: ExternalFilter;
  idx: number;
}

const TypeForm = ({ external, idx }: TypeFormProps) => {
  const { setFieldValue  } = useFormikContext();

  const [prevActionType, setPrevActionType] = useState<string | null>(null);
  useEffect(() => {
    // if (prevActionType !== null) {
    //   resetClientField(external, idx, prevActionType);
    // }
    setPrevActionType(external.type);
  }, [external.type, idx, setFieldValue]);

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
                  tooltip={<div><p>For custom commands you should specify the full path to the binary/program
                    you want to run. And you can include your own static variables:</p><a
                      href='https://autobrr.com/filters/actions#custom-commands--exec'
                      className='text-blue-400 visited:text-blue-400'
                      target='_blank'>https://autobrr.com/filters/actions#custom-commands--exec</a></div>}
              />
              <TextField
                  name={`external.${idx}.exec_args`}
                  label="Arguments"
                  columns={6}
                  placeholder={`Arguments eg. --test "{{ .TorrentName }}"`}
              />
            </div>
            <div className="mt-6 grid grid-cols-12 gap-6">
              <NumberField
                  name={`external.${idx}.script_expected_status`}
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
                name={`external.${idx}.webhook_expected_status`}
                label="Expected http status"
                placeholder="200"
            />
          </div>
      );

    default:
      return null;
  }
};
