import * as React from "react";
import type { ReactNode } from "react";
import { usePopperTooltip } from "react-popper-tooltip";
import { classNames } from "@utils";

interface TooltipProps {
  label: ReactNode;
  title?: ReactNode;
  maxWidth?: string;
  children: ReactNode;
}

export const Tooltip = ({
  label,
  title,
  children,
  maxWidth = "max-w-sm"
}: TooltipProps) => {
  const {
    // TODO?: getArrowProps,
    getTooltipProps,
    setTooltipRef,
    setTriggerRef,
    visible
  } = usePopperTooltip({
    trigger: ["click"],
    interactive: false
  });

  return (
    <>
      <div ref={setTriggerRef} className="truncate">
        {label}
      </div>
      {visible && (
        <div
          ref={setTooltipRef}
          {...getTooltipProps({
            className: classNames(
              maxWidth,
              "rounded-md border border-gray-300 text-black text-xs shadow-lg dark:text-white dark:border-gray-700 dark:shadow-2xl"
            )
          })}
        >
          {title ? (
            <div className="p-2 border-b border-gray-300 bg-gray-100 dark:border-gray-700 dark:bg-gray-800 rounded-t-md">
              {title}
            </div>
          ) : null}
          <div
            className={classNames(
              title ? "" : "rounded-t-md",
              "py-1 px-2 rounded-b-md bg-white dark:bg-gray-900"
            )}
          >
            {children}
          </div>
        </div>
      )}
    </>
  );
};