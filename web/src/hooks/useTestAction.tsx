/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect } from "react";
import { useMutation } from "@tanstack/react-query";

import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";

export type TestStatus = "idle" | "testing" | "passed" | "failed";

export interface TestAction<T> {
  run: (data: T) => void;
  status: TestStatus;
}

export const TEST_RESULT_VISIBLE_MS = 2500;

const testStatusByMutationStatus = {
  idle: "idle",
  pending: "testing",
  success: "passed",
  error: "failed"
} as const satisfies Record<string, TestStatus>;

export function useTestAction<T>(testFn: (data: T) => Promise<unknown>, successMessage: string): TestAction<T> {
  const { mutate, status, reset } = useMutation({ mutationFn: testFn });

  useEffect(() => {
    if (status !== "success" && status !== "error") {
      return;
    }

    const timer = setTimeout(reset, TEST_RESULT_VISIBLE_MS);
    return () => clearTimeout(timer);
  }, [status, reset]);

  return {
    run: (data) => mutate(data, {
      onSuccess: () => {
        toast.custom((toastInstance) => <Toast type="success" body={successMessage} t={toastInstance} />);
      }
    }),
    status: testStatusByMutationStatus[status]
  };
}
