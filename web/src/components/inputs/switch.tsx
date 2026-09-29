/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Field, Label, Description } from "@headlessui/react";

import { classNames } from "@utils";
import { useFormContext } from "@hooks/form";
import { DocsTooltip } from "@components/tooltips/DocsTooltip";
import { Checkbox } from "@components/Checkbox";

interface SwitchGroupProps {
  name: string;
  label?: string;
  description?: string | React.ReactNode;
  heading?: boolean;
  tooltip?: React.JSX.Element;
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
}: SwitchGroupProps) => {
  const form = useFormContext();

  return (
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

      <form.Field name={name}>
        {(field) => (
          <Checkbox
            name={name}
            className=""
            value={!!field.state.value}
            setValue={(value) => field.handleChange(value)}
            disabled={disabled}
          />
        )}
      </form.Field>
    </Field>
  );
};

interface SwitchButtonProps {
  name: string;
  defaultValue?: boolean;
  className?: string;
}

const SwitchButton = ({ name }: SwitchButtonProps) => {
  const form = useFormContext();

  return (
    <Field as="div" className="flex items-center justify-between">
      <form.Field name={name}>
        {(field) => (
          <Checkbox
            name={name}
            value={!!field.state.value}
            setValue={(value) => field.handleChange(value)}
          />
        )}
      </form.Field>
    </Field>
  );
};

export { SwitchGroup, SwitchButton };
