/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Form, Formik } from "formik";
import toast from "react-hot-toast";
import { UserIcon, QrCodeIcon } from "@heroicons/react/24/solid";

import { APIClient } from "@api/APIClient";
import { Section } from "./_components";
import { PasswordField, TextField } from "@components/inputs";
import Toast from "@components/notifications/Toast";
import { AuthContext } from "@utils/Context";
import { useState, useEffect, useRef, useCallback } from "react";

const AccountSettings = () => (
  <Section
    title="Account"
    description="Manage account settings."
  >
    <div className="py-0.5">
      <Credentials />
      <TwoFactorAuth />
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

interface VerificationValues {
  code: string;
}

interface UserUpdate {
  username_current: string;
  username_new: string;
  password_current: string;
  password_new: string;
}

interface Disable2FAVariables {
  silent: boolean;
}

interface Verify2FAVariables {
  code: string;
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

function TwoFactorAuth() {
  const [setupMode, setSetupMode] = useState(false);
  const [qrCode, setQrCode] = useState("");
  const [secret, setSecret] = useState("");
  const [setupStartTime, setSetupStartTime] = useState<number | null>(null);
  const verificationSuccessful = useRef(false);
  const queryClient = useQueryClient();
  const isProcessing = useRef(false);

  // Constants for setup timeout
  const SETUP_TIMEOUT_MS = 5 * 60 * 1000; // 5 minutes
  const CLEANUP_CHECK_INTERVAL_MS = 1000; // 1 second

  // Query 2FA status
  const { data: twoFactorStatus } = useQuery({
    queryKey: ["2fa-status"],
    queryFn: APIClient.auth.get2FAStatus
  });

  // Cleanup UI state only
  const cleanupUIState = useCallback(() => {
    setSetupMode(false);
    setQrCode("");
    setSecret("");
    setSetupStartTime(null);
  }, []);

  // Disable 2FA with silent option
  const disable2FAMutation = useMutation<void, Error, Disable2FAVariables>({
    mutationFn: APIClient.auth.disable2FA,
    onSuccess: (_, { silent } = { silent: false }) => {
      queryClient.invalidateQueries({ queryKey: ["2fa-status"] });
      if (!silent) {
        toast.custom((t) => (
          <Toast type="success" body="Two-factor authentication disabled" t={t} />
        ));
      }
    },
    onError: (_, { silent } = { silent: false }) => {
      if (!silent) {
        toast.custom((t) => (
          <Toast type="error" body="Failed to disable 2FA" t={t} />
        ));
      }
    }
  });

  // Start 2FA setup
  const startSetupMutation = useMutation({
    mutationFn: APIClient.auth.enable2FA,
    onSuccess: (data) => {
      setQrCode(data.url);
      setSecret(data.secret);
      setSetupMode(true);
      setSetupStartTime(Date.now());
      verificationSuccessful.current = false;
      toast.custom((t) => (
        <Toast type="success" body="Scan the QR code with your authenticator app" t={t} />
      ));
    },
    onError: () => {
      toast.custom((t) => (
        <Toast type="error" body="Failed to start 2FA setup" t={t} />
      ));
    }
  });

  // Silent cleanup for automatic cancellations
  const silentCleanup = useCallback(() => {
    if (!isProcessing.current) {
      isProcessing.current = true;
      disable2FAMutation.mutate({ silent: true }, {
        onSettled: () => {
          cleanupUIState();
          isProcessing.current = false;
        }
      });
    }
  }, [disable2FAMutation, cleanupUIState]);

  // Verify 2FA code
  const verify2FAMutation = useMutation<void, Error, Verify2FAVariables>({
    mutationFn: APIClient.auth.verify2FA,
    onMutate: () => {
      isProcessing.current = true;
    },
    onSuccess: () => {
      verificationSuccessful.current = true;
      queryClient.invalidateQueries({ queryKey: ["2fa-status"] });
      toast.custom((t) => (
        <Toast type="success" body="Two-factor authentication enabled successfully!" t={t} />
      ));
      cleanupUIState();
    },
    onError: () => {
      toast.custom((t) => (
        <Toast type="error" body="Invalid verification code" t={t} />
      ));
      return Promise.resolve();
    },
    onSettled: () => {
      isProcessing.current = false;
    }
  });

  // Handle explicit user cancellation
  const handleCancel = useCallback(() => {
    if (!isProcessing.current) {
      isProcessing.current = true;
      disable2FAMutation.mutate({ silent: true }, {
        onSettled: () => {
          cleanupUIState();
          isProcessing.current = false;
          toast.custom((t) => (
            <Toast type="info" body="2FA setup cancelled" t={t} />
          ));
        }
      });
    }
  }, [disable2FAMutation, cleanupUIState]);

  // Setup timeout check
  useEffect(() => {
    let timeoutInterval: NodeJS.Timeout | undefined;

    if (setupMode && setupStartTime && !isProcessing.current) {
      timeoutInterval = setInterval(() => {
        const elapsedTime = Date.now() - setupStartTime;
        
        if (elapsedTime >= SETUP_TIMEOUT_MS) {
          silentCleanup();
          toast.custom((t) => (
            <Toast type="error" body="2FA setup timed out. Please try again." t={t} />
          ));
        }
      }, CLEANUP_CHECK_INTERVAL_MS);
    }

    return () => {
      if (timeoutInterval) {
        clearInterval(timeoutInterval);
      }
    };
  }, [SETUP_TIMEOUT_MS, setupMode, setupStartTime, silentCleanup]);

  // Handle component unmounting and window unload
  useEffect(() => {
    const handleUnload = () => {
      if (setupMode && !verificationSuccessful.current && !isProcessing.current) {
        silentCleanup();
      }
    };

    window.addEventListener('beforeunload', handleUnload);

    return () => {
      window.removeEventListener('beforeunload', handleUnload);
      if (setupMode && !verificationSuccessful.current && !isProcessing.current) {
        silentCleanup();
      }
    };
  }, [setupMode, silentCleanup]);

  const validateVerificationCode = (values: VerificationValues) => {
    const errors: Record<string, string> = {};
    if (!values.code) {
      errors.code = "Verification code is required";
    } else if (!/^\d{6}$/.test(values.code)) {
      errors.code = "Code must be 6 digits";
    }
    return errors;
  };

  return (
    <Section
      title="Two-Factor Authentication"
      description="Enable two-factor authentication to add an extra layer of security to your account."
      noLeftPadding
    >
      <div className="px-2 pb-6 bg-white dark:bg-gray-800">
        {!setupMode ? (
          <div className="flex items-center justify-between">
            <div>
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {twoFactorStatus?.enabled
                  ? "Two-factor authentication is enabled"
                  : "Two-factor authentication is disabled"}
              </p>
            </div>
            <button
              onClick={() => {
                if (!isProcessing.current) {
                  // eslint-disable-next-line @typescript-eslint/no-unused-expressions
                  twoFactorStatus?.enabled 
                    ? disable2FAMutation.mutate({ silent: false }) 
                    : startSetupMutation.mutate();
                }
              }}
              disabled={isProcessing.current}
              className="flex items-center py-2 px-4 transition rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 disabled:opacity-50"
            >
              <QrCodeIcon className="w-4 h-4 mr-1" />
              {twoFactorStatus?.enabled ? "Disable 2FA" : "Enable 2FA"}
            </button>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="flex flex-col items-center space-y-4">
              <img src={qrCode} alt="2FA QR Code" className="w-48 h-48" />
              <p className="text-sm text-gray-600 dark:text-gray-400">
                Secret key: {secret}
              </p>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                Scan the QR code with your authenticator app and enter the verification code below to complete setup.
                This setup will expire in 5 minutes.
              </p>
            </div>
            <Formik
              initialValues={{ code: "" }}
              validate={validateVerificationCode}
              onSubmit={(values: VerificationValues, { resetForm }) => {
                if (!isProcessing.current) {
                  verify2FAMutation.mutate({ code: values.code }, {
                    onSettled: () => {
                      resetForm();
                    }
                  });
                }
              }}
            >
              {({ errors, touched }) => (
                <Form className="flex flex-col space-y-4">
                  <div>
                    <TextField
                      name="code"
                      label="Verification Code"
                      placeholder="Enter the 6-digit code"
                    />
                    {touched.code && errors.code && (
                      <p className="mt-1 text-sm text-red-600 dark:text-red-500">{errors.code}</p>
                    )}
                  </div>
                  <div className="flex justify-end space-x-4">
                    <button
                      type="button"
                      onClick={handleCancel}
                      disabled={isProcessing.current}
                      className="py-2 px-4 transition rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 disabled:opacity-50"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      disabled={isProcessing.current}
                      className="py-2 px-4 transition rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 disabled:opacity-50"
                    >
                      Verify
                    </button>
                  </div>
                </Form>
              )}
            </Formik>
          </div>
        )}
      </div>
    </Section>
  );
}

export default AccountSettings;
