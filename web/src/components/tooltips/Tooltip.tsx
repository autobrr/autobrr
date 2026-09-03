/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect, useRef } from 'react';
import type { ReactNode } from 'react';

import { Transition } from "@headlessui/react";
import { usePopperTooltip } from "react-popper-tooltip";
import { Placement } from '@popperjs/core';

import { classNames } from "@utils";
import { useMedia } from "@hooks/hooks";

interface TooltipProps {
  label: ReactNode;
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
  title,
  children,
  requiresClick,
  maxWidth = "max-w-sm"
}: TooltipProps) => {
  const [isTooltipVisible, setIsTooltipVisible] = useState(false);
  const tooltipNode = useRef<HTMLDivElement | null>(null);
  const triggerNode = useRef<HTMLDivElement | null>(null);

  // Below tailwind's sm breakpoint the tooltip sits above its trigger instead of to the right
  const isSmallScreen = useMedia("(max-width: 639px)", false);
  const placement: Placement = isSmallScreen ? "top" : "right";

  const {
    getTooltipProps,
    setTooltipRef: popperSetTooltipRef,
    setTriggerRef: popperSetTriggerRef,
    visible
  } = usePopperTooltip({
    trigger: requiresClick ? 'click' : ['click', 'hover'],
    interactive: true,
    delayHide: 200,
    placement,
    followCursor: placement === "right"
  });

  const handleClick = (e: React.MouseEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsTooltipVisible((visible) => !visible);
  };

  const setTooltipRef = (node: HTMLDivElement | null) => {
    popperSetTooltipRef(node);
    tooltipNode.current = node;
  };

  const setTriggerRef = (node: HTMLDivElement | null) => {
    popperSetTriggerRef(node);
    triggerNode.current = node;
  };

  // Only an open tooltip needs to hear about outside clicks, so a table full of
  // closed ones registers nothing on the document
  useEffect(() => {
    if (!isTooltipVisible) {
      return;
    }

    const handleClickOutside = (event: Event) => {
      const target = event.target as Node;
      if (tooltipNode.current && !tooltipNode.current.contains(target) && triggerNode.current && !triggerNode.current.contains(target)) {
        setIsTooltipVisible(false);
      }
    };

    document.addEventListener('touchstart', handleClickOutside, { capture: true, passive: true });
    document.addEventListener('mousedown', handleClickOutside, true);
    return () => {
      document.removeEventListener('touchstart', handleClickOutside, true);
      document.removeEventListener('mousedown', handleClickOutside, true);
    };
  }, [isTooltipVisible]);

  return (
    <>
      <div
        ref={setTriggerRef}
        className="truncate cursor-pointer"
        onClick={handleClick}
      >
        {label}
      </div>
      <Transition
        show={isTooltipVisible || visible}
        enter="transition-opacity duration-200 ease-out"
        enterFrom="opacity-0"
        enterTo="opacity-100"
        leave="transition-opacity duration-200 ease-in"
        leaveFrom="opacity-100"
        leaveTo="opacity-0"
      >
        <div
          ref={setTooltipRef}
          {...getTooltipProps({
            className: classNames(
              maxWidth,
              "z-10 rounded-md border border-gray-300 text-black text-xs normal-case tracking-normal font-normal shadow-lg dark:text-white dark:border-gray-700 dark:shadow-2xl"
            ),
            onClick: (e: React.MouseEvent) => e.stopPropagation()
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
