/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useRef } from "react";
import { ChevronDownIcon, ChevronRightIcon } from "@heroicons/react/24/solid";
import { ArrowDownIcon, ArrowUpIcon, SquaresPlusIcon } from "@heroicons/react/24/outline";
import { Field, FieldArray, FieldArrayRenderProps, FieldProps, useFormikContext } from "formik";
import { useTranslation } from "react-i18next";

import { classNames } from "@utils";
import { useToggle } from "@hooks/hooks";
import { TextAreaAutoResize } from "@components/inputs/input";
import { EmptyListState } from "@components/emptystates";
import { NumberField, Select, TextField } from "@components/inputs";
import {
  ExternalFilterOnErrorOptions,
  ExternalFilterTypeOptions,
  ExternalFilterWebhookMethodOptions
} from "@domain/constants";

import { DeleteModal } from "@components/modals";
import { DocsLink } from "@components/ExternalLink";
import { Checkbox } from "@components/Checkbox";
import { TitleSubtitle } from "@components/headings";
import { FilterLayout, FilterPage, FilterSection } from "@screens/filters/sections/_components.tsx";

export function External() {
  const { t } = useTranslation("filters");
  const { values } = useFormikContext<Filter>();

  const newItem: ExternalFilter = {
    id: values.external.length + 1,
    index: values.external.length,
    name: `External ${values.external.length + 1}`,
    enabled: false,
    type: "EXEC",
    on_error: "REJECT",
  };

  return (
    <div className="mt-5">
      <FieldArray name="external">
        {({ remove, push, move }: FieldArrayRenderProps) => (
          <>
            <div className="-ml-4 -mt-4 mb-6 flex justify-between items-center flex-wrap sm:flex-nowrap">
              <TitleSubtitle
                className="ml-4 mt-4"
                title={t("external.title")}
                subtitle={t("external.subtitle")}
              />
              <div className="ml-4 mt-4 shrink-0">
                <button
                  type="button"
                  className="relative inline-flex items-center px-4 py-2 transition border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                  onClick={() => push(newItem)}
                >
                  <SquaresPlusIcon
                    className="w-5 h-5 mr-1"
                    aria-hidden="true"
                  />
                  {t("external.addNew")}
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
              <EmptyListState text={t("external.empty")} />
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
  const { t } = useTranslation("filters");
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
          <div className="flex flex-col pr-3 justify-between">
            {idx > 0 && (
              <button type="button" className="cursor-pointer" onClick={moveUp}>
                <ArrowUpIcon
                  className="p-0.5 h-4 w-4 text-gray-700 dark:text-gray-400"
                  aria-hidden="true"
                />
              </button>
            )}

            {idx < values.external.length - 1 && (
              <button type="button" className="cursor-pointer" onClick={moveDown}>
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
                  {t(`external.types.${external.type}`)}
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
            title={t("external.removeTitle")}
            text={t("external.removeText")}
          />

          <FilterPage gap="sm:gap-y-6">
            <FilterSection
              title={t("external.sectionTitle")}
              subtitle={t("external.sectionSubtitle")}
            >
              <FilterLayout>
                <Select
                  name={`external.${idx}.type`}
                  label={t("external.type")}
                  optionDefaultText={t("external.selectType")}
                  options={ExternalFilterTypeOptions.map(option => ({
                    ...option,
                    label: t(`external.types.${option.value}`)
                  }))}
                  tooltip={<div><p>{t("external.typeTooltip")}</p></div>}
                  columns={4}
                />

                <TextField
                  name={`external.${idx}.name`}
                  label={t("external.name")} columns={4}
                />

                <Select
                  name={`external.${idx}.on_error`}
                  label={t("external.onError")}
                  optionDefaultText={t("external.selectType")}
                  options={ExternalFilterOnErrorOptions.map(option => ({
                    ...option,
                    label: t(`external.onErrorOptions.${option.value}`)
                  }))}
                  tooltip={<div><p>{t("external.onErrorTooltip")}</p></div>}
                  columns={4}
                />
              </FilterLayout>
            </FilterSection>

            <TypeForm external={external} idx={idx} />

            <div className="pt-6 pb-4 space-x-2 flex justify-between">
              <button
                type="button"
                className="inline-flex items-center justify-center px-4 py-2 rounded-md sm:text-sm bg-red-700 dark:bg-red-900 dark:hover:bg-red-700 hover:bg-red-800 text-white focus:outline-hidden"
                onClick={toggleDeleteModal}
              >
                {t("external.remove")}
              </button>

              <button
                type="button"
                className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden"
                onClick={toggleEdit}
              >
                {t("external.close")}
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
  const { t } = useTranslation("filters");
  switch (external.type) {
  case "EXEC": {
    return (
      <FilterSection
        title={t("external.execute.title")}
        subtitle={t("external.execute.subtitle")}
      >
        <FilterLayout>
          <TextAreaAutoResize
            name={`external.${idx}.exec_cmd`}
            label={t("external.execute.path")}
            columns={5}
            placeholder={t("external.execute.pathPlaceholder")}
            tooltip={
              <div>
                <p>{t("external.execute.pathTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters/actions#custom-commands--exec" />
              </div>
            }
          />
          <TextAreaAutoResize
            name={`external.${idx}.exec_args`}
            label={t("external.execute.args")}
            columns={5}
            placeholder={t("external.execute.argsPlaceholder")}
          />
          <div className="col-span-12 sm:col-span-2">
            <NumberField
              name={`external.${idx}.exec_expect_status`}
              label={t("external.execute.expectedExitStatus")}
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
          title={t("external.request.title")}
          subtitle={t("external.request.subtitle")}
        >
          <FilterLayout>
            <TextField
              name={`external.${idx}.webhook_host`}
              label={t("external.request.endpoint")}
              columns={6}
              placeholder={t("external.request.endpointPlaceholder")}
              tooltip={<p>{t("external.request.endpointTooltip")}</p>}
            />
            <Select
              name={`external.${idx}.webhook_method`}
              label={t("external.request.httpMethod")}
              optionDefaultText={t("external.request.httpMethodDefault")}
              options={ExternalFilterWebhookMethodOptions}
              tooltip={<div><p>{t("external.request.httpMethodTooltip")}</p></div>}
            />
            <TextField
              name={`external.${idx}.webhook_headers`}
              label={t("external.request.headers")}
              columns={6}
              placeholder={t("external.request.headersPlaceholder")}
            />
            <NumberField
              name={`external.${idx}.webhook_expect_status`}
              label={t("external.request.expectedStatus")}
              placeholder="200"
            />
          </FilterLayout>
        </FilterSection>
        <FilterSection
          title={t("external.retry.title")}
          subtitle={t("external.retry.subtitle")}
        >
          <FilterLayout>
            <TextField
              name={`external.${idx}.webhook_retry_status`}
              label={t("external.retry.retryStatus")}
              placeholder={t("external.retry.retryStatusPlaceholder")}
              columns={6}
            />
            <NumberField
              name={`external.${idx}.webhook_retry_attempts`}
              label={t("external.retry.retryAttempts")}
              placeholder="10"
            />
            <NumberField
              name={`external.${idx}.webhook_retry_delay_seconds`}
              label={t("external.retry.retryDelaySeconds")}
              placeholder="1"
            />
          </FilterLayout>
        </FilterSection>
        <FilterSection
          title={t("external.payload.title")}
          subtitle={t("external.payload.subtitle")}
        >
          <FilterLayout>
            <TextAreaAutoResize
              name={`external.${idx}.webhook_data`}
              label={t("external.payload.data")}
              placeholder={t("external.payload.dataPlaceholder")}
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
