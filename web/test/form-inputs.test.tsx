/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { ReactNode } from "react";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { useAppForm, fieldErrors } from "@hooks/form";
import { NumberField, RegexField, SwitchGroup, TextField } from "@components/inputs";
import { TextFieldWide } from "@components/inputs/input_wide";
import { SlideOver } from "@components/panels";
import "@app/i18n";

afterEach(() => {
  cleanup();
});

interface HarnessProps<T> {
  defaultValues: T;
  onSubmit: (value: T) => void;
  validate?: (value: T) => Record<string, string>;
  children: ReactNode;
}

function Harness<T>({ defaultValues, onSubmit, validate, children }: HarnessProps<T>) {
  const form = useAppForm({
    defaultValues,
    validators: {
      onChange: ({ value }) => validate ? fieldErrors(validate(value)) : undefined
    },
    onSubmit: ({ value }) => onSubmit(value)
  });

  return (
    <form.AppForm>
      <form onSubmit={(e) => { e.preventDefault(); form.handleSubmit(); }}>
        {children}
        <button type="submit">submit</button>
      </form>
    </form.AppForm>
  );
}

const submit = async () => {
  await act(async () => {
    fireEvent.submit(screen.getByText("submit"));
  });
};

test("TextField writes through to the form values", async () => {
  const onSubmit = vi.fn();
  render(
    <Harness defaultValues={{ name: "" }} onSubmit={onSubmit}>
      <TextField name="name" label="Name" />
    </Harness>
  );

  fireEvent.change(screen.getByLabelText("Name"), { target: { value: "My list" } });
  await submit();

  expect(onSubmit).toHaveBeenCalledWith({ name: "My list" });
});

test("TextField shows a form-level error once touched and blocks submit", async () => {
  const onSubmit = vi.fn();
  render(
    <Harness
      defaultValues={{ name: "" }}
      onSubmit={onSubmit}
      validate={(value): Record<string, string> => value.name ? {} : { name: "Required" }}
    >
      <TextField name="name" label="Name" />
    </Harness>
  );

  expect(screen.queryByText("Required")).toBeNull();
  await submit();
  expect(onSubmit).not.toHaveBeenCalled();
  expect(screen.getByText("Required")).toBeTruthy();

  fireEvent.change(screen.getByLabelText("Name"), { target: { value: "x" } });
  expect(screen.queryByText("Required")).toBeNull();
});

test("TextFieldWide runs its field-level validate prop", async () => {
  const onSubmit = vi.fn();
  render(
    <Harness defaultValues={{ settings: { apikey: "" } }} onSubmit={onSubmit}>
      <TextFieldWide name="settings.apikey" label="API key" validate={(value) => value ? undefined : "Key required"} />
    </Harness>
  );

  await submit();
  expect(onSubmit).not.toHaveBeenCalled();
  expect(screen.getByText("Key required")).toBeTruthy();

  fireEvent.change(screen.getByLabelText("API key"), { target: { value: "abc" } });
  await submit();
  expect(onSubmit).toHaveBeenCalledWith({ settings: { apikey: "abc" } });
});

test("SwitchGroup toggles a boolean", async () => {
  const onSubmit = vi.fn();
  render(
    <Harness defaultValues={{ enabled: false }} onSubmit={onSubmit}>
      <SwitchGroup name="enabled" label="Enabled" />
    </Harness>
  );

  fireEvent.click(screen.getByRole("switch"));
  await submit();

  expect(onSubmit).toHaveBeenCalledWith({ enabled: true });
});

test("NumberField stores numbers and falls back to 0 when cleared", async () => {
  const onSubmit = vi.fn();
  render(
    <Harness defaultValues={{ delay: 5 }} onSubmit={onSubmit}>
      <NumberField name="delay" label="Delay" />
    </Harness>
  );

  const input = screen.getByLabelText("Delay");
  fireEvent.change(input, { target: { value: "12" } });
  await submit();
  expect(onSubmit).toHaveBeenLastCalledWith({ delay: 12 });

  fireEvent.change(input, { target: { value: "" } });
  await submit();
  expect(onSubmit).toHaveBeenLastCalledWith({ delay: 0 });
});

test("RegexField blocks submit on an invalid pattern only while regex mode is on", async () => {
  const onSubmit = vi.fn();
  const { rerender } = render(
    <Harness defaultValues={{ shows: "" }} onSubmit={onSubmit}>
      <RegexField name="shows" label="Shows" useRegex />
    </Harness>
  );

  fireEvent.change(screen.getByLabelText("Shows"), { target: { value: "(unclosed" } });
  await submit();
  expect(onSubmit).not.toHaveBeenCalled();

  rerender(
    <Harness defaultValues={{ shows: "" }} onSubmit={onSubmit}>
      <RegexField name="shows" label="Shows" useRegex={false} />
    </Harness>
  );
  await submit();
  expect(onSubmit).toHaveBeenCalledWith({ shows: "(unclosed" });
});

test("SlideOver validates with the flat error map and submits typed values", async () => {
  const onSubmit = vi.fn();
  render(
    <SlideOver<{ name: string }>
      title="Proxy"
      type="CREATE"
      isOpen={true}
      toggle={() => {}}
      initialValues={{ name: "" }}
      validate={(values): Record<string, string> => values.name ? {} : { name: "Name required" }}
      onSubmit={onSubmit}
    >
      {() => <TextFieldWide name="name" label="Name" />}
    </SlideOver>
  );

  await act(async () => {
    fireEvent.click(screen.getByText("Create"));
  });
  expect(onSubmit).not.toHaveBeenCalled();
  expect(screen.getByText("Name required")).toBeTruthy();

  fireEvent.change(screen.getByLabelText("Name"), { target: { value: "socks" } });
  await act(async () => {
    fireEvent.click(screen.getByText("Create"));
  });
  expect(onSubmit).toHaveBeenCalledWith({ name: "socks" });
});
