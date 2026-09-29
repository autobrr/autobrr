/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test } from "vitest";

import { useAppForm } from "@hooks/form";
import { External } from "@screens/filters/sections/External";
import "@app/i18n";

afterEach(() => {
  cleanup();
});

const external = (id: number, name: string, index: number): ExternalFilter => ({
  id,
  index,
  name,
  enabled: true,
  type: "EXEC",
  on_error: "REJECT"
});

let submitted: ExternalFilter[] = [];

function Harness() {
  const form = useAppForm({
    defaultValues: {
      external: [external(1, "Alpha Entry", 2), external(2, "Beta Entry", 1), external(3, "Gamma Entry", 0)]
    } as Filter,
    onSubmit: ({ value }) => {
      submitted = value.external;
    }
  });

  return (
    <QueryClientProvider client={new QueryClient()}>
      <form.AppForm>
        <form onSubmit={(e) => { e.preventDefault(); form.handleSubmit(); }}>
          <External />
          <button type="submit">submit</button>
        </form>
      </form.AppForm>
    </QueryClientProvider>
  );
}

// Rows above the first have their move-up arrow as the first arrow button
const moveUp = (name: string) => {
  const row = screen.getByText(name).closest("li") as HTMLElement;
  fireEvent.click(row.querySelector(".flex.flex-col.pr-3 button") as HTMLButtonElement);
};

const rowNames = () => Array.from(document.querySelectorAll("li p.font-medium")).map((p) => p.textContent);

test("moving a row to the top persists the displayed order", async () => {
  render(<Harness />);

  moveUp("Gamma Entry");
  moveUp("Gamma Entry");
  expect(rowNames()).toEqual(["Gamma Entry", "Alpha Entry", "Beta Entry"]);

  await act(async () => {
    fireEvent.submit(screen.getByText("submit"));
  });
  expect(submitted.map((item) => [item.name, item.index])).toEqual([
    ["Gamma Entry", 2],
    ["Alpha Entry", 1],
    ["Beta Entry", 0]
  ]);
});

test("a new row keeps the existing order and is appended last", async () => {
  render(<Harness />);

  fireEvent.click(screen.getByText("Add new"));
  await act(async () => {
    fireEvent.submit(screen.getByText("submit"));
  });
  expect(submitted.map((item) => [item.name, item.index])).toEqual([
    ["Alpha Entry", 3],
    ["Beta Entry", 2],
    ["Gamma Entry", 1],
    ["External 4", 0]
  ]);
});
