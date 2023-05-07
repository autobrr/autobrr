/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { newRidgeState } from "react-ridge-state";

export const InitializeGlobalContext = () => {
  const auth_ctx = localStorage.getItem("auth");
  if (auth_ctx)
    AuthContext.set(JSON.parse(auth_ctx));

  const settings_ctx = localStorage.getItem("settings");
  if (settings_ctx) {
    SettingsContext.set(JSON.parse(settings_ctx));
  } else {
    // Only check for light theme, otherwise dark theme is the default
    SettingsContext.set((state) => ({
      ...state,
      darkTheme: !(
        window.matchMedia !== undefined &&
        window.matchMedia("(prefers-color-scheme: light)").matches
      )
    }));
  }
  const filterList_ctx = localStorage.getItem("filterList");
  if (filterList_ctx) {
    FilterListContext.set(JSON.parse(filterList_ctx));
  }
};
interface AuthInfo {
  username: string;
  isLoggedIn: boolean;
}

export const AuthContext = newRidgeState<AuthInfo>(
  {
    username: "",
    isLoggedIn: false
  },
  {
    onSet: (new_state) => {
      try {
        localStorage.setItem("auth", JSON.stringify(new_state));
      } catch (e) {
        console.log("An error occurred while trying to modify the local auth context state.");
        console.log("Error:", e);
      }
    }
  }
);

interface SettingsType {
  debug: boolean;
  checkForUpdates: boolean;
  darkTheme: boolean;
  scrollOnNewLog: boolean;
  indentLogLines: boolean;
  hideWrappedText: boolean;
}

export const SettingsContext = newRidgeState<SettingsType>(
  {
    debug: false,
    checkForUpdates: true,
    darkTheme: true,
    scrollOnNewLog: false,
    indentLogLines: false,
    hideWrappedText: false
  },
  {
    onSet: (new_state) => {
      try {
        if (new_state.darkTheme) {
          document.documentElement.classList.add("dark");
        } else {
          document.documentElement.classList.remove("dark");
        }

        localStorage.setItem("settings", JSON.stringify(new_state));
      } catch (e) {
        console.log("An error occurred while trying to modify the local settings context state.");
        console.log("Error:", e);
      }
    }
  }
);

export type FilterListState = {
  indexerFilter: string[],
  sortOrder: string;
  status: string;
};

export const FilterListContext = newRidgeState<FilterListState>(
  {
    indexerFilter: [],
    sortOrder: "",
    status: ""
  },
  {
    onSet: (new_state) => {
      try {
        localStorage.setItem("filterList", JSON.stringify(new_state));
      } catch (e) {
        console.log("An error occurred while trying to modify the local filter list context state.");
        console.log("Error:", e);
      }
    }
  }
);