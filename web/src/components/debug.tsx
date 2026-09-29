/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { FC } from "react";

import { SettingsContext } from "@utils/Context";
import { useFormValues } from "@hooks/form";

interface DebugProps {
    values: unknown;
}

export const DEBUG: FC<DebugProps> = ({ values }) => {
  const debug = SettingsContext.useSelector((s) => s.debug);

  if (!import.meta.env.DEV || !debug) {
    return null;
  }

  return (
    <div className="w-full p-2 flex flex-col mt-6 bg-gray-100 dark:bg-gray-900">
      <pre className="dark:text-gray-400 break-all whitespace-pre-wrap">{JSON.stringify(values, null, 2)}</pre>
    </div>
  );
};

const FormDebugValues = () => <DEBUG values={useFormValues()} />;

// Drop-in for <DEBUG values={values} /> inside a form.AppForm; the form subscription
// only exists while the panel is enabled.
export const FormDebug = () => {
  const debug = SettingsContext.useSelector((s) => s.debug);

  if (!import.meta.env.DEV || !debug) {
    return null;
  }

  return <FormDebugValues />;
};

export function LogDebug(...data: unknown[]): void {
  if (!import.meta.env.DEV) {
    return;
  }

  console.log(...data)
}
