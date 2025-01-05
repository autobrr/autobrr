/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { FC } from "react";
import { SettingsContext } from "@utils/Context";

interface DebugProps {
  values: unknown;
}

export const DEBUG: FC<DebugProps> = ({ values }) => {
  const settings = SettingsContext.useValue();

  if (process.env.NODE_ENV !== "development" || !settings.debug) {
    return null;
  }

  return (
    <div className="w-full p-2 flex flex-col mt-6 bg-gray-100 dark:bg-gray-900">
      <pre className="dark:text-gray-400 break-all whitespace-pre-wrap">{JSON.stringify(values, null, 2)}</pre>
    </div>
  );
};

export function LogDebug(...data: any[]): void {
  if (process.env.NODE_ENV !== "development") {
    return;
  }

  console.log(...data)
}
