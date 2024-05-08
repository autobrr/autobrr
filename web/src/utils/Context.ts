/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { StateWithValue } from "react-ridge-state";
import { newRidgeState } from "react-ridge-state";

interface SettingsType {
  debug: boolean;
  checkForUpdates: boolean;
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

// interface AuthInfo {
//   username: string;
//   isLoggedIn: boolean;
// }

// Default values
// const AuthContextDefaults: AuthInfo = {
//   username: "",
//   isLoggedIn: false
// };

const SettingsContextDefaults: SettingsType = {
  debug: false,
  checkForUpdates: true,
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

const SettingsKey = "autobrr_settings";
const FilterListKey = "autobrr_filter_list";

export const InitializeGlobalContext = () => {
  // ContextMerger<AuthInfo>(localStorageUserKey, AuthContextDefaults, AuthContextt);
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

// export const AuthContextt = newRidgeState<AuthInfo>(AuthContextDefaults, {
//   onSet: (newState, prevState) => DefaultSetter(localStorageUserKey, newState, prevState)
// });

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
const updateMetaThemeColor = (_darkTheme: boolean) => {
  const root = document.documentElement;
  const color = getComputedStyle(root).getPropertyValue('--rmsc-bg').trim();

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

export type AuthCtx = {
  isLoggedIn: boolean
  username?: string
  login: (username: string) => void
  logout: () => void
}

export const localStorageUserKey = "autobrr_user_auth"

export const AuthContext: AuthCtx = {
  isLoggedIn: false,
  username: undefined,
  login: (username: string) => {
    AuthContext.isLoggedIn = true
    AuthContext.username = username

    localStorage.setItem(localStorageUserKey, JSON.stringify(AuthContext));
  },
  logout: () => {
    AuthContext.isLoggedIn = false
    AuthContext.username = undefined

    localStorage.removeItem(localStorageUserKey);
  },
}