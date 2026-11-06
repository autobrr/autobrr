/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { XMarkIcon } from "@heroicons/react/24/solid";
import { ArrowTopRightOnSquareIcon } from "@heroicons/react/24/outline";
import { useTranslation } from "react-i18next";

import { APIClient } from "@api/APIClient";
import { MatrixKeys } from "@api/query_keys";
import { MatrixNetworkQueryOptions } from "@api/queries";
import { fieldHasError, useAppForm, useFormContext } from "@hooks/form";
import type { FormFieldErrors } from "@hooks/form";
import { PasswordFieldWide, SwitchButton, SwitchGroupWide, TextFieldWide } from "@components/inputs";
import { SlideOver, SlideOverShell, SlideOverTitle } from "@components/panels";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { classNames } from "@utils";
import { AddFormProps, UpdateFormProps } from "@forms/_shared";

const isRoomName = (name: string) => /^[#!][^:]+:.+$/.test(name.trim());

const validateNetwork = (values: MatrixNetworkCreate, t: (key: string) => string): FormFieldErrors => {
  const errors: FormFieldErrors = {};

  if (!values.name) {
    errors.name = t("forms.matrix.required");
  }

  if (!values.homeserver) {
    errors.homeserver = t("forms.matrix.required");
  }

  values.rooms.forEach((room, index) => {
    if (!isRoomName(room.name)) {
      errors[`rooms[${index}].name`] = t("forms.matrix.invalidRoom");
    }
  });

  return errors;
};

interface RoomsFieldArrayProps {
  rooms: MatrixRoom[];
}

const RoomsFieldArray = ({ rooms }: RoomsFieldArrayProps) => {
  const { t } = useTranslation("settings");
  const form = useFormContext();

  return (
    <div className="px-4">
      <div className="flex flex-col space-y-2">
        {rooms.length > 0 ? (
          rooms.map((room, index) => (
            <div key={room.id || `new-${index}`} className="flex justify-between border dark:border-gray-700 dark:bg-gray-815 p-2 rounded-md">
              <div className="flex w-full gap-2 items-center">
                <div className="shrink-0" title={t("forms.matrix.roomEnabled")}>
                  <SwitchButton name={`rooms[${index}].enabled`} defaultValue={true} />
                </div>
                <form.Field name={`rooms[${index}].name`}>
                  {(field) => (
                    <input
                      name={field.name}
                      type="text"
                      value={field.state.value ?? ""}
                      onChange={(e) => field.handleChange(e.target.value)}
                      onBlur={field.handleBlur}
                      placeholder={t("forms.matrix.roomPlaceholder")}
                      className={classNames(
                        fieldHasError(field.state.meta)
                          ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                          : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500",
                        "block w-full shadow-xs sm:text-sm rounded-md border py-2.5 bg-gray-100 dark:bg-gray-850 dark:text-gray-100"
                      )}
                    />
                  )}
                </form.Field>
              </div>

              <button
                type="button"
                className="ml-2 bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer"
                onClick={() => form.removeFieldValue("rooms", index)}
              >
                <span className="sr-only">{t("forms.matrix.remove")}</span>
                <XMarkIcon className="h-6 w-6" aria-hidden="true" />
              </button>
            </div>
          ))
        ) : (
          <span className="text-center text-sm text-gray-500 dark:text-gray-400">
            {t("forms.matrix.noRooms")}
          </span>
        )}
        <button
          type="button"
          className="border dark:border-gray-600 dark:bg-gray-700 my-4 px-4 py-2 text-sm text-gray-700 dark:text-white hover:bg-gray-50 dark:hover:bg-gray-600 rounded-sm self-center text-center cursor-pointer"
          onClick={() => form.pushFieldValue("rooms", { id: 0, name: "", room_id: "", enabled: true })}
        >
          {t("forms.matrix.addRoom")}
        </button>
      </div>
    </div>
  );
};

interface NetworkFieldsProps {
  rooms: MatrixRoom[];
}

const NetworkFields = ({ rooms }: NetworkFieldsProps) => {
  const { t } = useTranslation("settings");

  return (
    <div className="divide-y divide-gray-200 dark:divide-gray-700">
      <TextFieldWide name="name" label={t("forms.matrix.name")} required={true} />
      <SwitchGroupWide name="enabled" label={t("forms.matrix.enabled")} />
      <TextFieldWide
        name="homeserver"
        label={t("forms.matrix.homeserver")}
        placeholder={t("forms.matrix.homeserverPlaceholder")}
        help={t("forms.matrix.homeserverHelp")}
        required={true}
      />

      <div className="py-5">
        <div className="px-4 space-y-1 mb-8">
          <h2 className="text-lg font-medium text-gray-900 dark:text-white">{t("forms.matrix.rooms")}</h2>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {t("forms.matrix.roomsDesc")}
          </p>
        </div>

        <RoomsFieldArray rooms={rooms} />
      </div>
    </div>
  );
};

export function MatrixNetworkAddForm({ isOpen, toggle }: AddFormProps) {
  const { t } = useTranslation("settings");
  const queryClient = useQueryClient();

  const createMutation = useMutation({
    mutationFn: (network: MatrixNetworkCreate) => APIClient.matrix.store(network),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MatrixKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.matrix.added")} t={toastInstance} />);
      toggle();
    },
    onError: () => {
      toast.custom((toastInstance) => <Toast type="error" body={t("forms.matrix.addFailed")} t={toastInstance} />);
    }
  });

  const onSubmit = (formData: MatrixNetworkCreate) => {
    createMutation.mutate(formData);
  };

  const initialValues: MatrixNetworkCreate = {
    name: "",
    enabled: true,
    homeserver: "",
    access_token: "",
    rooms: []
  };

  return (
    <SlideOver<MatrixNetworkCreate>
      title={t("forms.matrix.title")}
      type="CREATE"
      isOpen={isOpen}
      toggle={toggle}
      initialValues={initialValues}
      validate={(values) => validateNetwork(values, t)}
      onSubmit={onSubmit}
    >
      {(values) => <NetworkFields rooms={values.rooms} />}
    </SlideOver>
  );
}

