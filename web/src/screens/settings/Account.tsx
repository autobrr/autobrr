/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation } from "@tanstack/react-query";
import { APIClient } from "@api/APIClient";
import Toast from "@components/notifications/Toast";
import { Section } from "./_components";
import { Form, Formik } from "formik";
import { PasswordField, TextField } from "@components/inputs";
import { AuthContext } from "@utils/Context";
import toast from "react-hot-toast";
import { UserIcon } from "@heroicons/react/24/solid";

const AccountSettings = () => (
  <Section
    title="Account"
    description="Manage account settings."
  >
    <div className="py-0.5">
      <Credentials />
    </div>
  </Section>
);

interface InputValues {
  username: string;
  newUsername: string;
  oldPassword: string;
  newPassword: string;
  confirmPassword: string;
}

function Credentials() {
  const [ getAuthContext ] = AuthContext.use();


  const validate = (values: InputValues) => {
    const errors: Record<string, string> = {};

    if (!values.username)
      errors.username = "Required";

    if (values.newPassword !== values.confirmPassword)
      errors.confirmPassword = "Passwords don't match!";

    return errors;
  };

  const logoutMutation = useMutation({
    mutationFn: APIClient.auth.logout,
    onSuccess: () => {
      AuthContext.reset();
      toast.custom((t) => (
        <Toast type="success" body="User updated successfully. Please sign in again!" t={t} />
      ));
    }
  });

  const changeCredentialMutation = useMutation({
    mutationFn: (data: InputValues) => APIClient.auth.updateUser(data.username, data.newUsername, data.oldPassword, data.newPassword),
    onSuccess: () => {
      logoutMutation.mutate();
    }
  });

  const containerClass = "px-2 pb-6 bg-white dark:bg-gray-800";
  const buttonClass = "mt-6 w-auto flex items-center py-2 px-4 transition rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500";
  const iconClass = "w-4 h-4 mr-1";

  return (
    <Section
      title="Change credentials"
      description="The username and password can be changed either separately or simultaneously. Note that you will be logged out after changing credentials."
    >
      <div className={containerClass}>
        <Formik
          initialValues={{
            username: getAuthContext.username,
            newUsername: "",
            oldPassword: "",
            newPassword: "",
            confirmPassword: ""
          }}
          onSubmit={(data) => {
            changeCredentialMutation.mutate(data);
          }}
          validate={validate}
        >
          {({ values }) => (
            <Form>
              <div className="grid grid-cols-1 gap-5">
                <TextField name="username" label="Current Username" columns={6} autoComplete="username" disabled />
                <PasswordField name="oldPassword" placeholder="Required" label="Current Password" columns={6} autoComplete="current-password" required tooltip={
                  <div>
                    <p>Required if updating credentials</p>
                  </div>
                } />
                <TextField name="newUsername" label="New Username" columns={6} tooltip={
                  <div>
                    <p>Optional</p>
                  </div>
                } />
                <PasswordField name="newPassword" label="New Password" columns={6} autoComplete="new-password" tooltip={
                  <div>
                    <p>Optional</p>
                  </div>
                } />
                {values.newPassword && (
                  <PasswordField name="confirmPassword" label="Confirm New Password" columns={6} autoComplete="new-password" />
                )}
              </div>
              <div className="flex justify-end">
                <button
                  type="submit"
                  className={buttonClass}
                >
                  <UserIcon className={iconClass} />
                  Update Credentials
                </button>
              </div>
            </Form>
          )}
        </Formik>
      </div>
    </Section>
  );
}

export default AccountSettings;
