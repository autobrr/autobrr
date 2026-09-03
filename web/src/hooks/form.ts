/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { createFormHook, createFormHookContexts, useSelector } from "@tanstack/react-form";
import type { AnyFieldMeta, AnyFormApi, ReactFormExtendedApi, UpdateMetaOptions } from "@tanstack/react-form";

// The array helpers are keyed by DeepKeysOfType, which collapses to never for an any-typed form.
interface ArrayFieldHelpers {
  pushFieldValue: (field: string, value: unknown, opts?: UpdateMetaOptions) => void;
  insertFieldValue: (field: string, index: number, value: unknown, opts?: UpdateMetaOptions) => Promise<void>;
  replaceFieldValue: (field: string, index: number, value: unknown, opts?: UpdateMetaOptions) => Promise<void>;
  removeFieldValue: (field: string, index: number, opts?: UpdateMetaOptions) => Promise<void>;
  swapFieldValues: (field: string, index1: number, index2: number, opts?: UpdateMetaOptions) => void;
  moveFieldValues: (field: string, index1: number, index2: number, opts?: UpdateMetaOptions) => void;
  clearFieldValues: (field: string, opts?: UpdateMetaOptions) => void;
}

export type AnyReactFormApi =
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  Omit<ReactFormExtendedApi<any, any, any, any, any, any, any, any, any, any, any, any>, keyof ArrayFieldHelpers>
  & ArrayFieldHelpers;

const contexts = createFormHookContexts();

export const { fieldContext, formContext } = contexts;

export const { useAppForm } = createFormHook({
  fieldContext,
  formContext,
  fieldComponents: {},
  formComponents: {}
});

// The library types the context form for an empty record, which rejects every
// dynamic name; the house inputs address fields by their snake_case wire name.
export const useFormContext = () => contexts.useFormContext() as unknown as AnyReactFormApi;

export const useFormValues = <T>() => {
  const form = useFormContext();

  return useSelector(form.store, (state) => state.values as T);
};

export type FormFieldErrors = Record<string, string>;

// Adapts a validate function returning { fieldName: message } to a form-level validator.
export const fieldErrors = (errors: FormFieldErrors) => {
  if (Object.keys(errors).length === 0) {
    return undefined;
  }

  return { fields: errors };
};

// Standard Schema issues arrive as objects, hand-rolled validators as strings.
export const errorMessage = (error: unknown) => {
  if (typeof error === "string") {
    return error;
  }
  if (error && typeof error === "object" && "message" in error) {
    return String((error as { message: unknown }).message);
  }

  return undefined;
};

export const errorMessages = (errors: unknown[]) => errors.map(errorMessage).filter((message): message is string => !!message);

export const fieldHasError = (meta: AnyFieldMeta) => meta.isTouched && meta.errors.length > 0;

// Submit only touches mounted fields; schema errors on another tab must show once that tab mounts.
export const touchInvalidFields = (form: AnyFormApi) => {
  for (const [field, meta] of Object.entries(form.state.fieldMeta)) {
    if (!meta || meta.isTouched || meta.errors.length === 0) {
      continue;
    }
    form.setFieldMeta(field, (prev) => ({ ...prev, isTouched: true }));
  }
};
