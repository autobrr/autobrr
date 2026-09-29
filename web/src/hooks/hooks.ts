/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useCallback, useState, useSyncExternalStore } from "react";

export function useToggle(initialValue = false): [boolean, () => void] {
  const [value, setValue] = useState(initialValue);
  const toggle = useCallback(() => setValue((v) => !v), []);

  return [value, toggle];
}

// One MediaQueryList per query, shared by every subscriber, instead of one per hook instance.
const mediaQueries = new Map<string, MediaQueryList>();

const getMediaQuery = (query: string) => {
  let mql = mediaQueries.get(query);
  if (!mql) {
    mql = window.matchMedia(query);
    mediaQueries.set(query, mql);
  }

  return mql;
};

export const useMedia = (query: string, defaultState = false) => {
  const subscribe = useCallback((onChange: () => void) => {
    const mql = getMediaQuery(query);
    mql.addEventListener("change", onChange);
    return () => mql.removeEventListener("change", onChange);
  }, [query]);

  return useSyncExternalStore(subscribe, () => getMediaQuery(query).matches, () => defaultState);
};
