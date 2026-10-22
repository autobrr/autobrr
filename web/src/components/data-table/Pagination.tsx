/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useTranslation } from "react-i18next";
import {
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon
} from "@heroicons/react/24/solid";

import { TableButton, TablePageButton } from "./Buttons";

interface TablePaginationProps {
  pageIndex: number;
  pageCount: number;
  pageSize: number;
  pageSizes: readonly number[];
  onPageChange: (pageIndex: number) => void;
  onPageSizeChange: (pageSize: number) => void;
}

/** TablePagination is the footer row shared by the data tables. pageIndex is zero-based. */
export const TablePagination = ({ pageIndex, pageCount, pageSize, pageSizes, onPageChange, onPageSizeChange }: TablePaginationProps) => {
  const { t } = useTranslation("common");
  const canPrevious = pageIndex > 0;
  const canNext = pageIndex + 1 < pageCount;

  return (
    <div className="flex items-center justify-between px-6 py-3 border-t border-gray-200 dark:border-gray-700">
      <div className="flex justify-between flex-1 sm:hidden">
        <TableButton onClick={() => onPageChange(pageIndex - 1)} disabled={!canPrevious}>{t("releaseTable.previous")}</TableButton>
        <TableButton onClick={() => onPageChange(pageIndex + 1)} disabled={!canNext}>{t("releaseTable.next")}</TableButton>
      </div>
      <div className="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
        <div className="flex items-baseline gap-x-2">
          <span className="text-sm text-gray-700 dark:text-gray-400">
            {t("releaseTable.pageOf", { page: pageIndex + 1, total: pageCount })}
          </span>
          <label>
            <span className="sr-only">{t("releaseTable.itemsPerPage")}</span>
            <select
              className="py-1 pl-2 pr-8 text-sm block w-full border-gray-300 rounded-md shadow-xs cursor-pointer transition-colors dark:bg-gray-800 dark:border-gray-600 dark:text-gray-400 dark:hover:text-gray-200 focus:border-blue-300 focus:ring-3 focus:ring-blue-200/50"
              value={pageSize}
              onChange={e => onPageSizeChange(Number(e.target.value))}
            >
              {pageSizes.map(size => (
                <option key={size} value={size}>
                  {t("releaseTable.entries", { count: size })}
                </option>
              ))}
            </select>
          </label>
        </div>
        <div>
          <nav className="inline-flex -space-x-px rounded-md shadow-xs" aria-label={t("releaseTable.pagination")}>
            <TablePageButton
              className="rounded-l-md"
              onClick={() => onPageChange(0)}
              disabled={!canPrevious}
            >
              <span className="sr-only">{t("releaseTable.first")}</span>
              <ChevronDoubleLeftIcon className="w-4 h-4" aria-hidden="true"/>
            </TablePageButton>
            <TablePageButton
              className="pl-1 pr-2"
              onClick={() => onPageChange(pageIndex - 1)}
              disabled={!canPrevious}
            >
              <ChevronLeftIcon className="w-4 h-4 mr-1" aria-hidden="true"/>
              <span>{t("releaseTable.prev")}</span>
            </TablePageButton>
            <TablePageButton
              className="pl-2 pr-1"
              onClick={() => onPageChange(pageIndex + 1)}
              disabled={!canNext}
            >
              <span>{t("releaseTable.next")}</span>
              <ChevronRightIcon className="w-4 h-4 ml-1" aria-hidden="true"/>
            </TablePageButton>
            <TablePageButton
              className="rounded-r-md"
              onClick={() => onPageChange(pageCount - 1)}
              disabled={!canNext}
            >
              <ChevronDoubleRightIcon className="w-4 h-4" aria-hidden="true"/>
              <span className="sr-only">{t("releaseTable.last")}</span>
            </TablePageButton>
          </nav>
        </div>
      </div>
    </div>
  );
};
