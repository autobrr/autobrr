/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, useEffect, useRef, useCallback, Fragment } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Form, Formik, FormikProps } from "formik";
import toast from "react-hot-toast";
import { QrCodeIcon } from "@heroicons/react/24/solid";
import { Dialog, DialogPanel, DialogTitle, Transition, TransitionChild } from "@headlessui/react";

import { APIClient } from "@api/APIClient";
import { Section } from "../_components";
import { TextField } from "@components/inputs";
import Toast from "@components/notifications/Toast";
import { DeleteModal } from "@components/modals";

interface VerificationValues {
  code: string;
}

interface Disable2FAVariables {
  silent: boolean;
}

interface Verify2FAVariables {
  code: string;
}

// Constants for setup timeout
const SETUP_TIMEOUT_MS = 5 * 60 * 1000; // 5 minutes
const CLEANUP_CHECK_INTERVAL_MS = 1000; // 1 second

const SetupModalContent = ({ qrCode, secret, formikRef, isProcessing, handleCancel, verify2FAMutation }: any) => (
  <>
    <div className="bg-white dark:bg-gray-800 px-4 pt-5 pb-4 sm:py-6 sm:px-4 sm:pb-4">
      <div className="mt-3 text-left sm:mt-0">
        <DialogTitle as="h3" className="mb-3 text-lg leading-6 pb-2 text-center font-medium text-gray-900 dark:text-white">
          Set up Two-Factor Authentication
        </DialogTitle>
        <div className="space-y-4">
          <div className="flex flex-col items-center space-y-4">
            <img src={qrCode} alt="2FA QR Code" className="w-48 h-48" />
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Secret key: <span className="font-medium">{secret}</span>
            </p>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Scan the QR code with your authenticator app and enter the verification code below to complete setup.
              This setup will expire in 5 minutes.
            </p>
          </div>
          <Formik
            innerRef={formikRef}
            initialValues={{ code: "" }}
            validate={(values: VerificationValues) => {
              const errors: Record<string, string> = {};
              if (!values.code) {
                errors.code = "Verification code is required";
              } else if (!/^\d{6}$/.test(values.code)) {
                errors.code = "Code must be 6 digits";
              }
              return errors;
            }}
            onSubmit={(values: VerificationValues) => {
              if (!isProcessing.current) {
                verify2FAMutation.mutate({ code: values.code });
              }
            }}
          >
            <Form className="flex flex-col space-y-4">
              <div>
                <TextField
                  name="code"
                  label="Verification Code:"
                  placeholder="Enter the 6-digit code"
                />
              </div>
            </Form>
          </Formik>
        </div>
      </div>
    </div>
    <div className="bg-gray-50 dark:bg-gray-800 px-4 py-4 sm:px-4 sm:flex sm:flex-row-reverse">
      <button
        type="button"
        onClick={() => {
          if (!isProcessing.current) {
            const form = formikRef.current;
            if (form) {
              form.submitForm();
            }
          }
        }}
        disabled={isProcessing.current}
        className="flex items-center py-2 px-4 ml-2 transition rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
      >
        Verify
      </button>
      <button
        type="button"
        onClick={handleCancel}
        disabled={isProcessing.current}
        className="mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 dark:border-gray-600 shadow-sm px-4 py-2 bg-white dark:bg-gray-700 text-base font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 sm:mt-0 sm:ml-3 sm:w-auto sm:text-sm disabled:opacity-50"
      >
        Cancel
      </button>
    </div>
  </>
);

