/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Tooltip } from "./Tooltip";

interface DocsTooltipProps {
  label?: React.ReactNode;
  children?: React.ReactNode;
}

export const DocsTooltip = ({ label, children }: DocsTooltipProps) => (
  <Tooltip
    label={
      <div className="flex flex-row items-center">
        {label ?? null}
        <svg className="ml-1 w-4 h-4 text-gray-500 dark:text-gray-400 fill-current" viewBox="0 0 72 72"><path d="M32 2C15.432 2 2 15.432 2 32s13.432 30 30 30s30-13.432 30-30S48.568 2 32 2m5 49.75H27v-24h10v24m-5-29.5a5 5 0 1 1 0-10a5 5 0 0 1 0 10" /></svg>
      </div>
    }
  >
    {children}
  </Tooltip>
);

