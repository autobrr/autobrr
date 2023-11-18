/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { classNames } from "@utils";

type SectionProps = {
  title: string;
  description: string | React.ReactNode;
  rightSide?: React.ReactNode;
  children?: React.ReactNode;
};

export const Section = ({
  title,
  description,
  rightSide,
  children
}: SectionProps) => (
  <div className="pb-6 px-4 lg:col-span-9">
    <div
      className={classNames(
        "mt-6 mb-4",
        rightSide
          ? "flex justify-between items-start flex-wrap sm:flex-nowrap gap-2"
          : ""
      )}
    >
      <div className="sm:px-2">
        <h2 className="text-lg leading-4 font-bold text-gray-900 dark:text-white">{title}</h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{description}</p>
      </div>
      <div className="flex-shrink-0">
        {rightSide ?? null}
      </div>
    </div>
    {children}
  </div>
);

interface RowItemProps {
  label: string;
  value?: string | React.ReactNode;
  title?: string;
  emptyText?: string;
  rightSide?: React.ReactNode;
  className?: string;
}

export const RowItem = ({
  label,
  value,
  title,
  emptyText,
  rightSide,
  className = "sm:col-span-3"
}: RowItemProps) => (
  <div className="p-4 sm:px-6 sm:grid sm:grid-cols-4 sm:gap-4">
    <div className="font-medium text-gray-900 dark:text-white text-sm self-center" title={title}>
      {label}
    </div>
    <div
      className={classNames(
        className,
        "mt-1 text-gray-900 dark:text-gray-300 text-sm break-all sm:mt-0"
      )}
    >
      {value
        ? (
          <>
            {typeof (value) === "string" ? (
              <span className="px-1.5 py-1 bg-gray-200 dark:bg-gray-700 rounded shadow text-ellipsis leading-7">
                {value}
              </span>
            ) : value}
            {rightSide ?? null}
          </>
        )
        : (emptyText ?? null)
      }
    </div>
  </div>
);
