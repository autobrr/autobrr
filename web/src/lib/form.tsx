/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { ReactNode } from "react";
import { createFormHookContexts, createFormHook } from "@tanstack/react-form";
export { useStore } from "@tanstack/react-store";

export const { fieldContext, formContext, useFieldContext, useFormContext } =
  createFormHookContexts();

export const { useAppForm, withForm } = createFormHook({
  fieldContext,
  formContext,
  fieldComponents: {},
  formComponents: {},
});

/**
 * ContextField - A field wrapper for sub-components that need to render form
 * fields using the tanstack input components (which use useFieldContext).
 *
 * Use this inside components wrapped by <form.AppForm> where you don't have
 * direct access to the form instance's AppField.
 *
 * Usage:
 *   <ContextField name="host">
 *     <TextFieldWide label="Host" />
 *   </ContextField>
 */
export function ContextField({ name, children, validators, ...rest }: {
  name: string;
  children: ReactNode;
  validators?: Record<string, unknown>;
  [key: string]: unknown;
}) {
  const form = useFormContext();
  return (
    // @ts-expect-error - dynamic field name typing
    <form.Field name={name} validators={validators} {...rest}>
      {(field: any) => (
        <fieldContext.Provider value={field}>
          {children}
        </fieldContext.Provider>
      )}
    </form.Field>
  );
}
