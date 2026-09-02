/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useSyncExternalStore } from "react";
import { useTranslation } from "react-i18next";
import { SettingsContext } from "@utils/Context";
import { classNames } from "@utils";

const subscribeToColorScheme = (onChange: () => void) => {
  const mq = window.matchMedia("(prefers-color-scheme: dark)");
  mq.addEventListener("change", onChange);
  return () => mq.removeEventListener("change", onChange);
};

export const useIsDark = (): boolean => {
  const settings = SettingsContext.useValue();
  const prefersDark = useSyncExternalStore(
    subscribeToColorScheme,
    () => window.matchMedia("(prefers-color-scheme: dark)").matches
  );
  return settings.theme === "dark" || (settings.theme === "system" && prefersDark);
};

// Tailwind hues the app already uses (green-600, blue-500, zinc, red),
// validated for CVD separation and surface contrast against the card
// surfaces (#ffffff / #27272a). Rejected is deliberately gray: it is
// context next to the series that matter, and a warm hue beside the green
// fails colorblind separation. Keep the series order approved, matches,
// rejected, errored when they share a chart.
export const seriesColors = (dark: boolean) => ({
  approved: "#16a34a",
  matches: "#3b82f6",
  rejected: dark ? "#71717a" : "#a1a1aa",
  errored: dark ? "#ef4444" : "#dc2626"
});

export const sequentialRamp = (dark: boolean): string[] =>
  dark
    ? ["#1e3a8a", "#1d4ed8", "#3b82f6", "#60a5fa", "#bfdbfe"]
    : ["#bfdbfe", "#93c5fd", "#60a5fa", "#2563eb", "#1e40af"];

export const chartTheme = (dark: boolean) => ({
  foreground: dark ? "#d4d4d8" : "#3f3f46",
  muted: dark ? "#a1a1aa" : "#71717a",
  grid: dark ? "#3f3f46" : "#e4e4e7",
  background: dark ? "#27272a" : "#ffffff",
  palette: Object.values(seriesColors(dark))
});

interface ChartCardProps {
  title: string;
  className?: string;
  action?: React.ReactNode;
  children: React.ReactNode;
}

export const ChartCard = ({ title, className, action, children }: ChartCardProps) => (
  <div className={classNames("flex h-full min-w-0 flex-col overflow-hidden rounded-lg bg-white px-4 py-3 shadow-lg dark:bg-gray-800", className ?? "")}>
    <div className="flex min-w-0 flex-wrap items-center justify-between gap-2">
      <h3 className="min-w-0 flex-1 truncate text-sm font-medium text-gray-500 dark:text-gray-400">{title}</h3>
      {action}
    </div>
    {children}
  </div>
);

interface RangeSelectProps<T extends string> {
  options: { value: T; label: string }[];
  value: T;
  onChange: (value: T) => void;
}

export const RangeSelect = <T extends string>({ options, value, onChange }: RangeSelectProps<T>) => (
  <div className="flex items-center gap-0.5 shrink-0" role="group">
    {options.map((option) => (
      <button
        key={option.value}
        type="button"
        aria-pressed={option.value === value}
        onClick={() => onChange(option.value)}
        className={classNames(
          "inline-flex min-h-9 items-center rounded-md px-2.5 text-xs transition-colors duration-200 ease-in-out focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500",
          option.value === value
            ? "bg-gray-150 dark:bg-gray-750 font-medium text-gray-900 dark:text-gray-200"
            : "text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 cursor-pointer"
        )}
      >
        {option.label}
      </button>
    ))}
  </div>
);

interface ChartLegendProps {
  items: { label: string; color: string }[];
}

export const ChartLegend = ({ items }: ChartLegendProps) => (
  <div className="flex flex-wrap items-center gap-x-4 gap-y-1 mt-2">
    {items.map((item) => (
      <span key={item.label} className="flex items-center gap-1.5 text-xs text-gray-600 dark:text-gray-400">
        <span
          className="inline-block w-3.5 h-0.5 rounded-full"
          style={{ backgroundColor: item.color }}
          aria-hidden="true"
        />
        {item.label}
      </span>
    ))}
  </div>
);

export const ChartSkeleton = ({ heightClass }: { heightClass: string }) => (
  <div className={classNames("mt-3 rounded animate-pulse bg-gray-100 dark:bg-gray-815", heightClass)} />
);

export const ChartError = ({ heightClass }: { heightClass: string }) => {
  const { t } = useTranslation("common");
  return (
    <div
      className={classNames(
        "flex items-center justify-center mt-3 rounded border border-dashed border-gray-250 dark:border-gray-750 text-sm text-gray-500 dark:text-gray-400",
        heightClass
      )}
    >
      {t("dashboardCharts.loadError")}
    </div>
  );
};
