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
  darkTheme: true,
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
  let values = defaults;

  const storage = localStorage.getItem(key);
  if (storage) {
    try {
      const json = JSON.parse(storage);
      if (json === null) {
        console.warn(`JSON localStorage value for '${key}' context state is null`);
      } else {
        values = { ...defaults, ...json };
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
  // ContextMerger<AuthInfo>("auth", AuthContextDefaults, AuthContext);
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

// export const AuthContext = newRidgeState<AuthInfo>(AuthContextDefaults, {
//   onSet: (newState, prevState) => DefaultSetter("auth", newState, prevState)
// });

export const SettingsContext = newRidgeState<SettingsType>(
  SettingsContextDefaults,
  {
    onSet: (newState, prevState) => {
      document.documentElement.classList.toggle("dark", newState.darkTheme);
      DefaultSetter(SettingsKey, newState, prevState);
    }
  }
);

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