/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation } from "@tanstack/react-query";
import { Form, Formik } from "formik";
import { UserIcon } from "@heroicons/react/24/solid";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faOpenid } from "@fortawesome/free-brands-svg-icons";

import { APIClient } from "@api/APIClient";
import { Section } from "./_components";
import { PasswordField, TextField } from "@components/inputs";
import toast from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { AuthContext } from "@utils/Context";

const AccountSettings = () => {
  const auth = AuthContext.get();

  return (
    <Section
      title="Account"
      description="Manage account settings."
    >
      <div className="py-0.5">
        {auth.authMethod === 'oidc' ? <OIDCAccount /> : <Credentials />}
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
                  className="mt-4 w-auto flex items-center py-2 px-4 transition rounded-md shadow-xs text-sm font-medium text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
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
  const auth = AuthContext.get();
  
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
      <div className="px-4 py-5 sm:p-6 bg-white dark:bg-gray-800 rounded-lg transition duration-150 dark:shadow-gray-900">
        <div className="flex flex-col sm:flex-row items-center">
          <div className="flex-shrink-0 relative">
            {auth.profilePicture ? (
              <img
                src={auth.profilePicture}
                alt={`${auth.username}'s profile picture`}
                className="h-16 w-16 sm:h-16 sm:w-16 rounded-full object-cover border-2 border-gray-200 dark:border-gray-700 transition duration-200 shadow-sm"
                onError={(e) => {
                  const target = e.target as HTMLImageElement;
                  target.style.display = 'none';
                  const parent = target.parentElement;
                  if (parent) {
                    const iconContainer = document.createElement('div');
                    iconContainer.className = "h-20 w-20 sm:h-24 sm:w-24 rounded-full bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-800 flex items-center justify-center shadow-sm border-2 border-gray-200 dark:border-gray-700";
                    iconContainer.innerHTML = '<svg aria-hidden="true" focusable="false" data-prefix="fab" data-icon="openid" class="h-10 w-10 text-gray-500 dark:text-gray-400" role="img" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 448 512"><path fill="currentColor" d="M271.5 432l-68 32C88.5 453.7 0 392.5 0 318.2c0-71.5 82.5-131 191.7-144.3v43c-71.5 12.5-124 53-124 101.3 0 51 58.5 93.3 135.5 103v-340l68-33.2v384zM448 291l-131.3-28.5 36.8-20.7c-19.5-11.5-43.5-20-70-24.8v-43c46.2 5.5 87.7 19.5 120.3 39.3l35-19.8L448 291z"></path></svg>';
                    parent.appendChild(iconContainer);
                  }
                }}
              />
            ) : (
              <div className="h-20 w-20 sm:h-24 sm:w-24 rounded-full bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-800 flex items-center justify-center shadow-sm border-2 border-gray-200 dark:border-gray-700 transition duration-200 hover:border-blue-500 dark:hover:border-blue-400">
                <FontAwesomeIcon icon={faOpenid} className="h-10 w-10 text-gray-500 dark:text-gray-400" />
              </div>
            )}
          </div>
          <div className="mt-4 sm:mt-0 sm:ml-6 text-center sm:text-left">
            <h3 className="text-xl font-semibold leading-6 text-gray-900 dark:text-gray-100">
              {auth.username}
            </h3>
            <div className="mt-2 flex items-center text-sm text-gray-500 dark:text-gray-400">
              <FontAwesomeIcon icon={faOpenid} className="mr-1.5 h-4 w-4 flex-shrink-0" />
              <p>Authenticated via OpenID Connect</p>
            </div>
          </div>
        </div>
      </div>
    </Section>
  );
}

export default AccountSettings;
