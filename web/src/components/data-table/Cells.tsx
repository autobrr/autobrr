/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import * as React from "react";
import { formatDistanceToNowStrict } from "date-fns";
import { CheckIcon } from "@heroicons/react/24/solid";
import { ClockIcon, ExclamationCircleIcon, NoSymbolIcon } from "@heroicons/react/24/outline";

import { classNames, simplifyDate } from "@utils";
import { Tooltip } from "../tooltips/Tooltip";

interface CellProps {
    value: string;
}

export const AgeCell = ({ value }: CellProps) => (
  <div className="text-sm text-gray-500" title={simplifyDate(value)}>
    {formatDistanceToNowStrict(new Date(value), { addSuffix: false })}
  </div>
);

export const IndexerCell = ({ value }: CellProps) => (
  <div
    className={classNames(
      "py-3 text-sm font-medium box-content text-gray-900 dark:text-gray-300",
      "max-w-[96px] sm:max-w-[216px] md:max-w-[360px] lg:max-w-[640px] xl:max-w-[840px]"
    )}
  >
    <Tooltip
      label={value}
      maxWidth="max-w-[90vw]"
    >
      <span className="whitespace-pre-wrap break-words">
        {value}
      </span>
    </Tooltip>
  </div>
);

export const TitleCell = ({ value }: CellProps) => (
  <div
    className={classNames(
      "py-3 text-sm font-medium box-content text-gray-900 dark:text-gray-300",
      "max-w-[96px] sm:max-w-[216px] md:max-w-[360px] lg:max-w-[640px] xl:max-w-[840px]"
    )}
  >
    <Tooltip
      label={value}
      maxWidth="max-w-[90vw]"
    >
      <span className="whitespace-pre-wrap break-words">
        {value}
      </span>
    </Tooltip>
  </div>
);

interface ReleaseStatusCellProps {
    value: ReleaseActionStatus[];
}

interface StatusCellMapEntry {
    colors: string;
    icon: React.ReactElement;
    textFormatter: (text: string) => React.ReactElement;
}

const StatusCellMap: Record<string, StatusCellMapEntry> = {
  "PUSH_ERROR": {
    colors: "bg-pink-100 text-pink-800 hover:bg-pink-300",
    icon: <ExclamationCircleIcon className="h-5 w-5" aria-hidden="true" />,
    textFormatter: (text: string) => (
      <>
        Action
        {" "}
        <span className="font-bold underline underline-offset-2 decoration-2 decoration-red-500">
          error
        </span>
        {": "}
        {text}
      </>
    )
  },
  "PUSH_REJECTED": {
    colors: "bg-blue-100 dark:bg-blue-100 text-blue-400 dark:text-blue-800 hover:bg-blue-300 dark:hover:bg-blue-400",
    icon: <NoSymbolIcon className="h-5 w-5" aria-hidden="true" />,
    textFormatter: (text: string) => (
      <>
        Action
        {" "}
        <span
          className="font-bold underline underline-offset-2 decoration-2 decoration-sky-500"
        >
          rejected
        </span>
        {": "}
        {text}
      </>
    )
  },
  "PUSH_APPROVED": {
    colors: "bg-green-100 text-green-800 hover:bg-green-300",
    icon: <CheckIcon className="h-5 w-5" aria-hidden="true" />,
    textFormatter: (text: string) => (
      <>
        Action
        {" "}
        <span className="font-bold underline underline-offset-2 decoration-2 decoration-green-500">
          approved
        </span>
        {": "}
        {text}
      </>
    )
  },
  "PENDING": {
    colors: "bg-yellow-100 text-yellow-800 hover:bg-yellow-200",
    icon: <ClockIcon className="h-5 w-5" aria-hidden="true" />,
    textFormatter: (text: string) => (
      <>
        Action
        {" "}
        <span className="font-bold underline underline-offset-2 decoration-2 decoration-yellow-500">
          pending
        </span>
        {": "}
        {text}
      </>
    )
  }
};

const CellLine = ({ title, children }: { title: string; children?: string; }) => {
  if (!children)
    return null;

  return (
    <div className="mt-0.5">
      <span className="font-bold">{title}: </span>
      <span className="whitespace-pre-wrap break-words leading-5">{children}</span>
    </div>
  );
};

export const ReleaseStatusCell = ({ value }: ReleaseStatusCellProps) => (
  <div className="flex text-sm font-medium text-gray-900 dark:text-gray-300">
    {value.map((v, idx) => (
      <div
        key={idx}
        className={classNames(
          StatusCellMap[v.status].colors,
          "mr-1 inline-flex items-center rounded text-xs cursor-pointer"
        )}
      >
        <Tooltip
          label={StatusCellMap[v.status].icon}
          title={StatusCellMap[v.status].textFormatter(v.action)}
        >
          <div className="mb-1">
            <CellLine title="Type">{v.type}</CellLine>
            <CellLine title="Client">{v.client}</CellLine>
            <CellLine title="Filter">{v.filter}</CellLine>
            <CellLine title="Time">{simplifyDate(v.timestamp)}</CellLine>
            {v.rejections.length ? (
              <CellLine title="Rejected">
                {v.rejections.toString()}
              </CellLine>
            ) : null}
          </div>
        </Tooltip>
      </div>
    ))}
  </div>
);