interface MatrixNetworkUpdateFormProps extends UpdateFormProps<MatrixNetwork> {
  onLogin: () => void;
}

export function MatrixNetworkUpdateForm({ isOpen, toggle, data, onLogin }: MatrixNetworkUpdateFormProps) {
  const { t } = useTranslation("settings");
  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: (network: MatrixNetwork) => APIClient.matrix.update(network),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MatrixKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.matrix.updated", { name: data.name })} t={toastInstance} />);
      toggle();
    },
    onError: () => {
      toast.custom((toastInstance) => <Toast type="error" body={t("forms.matrix.updateFailed")} t={toastInstance} />);
    }
  });

  const onSubmit = (formData: MatrixNetwork) => {
    updateMutation.mutate(formData);
  };

  const deleteMutation = useMutation({
    mutationFn: (id: number) => APIClient.matrix.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MatrixKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.matrix.deleted", { name: data.name })} t={toastInstance} />);
      toggle();
    }
  });

  const deleteAction = () => deleteMutation.mutate(data.id);

  const initialValues: MatrixNetwork = {
    id: data.id,
    name: data.name,
    enabled: data.enabled,
    homeserver: data.homeserver,
    auth_method: data.auth_method,
    user_id: data.user_id,
    device_id: data.device_id,
    access_token: data.access_token,
    rooms: data.rooms.map((room) => ({ id: room.id, enabled: room.enabled, name: room.name, room_id: room.room_id }))
  };

  return (
    <SlideOver<MatrixNetwork>
      title={t("forms.matrix.title")}
      type="UPDATE"
      isOpen={isOpen}
      toggle={toggle}
      initialValues={initialValues}
      validate={(values) => validateNetwork(values, t)}
      onSubmit={onSubmit}
      deleteAction={deleteAction}
    >
      {(values) => (
        <>
          <AccountSection network={data} onLogin={onLogin} />
          <NetworkFields rooms={values.rooms} />
        </>
      )}
    </SlideOver>
  );
}

interface AccountSectionProps {
  network: MatrixNetwork;
  onLogin: () => void;
}

