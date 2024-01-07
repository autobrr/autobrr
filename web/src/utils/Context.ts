/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { newRidgeState } from "react-ridge-state";
import type { StateWithValue } from "react-ridge-state";

interface AuthInfo {
  username: string;
  isLoggedIn: boolean;
}

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

// Default values
const AuthContextDefaults: AuthInfo = {
  username: "",
  isLoggedIn: false
};

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

export const InitializeGlobalContext = () => {
  ContextMerger<AuthInfo>("auth", AuthContextDefaults, AuthContext);
  ContextMerger<SettingsType>(
    "settings",
    SettingsContextDefaults,
    SettingsContext
  );
  ContextMerger<FilterListState>(
    "filterList",
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

export const AuthContext = newRidgeState<AuthInfo>(AuthContextDefaults, {
  onSet: (newState, prevState) => DefaultSetter("auth", newState, prevState)
});

export const SettingsContext = newRidgeState<SettingsType>(
  SettingsContextDefaults,
  {
    onSet: (newState, prevState) => {
      document.documentElement.classList.toggle("dark", newState.darkTheme);
      DefaultSetter("settings", newState, prevState);
    }
  }
);

export const FilterListContext = newRidgeState<FilterListState>(
  FilterListContextDefaults,
  {
    onSet: (newState, prevState) => DefaultSetter("filterList", newState, prevState)
  }
);
