/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQuery } from "@tanstack/react-query";
import { Form, Formik } from "formik";
import toast from "react-hot-toast";
import { UserIcon } from "@heroicons/react/24/solid";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faOpenid } from "@fortawesome/free-brands-svg-icons";

import { APIClient } from "@api/APIClient";
import { Section } from "./_components";
import { PasswordField, TextField } from "@components/inputs";
import Toast from "@components/notifications/Toast";
import { AuthContext } from "@utils/Context";

const AccountSettings = () => {
  const { data: oidcConfig, isLoading } = useQuery({
    queryKey: ["oidc-config"],
    queryFn: () => APIClient.auth.getOIDCConfig(),
  });

  if (isLoading) {
    return null;
  }

  return (
    <Section
      title="Account"
      description="Manage account settings."
    >
      <div className="py-0.5">
        {oidcConfig?.enabled ? <OIDCAccount /> : <Credentials />}
      </div>
    </Section>
  );
};

interface InputValues {
  username: string;
  newUsername: string;
  oldPassword: string;
  newPassword: string;
  confirmPassword: string;
}

function Credentials() {
  const username = AuthContext.useSelector((s) => s.username);

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

  const updateUserMutation = useMutation({
    mutationFn: (data: UserUpdate) => APIClient.auth.updateUser(data),
    onError: () => {
      toast.custom((t) => (
        <Toast type="error" body="Error updating credentials. Did you provide the correct current password?" t={t} />
      ));
    },
    onSuccess: () => {
      logoutMutation.mutate();
    }
  });

  const separatorClass = "mb-6";

  return (
    <Section
      title="Change credentials"
      description="The username and password can be changed either separately or simultaneously. Note that you will be logged out after changing credentials."
      noLeftPadding
    >
      <div className="px-2 pb-0 sm:pb-6 bg-white dark:bg-gray-800">
        <Formik
          initialValues={{
            username: username,
            newUsername: "",
            oldPassword: "",
            newPassword: "",
            confirmPassword: ""
          }}
          onSubmit={(data) => {
            updateUserMutation.mutate({
              username_current: data.username,
              username_new: data.newUsername,
              password_current: data.oldPassword,
              password_new: data.newPassword,
            });
          }}
          validate={validate}
        >
          {({ values }) => (
            <Form>
              <div className="flex flex-col sm:grid sm:grid-cols-2 gap-x-10 pt-2">
                <div className={separatorClass}>
                  <TextField name="username" label="Current Username" autoComplete="username" disabled />
                </div>
                <div className={separatorClass}>
                  <TextField name="newUsername" label="New Username" tooltip={
                    <div>
                      <p>Optional</p>
                    </div>
                  } />
                </div>

                <hr className="col-span-2 mb-6 border-t border-gray-300 dark:border-gray-750" />

                <div className={separatorClass}>
                  <PasswordField name="oldPassword" placeholder="Required" label="Current Password" autoComplete="current-password" required tooltip={
                    <div>
                      <p>Required if updating credentials</p>
                    </div>
                  } />
                </div>
                <div>
                  <div className={separatorClass}>
                    <PasswordField name="newPassword" label="New Password" autoComplete="new-password" tooltip={
                      <div>
                        <p>Optional</p>
                      </div>
                    } />
                  </div>
                  {values.newPassword && (
                    <div className={separatorClass}>
                      <PasswordField name="confirmPassword" label="Confirm New Password" autoComplete="new-password" />
                    </div>
                  )}
                </div>
              </div>
              <div className="flex justify-end">
                <button
                  type="submit"
                  className="mt-4 w-auto flex items-center py-2 px-4 transition rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                >
                  <UserIcon className="w-4 h-4 mr-1" />
                  Save
                </button>
              </div>
            </Form>
          )}
        </Formik>
      </div>
    </Section>
  );
}

function OIDCAccount() {
  return (
    <Section
      titleElement={
        <div className="flex items-center space-x-2">
          <span className="text-gray-700 dark:text-gray-300 font-bold">OpenID Connect Account</span>
          <FontAwesomeIcon icon={faOpenid} className="h-5 w-5 text-gray-500 dark:text-gray-400" />
        </div>
      }
      title="OpenID Connect Account"
      description="Your account credentials are managed by your OpenID Connect provider. To change your username, please visit your provider's settings page and log in again."
      noLeftPadding
    >
    </Section>
  );
}

export default AccountSettings;
