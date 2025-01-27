/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useEffect } from "react";
import { useForm } from "react-hook-form";
import { useMutation, useQuery, useQueryErrorResetBoundary } from "@tanstack/react-query";
import { getRouteApi, useRouter } from "@tanstack/react-router";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faOpenid } from "@fortawesome/free-brands-svg-icons";

import { RocketLaunchIcon } from "@heroicons/react/24/outline";

import { APIClient } from "@api/APIClient";
import toast from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { Tooltip } from "@components/tooltips/Tooltip";
import { PasswordInput, TextInput } from "@components/inputs/text";

import Logo from "@app/logo.svg?react";
import { AuthContext, AuthInfo } from "@utils/Context";
// import { WarningAlert } from "@components/alerts";

type LoginFormFields = {
  username: string;
  password: string;
};

type ValidateResponse = {
  username?: AuthInfo['username'];
  auth_method?: AuthInfo['authMethod'];
}

export const Login = () => {
  const [auth, setAuth] = AuthContext.use();
  const queryErrorResetBoundary = useQueryErrorResetBoundary()
  const router = useRouter()

  const loginRoute = getRouteApi('/login');
  const search = loginRoute.useSearch();

  // Query to check if onboarding is available
  const { data: canOnboard } = useQuery({
    queryKey: ["can-onboard"],
    queryFn: async () => {
      try {
        await APIClient.auth.canOnboard();
        return true;
      } catch {
        return false;
      }
    },
  });

  // Query to check if OIDC is enabled
  const { data: oidcConfig } = useQuery({
    queryKey: ["oidc-config"],
    queryFn: async () => {
      const config = await APIClient.auth.getOIDCConfig();
      console.debug("OIDC config:", config);
      return config;
    },
  });

  const { handleSubmit, register, formState } = useForm<LoginFormFields>({
    defaultValues: { username: "", password: "" },
    mode: "onBlur"
  });

  useEffect(() => {
    queryErrorResetBoundary.reset()
    // remove user session when visiting login page
    AuthContext.reset();

    // Check if this is an OIDC callback
    const urlParams = new URLSearchParams(window.location.search);
    const code = urlParams.get('code');
    const state = urlParams.get('state');

    if (code && state) {
      // This is an OIDC callback, validate the session
      APIClient.auth.validate().then((response: ValidateResponse) => {
        // If validation succeeds, set the user as logged in
        setAuth({
          isLoggedIn: true,
          username: response.username || 'unknown',
          authMethod: response.auth_method || (oidcConfig?.enabled ? 'oidc' : 'password')
        });
        router.invalidate();
      }).catch((error) => {
        // If validation fails, show an error
        toast.custom((t) => (
          <Toast type="error" body={error.message || "OIDC authentication failed"} t={t} />
        ));
      });
    }
  }, [queryErrorResetBoundary, oidcConfig, setAuth, router]);

  const loginMutation = useMutation({
    mutationFn: (data: LoginFormFields) => APIClient.auth.login(data.username, data.password),
    onSuccess: (_, variables: LoginFormFields) => {
      queryErrorResetBoundary.reset()
      setAuth({
        isLoggedIn: true,
        username: variables.username,
        authMethod: 'password'
      });
      router.invalidate()
    },
    onError: (error) => {
      toast.custom((t) => (
        <Toast type="error" body={error.message || "An error occurred!"} t={t} />
      ));
    }
  });

  const onSubmit = (data: LoginFormFields) => loginMutation.mutate(data);

  const handleOIDCLogin = () => {
    if (oidcConfig?.enabled && oidcConfig.authorizationUrl) {
      window.location.href = oidcConfig.authorizationUrl;
    }
  };

  React.useLayoutEffect(() => {
    if (auth.isLoggedIn && search.redirect) {
      router.history.push(search.redirect)
    } else if (auth.isLoggedIn) {
      router.history.push("/")
    }
  }, [auth.isLoggedIn, search.redirect]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="flex min-h-full flex-1 flex-col justify-center py-12 sm:px-6 lg:px-8">
      <div className="sm:mx-auto sm:w-full sm:max-w-md">
        <Logo className="mx-auto h-12 w-auto" />
        <h2 className="mt-6 text-center text-2xl font-bold tracking-tight text-gray-900 dark:text-gray-200">
          autobrr
        </h2>
      </div>

      {/* Wait for OIDC config to load before rendering any login forms */}
      {typeof oidcConfig !== 'undefined' && (
        <div className="mt-10 sm:mx-auto sm:w-full sm:max-w-[480px]">
          <div className={`px-6 ${(!canOnboard && (!oidcConfig?.enabled || !oidcConfig?.disableBuiltInLogin)) ? 'py-12 bg-white dark:bg-gray-800 shadow-sm sm:rounded-lg sm:px-12 border border-gray-150 dark:border-gray-775' : ''}`}>
            {/* Built-in login form */}
            {!canOnboard && (!oidcConfig?.enabled || !oidcConfig?.disableBuiltInLogin) && (
              <>
                <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
                  <TextInput<LoginFormFields>
                    name="username"
                    id="username"
                    label="Username"
                    type="text"
                    register={register}
                    rules={{ required: "Username is required" }}
                    errors={formState.errors}
                    autoComplete="username"
                  />
                  <PasswordInput<LoginFormFields>
                    name="password"
                    id="password"
                    label="Password"
                    register={register}
                    rules={{ required: "Password is required" }}
                    errors={formState.errors}
                    autoComplete="current-password"
                  />

                  <div className="flex items-center justify-end">
                    <div className="text-sm">
                      <Tooltip
                        label={
                          <div className="flex flex-row items-center cursor-pointer text-gray-700 dark:text-gray-200">
                            Forgot password? <svg className="ml-1 w-3 h-3 text-gray-500 dark:text-gray-400 fill-current" viewBox="0 0 72 72"><path d="M32 2C15.432 2 2 15.432 2 32s13.432 30 30 30s30-13.432 30-30S48.568 2 32 2m5 49.75H27v-24h10v24m-5-29.5a5 5 0 1 1 0-10a5 5 0 0 1 0 10" /></svg>
                          </div>
                        }
                      >
                        <p className="py-1">Reset via terminal: <code>autobrrctl --config /home/username/.config/autobrr change-password $USERNAME</code></p>
                      </Tooltip>
                    </div>
                  </div>

                <button
                  type="submit"
                  className="w-full flex items-center justify-center py-2 px-4 border border-transparent rounded-md shadow-xs text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                >
                  <RocketLaunchIcon className="w-4 h-4 mr-1.5" />
                  Sign in
                </button>
              </form>

                {oidcConfig?.enabled && (
                  <div className="relative mt-10">
                    <div aria-hidden="true" className="absolute inset-0 flex items-center">
                      <div className="w-full border-t border-gray-200 dark:border-gray-700" />
                    </div>
                    <div className="relative flex justify-center text-sm">
                      <span className="bg-white dark:bg-gray-800 px-6 text-gray-900 dark:text-gray-200">Or continue with</span>
                    </div>
                  </div>
                )}
              </>
            )}

            {/* OIDC button */}
            {oidcConfig?.enabled && (
              <div className={(!canOnboard && !oidcConfig?.disableBuiltInLogin) ? 'mt-6' : ''}>
                <button
                  type="button"
                  onClick={handleOIDCLogin}
                  className="w-full flex items-center justify-center gap-3 py-2 px-4 border border-gray-300 dark:border-gray-700 rounded-md shadow-xs text-sm font-medium text-gray-900 dark:text-gray-200 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
                >
                  <FontAwesomeIcon icon={faOpenid} className="h-5 w-5" />
                  <span>OpenID Connect</span>
                </button>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
