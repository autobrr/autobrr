/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { useMedia } from "@hooks/hooks";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const Probe = ({ id, query }: { id: string; query: string }) => {
  const matches = useMedia(query);
  return <span data-testid={id}>{matches ? "narrow" : "wide"}</span>;
};

test("consumers of one query share a single MediaQueryList and update together", () => {
  const listeners = new Set<() => void>();
  const mql = {
    matches: false,
    media: "",
    onchange: null,
    addEventListener: (_: string, cb: () => void) => listeners.add(cb),
    removeEventListener: (_: string, cb: () => void) => listeners.delete(cb),
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false
  } as unknown as MediaQueryList;
  const matchMedia = vi.spyOn(window, "matchMedia").mockReturnValue(mql);

  const query = "(max-width: 123px)";
  render(
    <>
      <Probe id="a" query={query} />
      <Probe id="b" query={query} />
    </>
  );

  expect(matchMedia).toHaveBeenCalledTimes(1);
  expect(screen.getByTestId("a").textContent).toBe("wide");
  expect(screen.getByTestId("b").textContent).toBe("wide");

  act(() => {
    (mql as { matches: boolean }).matches = true;
    listeners.forEach((cb) => cb());
  });

  expect(screen.getByTestId("a").textContent).toBe("narrow");
  expect(screen.getByTestId("b").textContent).toBe("narrow");
});
