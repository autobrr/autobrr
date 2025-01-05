/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { ChevronDownIcon } from "@heroicons/react/24/solid";

import { classNames } from "@utils";
import { useToggle } from "@hooks/hooks";
import { TitleSubtitle } from "@components/headings";

type FilterSectionProps = {
  children: React.ReactNode;
  title?: string;
  subtitle?: string | React.ReactNode;
  gap?: string;
};

type OwningComponent = {
  children: React.ReactNode;
  className?: string;
  gap?: string;
};

const VerticalGap = "gap-y-6 sm:gap-y-4";

export const FilterNormalGridGapClass = `gap-x-0.5 sm:gap-x-3 ${VerticalGap}`;
export const FilterTightGridGapClass = `gap-x-0.5 sm:gap-x-1.5 ${VerticalGap}`;
export const FilterWideGridGapClass = `gap-x-0.5 sm:gap-x-6 ${VerticalGap}`;

export const FilterLayoutClass = "grid grid-cols-12 col-span-12";

export const FilterLayout = ({
  children,
  className = "",
  gap = FilterNormalGridGapClass
}: OwningComponent) => (
  <div className={classNames(className, FilterLayoutClass, gap)}>{children}</div>
);

export const FilterRow = ({
  children,
  className = "",
  gap = FilterNormalGridGapClass
}: OwningComponent) => (
  <div className={classNames(className, gap, "col-span-12")}>{children}</div>
);

export const FilterHalfRow = ({
  children,
  className = "",
  gap = FilterNormalGridGapClass
}: OwningComponent) => (
  <div className={classNames(className, gap, "col-span-12 sm:col-span-6")}>{children}</div>
);

export const FilterSection = ({
  title,
  subtitle,
  children,
  gap = FilterNormalGridGapClass
}: FilterSectionProps) => (
  <div
    className={classNames(
      title ? "py-6" : "pt-5 pb-4",
      "flex flex-col",
      gap
    )}
  >
    {(title && subtitle) ? (
      <TitleSubtitle title={title} subtitle={subtitle} />
    ) : null}
    {children}
  </div>
);

type FilterPageProps = {
  gap?: string;
  children: React.ReactNode;
};

export const FilterPage = ({
  gap = VerticalGap,
  children
}: FilterPageProps) => (
  <div
    className={classNames(
      gap,
      "flex flex-col w-full divide-y divide-gray-150 dark:divide-gray-750"
    )}
  >
    {children}
  </div>
);

interface CollapsibleSectionProps {
  title: string;
  subtitle?: string | React.ReactNode;
  children: React.ReactNode;
  defaultOpen?: boolean;
  noBottomBorder?: boolean;
  childClassName?: string;
}

export const CollapsibleSection = ({
  title,
  subtitle,
  children,
  defaultOpen = false,
  noBottomBorder = false,
  childClassName = FilterNormalGridGapClass
}: CollapsibleSectionProps) => {
  const [isOpen, toggleOpen] = useToggle(defaultOpen);

  return (
    <div
      className={classNames(
        isOpen ? "pb-10" : "pb-4",
        noBottomBorder ? "" : "border-dashed border-b-2 border-gray-150 dark:border-gray-775",
        "rounded-t-lg"
      )}
    >
      <div
        className="flex select-none items-center py-3.5 px-1 -ml-1 cursor-pointer transition rounded-lg hover:bg-gray-100 dark:hover:bg-gray-725"
        onClick={toggleOpen}
      >
        <div className="flex flex-row gap-2 items-center">
          <button
            type="button"
            className={classNames(
              isOpen ? "rotate-0" : "-rotate-90",
              "text-sm font-medium text-white transition-transform"
            )}
          >
            <ChevronDownIcon className="h-6 w-6 text-gray-400" aria-hidden="true" />
          </button>
          <div
            className={classNames(
              isOpen ? "flex-col gap-0" : "flex-col sm:flex-row sm:items-end sm:gap-2",
              "flex"
            )}
          >
            <h3 className="text-xl leading-6 font-bold break-all dark:shadow-gray-900 text-gray-900 dark:text-gray-200">
              {title}
            </h3>
            <p className="text-sm text-gray-500 dark:text-gray-400 truncate whitespace-normal break-words">
              {subtitle}
            </p>
          </div>
        </div>
      </div>
      {/*TODO: Animate this too*/}
      {isOpen && (
        <div className={classNames(childClassName, "grid grid-cols-12 col-span-12 sm:px-1 mt-2")}>
          {children}
        </div>
      )}
    </div>
  );
};
