/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { ReactNode } from "react";
import { Transition } from "@headlessui/react";
import { usePopperTooltip } from "react-popper-tooltip";

import { classNames } from "@utils";

interface TooltipProps {
  label: ReactNode;
  onLabelClick?: (e: React.MouseEvent<HTMLDivElement>) => void;
  title?: ReactNode;
  maxWidth?: string;
  requiresClick?: boolean;
  children: ReactNode;
}

// NOTE(stacksmash76): onClick is not propagated
// to the label (always-visible) component, so you will have
// to use the `onLabelClick` prop in this case.
export const Tooltip = ({
  label,
  onLabelClick,
  title,
  children,
  requiresClick,
  maxWidth = "max-w-sm"
}: TooltipProps) => {
  const {
    // TODO?: getArrowProps,
    getTooltipProps,
    setTooltipRef,
    setTriggerRef,
    visible
  } = usePopperTooltip({
    trigger: requiresClick ? ["click"] : ["click", "hover"],
    interactive: true,
    delayHide: 200,
    placement: "right"
  });

  if (!children || Array.isArray(children) && !children.length) {
    return null;
  }

  return (
    <>
      <div
        ref={setTriggerRef}
        className="truncate"
        onClick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          e.nativeEvent.stopImmediatePropagation();

          onLabelClick?.(e);
        }}
      >
        {label}
      </div>
      <Transition
        show={visible}
        className="z-10"
        enter="transition duration-200 ease-out"
        enterFrom="opacity-0"
        enterTo="opacity-100"
        leave="transition duration-200 ease-in"
        leaveFrom="opacity-100"
        leaveTo="opacity-0"
      >
        <div
          ref={setTooltipRef}
          {...getTooltipProps({
            className: classNames(
              maxWidth,
              "rounded-md border border-gray-300 text-black text-xs normal-case tracking-normal font-normal shadow-lg dark:text-white dark:border-gray-700 dark:shadow-2xl"
            )
          })}
        >
          {title ? (
            <div className="flex justify-between items-center p-2 border-b border-gray-300 bg-gray-100 dark:border-gray-700 dark:bg-gray-800 rounded-t-md">
              {title}
            </div>
          ) : null}
          <div
            className={classNames(
              title ? "" : "rounded-t-md",
              "whitespace-normal break-words py-1 px-2 rounded-b-md bg-white dark:bg-gray-900"
            )}
          >
            {children}
          </div>
        </div>
      </Transition>
    </>
  );
};
