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
import { KeyIcon } from "@heroicons/react/24/outline";
import { AuthContext } from "@utils/Context";
import toast from "react-hot-toast";

const ProfileSettings = () => (
  <Section
    title="Profile"
    description="Manage profile."
  >
    <div className="py-6 px-4 sm:p-6">
      <UserProfile />
    </div>
  </Section>
);

interface InputValues {
  username: string;
  oldPassword: string;
  newPassword: string;
  confirmPassword: string;
}

function UserProfile() {
  const [ getAuthContext, _] = AuthContext.use();


  const validate = (values: InputValues) => {
    const errors: Record<string, string> = {};

    if (!values.username)
      errors.username = "Required";

    if (!values.oldPassword)
      errors.oldPassword = "Required";

    if (!values.newPassword)
      errors.newPassword = "Required";

    if (values.newPassword !== values.confirmPassword)
      errors.confirmPassword = "Passwords don't match!";

    return errors;
  };

  const logoutMutation = useMutation({
    mutationFn: APIClient.auth.logout,
    onSuccess: () => {
      AuthContext.reset();
      toast.custom((t) => (
        <Toast type="success" body="Your password has been updated successfully. Please sign in again with your new password!" t={t} />
      ));
    }
  });

  const changePasswordMutation = useMutation({
    mutationFn: (data: InputValues) => APIClient.auth.changePassword(data.username, data.oldPassword, data.newPassword),
    onSuccess: () => {
      logoutMutation.mutate();
    }
  });

  return (

    <div className="mx-auto w-full rounded-2xl shadow-lg">
      <div className="px-8 pt-8 pb-6 rounded-2xl bg-white dark:bg-gray-800 border border-gray-150 dark:border-gray-775">
        <Formik
          initialValues={{
            username: getAuthContext.username,
            oldPassword: "",
            newPassword: "",
            confirmPassword: ""
          }}
          onSubmit={(data) => {
            changePasswordMutation.mutate(data);
          }}
          validate={validate}
        >
          <Form>
            <div className="grid grid-cols-1 gap-5">
              <TextField name="username" label="Username" columns={6} autoComplete="username" disabled  />
              <PasswordField name="oldPassword" label="Current Password" columns={6} autoComplete="current-password" required />
              <PasswordField name="newPassword" label="New Password" columns={6} autoComplete="current-password" required />
              <PasswordField name="confirmPassword" label="Confirm password" columns={6} autoComplete="current-password" required  />
            </div>
            <button
              type="submit"
              className="mt-6 w-full flex items-center justify-center py-2 px-4 border border-transparent transition rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
            >
              <KeyIcon className="w-4 h-4 mr-1.5" />
                Change Password
            </button>
          </Form>
        </Formik>
      </div>
    </div>

  );
}

export default ProfileSettings;
