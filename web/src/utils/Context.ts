/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { StateWithValue } from "react-ridge-state";
import { newRidgeState } from "react-ridge-state";
import { getInitialLanguage } from "@app/i18n";

export type Theme = "light" | "dark" | "system";
export type Language = "en" | "fr" | "de" | "cs" | "no" | "ru" | "es" | "zh-CN";

interface SettingsType {
  debug: boolean;
  theme: Theme;
  language: Language;
  scrollOnNewLog: boolean;
  indentLogLines: boolean;
  hideWrappedText: boolean;
  incognitoMode: boolean;
}

export const isDarkTheme = (theme: Theme): boolean => {
  if (theme === "system") {
    return window.matchMedia("(prefers-color-scheme: dark)").matches;
  }
  return theme === "dark";
};

export type FilterListState = {
  indexerFilter: string[];
  sortOrder: string;
  status: string;
};

export interface DashboardWidgetConfig {
  id: string;
  hidden: boolean;
}

// An empty widgets list means "registry defaults"; the dashboard grid
// reconciles stored entries against the widget registry on render.
export interface DashboardConfigType {
  version: number;
  widgets: DashboardWidgetConfig[];
}

export interface AuthInfo {
  username: string;
  isLoggedIn: boolean;
  authMethod?: 'password' | 'oidc';
  profilePicture?: string;
  issuerUrl?: string;
}

// Default values
const AuthContextDefaults: AuthInfo = {
  username: "",
  isLoggedIn: false,
  authMethod: undefined,
  profilePicture: undefined,
  issuerUrl: undefined
};

const SettingsContextDefaults: SettingsType = {
  debug: false,
  theme: "system",
  language: getInitialLanguage(),
  scrollOnNewLog: false,
  indentLogLines: false,
  hideWrappedText: false,
  incognitoMode: false
};

const FilterListContextDefaults: FilterListState = {
  indexerFilter: [],
  sortOrder: "",
  status: ""
};

const DashboardConfigDefaults: DashboardConfigType = {
  version: 1,
  widgets: []
};

// Reads throw in some private-browsing modes and parse can fail on hand-edited storage.
const readStored = (key: string): Record<string, unknown> | undefined => {
  try {
    const storage = localStorage.getItem(key);
    if (!storage) {
      return undefined;
    }

    const json = JSON.parse(storage);
    if (json === null || typeof json !== "object") {
      console.warn(`JSON localStorage value for '${key}' context state is not an object`);
      return undefined;
    }

    return json;
  } catch (e) {
    console.error(`Failed to read ${key} context state: ${e}`);
    return undefined;
  }
};

// eslint-disable-next-line
function ContextMerger<T extends {}>(
  key: string,
  defaults: T,
  ctxState: StateWithValue<T>,
  stored: Record<string, unknown> | undefined = readStored(key)
) {
  ctxState.set({ ...structuredClone(defaults), ...stored });
}

const AuthKey = "autobrr_user_auth";
const SettingsKey = "autobrr_settings";
const FilterListKey = "autobrr_filter_list";
const DashboardKey = "autobrr_dashboard";

export const InitializeGlobalContext = () => {
  // Migrate old darkTheme boolean to new theme setting
  const storedSettings = readStored(SettingsKey);
  if (storedSettings && "darkTheme" in storedSettings && !("theme" in storedSettings)) {
    storedSettings.theme = storedSettings.darkTheme ? "dark" : "light";
    delete storedSettings.darkTheme;
    try {
      localStorage.setItem(SettingsKey, JSON.stringify(storedSettings));
    } catch {
      // ignore migration errors
    }
  }

  ContextMerger<AuthInfo>(AuthKey, AuthContextDefaults, AuthContext);
  ContextMerger<SettingsType>(
    SettingsKey,
    SettingsContextDefaults,
    SettingsContext,
    storedSettings
  );
  ContextMerger<FilterListState>(
    FilterListKey,
    FilterListContextDefaults,
    FilterListContext
  );
  ContextMerger<DashboardConfigType>(
    DashboardKey,
    DashboardConfigDefaults,
    DashboardConfigContext
  );
};

function DefaultSetter<T>(name: string, newState: T, prevState: T) {
  try {
    localStorage.setItem(name, JSON.stringify(newState));
  } catch (e) {
    console.error(
      `An error occurred while trying to modify '${name}' context state: ${e}`
    );
    console.warn(`  --> prevState: ${prevState}`);
    console.warn(`  --> newState: ${newState}`);
  }
}

export const AuthContext = newRidgeState<AuthInfo>(
  AuthContextDefaults,
  {
    onSet: (newState, prevState) => DefaultSetter(AuthKey, newState, prevState)
  }
);

export const DashboardConfigContext = newRidgeState<DashboardConfigType>(
  DashboardConfigDefaults,
  {
    onSet: (newState, prevState) => DefaultSetter(DashboardKey, newState, prevState)
  }
);

export const SettingsContext = newRidgeState<SettingsType>(
  SettingsContextDefaults,
  {
    onSet: (newState, prevState) => {
      const dark = isDarkTheme(newState.theme);
      document.documentElement.classList.toggle("dark", dark);
      DefaultSetter(SettingsKey, newState, prevState);
      updateMetaThemeColor(dark);
    }
  }
);

/**
 * Updates the meta theme color based on the current theme state.
 * Used by Safari to color the compact tab bar on both iOS and MacOS.
 */
const updateMetaThemeColor = (darkTheme: boolean) => {
  const color = darkTheme ? '#121315' : '#f4f4f5';
  let metaThemeColor: HTMLMetaElement | null = document.querySelector('meta[name="theme-color"]');
  if (!metaThemeColor) {
    metaThemeColor = document.createElement('meta') as HTMLMetaElement;
    metaThemeColor.name = "theme-color";
    document.head.appendChild(metaThemeColor);
  }

  metaThemeColor.content = color;
};

export const FilterListContext = newRidgeState<FilterListState>(
  FilterListContextDefaults,
  {
    onSet: (newState, prevState) => DefaultSetter(FilterListKey, newState, prevState)
  }
);
