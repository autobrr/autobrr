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
  
  // Helper function to format the issuer URL for display
  const getFormattedIssuerName = () => {
    if (!auth.issuerUrl) return "your identity provider";
    
    try {
      const url = new URL(auth.issuerUrl);
      // Return domain name without 'www.'
      return url.hostname.replace(/^www\./i, '');
    } catch {
      return "your identity provider";
    }
  };
  
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
      <div className="px-4 py-5 sm:p-6 bg-white dark:bg-gray-800 rounded-lg border border-gray-100 dark:border-gray-700 transition duration-150">
        <div className="flex flex-col sm:flex-row items-center">
          <div className="flex-shrink-0 relative">
            {auth.profilePicture ? (
              <img
                src={auth.profilePicture}
                alt={`${auth.username}'s profile picture`}
                className="h-16 w-16 sm:h-20 sm:w-20 rounded-full object-cover border-2 border-gray-200 dark:border-gray-400 transition duration-200"
                onError={(e) => {
                  // Fallback to OIDC icon if image fails to load
                  const target = e.target as HTMLImageElement;
                  target.style.display = 'none';
                  // Create and append OIDC icon element
                  const parent = target.parentElement;
                  if (parent) {
                    const iconContainer = document.createElement('div');
                    iconContainer.className = "h-16 w-16 sm:h-20 sm:w-20 rounded-full bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-800 dark:to-gray-700 flex items-center justify-center border-2 border-blue-100 dark:border-blue-900";
                    iconContainer.innerHTML = '<svg aria-hidden="true" focusable="false" data-prefix="fab" data-icon="openid" class="h-8 w-8 text-gray-500 dark:text-gray-400" role="img" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 448 512"><path fill="currentColor" d="M271.5 432l-68 32C88.5 453.7 0 392.5 0 318.2c0-71.5 82.5-131 191.7-144.3v43c-71.5 12.5-124 53-124 101.3 0 51 58.5 93.3 135.5 103v-340l68-33.2v384zM448 291l-131.3-28.5 36.8-20.7c-19.5-11.5-43.5-20-70-24.8v-43c46.2 5.5 87.7 19.5 120.3 39.3l35-19.8L448 291z"></path></svg>';
                    parent.appendChild(iconContainer);
                  }
                }}
              />
            ) : (
              <div className="h-16 w-16 sm:h-20 sm:w-20 rounded-full bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-800 dark:to-gray-700 flex items-center justify-center border-2 border-blue-100 dark:border-blue-900 transition duration-200">
                <FontAwesomeIcon icon={faOpenid} className="h-7 w-7 text-gray-500 dark:text-gray-400" />
              </div>
            )}
          </div>
          <div className="mt-4 sm:mt-0 sm:ml-6 text-center sm:text-left">
            <h3 className="text-lg font-medium leading-6 text-gray-900 dark:text-gray-100">
              {auth.username}
            </h3>
            <div className="mt-1 flex items-center text-sm text-gray-500 dark:text-gray-400">
              <FontAwesomeIcon icon={faOpenid} className="mr-1.5 h-4 w-4 flex-shrink-0" />
              <p>Authenticated via OpenID Connect</p>
            </div>
            {auth.issuerUrl && (
              <div className="mt-2">
                <a 
                  href={auth.issuerUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center px-2.5 py-1.5 text-xs font-medium rounded-md text-blue-700 dark:text-blue-300 bg-blue-50 dark:bg-blue-900/30 hover:bg-blue-100 dark:hover:bg-blue-900/50 border border-blue-200 dark:border-blue-800 transition-colors duration-150"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-3.5 w-3.5 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                  </svg>
                  {getFormattedIssuerName()}
                </a>
              </div>
            )}
          </div>
        </div>
      </div>
    </Section>
  );
}

export default AccountSettings;
