/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import * as React from "react";
import { toast } from "react-hot-toast";
import { formatDistanceToNowStrict } from "date-fns";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowPathIcon, CheckIcon } from "@heroicons/react/24/solid";
import { ArrowDownTrayIcon, ArrowTopRightOnSquareIcon } from "@heroicons/react/24/outline";
import { ExternalLink } from "../ExternalLink"; 
import { ClockIcon, XMarkIcon, NoSymbolIcon } from "@heroicons/react/24/outline";

import { APIClient } from "@api/APIClient";
import { classNames, simplifyDate } from "@utils";
import { filterKeys } from "@screens/filters/List";
import Toast from "@components/notifications/Toast";
import { RingResizeSpinner } from "@components/Icons";
import { Tooltip } from "@components/tooltips/Tooltip";

interface CellProps {
    value: string;
}

interface LinksCellProps {
  value: Release;
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
      requiresClick
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
      requiresClick
      label={value}
      maxWidth="max-w-[90vw]"
    >
      <span className="whitespace-pre-wrap break-words">
        {value}
      </span>
    </Tooltip>
  </div>
);

interface RetryActionButtonProps {
  status: ReleaseActionStatus;
}

interface RetryAction {
  releaseId: number;
  actionId: number;
}

const RetryActionButton = ({ status }: RetryActionButtonProps) => {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (vars: RetryAction) => APIClient.release.replayAction(vars.releaseId, vars.actionId),
    onSuccess: () => {
      // Invalidate filters just in case, most likely not necessary but can't hurt.
      queryClient.invalidateQueries({ queryKey: filterKeys.lists() });

      toast.custom((t) => (
        <Toast type="success" body={`${status?.action} replayed`} t={t} />
      ));
    }
  });

  const replayAction = () => {
    console.log("replay action");
    mutation.mutate({ releaseId: status.release_id,actionId: status.id });
  };

  return (
    <button className="flex items-center px-1.5 py-1 ml-2 rounded transition border-gray-500 bg-gray-250 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600" onClick={replayAction}>
      <span className="mr-1.5">Retry</span>
      {mutation.isLoading
        ? <RingResizeSpinner className="text-blue-500 w-4 h-4 iconHeight" aria-hidden="true" />
        : <ArrowPathIcon className="h-4 w-4" />
      }
    </button>
  );
};

interface ReleaseStatusCellProps {
    value: ReleaseActionStatus[];
}

interface StatusCellMapEntry {
    colors: string;
    icon: React.ReactElement;
    textFormatter: (status: ReleaseActionStatus) => React.ReactElement;
}

const StatusCellMap: Record<string, StatusCellMapEntry> = {
  "PUSH_ERROR": {
    colors: "bg-red-100 text-red-800 hover:bg-red-275",
    icon: <XMarkIcon className="h-5 w-5" aria-hidden="true" />,
    textFormatter: (status: ReleaseActionStatus) => (
      <>
        <span>
        Action
          {" "}
          <span className="font-bold underline underline-offset-2 decoration-2 decoration-red-500">
          error
          </span>
          {": "}
          {status.action}
        </span>
        <div>
          {status.action_id > 0 && <RetryActionButton status={status} />}
        </div>
      </>
    )
  },
  "PUSH_REJECTED": {
    colors: "bg-blue-100 dark:bg-blue-100 text-blue-400 dark:text-blue-800 hover:bg-blue-300 dark:hover:bg-blue-400",
    icon: <NoSymbolIcon className="h-5 w-5" aria-hidden="true" />,
    textFormatter: (status: ReleaseActionStatus) => (
      <>
        <span>
        Action
          {" "}
          <span
            className="font-bold underline underline-offset-2 decoration-2 decoration-sky-500"
          >
          rejected
          </span>
          {": "}
          {status.action}
        </span>
        <div>
          {status.action_id > 0 && <RetryActionButton status={status} />}
        </div>
      </>
    )
  },
  "PUSH_APPROVED": {
    colors: "bg-green-175 text-green-900 hover:bg-green-300",
    icon: <CheckIcon className="h-5 w-5" aria-hidden="true" />,
    textFormatter: (status: ReleaseActionStatus) => (
      <>
        <span>
          Action
          {" "}
          <span className="font-bold underline underline-offset-2 decoration-2 decoration-green-500">
          approved
          </span>
          {": "}
          {status.action}
        </span>
        {/*<div>*/}
        {/*  {status.action_id > 0 && <RetryActionButton status={status} />}*/}
        {/*</div>*/}
      </>
    )
  },
  "PENDING": {
    colors: "bg-yellow-100 text-yellow-800 hover:bg-yellow-200",
    icon: <ClockIcon className="h-5 w-5" aria-hidden="true" />,
    textFormatter: (status: ReleaseActionStatus) => (
      <>
        <span>
          Action
          {" "}
          <span className="font-bold underline underline-offset-2 decoration-2 decoration-yellow-500">
          pending
          </span>
          {": "}
          {status.action}
        </span>
        <div>
          {status.action_id > 0 && <RetryActionButton status={status} />}
        </div>
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
          requiresClick
          label={StatusCellMap[v.status].icon}
          title={StatusCellMap[v.status].textFormatter(v)}
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

export const LinksCell = ({ value }: LinksCellProps) => {
  return (
    <div className="flex space-x-2">
      {value.download_url && (
        <ExternalLink href={value.download_url}>
          <ArrowDownTrayIcon title="Download torrent file" className="h-5 w-5 text-blue-400 hover:text-blue-500 dark:text-blue-500 dark:hover:text-blue-600" aria-hidden="true" />
        </ExternalLink>
      )}
      {value.info_url && (
        <ExternalLink href={value.info_url}>
          <ArrowTopRightOnSquareIcon title="Visit torrentinfo url" className="h-5 w-5 text-blue-400 hover:text-blue-500 dark:text-blue-500 dark:hover:text-blue-600" aria-hidden="true" />
        </ExternalLink>
      )}
    </div>
  );
};