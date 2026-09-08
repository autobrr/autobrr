/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { SettingsContext } from "@utils/Context";
import { useMedia } from "@hooks/hooks";

export const useIsDark = (): boolean => {
  const theme = SettingsContext.useSelector((s) => s.theme);
  const prefersDark = useMedia("(prefers-color-scheme: dark)");
  return theme === "dark" || (theme === "system" && prefersDark);
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
