/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { toast } from "@components/hot-toast";
import { TEST_RESULT_VISIBLE_MS, useTestAction } from "@hooks/useTestAction";

afterEach(() => {
  toast.remove();
  vi.useRealTimers();
});

const wrapper = ({ children }: { children: ReactNode }) => (
  <QueryClientProvider client={new QueryClient()}>{children}</QueryClientProvider>
);

test.each([
  ["passed", () => Promise.resolve()],
  ["failed", () => Promise.reject(new Error("boom"))]
] as const)("a %s result goes back to idle after it has been shown", async (shown, testFn) => {
  vi.useFakeTimers({ shouldAdvanceTime: true });
  const { result } = renderHook(() => useTestAction(testFn, "Test passed"), { wrapper });

  expect(result.current.status).toBe("idle");
  act(() => result.current.run(undefined));
  await waitFor(() => expect(result.current.status).toBe(shown));

  act(() => vi.advanceTimersByTime(TEST_RESULT_VISIBLE_MS - 100));
  expect(result.current.status).toBe(shown);

  act(() => vi.advanceTimersByTime(100));
  expect(result.current.status).toBe("idle");
});
