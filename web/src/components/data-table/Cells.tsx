/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import * as React from "react";
import { formatDistanceToNowStrict } from "date-fns";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CellContext } from "@tanstack/react-table";
import { useTranslation } from "react-i18next";
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
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { RingResizeSpinner } from "@components/Icons";
import { Tooltip } from "@components/tooltips/Tooltip";

export const NameCell = (props: CellContext<Release, unknown>) => {
  const { t } = useTranslation("common");

  return (
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
          <span className="text-xs text-gray-500 dark:text-gray-400">{t("releaseTable.labels.category")}:</span> {props.row.original.category}
          <span className="text-xs text-gray-500 dark:text-gray-400"> {t("releaseTable.labels.size")}:</span> {humanFileSize(props.row.original.size)}
          <span className="text-xs text-gray-500 dark:text-gray-400"> {t("releaseTable.labels.misc")}:</span> {`${props.row.original.resolution} ${props.row.original.source} ${props.row.original.codec ?? ""} ${props.row.original.container}`}
        </div>
      </div>
    </div>
  );
};

export const LinksCell = (props: CellContext<Release, unknown>) => {
  const { t } = useTranslation("common");
  return (
    <div className="flex space-x-2 text-blue-400 dark:text-blue-500">
      <div>
        <Tooltip
          requiresClick
          label={<DocumentTextIcon
            className="h-5 w-5 cursor-pointer text-blue-400 hover:text-blue-500 dark:text-blue-500 dark:hover:text-blue-600"
            aria-hidden={true}/>}
          title={t("releaseTable.details")}
        >
          <div className="mb-1">
            <CellLine title={t("releaseTable.fields.release")}>{props.row.original.name}</CellLine>
            <CellLine title={t("releaseTable.fields.indexer")}>{props.row.original.indexer.identifier}</CellLine>
            <CellLine title={t("releaseTable.fields.protocol")}>{props.row.original.protocol}</CellLine>
            <CellLine title={t("releaseTable.fields.implementation")}>{props.row.original.implementation}</CellLine>
            <CellLine title={t("releaseTable.fields.announceType")}>{props.row.original.announce_type}</CellLine>
            <CellLine title={t("releaseTable.fields.category")}>{props.row.original.category}</CellLine>
            <CellLine title={t("releaseTable.fields.uploader")}>{props.row.original.uploader}</CellLine>
            <CellLine title={t("releaseTable.fields.size")}>{humanFileSize(props.row.original.size)}</CellLine>
            <CellLine title={t("releaseTable.fields.title")}>{props.row.original.title}</CellLine>
            {props.row.original.year > 0 && <CellLine title={t("releaseTable.fields.year")}>{props.row.original.year.toString()}</CellLine>}
            {props.row.original.season > 0 &&
                <CellLine title={t("releaseTable.fields.season")}>{props.row.original.season.toString()}</CellLine>}
            {props.row.original.episode > 0 &&
                <CellLine title={t("releaseTable.fields.episode")}>{props.row.original.episode.toString()}</CellLine>}
            <CellLine title={t("releaseTable.fields.resolution")}>{props.row.original.resolution}</CellLine>
            <CellLine title={t("releaseTable.fields.source")}>{props.row.original.source}</CellLine>
            <CellLine title={t("releaseTable.fields.codec")}>{props.row.original.codec}</CellLine>
            <CellLine title={t("releaseTable.fields.hdr")}>{props.row.original.hdr}</CellLine>
            <CellLine title={t("releaseTable.fields.group")}>{props.row.original.group}</CellLine>
            <CellLine title={t("releaseTable.fields.container")}>{props.row.original.container}</CellLine>
            <CellLine title={t("releaseTable.fields.origin")}>{props.row.original.origin}</CellLine>
          </div>
        </Tooltip>
      </div>
      {props.row.original.download_url && (
        <ExternalLink href={props.row.original.download_url}>
          <ArrowDownTrayIcon
            title={t("releaseTable.downloadTorrent")}
            className="h-5 w-5 hover:text-blue-500 dark:hover:text-blue-600"
            aria-hidden="true"
          />
        </ExternalLink>
      )}
      {props.row.original.info_url && (
        <ExternalLink href={props.row.original.info_url}>
          <ArrowTopRightOnSquareIcon
            title={t("releaseTable.visitTorrentInfo")}
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
  const { t } = useTranslation("common");
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (vars: RetryAction) => APIClient.release.replayAction(vars.releaseId, vars.actionId),
    onSuccess: () => {
      // Invalidate filters just in case, most likely not necessary but can't hurt.
      queryClient.invalidateQueries({ queryKey: FilterKeys.lists() });

      toast.custom((tst) => (
        <Toast type="success" body={t("releaseTable.actionReplayed", { action: status?.action })} t={tst} />
      ));
    }
  });

  const replayAction = () => {
    mutation.mutate({ releaseId: status.release_id,actionId: status.id });
  };

  return (
    <button className="flex items-center px-1.5 py-1 ml-2 rounded-sm transition border-gray-500 bg-gray-250 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600" onClick={replayAction}>
      <span className="mr-1.5">{t("releaseTable.retry")}</span>
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

const getStatusCellMap = (t: (key: string, options?: Record<string, unknown>) => string): Record<string, StatusCellMapEntry> => ({
  "PUSH_ERROR": {
    colors: "bg-red-100 text-red-800 hover:bg-red-275",
    icon: <XMarkIcon className="h-5 w-5" aria-hidden="true" />,
    textFormatter: (status: ReleaseActionStatus) => (
      <>
        <span>
          {t("releaseTable.action")}
          {" "}
          <span className="font-bold underline underline-offset-2 decoration-2 decoration-red-500">
            {t("releaseTable.status.error")}
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
          {t("releaseTable.action")}
          {" "}
          <span
            className="font-bold underline underline-offset-2 decoration-2 decoration-sky-500"
          >
            {t("releaseTable.status.rejected")}
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
          {t("releaseTable.action")}
          {" "}
          <span className="font-bold underline underline-offset-2 decoration-2 decoration-green-500">
            {t("releaseTable.status.approved")}
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
  "PENDING": {
    colors: "bg-yellow-100 text-yellow-800 hover:bg-yellow-200",
    icon: <ClockIcon className="h-5 w-5" aria-hidden="true" />,
    textFormatter: (status: ReleaseActionStatus) => (
      <>
        <span>
          {t("releaseTable.action")}
          {" "}
          <span className="font-bold underline underline-offset-2 decoration-2 decoration-yellow-500">
            {t("releaseTable.status.pending")}
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
});

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

export const ReleaseStatusCell = ({ row }: CellContext<Release, unknown>) => {
  const { t } = useTranslation("common");
  const statusCellMap = getStatusCellMap(t);

  return (
    <div className="flex text-sm font-medium text-gray-900 dark:text-gray-300">
      {row.original.action_status.map((v, idx) => (
        <div
          key={idx}
          className={classNames(
            statusCellMap[v.status].colors,
            "mr-1 inline-flex items-center rounded-sm text-xs cursor-pointer"
          )}
        >
          <Tooltip
            requiresClick
            label={statusCellMap[v.status].icon}
            title={statusCellMap[v.status].textFormatter(v)}
          >
            <div className="mb-1">
              <CellLine title={t("releaseTable.fields.type")}>{v.type}</CellLine>
              <CellLine title={t("releaseTable.fields.client")}>{v.client}</CellLine>
              <CellLine title={t("releaseTable.fields.filter")}>{v.filter}</CellLine>
              <CellLine title={t("releaseTable.fields.time")}>{simplifyDate(v.timestamp)}</CellLine>
              {v.rejections.length ? (
                <CellLine title={t("releaseTable.fields.rejected")}>
                  {v.rejections.toString()}
                </CellLine>
              ) : null}
            </div>
          </Tooltip>
        </div>
      ))}
    </div>
  );
};
