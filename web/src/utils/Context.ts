/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { StateWithValue } from "react-ridge-state";
import { newRidgeState } from "react-ridge-state";

interface SettingsType {
  debug: boolean;
  darkTheme: boolean;
  scrollOnNewLog: boolean;
  indentLogLines: boolean;
  hideWrappedText: boolean;
}

export type FilterListState = {
  indexerFilter: string[];
  sortOrder: string;
  status: string;
};

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
  darkTheme: window.matchMedia('(prefers-color-scheme: dark)').matches,
  scrollOnNewLog: false,
  indentLogLines: false,
  hideWrappedText: false
};

const FilterListContextDefaults: FilterListState = {
  indexerFilter: [],
  sortOrder: "",
  status: ""
};

// eslint-disable-next-line
function ContextMerger<T extends {}>(
  key: string,
  defaults: T,
  ctxState: StateWithValue<T>
) {
  let values = structuredClone(defaults);

  const storage = localStorage.getItem(key);
  if (storage) {
    try {
      const json = JSON.parse(storage);
      if (json === null) {
        console.warn(`JSON localStorage value for '${key}' context state is null`);
      } else {
        values = { ...values, ...json };
      }
    } catch (e) {
      console.error(`Failed to merge ${key} context state: ${e}`);
    }
  }

  ctxState.set(values);
}

const AuthKey = "autobrr_user_auth";
const SettingsKey = "autobrr_settings";
const FilterListKey = "autobrr_filter_list";

export const InitializeGlobalContext = () => {
  ContextMerger<AuthInfo>(AuthKey, AuthContextDefaults, AuthContext);
  ContextMerger<SettingsType>(
    SettingsKey,
    SettingsContextDefaults,
    SettingsContext
  );
  ContextMerger<FilterListState>(
    FilterListKey,
    FilterListContextDefaults,
    FilterListContext
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

export const SettingsContext = newRidgeState<SettingsType>(
  SettingsContextDefaults,
  {
    onSet: (newState, prevState) => {
      document.documentElement.classList.toggle("dark", newState.darkTheme);
      DefaultSetter(SettingsKey, newState, prevState);
      updateMetaThemeColor(newState.darkTheme);
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
