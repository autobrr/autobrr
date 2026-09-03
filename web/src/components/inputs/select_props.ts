/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { CSSObjectWithLabel, Theme } from "react-select";

import { DropdownIndicator, IndicatorSeparator, SelectControl, SelectInput, SelectMenu, SelectOption } from "@components/inputs/common";
// Shared react-select props. Passing fresh objects per render invalidates react-select's
// internal component and style caches on every keystroke.
export const selectComponents = {
  Input: SelectInput,
  Control: SelectControl,
  Menu: SelectMenu,
  Option: SelectOption,
  IndicatorSeparator,
  DropdownIndicator
};

export const selectStyles = {
  singleValue: (base: CSSObjectWithLabel) => ({ ...base, color: "unset" })
};

export const selectTheme = (theme: Theme): Theme => ({
  ...theme,
  spacing: { ...theme.spacing, controlHeight: 30, baseUnit: 2 }
});