export function TwoFactorAuth() {
  const [setupMode, setSetupMode] = useState(false);
  const [qrCode, setQrCode] = useState("");
  const [secret, setSecret] = useState("");
  const [setupStartTime, setSetupStartTime] = useState<number | null>(null);
  const [showDisableModal, setShowDisableModal] = useState(false);
  const verificationSuccessful = useRef(false);
  const queryClient = useQueryClient();
  const isProcessing = useRef(false);
  const formikRef = useRef<FormikProps<VerificationValues>>(null);

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
      setShowDisableModal(false);
    },
    onError: (_, { silent } = { silent: false }) => {
      if (!silent) {
        toast.custom((t) => (
          <Toast type="error" body="Failed to disable 2FA" t={t} />
        ));
      }
      setShowDisableModal(false);
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
      isProcessing.current = false;
      // Set the field error using Formik
      if (formikRef.current) {
        formikRef.current.setFieldError('code', 'Invalid verification code');
        formikRef.current.setFieldValue('code', '', false);
      }
      // You can keep the toast as well, or remove it if you prefer just the field error
      toast.custom((t) => (
        <Toast type="error" body="Invalid verification code. Please check your authenticator app and try again." t={t} />
      ));
    }
  });

  // Handle explicit user cancellation
  const handleCancel = useCallback(() => {
    if (!isProcessing.current && setupMode) {
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
  }, [disable2FAMutation, cleanupUIState, setupMode]);

  // Setup timeout check
  useEffect(() => {
    let timeoutInterval: NodeJS.Timeout | undefined;

    if (setupMode && setupStartTime && !verificationSuccessful.current) {
      timeoutInterval = setInterval(() => {
        const elapsedTime = Date.now() - setupStartTime;

        if (elapsedTime >= SETUP_TIMEOUT_MS && !isProcessing.current) {
          handleCancel();
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
  }, [setupMode, setupStartTime, handleCancel]);

  const cancelModalButtonRef = useRef(null);

  return (
    <>
      <Section
        title="Two-Factor Authentication"
        description="Enable two-factor authentication to add an extra layer of security to your account."
        noLeftPadding
      >
        <div className="px-2 pb-6 bg-white dark:bg-gray-800">
          <div className="flex mt-10 items-center justify-between">
            <div>
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {twoFactorStatus?.enabled
                  ? "Two-factor authentication is enabled."
                  : "Two-factor authentication is disabled."}
              </p>
            </div>
            <button
              onClick={() => {
                if (!isProcessing.current) {
                  if (twoFactorStatus?.enabled) {
                    setShowDisableModal(true);
                  } else {
                    startSetupMutation.mutate();
                  }
                }
              }}
              className="flex items-center py-2 px-4 transition rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
            >
              <QrCodeIcon className="w-4 h-4 mr-1" />
              {twoFactorStatus?.enabled ? "Disable 2FA" : "Enable 2FA"}
            </button>
          </div>
        </div>
      </Section>

      <Transition show={setupMode} as={Fragment}>
        <Dialog
          as="div"
          static
          className="fixed z-10 inset-0 overflow-y-auto bg-gray-700/60 dark:bg-black/60 transition-opacity"
          open={setupMode}
          onClose={() => handleCancel()}
        >
          <div className="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
            <span className="hidden sm:inline-block sm:align-middle sm:h-screen" aria-hidden="true">
              &#8203;
            </span>
            <TransitionChild
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
              enterTo="opacity-100 translate-y-0 sm:scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 translate-y-0 sm:scale-100"
              leaveTo="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
            >
              <DialogPanel className="inline-block align-bottom border border-transparent dark:border-gray-700 rounded-lg text-left overflow-hidden shadow-xl transform transition sm:my-8 sm:align-middle w-full sm:max-w-lg">
                <SetupModalContent
                  qrCode={qrCode}
                  secret={secret}
                  formikRef={formikRef}
                  isProcessing={isProcessing}
                  handleCancel={handleCancel}
                  verify2FAMutation={verify2FAMutation}
                />
              </DialogPanel>
            </TransitionChild>
          </div>
        </Dialog>
      </Transition>

      <DeleteModal
        title="Disable Two-Factor Authentication"
        text="Are you sure you want to disable two-factor authentication?"
        isOpen={showDisableModal}
        isLoading={disable2FAMutation.isPending}
        toggle={() => setShowDisableModal(false)}
        buttonRef={cancelModalButtonRef}
        deleteAction={() => disable2FAMutation.mutate({ silent: false })}
      />
    </>
  );
}
