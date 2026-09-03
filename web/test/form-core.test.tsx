/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { z } from "zod";

import { useSelector } from "@tanstack/react-form";

import { useAppForm, useFormContext, useFormValues, fieldErrors, errorMessages, touchInvalidFields } from "@hooks/form";

afterEach(() => {
  cleanup();
});

const actionSchema = z.object({
  type: z.string(),
  client_id: z.number().optional()
}).superRefine((value, ctx) => {
  if (value.type === "QBITTORRENT" && !value.client_id) {
    ctx.addIssue({ message: "Must select client", code: "custom", path: ["client_id"] });
  }
});

const schema = z.object({
  name: z.string(),
  indexers: z.array(z.object({ id: z.number() })).min(1, { message: "Must select at least one indexer" }),
  actions: z.array(actionSchema)
});

type Values = z.infer<typeof schema>;

const defaults: Values = {
  name: "",
  indexers: [],
  actions: [{ type: "QBITTORRENT", client_id: 0 }]
};

// Mirrors the filter editor: the tab that is mounted only renders a subset of
// the fields the schema validates.
function SchemaForm({ onSubmit, onSubmitInvalid, showClient }: {
  onSubmit: (value: Values) => void;
  onSubmitInvalid: (errors: string[]) => void;
  showClient?: boolean;
}) {
  const form = useAppForm({
    defaultValues: defaults,
    validators: { onChange: schema },
    onSubmit: ({ value }) => onSubmit(value),
    onSubmitInvalid: ({ formApi }) => {
      touchInvalidFields(formApi);
      const messages: string[] = [];
      for (const [field, meta] of Object.entries(formApi.state.fieldMeta)) {
        for (const message of errorMessages(meta?.errors ?? [])) {
          messages.push(`${field}: ${message}`);
        }
      }
      onSubmitInvalid(messages);
    }
  });

  return (
    <form.AppForm>
      <form onSubmit={(e) => { e.preventDefault(); form.handleSubmit(); }}>
        <form.Field name="name">
          {(field) => (
            <input aria-label="name" value={field.state.value} onChange={(e) => field.handleChange(e.target.value)} />
          )}
        </form.Field>
        {showClient && <ClientField />}
        <button type="submit">save</button>
        <button type="button" onClick={() => form.reset({ ...defaults, name: "reset" })}>reset</button>
        <Dirty />
      </form>
    </form.AppForm>
  );
}

function ClientField() {
  const form = useFormContext();

  return (
    <form.Field name="actions[0].client_id">
      {(field) => (
        <div>
          <input aria-label="client" type="number" value={field.state.value} onChange={(e) => field.handleChange(parseInt(e.target.value))} />
          {field.state.meta.isTouched && field.state.meta.errors.length > 0 && <span role="alert">{errorMessages(field.state.meta.errors).join(",")}</span>}
        </div>
      )}
    </form.Field>
  );
}

function Dirty() {
  const values = useFormValues<Values>();
  const form = useFormContext();
  const isDirty = useSelector(form.store, (state) => state.isDirty);

  return <output aria-label="dirty">{isDirty ? "dirty" : "clean"}:{values.name}</output>;
}

test("blocks submit on schema issues for fields that are not mounted", async () => {
  const onSubmit = vi.fn();
  const onSubmitInvalid = vi.fn();
  render(<SchemaForm onSubmit={onSubmit} onSubmitInvalid={onSubmitInvalid} />);

  fireEvent.change(screen.getByLabelText("name"), { target: { value: "My filter" } });
  await act(async () => {
    fireEvent.submit(screen.getByText("save"));
  });

  expect(onSubmit).not.toHaveBeenCalled();
  expect(onSubmitInvalid).toHaveBeenCalledTimes(1);
  const messages = onSubmitInvalid.mock.calls[0][0] as string[];
  expect(messages).toContain("indexers: Must select at least one indexer");
  expect(messages).toContain("actions[0].client_id: Must select client");
});

test("delivers nested schema issues to a mounted bracket-named field", async () => {
  const onSubmit = vi.fn();
  const onSubmitInvalid = vi.fn();
  render(<SchemaForm onSubmit={onSubmit} onSubmitInvalid={onSubmitInvalid} showClient />);

  await act(async () => {
    fireEvent.submit(screen.getByText("save"));
  });
  expect(screen.getByRole("alert").textContent).toBe("Must select client");

  fireEvent.change(screen.getByLabelText("client"), { target: { value: "3" } });
  expect(screen.queryByRole("alert")).toBeNull();
});

