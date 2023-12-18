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
import { KeyIcon, UserIcon } from "@heroicons/react/24/outline";
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

interface ChangePasswordValues {
  username: string;
  oldPassword: string;
  newPassword: string;
  confirmPassword: string;
}

interface ChangeUsernameValues {
  username: string;
  newUsername: string;
}


function UserProfile() {
  const [getAuthContext] = AuthContext.use();


  const validateChangePassword = (values: ChangePasswordValues) => {
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

  const validateChangeUsername = (values: ChangeUsernameValues) => {
    const errors: Record<string, string> = {};

    if (!values.username)
      errors.username = "Required";

    if (!values.newUsername)
      errors.newUsername = "Required";

    return errors;
  };

  const logoutMutation = useMutation({
    mutationFn: APIClient.auth.logout,
    onSuccess: () => {
      AuthContext.reset();
      toast.custom((t) => (
        <Toast type="success" body="Your username or password has been updated successfully. Please sign in again!" t={t} />
      ));
    }
  });

  const changePasswordMutation = useMutation({
    mutationFn: (data: ChangePasswordValues) => APIClient.auth.changePassword(data.username, data.oldPassword, data.newPassword),
    onSuccess: () => {
      logoutMutation.mutate();
    }
  });

  const changeUsernameMutation = useMutation({
    mutationFn: (data: ChangeUsernameValues) => APIClient.auth.changeUsername(data.username, data.newUsername),
    onSuccess: () => {
      logoutMutation.mutate();
    }
  });

  const containerClass = "flex-1 px-8 pt-4 pb-6 bg-white dark:bg-gray-800";
  const headerClass = "text-lg leading-6 font-medium text-gray-900 dark:text-gray-100";
  const buttonClass = "mt-6 w-full flex items-center justify-center py-2 px-4 transition rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500";
  const iconClass = "w-4 h-4 mr-1.5";

  return (
    <div className="mx-auto w-full">
      <div className="flex gap-6">
        {/* Password Change Form */}
        <div className={containerClass}>
          <Formik
            initialValues={{
              username: getAuthContext.username,
              oldPassword: '',
              newPassword: '',
              confirmPassword: ''
            }}
            onSubmit={(data) => {
              changePasswordMutation.mutate(data);
            }}
            validate={validateChangePassword}
          >
            <Form>
              <div className="grid grid-cols-1 gap-5">
                <h3 className={headerClass}>Change Password</h3>
                <TextField name="username" label="Username" columns={6} autoComplete="username" disabled />
                <PasswordField name="oldPassword" label="Current Password" columns={6} autoComplete="current-password" required />
                <PasswordField name="newPassword" label="New Password" columns={6} autoComplete="new-password" required />
                <PasswordField name="confirmPassword" label="Confirm Password" columns={6} autoComplete="new-password" required />
              </div>
              <button type="submit" className={buttonClass}>
                <KeyIcon className={iconClass} />
                Change Password
              </button>
            </Form>
          </Formik>
        </div>
        {/* Username Change Form */}
        <div className={containerClass}>
          <Formik
            initialValues={{
              username: getAuthContext.username,
              newUsername: '',
            }}
            onSubmit={(data) => {
              changeUsernameMutation.mutate(data);
            }}
            validate={validateChangeUsername}
          >
            <Form>
              <div className="grid grid-cols-1 gap-5">
                <h3 className={headerClass}>Change Username</h3>
                <TextField name="username" label="Username" columns={6} autoComplete="username" disabled />
                <TextField name="newUsername" label="New Username" columns={6} autoComplete="username" required />
              </div>
              <button type="submit" className={buttonClass}>
                <UserIcon className={iconClass} />
                Change Username
              </button>
            </Form>
          </Formik>
        </div>
      </div>
    </div>
  );
}


export default ProfileSettings;
