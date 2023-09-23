/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { PlacesType, Tooltip } from "react-tooltip";
import "./CustomTooltip.css";


interface CustomTooltipProps {
    anchorId: string;
    children?: React.ReactNode;
    clickable?: boolean;
    place?: PlacesType;
  }

export const CustomTooltip = ({
  anchorId,
  children,
  clickable = true,
  place = "top"
}: CustomTooltipProps) => {
  const id = `${anchorId}-tooltip`;
  return (
    <div className="flex items-center">
      <svg id={id} className="ml-1 w-4 h-4 text-gray-500 dark:text-gray-400 fill-current" viewBox="0 0 72 72"><path d="M32 2C15.432 2 2 15.432 2 32s13.432 30 30 30s30-13.432 30-30S48.568 2 32 2m5 49.75H27v-24h10v24m-5-29.5a5 5 0 1 1 0-10a5 5 0 0 1 0 10"/></svg>
      <Tooltip
        style={{ maxWidth: "350px", fontSize: "12px", textTransform: "none", fontWeight: "normal", borderRadius: "0.375rem", backgroundColor: "#34343A", color: "#fff" }}
        delayShow={100}
        delayHide={150}
        place={place}
        anchorSelect={id}
        data-html={true}
        clickable={clickable}
      >
        {children}
      </Tooltip>
    </div>
  );
};