test("shows a schema error inline when the field mounts after a failed submit", async () => {
  const { rerender } = render(<SchemaForm onSubmit={vi.fn()} onSubmitInvalid={vi.fn()} />);

  await act(async () => {
    fireEvent.submit(screen.getByText("save"));
  });
  expect(screen.queryByRole("alert")).toBeNull();

  rerender(<SchemaForm onSubmit={vi.fn()} onSubmitInvalid={vi.fn()} showClient />);
  expect(screen.getByRole("alert").textContent).toBe("Must select client");
});

test("submits once every schema issue is resolved", async () => {
  const onSubmit = vi.fn();
  const onSubmitInvalid = vi.fn();
  render(<SchemaForm onSubmit={onSubmit} onSubmitInvalid={onSubmitInvalid} showClient />);

  await act(async () => {
    fireEvent.submit(screen.getByText("save"));
  });
  expect(onSubmit).not.toHaveBeenCalled();

  fireEvent.change(screen.getByLabelText("client"), { target: { value: "3" } });
  await act(async () => {
    fireEvent.submit(screen.getByText("save"));
  });
  expect(onSubmit).not.toHaveBeenCalled();
  expect(onSubmitInvalid).toHaveBeenCalledTimes(2);
});

test("reset replaces values and clears dirty state", () => {
  render(<SchemaForm onSubmit={vi.fn()} onSubmitInvalid={vi.fn()} />);

  fireEvent.change(screen.getByLabelText("name"), { target: { value: "typed" } });
  expect(screen.getByLabelText("dirty").textContent).toBe("dirty:typed");

  fireEvent.click(screen.getByText("reset"));
  expect(screen.getByLabelText("dirty").textContent).toBe("clean:reset");
});

test("fieldErrors adapts a hand-rolled validate map", () => {
  expect(fieldErrors({})).toBeUndefined();
  expect(fieldErrors({ name: "Required", "auth.account": "Required" })).toEqual({
    fields: { name: "Required", "auth.account": "Required" }
  });
});

function ValidateForm({ onSubmit }: { onSubmit: (value: { name: string; auth: { account: string } }) => void }) {
  const form = useAppForm({
    defaultValues: { name: "", auth: { account: "" } },
    validators: {
      onChange: ({ value }) => {
        const errors: Record<string, string> = {};
        if (!value.name) {
          errors.name = "Required";
        }
        if (!value.auth.account) {
          errors["auth.account"] = "Account required";
        }

        return fieldErrors(errors);
      }
    },
    onSubmit: ({ value }) => onSubmit(value)
  });

  return (
    <form.AppForm>
      <form onSubmit={(e) => { e.preventDefault(); form.handleSubmit(); }}>
        <form.Field name="name">
          {(field) => (
            <div>
              <input aria-label="name" value={field.state.value} onChange={(e) => field.handleChange(e.target.value)} onBlur={field.handleBlur} />
              {field.state.meta.isTouched && field.state.meta.errors.length > 0 && <span data-testid="name-error">{errorMessages(field.state.meta.errors)[0]}</span>}
            </div>
          )}
        </form.Field>
        <form.Field name="auth.account">
          {(field) => (
            <div>
              <input aria-label="account" value={field.state.value} onChange={(e) => field.handleChange(e.target.value)} onBlur={field.handleBlur} />
              {field.state.meta.isTouched && field.state.meta.errors.length > 0 && <span data-testid="account-error">{errorMessages(field.state.meta.errors)[0]}</span>}
            </div>
          )}
        </form.Field>
        <button type="submit">save</button>
      </form>
    </form.AppForm>
  );
}

test("hand-rolled validate surfaces dotted nested errors after submit and clears them on fix", async () => {
  const onSubmit = vi.fn();
  render(<ValidateForm onSubmit={onSubmit} />);

  expect(screen.queryByTestId("name-error")).toBeNull();
  await act(async () => {
    fireEvent.submit(screen.getByText("save"));
  });
  expect(onSubmit).not.toHaveBeenCalled();
  expect(screen.getByTestId("name-error").textContent).toBe("Required");
  expect(screen.getByTestId("account-error").textContent).toBe("Account required");

  fireEvent.change(screen.getByLabelText("name"), { target: { value: "net" } });
  fireEvent.change(screen.getByLabelText("account"), { target: { value: "acc" } });
  expect(screen.queryByTestId("name-error")).toBeNull();
  expect(screen.queryByTestId("account-error")).toBeNull();

  await act(async () => {
    fireEvent.submit(screen.getByText("save"));
  });
  expect(onSubmit).toHaveBeenCalledWith({ name: "net", auth: { account: "acc" } });
});