// Reads the network from props, not the form: logging in and out applies right away,
// so it must not wait for, or be undone by, the form's Save and Cancel.
const AccountSection = ({ network, onLogin }: AccountSectionProps) => {
  const { t } = useTranslation("settings");
  const queryClient = useQueryClient();

  const logoutMutation = useMutation({
    mutationFn: (id: number) => APIClient.matrix.logout(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MatrixKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("matrix.loggedOut", { name: network.name })} t={toastInstance} />);
    }
  });

  const loggedIn = network.user_id !== "";

  return (
    <div className="px-4 py-5 border-b border-gray-200 dark:border-gray-700">
      <h2 className="text-lg font-medium text-gray-900 dark:text-white">{t("forms.matrix.account")}</h2>
      <p className="text-sm text-gray-500 dark:text-gray-400">{t("forms.matrix.accountDesc")}</p>

      <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
        <div className="min-w-0 text-sm">
          {loggedIn ? (
            <>
              <p className="font-medium text-gray-900 dark:text-white truncate">{network.user_id}</p>
              {network.auth_method && (
                <p className="text-gray-500 dark:text-gray-400">{t(`forms.matrix.authMethod.${network.auth_method}`)}</p>
              )}
            </>
          ) : (
            <p className="text-gray-500 dark:text-gray-400">{t("matrix.notLoggedIn")}</p>
          )}
          {loggedIn && network.health?.login_required && network.health.error && (
            <p className="text-red-600 dark:text-red-400">{network.health.error}</p>
          )}
        </div>

        <div className="flex gap-2">
          {loggedIn && (
            <button
              type="button"
              disabled={logoutMutation.isPending}
              onClick={() => logoutMutation.mutate(network.id)}
              className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer disabled:cursor-not-allowed disabled:opacity-60"
            >
              {t("matrix.logout")}
            </button>
          )}
          <button
            type="button"
            onClick={onLogin}
            className="inline-flex justify-center py-2 px-4 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer"
          >
            {t("matrix.login")}
          </button>
        </div>
      </div>
    </div>
  );
};

export function MatrixLoginForm({ isOpen, toggle, data }: UpdateFormProps<MatrixNetwork>) {
  return (
    <SlideOverShell isOpen={isOpen} toggle={toggle}>
      <MatrixLoginPanel toggle={toggle} network={data} />
    </SlideOverShell>
  );
}

interface MatrixLoginPanelProps {
  toggle: () => void;
  network: MatrixNetwork;
}

