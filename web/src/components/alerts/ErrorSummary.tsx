/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import StackTracey from "stacktracey";

interface ErrorSummaryProps {
  error: Error;
}

export const ErrorSummary = ({ error }: ErrorSummaryProps) => {
  const summary = new StackTracey(error).clean().asTable({
    maxColumnWidths: {
      callee: 48,
      file: 48,
      sourceLine: 384
    }
  });

  if (!summary) {
    return null;
  }

  return (
    <pre className="mt-2 mb-4 text-sm text-red-700 dark:text-red-800 overflow-x-auto">
      {summary}
    </pre>
  );
};
