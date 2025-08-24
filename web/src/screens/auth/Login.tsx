/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useEffect } from "react";
import { useMutation, useQuery, useQueryErrorResetBoundary } from "@tanstack/react-query";
import { getRouteApi, useRouter } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form"
import { Checkbox, Field, Label } from "@headlessui/react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faOpenid } from "@fortawesome/free-brands-svg-icons";
import { RocketLaunchIcon } from "@heroicons/react/24/outline";
import { EyeIcon, EyeSlashIcon } from "@heroicons/react/24/solid";

import { APIClient } from "@api/APIClient";
import toast from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { Tooltip } from "@components/tooltips/Tooltip";

import Logo from "@app/logo.svg?react";
import { AuthContext, AuthInfo } from "@utils/Context";
import { classNames } from "@utils";
import { useToggle } from "@hooks/hooks";

type LoginFormFields = {
    username: string;
    password: string;
    remember_me: boolean;
};

type ValidateResponse = {
    username?: AuthInfo['username'];
    auth_method?: AuthInfo['authMethod'];
    profile_picture?: AuthInfo['profilePicture'];
}

export const Login = () => {
    const [auth, setAuth] = AuthContext.use();
    const queryErrorResetBoundary = useQueryErrorResetBoundary()
    const router = useRouter()

    const loginRoute = getRouteApi('/login');
    const search = loginRoute.useSearch();

    // Query to check if onboarding is available
    const {data: canOnboard} = useQuery({
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
    const {data: oidcConfig} = useQuery({
        queryKey: ["oidc-config"],
        queryFn: async () => {
            const config = await APIClient.auth.getOIDCConfig();
            console.debug("OIDC config:", config);
            return config;
        },
    });

    const form = useForm({
        defaultValues: {
            username: "",
            password: "",
            remember_me: true
        },
        onSubmit: (data) => {
            console.log("submit form", data)

            loginMutation.mutate(data.value)
        }
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
                    authMethod: response.auth_method || (oidcConfig?.enabled ? 'oidc' : 'password'),
                    profilePicture: response.profile_picture,
                });
                router.invalidate();
            }).catch((error) => {
                // If validation fails, show an error
                toast.custom((t) => (
                    <Toast type="error" body={error.message || "OIDC authentication failed"} t={t}/>
                ));
            });
        }
    }, [queryErrorResetBoundary, oidcConfig, setAuth, router]);

    const loginMutation = useMutation({
        mutationFn: (data: LoginFormFields) => APIClient.auth.login(data.username, data.password, data.remember_me),
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
                <Toast type="error" body={error.message || "An error occurred!"} t={t}/>
            ));
        }
    });

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
    }, [auth.isLoggedIn, search.redirect]) // eslint-disable-line

    return (
        <div className="flex min-h-full flex-1 flex-col justify-center py-12 sm:px-6 lg:px-8">
            <div className="sm:mx-auto sm:w-full sm:max-w-md">
                <Logo className="mx-auto h-12 w-auto"/>
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
                                <form onSubmit={(e) => {
                                    e.preventDefault()
                                    e.stopPropagation()
                                    form.handleSubmit()
                                }} className="space-y-6">
                                    <form.Field
                                        name="username"
                                        validators={{
                                            onChange: ({value}) => !value ? 'Username is required' : undefined,
                                        }}
                                        children={({state, handleChange, handleBlur}) => {
                                            return (
                                                <div className="col-span-12">
                                                    <div>
                                                        <label
                                                            htmlFor="username"
                                                            className="block ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide"
                                                        >
                                                            Username
                                                        </label>
                                                    </div>
                                                    <div className="">
                                                        <input
                                                            type="text"
                                                            id="username"
                                                            onChange={(e) => handleChange(e.target.value)}
                                                            onBlur={handleBlur}
                                                            value={state.value}
                                                            className={classNames(
                                                                "block mt-1 w-full shadow-xs sm:text-sm rounded-md py-2.5 bg-gray-100 dark:bg-gray-850 dark:text-gray-100",
                                                                !state.meta.isValid
                                                                    ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                                                                    : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500"
                                                            )}
                                                        />
                                                        {state.meta.errors && (
                                                            <p className="mt-1 text-sm text-left block text-red-600">{state.meta.errors[0]}</p>
                                                        )}
                                                    </div>
                                                </div>
                                            )
                                        }}
                                    />

                                    <form.Field
                                        name="password"
                                        validators={{
                                            onChange: ({value}) => !value ? 'Password is required' : undefined,
                                        }}
                                        children={({name, state, handleChange, handleBlur }) => (

                                            <div className="col-span-12">
                                            <div>
                                            <label
                                            htmlFor={name}
                                        className="block ml-px text-xs font-bold text-gray-800 dark:text-gray-100 uppercase tracking-wide"
                                    >
                                        Password
                                    </label>
                                </div>
                                <div className="sm:col-span-2 relative">
                                    <PasswordInputField name={name} value={state.value} onChange={handleChange} onBlur={handleBlur} isValid={state.meta.isValid} />
                                </div>
                                {state.meta.errors && (
                                    <p className="mt-1 text-sm text-left block text-red-600">{state.meta.errors[0]}</p>
                                )}
                            </div>

                                        )}
                                    />
                                    <div className="col-span-12">
                                        <div className="flex items-center justify-between">
                                            <form.Field
                                                name="remember_me"
                                                children={({name, state, handleChange, handleBlur}) => (
                                                    <Field className="flex items-center gap-2">
                                                        <Checkbox
                                                            id={name}
                                                            checked={state.value}
                                                            onChange={handleChange}
                                                            onBlur={handleBlur}
                                                            className="group block size-4 rounded border bg-white data-checked:bg-blue-500"
                                                        >
                                                            <svg className="stroke-white opacity-0 group-data-checked:opacity-100" viewBox="0 0 14 14" fill="none">
                                                                <path d="M3 8L6 11L11 3.5" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"/>
                                                            </svg>
                                                        </Checkbox>
                                                        <Label className="text-sm text-gray-700 dark:text-gray-200">Remember me</Label>
                                                    </Field>
                                                )}
                                            />
                                            <div className="text-sm">
                                                <Tooltip
                                                    label={
                                                        <div className="flex flex-row items-center cursor-pointer text-gray-700 dark:text-gray-200">
                                                            Forgot password? <svg className="ml-1 w-3 h-3 text-gray-500 dark:text-gray-400 fill-current" viewBox="0 0 72 72">
                                                            <path d="M32 2C15.432 2 2 15.432 2 32s13.432 30 30 30s30-13.432 30-30S48.568 2 32 2m5 49.75H27v-24h10v24m-5-29.5a5 5 0 1 1 0-10a5 5 0 0 1 0 10"/>
                                                        </svg>
                                                        </div>
                                                    }
                                                >
                                                    <p className="py-1">Reset via terminal: <code>autobrrctl --config /home/username/.config/autobrr change-password $USERNAME</code></p>
                                                </Tooltip>
                                            </div>
                                        </div>
                                    </div>

                                    <button
                                        type="submit"
                                        className="w-full flex items-center justify-center py-2 px-4 border border-transparent rounded-md shadow-xs text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                                    >
                                        <RocketLaunchIcon className="w-4 h-4 mr-1.5"/>
                                        Sign in
                                    </button>
                                </form>

                                {oidcConfig?.enabled && (
                                    <div className="relative mt-10">
                                        <div aria-hidden="true" className="absolute inset-0 flex items-center">
                                            <div className="w-full border-t border-gray-200 dark:border-gray-700"/>
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
                                    <FontAwesomeIcon icon={faOpenid} className="h-5 w-5"/>
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

interface PasswordFieldProps {
    name: string;
    value: string;
    onChange: (value: string) => void;
    onBlur: () => void;
    isValid: boolean;
}

function PasswordInputField({name, value, onChange, onBlur, isValid}: PasswordFieldProps) {
    const [isVisible, toggleVisibility] = useToggle(false);

    return (
            <div className="sm:col-span-2 relative">
                <input
                    // type="text"
                    type={isVisible ? "text" : "password"}
                    id={name}
                    onChange={(e) => onChange(e.target.value)}
                    onBlur={onBlur}
                    value={value}
                    className={classNames(
                        "block mt-1 w-full shadow-xs sm:text-sm rounded-md py-2.5 bg-gray-100 dark:bg-gray-850 dark:text-gray-100",
                        !isValid
                            ? "border-red-500 focus:ring-red-500 focus:border-red-500"
                            : "border-gray-300 dark:border-gray-700 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500"
                    )}
                />
                <div className="absolute inset-y-0 right-0 px-3 flex items-center" onClick={toggleVisibility}>
                    {!isVisible ? <EyeIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true"/> : <EyeSlashIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true"/>}
                </div>
            </div>
    )
}