function MatrixLoginPanel({ toggle, network }: MatrixLoginPanelProps) {
  const { t } = useTranslation("settings");
  const queryClient = useQueryClient();

  const [startedLogin, setStartedLogin] = useState<MatrixLogin>();
  const started = startedLogin !== undefined;

  const networkQuery = useQuery(MatrixNetworkQueryOptions(network.id, 3000));
  const serverLogin = networkQuery.data?.health?.login;

  // A pending login survives closing the panel or a page reload. A finished one is only
  // shown for a login started here, so an old failure does not greet the next visit.
  const login = serverLogin?.state === "PENDING" || started ? (serverLogin ?? startedLogin) : undefined;
  const pending = login?.state === "PENDING";

  const loginMutation = useMutation({
    mutationFn: (id: number) => APIClient.matrix.login(id),
    onSuccess: (data) => {
      queryClient.setQueryData<MatrixNetwork>(MatrixKeys.detail(network.id), (current) =>
        current ? { ...current, health: { ...(current.health ?? { connected: false, healthy: false }), login: data } } : current
      );
      setStartedLogin(data);
    },
    onError: () => {
      toast.custom((toastInstance) => <Toast type="error" body={t("forms.matrix.loginFailed")} t={toastInstance} />);
    }
  });

  const tokenMutation = useMutation({
    mutationFn: (accessToken: string) => APIClient.matrix.setAccessToken(network.id, accessToken),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MatrixKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.matrix.tokenSaved", { name: network.name })} t={toastInstance} />);
      toggle();
    },
    onError: () => {
      toast.custom((toastInstance) => <Toast type="error" body={t("forms.matrix.tokenFailed")} t={toastInstance} />);
    }
  });

  const close = () => {
    queryClient.invalidateQueries({ queryKey: MatrixKeys.lists() });
    toggle();
  };

  const tokenForm = useAppForm({
    defaultValues: { access_token: "" },
    onSubmit: ({ value }) => tokenMutation.mutate(value.access_token)
  });

  return (
    <div className="h-full min-h-0 flex flex-col bg-white dark:bg-gray-800">
      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
          <div className="flex items-start justify-between space-x-3">
            <div className="space-y-1">
              <SlideOverTitle>
                {t("forms.matrix.loginTitle", { name: network.name })}
              </SlideOverTitle>
              <p className="text-sm text-gray-500 dark:text-gray-200">
                {t("forms.matrix.loginDescription")}
              </p>
            </div>
            <div className="h-7 flex items-center">
              <button
                type="button"
                className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500 cursor-pointer"
                onClick={close}
              >
                <span className="sr-only">{t("forms.matrix.close")}</span>
                <XMarkIcon className="h-6 w-6" aria-hidden="true" />
              </button>
            </div>
          </div>
        </div>

        <div className="px-4 py-5 sm:px-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-medium text-gray-900 dark:text-white">{t("forms.matrix.deviceLogin")}</h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("forms.matrix.deviceLoginDesc")}</p>

          {login?.state === "SUCCEEDED" ? (
            <p className="mt-4 text-sm font-medium text-green-700 dark:text-green-400">
              {t("forms.matrix.loginSucceeded", { user: login.user_id })}
            </p>
          ) : pending ? (
            <div className="mt-4 space-y-3">
              <p className="text-sm text-gray-700 dark:text-gray-300">{t("forms.matrix.enterCode")}</p>
              <p className="font-mono text-2xl tracking-widest text-gray-900 dark:text-white select-all">{login.user_code}</p>
              <a
                href={login.verification_uri_complete || login.verification_uri}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 text-sm font-medium text-blue-600 dark:text-blue-400 hover:underline cursor-pointer"
              >
                {login.verification_uri}
                <ArrowTopRightOnSquareIcon className="h-4 w-4" aria-hidden="true" />
              </a>
              <p className="text-xs text-gray-500 dark:text-gray-400">{t("forms.matrix.waitingForApproval")}</p>
            </div>
          ) : (
            <div className="mt-4 space-y-3">
              {login?.state === "FAILED" && (
                <p className="text-sm text-red-600 dark:text-red-400">{t("forms.matrix.loginError", { error: login.error })}</p>
              )}
              <button
                type="button"
                disabled={loginMutation.isPending}
                onClick={() => loginMutation.mutate(network.id)}
                className="inline-flex justify-center py-2 px-4 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer disabled:cursor-not-allowed disabled:opacity-60"
              >
                {t("forms.matrix.startLogin")}
              </button>
            </div>
          )}
        </div>

        <tokenForm.AppForm>
          <form
            className="py-5"
            onSubmit={(e) => {
              e.preventDefault();
              e.stopPropagation();
              tokenForm.handleSubmit();
            }}
          >
            <div className="px-4 sm:px-6 space-y-1">
              <h2 className="text-lg font-medium text-gray-900 dark:text-white">{t("forms.matrix.accessToken")}</h2>
              <p className="text-sm text-gray-500 dark:text-gray-400">{t("forms.matrix.accessTokenDesc")}</p>
            </div>
            <PasswordFieldWide name="access_token" label={t("forms.matrix.accessToken")} required={true} autoComplete="off" />
            <div className="px-4 sm:px-6 flex justify-end">
              <button
                type="submit"
                disabled={tokenMutation.isPending}
                className="inline-flex justify-center py-2 px-4 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer disabled:cursor-not-allowed disabled:opacity-60"
              >
                {t("forms.matrix.saveToken")}
              </button>
            </div>
          </form>
        </tokenForm.AppForm>
      </div>

      <div className="shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-5 sm:px-6">
        <div className="flex justify-end">
          <button
            type="button"
            className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer"
            onClick={close}
          >
            {t("forms.matrix.close")}
          </button>
        </div>
      </div>
    </div>
  );
}
