/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { FieldProps } from "formik";
import { Field as FormikField } from "formik";
import { Field, Label, Description } from "@headlessui/react";

import { classNames } from "@utils";
import { DocsTooltip } from "@components/tooltips/DocsTooltip";
import { Checkbox } from "@components/Checkbox";

interface SwitchGroupProps {
  name: string;
  label?: string;
  description?: string | React.ReactNode;
  heading?: boolean;
  tooltip?: JSX.Element;
  disabled?: boolean;
  className?: string;
}

const SwitchGroup = ({
  name,
  label,
  description,
  tooltip,
  heading,
  disabled,
  className
}: SwitchGroupProps) => (
  <Field
    as="div"
    className={classNames(
      className ?? "py-2",
      "flex items-center justify-between"
    )}
  >
    {label && <div className="flex flex-col">
      <Label
        passive
        as={heading ? "h2" : "span"}
        className={classNames(
          "flex float-left ml-px cursor-default text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide",
          heading ? "text-lg" : "text-sm"
        )}
      >
        <div className="flex">
          {tooltip ? (
            <DocsTooltip label={label}>{tooltip}</DocsTooltip>
          ) : label}
        </div>
      </Label>
      {description && (
        <Description as="span" className="text-sm mt-1 pr-4 text-gray-500 dark:text-gray-400">
          {description}
        </Description>
      )}
    </div>
    }

    <FormikField name={name} type="checkbox">
      {({
        field,
        form: { setFieldValue }
      }: FieldProps) => (
        <Checkbox
          {...field}
          className=""
          value={!!field.checked}
          setValue={(value) => {
            setFieldValue(field?.name ?? "", value);
          }}
          disabled={disabled}
        />
      )}
    </FormikField>
  </Field>
);

interface SwitchButtonProps {
  name: string;
  defaultValue?: boolean;
  className?: string;
}

const SwitchButton = ({ name, defaultValue }: SwitchButtonProps) => (
  <Field as="div" className="flex items-center justify-between">
    <FormikField
      name={name}
      defaultValue={defaultValue as boolean}
      type="checkbox"
    >
      {({
        field,
        form: { setFieldValue }
      }: FieldProps) => (
        <Checkbox
          {...field}
          value={!!field.checked}
          setValue={(value) => {
            setFieldValue(field?.name ?? "", value);
          }}
        />
      )}
    </FormikField>
  </Field>
);

export { SwitchGroup, SwitchButton };
