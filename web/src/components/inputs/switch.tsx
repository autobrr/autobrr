/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import type { FieldInputProps, FieldMetaProps, FieldProps, FormikProps, FormikValues } from "formik";
import { Field } from "formik";
import { Switch as HeadlessSwitch } from "@headlessui/react";

import { classNames } from "@utils";
import { CustomTooltip } from "@components/tooltips/CustomTooltip";

type SwitchProps<V = unknown> = {
    label?: string
    checked: boolean
    value: boolean
    disabled?: boolean
    onChange: (value: boolean) => void
    field?: FieldInputProps<V>
    form?: FormikProps<FormikValues>
    meta?: FieldMetaProps<V>
    children: React.ReactNode
  className: string
};

export const Switch = ({
  label,
  checked: $checked,
  disabled = false,
  onChange: $onChange,
  field,
  form
}: SwitchProps) => {
  const checked = field?.checked ?? $checked;

  return (
    <HeadlessSwitch.Group as="div" className="flex items-center space-x-4">
      <HeadlessSwitch.Label>{label}</HeadlessSwitch.Label>
      <HeadlessSwitch
        as="button"
        name={field?.name}
        disabled={disabled}
        checked={checked}
        onChange={value => {
          form?.setFieldValue(field?.name ?? "", value);
          $onChange && $onChange(value);
        }}

        className={classNames(
          checked ? "bg-blue-500" : "bg-gray-200 dark:bg-gray-600",
          "ml-4 relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        )}
      >
        {({ checked }) => (
          <span
            aria-hidden="true"
            className={classNames(
              checked ? "translate-x-5" : "translate-x-0",
              "inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200"
            )}
          />
        )}
      </HeadlessSwitch>
    </HeadlessSwitch.Group>
  );
};

export type SwitchFormikProps = SwitchProps & FieldProps & React.InputHTMLAttributes<HTMLInputElement>;

export const SwitchFormik = (props: SwitchProps) => <Switch {...props}  children={props.children}/>;

interface SwitchGroupProps {
    name: string;
    label?: string;
    description?: string;
    className?: string;
    heading?: boolean;
    tooltip?: JSX.Element;
}

const SwitchGroup = ({
  name,
  label,
  description,
  tooltip,
  heading
}: SwitchGroupProps) => (
  <HeadlessSwitch.Group as="ol" className="py-4 flex items-center justify-between">
    {label && <div className="flex flex-col">
      <HeadlessSwitch.Label as={heading ? "h2" : "span"} className={classNames("flex float-left cursor-default mb-2 text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide", heading ? "text-lg" : "text-sm")}
        passive>
        <div className="flex">
          {label}
          {tooltip && (
            <CustomTooltip anchorId={name}>{tooltip}</CustomTooltip>
          )}
        </div>
      </HeadlessSwitch.Label>
      {description && (
        <HeadlessSwitch.Description className="text-sm mt-1 text-gray-500 dark:text-gray-400">
          {description}
        </HeadlessSwitch.Description>
      )}
    </div>
    }

    <Field name={name} type="checkbox">
      {({
        field,
        form: { setFieldValue }
      }: FieldProps) => (
        <Switch
          {...field}
          // type="button"
          value={field.value}
          checked={field.checked ?? false}
          onChange={value => {
            setFieldValue(field?.name ?? "", value);
          }}
          className={classNames(
            field.value ? "bg-blue-500" : "bg-gray-200",
            "ml-4 relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          )}
        >
          <span
            aria-hidden="true"
            className={classNames(
              field.value ? "translate-x-5" : "translate-x-0",
              "inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200"
            )}
          />
        </Switch>

      )}
    </Field>
  </HeadlessSwitch.Group>
);

export { SwitchGroup };
