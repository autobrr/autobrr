/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation } from "@tanstack/react-query";
import { UserIcon } from "@heroicons/react/24/solid";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faOpenid } from "@fortawesome/free-brands-svg-icons";

import { APIClient } from "@api/APIClient";
import { Section } from "./_components";
import { PasswordField, TextField } from "@components/inputs/tanstack";
import toast from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { AuthContext } from "@utils/Context";
import { useAppForm, useStore } from "@app/lib/form";

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

function Credentials() {
  const username = AuthContext.useSelector((s) => s.username);

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

  const form = useAppForm({
    defaultValues: {
      username: username,
      newUsername: "",
      oldPassword: "",
      newPassword: "",
      confirmPassword: ""
    },
    onSubmit: async ({ value }) => {
      updateUserMutation.mutate({
        username_current: value.username,
        username_new: value.newUsername,
        password_current: value.oldPassword,
        password_new: value.newPassword,
      });
    },
  });

  const newPassword = useStore(form.store, (s) => s.values.newPassword);

  const separatorClass = "mb-6";

  return (
    <Section
      title="Change credentials"
      description="The username and password can be changed either separately or simultaneously. Note that you will be logged out after changing credentials."
      noLeftPadding
    >
      <div className="px-2 pb-0 sm:pb-6 bg-white dark:bg-gray-800">
        <form.AppForm>
          <form onSubmit={(e) => { e.preventDefault(); form.handleSubmit(); }}>
              <div className="flex flex-col sm:grid sm:grid-cols-2 gap-x-10 pt-2">
                <div className={separatorClass}>
                  <form.AppField name="username">
                    {() => <TextField label="Current Username" autoComplete="username" disabled />}
                  </form.AppField>
                </div>
                <div className={separatorClass}>
                  <form.AppField name="newUsername">
                    {() => <TextField label="New Username" tooltip={
                      <div>
                        <p>Optional</p>
                      </div>
                    } />}
                  </form.AppField>
                </div>

                <hr className="col-span-2 mb-6 border-t border-gray-300 dark:border-gray-750" />

                <div className={separatorClass}>
                  <form.AppField name="oldPassword">
                    {() => <PasswordField placeholder="Required" label="Current Password" autoComplete="current-password" required tooltip={
                      <div>
                        <p>Required if updating credentials</p>
                      </div>
                    } />}
                  </form.AppField>
                </div>
                <div>
                  <div className={separatorClass}>
                    <form.AppField name="newPassword">
                      {() => <PasswordField label="New Password" autoComplete="new-password" tooltip={
                        <div>
                          <p>Optional</p>
                        </div>
                      } />}
                    </form.AppField>
                  </div>
                  {newPassword && (
                    <div className={separatorClass}>
                      <form.AppField name="confirmPassword" validators={{
                        onSubmit: ({ value, fieldApi }) => {
                          if (fieldApi.form.getFieldValue("newPassword") && value !== fieldApi.form.getFieldValue("newPassword")) return "Passwords don't match!";
                          return undefined;
                        }
                      }}>
                        {() => <PasswordField label="Confirm New Password" autoComplete="new-password" />}
                      </form.AppField>
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
          </form>
        </form.AppForm>
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
                className="h-16 w-16 sm:h-20 sm:w-20 rounded-full object-cover border-1 border-gray-200 dark:border-gray-700 transition duration-200"
                onError={() => auth.profilePicture = undefined}
              />
            ) : (
              <div className="h-16 w-16 sm:h-20 sm:w-20 rounded-full flex items-center justify-center bg-gray-100 dark:bg-gray-700 border-2 border-gray-200 dark:border-gray-700 transition duration-200">
                <FontAwesomeIcon 
                  icon={faOpenid} 
                  className="h-16 w-16 text-gray-500 dark:text-gray-400" 
                  aria-hidden="true"
                />
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
