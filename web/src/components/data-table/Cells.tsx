/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import * as React from "react";
import { toast } from "react-hot-toast";
import { formatDistanceToNowStrict } from "date-fns";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CellContext } from "@tanstack/react-table";
import { ArrowPathIcon, CheckIcon } from "@heroicons/react/24/solid";
import {
  ClockIcon,
  XMarkIcon,
  NoSymbolIcon,
  ArrowDownTrayIcon,
  ArrowTopRightOnSquareIcon, DocumentTextIcon
} from "@heroicons/react/24/outline";

import { APIClient } from "@api/APIClient";
import { FilterKeys } from "@api/query_keys";
import { classNames, humanFileSize, simplifyDate } from "@utils";
import { ExternalLink } from "../ExternalLink";
import Toast from "@components/notifications/Toast";
import { RingResizeSpinner } from "@components/Icons";
import { Tooltip } from "@components/tooltips/Tooltip";

export const NameCell = (props: CellContext<Release, unknown>) => (
  <div
    className={classNames(
      "flex justify-between items-center py-2 text-sm font-medium box-content text-gray-900 dark:text-gray-300",
      "max-w-[82px] sm:max-w-[130px] md:max-w-[260px] lg:max-w-[500px] xl:max-w-[760px]"
    )}
  >
    <div className="flex flex-col truncate">
      <span className="truncate">
        {String(props.cell.getValue())}
      </span>
      <div className="text-xs truncate">
        <span className="text-xs text-gray-500 dark:text-gray-400">Category:</span> {props.row.original.category}
        <span
          className="text-xs text-gray-500 dark:text-gray-400"> Size:</span> {humanFileSize(props.row.original.size)}
        <span
          className="text-xs text-gray-500 dark:text-gray-400"> Misc:</span> {`${props.row.original.resolution} ${props.row.original.source} ${props.row.original.codec ?? ""} ${props.row.original.container}`}
      </div>
    </div>
  </div>
);

export const LinksCell = (props: CellContext<Release, unknown>) => {
  return (
    <div className="flex space-x-2 text-blue-400 dark:text-blue-500">
      <div>
        <Tooltip
          requiresClick
          label={<DocumentTextIcon
            className="h-5 w-5 cursor-pointer text-blue-400 hover:text-blue-500 dark:text-blue-500 dark:hover:text-blue-600"
            aria-hidden={true}/>}
          title="Details"
        >
          <div className="mb-1">
            <CellLine title="Release">{props.row.original.name}</CellLine>
            <CellLine title="Indexer">{props.row.original.indexer.identifier}</CellLine>
            <CellLine title="Protocol">{props.row.original.protocol}</CellLine>
            <CellLine title="Implementation">{props.row.original.implementation}</CellLine>
            <CellLine title="Category">{props.row.original.category}</CellLine>
            <CellLine title="Uploader">{props.row.original.uploader}</CellLine>
            <CellLine title="Size">{humanFileSize(props.row.original.size)}</CellLine>
            <CellLine title="Title">{props.row.original.title}</CellLine>
            {props.row.original.year > 0 && <CellLine title="Year">{props.row.original.year.toString()}</CellLine>}
            {props.row.original.season > 0 &&
                <CellLine title="Season">{props.row.original.season.toString()}</CellLine>}
            {props.row.original.episode > 0 &&
                <CellLine title="Episode">{props.row.original.episode.toString()}</CellLine>}
            <CellLine title="Resolution">{props.row.original.resolution}</CellLine>
            <CellLine title="Source">{props.row.original.source}</CellLine>
            <CellLine title="Codec">{props.row.original.codec}</CellLine>
            <CellLine title="HDR">{props.row.original.hdr}</CellLine>
            <CellLine title="Group">{props.row.original.group}</CellLine>
            <CellLine title="Container">{props.row.original.container}</CellLine>
            <CellLine title="Origin">{props.row.original.origin}</CellLine>
          </div>
        </Tooltip>
      </div>
      {props.row.original.download_url && (
        <ExternalLink href={props.row.original.download_url}>
          <ArrowDownTrayIcon
            title="Download torrent file"
            className="h-5 w-5 hover:text-blue-500 dark:hover:text-blue-600"
            aria-hidden="true"
          />
        </ExternalLink>
      )}
      {props.row.original.info_url && (
        <ExternalLink href={props.row.original.info_url}>
          <ArrowTopRightOnSquareIcon
            title="Visit torrentinfo url"
            className="h-5 w-5 hover:text-blue-500 dark:hover:text-blue-600"
            aria-hidden="true"
          />
        </ExternalLink>
      )}
    </div>
  );
};

export const AgeCell = ({cell}: CellContext<Release, unknown>) => (
  <div className="text-sm text-gray-500" title={simplifyDate(cell.getValue() as string)}>
    {formatDistanceToNowStrict(new Date(cell.getValue() as string), {addSuffix: false})}
  </div>
);

export const IndexerCell = (props: CellContext<Release, unknown>) => (
    <div
      className={classNames(
        "py-3 text-sm font-medium box-content text-gray-900 dark:text-gray-300",
        "max-w-[96px] sm:max-w-[216px] md:max-w-[360px] lg:max-w-[640px] xl:max-w-[840px]"
      )}
    >
      <Tooltip
        requiresClick
        label={props.row.original.indexer.name ? props.row.original.indexer.name : props.row.original.indexer.identifier}
        maxWidth="max-w-[90vw]"
      >
      <span className="whitespace-pre-wrap break-words">
        {props.row.original.indexer.name ? props.row.original.indexer.name : props.row.original.indexer.identifier}
      </span>
      </Tooltip>
    </div>
);

export const TitleCell = ({cell}: CellContext<Release, string>) => (
  <div
    className={classNames(
      "py-3 text-sm font-medium box-content text-gray-900 dark:text-gray-300",
      "max-w-[96px] sm:max-w-[216px] md:max-w-[360px] lg:max-w-[640px] xl:max-w-[840px]"
    )}
  >
    <Tooltip
      requiresClick
      label={cell.getValue()}
      maxWidth="max-w-[90vw]"
    >
      <span className="whitespace-pre-wrap break-words">
        {cell.getValue()}
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
      queryClient.invalidateQueries({ queryKey: FilterKeys.lists() });

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
      {mutation.isPending
        ? <RingResizeSpinner className="text-blue-500 w-4 h-4 iconHeight" aria-hidden="true" />
        : <ArrowPathIcon className="h-4 w-4" />
      }
    </button>
  );
};

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

export const ReleaseStatusCell = ({ row }: CellContext<Release, unknown>) => (
  <div className="flex text-sm font-medium text-gray-900 dark:text-gray-300">
    {row.original.action_status.map((v, idx) => (
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

