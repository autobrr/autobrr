/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Form, Formik } from "formik";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";

import { APIClient } from "@api/APIClient";
import { TextField, PasswordField } from "@components/inputs";

import { UserPlusIcon } from "@heroicons/react/24/outline";

import Logo from "@app/logo.svg?react";

interface InputValues {
  username: string;
  password1: string;
  password2: string;
}

export const Onboarding = () => {
  const validate = (values: InputValues) => {
    const obj: Record<string, string> = {};

    if (!values.username)
      obj.username = "Required";

    if (!values.password1)
      obj.password1 = "Required";

    if (!values.password2)
      obj.password2 = "Required";

    if (values.password1 !== values.password2)
      obj.password2 = "Passwords don't match!";

    return obj;
  };

  const navigate = useNavigate();

  // Query to check if OIDC is enabled
  const { data: oidcConfig } = useQuery({
    queryKey: ["oidc-config"],
    queryFn: () => APIClient.auth.getOIDCConfig(),
  });

  const mutation = useMutation({
    mutationFn: (data: InputValues) => APIClient.auth.onboard(data.username, data.password1),
    onSuccess: () => navigate({ to: "/login" })
  });

  // If OIDC is enabled, redirect to login
  if (oidcConfig?.enabled) {
    navigate({ to: "/login" });
    return null;
  }

  return (
    <div className="min-h-screen flex flex-col justify-center py-12 sm:px-6 lg:px-8">
      <div className="sm:mx-auto sm:w-full sm:max-w-md mb-6">
        <Logo className="mx-auto h-12" />
        <h1 className="text-center text-gray-900 dark:text-gray-200 font-bold pt-2 text-2xl">
          autobrr
        </h1>
      </div>
      <div className="mx-auto w-full max-w-md rounded-2xl shadow-lg">
        <div className="px-8 pt-8 pb-6 rounded-2xl bg-white dark:bg-gray-800 border border-gray-150 dark:border-gray-775">
          <Formik
            initialValues={{
              username: "",
              password1: "",
              password2: ""
            }}
            onSubmit={(data) => mutation.mutate(data)}
            validate={validate}
          >
            <Form>
              <div className="space-y-6">
                <TextField name="username" label="Username" columns={6} autoComplete="username" />
                <PasswordField name="password1" label="Password" columns={6} autoComplete="current-password" />
                <PasswordField name="password2" label="Confirm password" columns={6} autoComplete="current-password" />
              </div>
              <button
                type="submit"
                className="mt-6 w-full flex items-center justify-center py-2 px-4 border border-transparent transition rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
              >
                <UserPlusIcon className="w-4 h-4 mr-1.5" />
                Create account
              </button>
            </Form>
          </Formik>
        </div>
      </div>
    </div>
  );
};
