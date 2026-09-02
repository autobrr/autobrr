/*
 * Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { XMarkIcon } from "@heroicons/react/24/solid";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useSelector } from "@tanstack/react-form";
import { useTranslation } from "react-i18next";

import { AddFormProps } from "@forms/_shared";
import { useAppForm } from "@hooks/form";
import { DEBUG } from "@components/debug.tsx";
import { PasswordFieldWide, SwitchGroupWide, TextFieldWide } from "@components/inputs";
import { SelectFieldBasic } from "@components/inputs/select_wide";
import { ProxyTypeOptions } from "@domain/constants";
import { APIClient } from "@api/APIClient";
import { FeedKeys, IndexerKeys, IrcKeys, ProxyKeys } from "@api/query_keys";
import { ProxyUsageQueryOptions } from "@api/queries";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { SlideOver, SlideOverShell, SlideOverTitle } from "@components/panels";

export function ProxyAddForm({ isOpen, toggle }: AddFormProps) {
  return (
    <SlideOverShell isOpen={isOpen} toggle={toggle}>
      <ProxyAddFormPanel toggle={toggle} />
    </SlideOverShell>
  );
}

interface ProxyAddFormPanelProps {
  toggle: () => void;
}

function ProxyAddFormPanel({ toggle }: ProxyAddFormPanelProps) {
  const { t } = useTranslation("settings");
  const queryClient = useQueryClient();

  const createMutation = useMutation({
    mutationFn: (req: ProxyCreate) => APIClient.proxy.store(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ProxyKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.proxy.added")} t={toastInstance} />);
      toggle();
    },
    onError: () => {
      toast.custom((toastInstance) => <Toast type="error" body={t("forms.proxy.addFailed")} t={toastInstance} />);
    }
  });

  const onSubmit = (formData: ProxyCreate) => {
    createMutation.mutate(formData);
  }

  const testMutation = useMutation({
    mutationFn: (data: Proxy) => APIClient.proxy.test(data),
    onError: (err) => {
      console.error(err);
    }
  });

  const testProxy = (data: unknown) => testMutation.mutate(data as Proxy);

  const initialValues: ProxyCreate = {
    enabled: true,
    name: "Proxy",
    type: "SOCKS5",
    addr: "socks5://ip:port",
    user: "",
    pass: "",
  }

  const form = useAppForm({
    defaultValues: initialValues,
    onSubmit: ({ value }) => onSubmit(value)
  });

  const values = useSelector(form.store, (state) => state.values);

  return (
    <form.AppForm>
      <form
        className="h-full min-h-0 flex flex-col bg-white dark:bg-gray-800"
        onSubmit={(e) => {
          e.preventDefault();
          e.stopPropagation();
          form.handleSubmit();
        }}
      >
        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
            <div className="flex items-start justify-between space-x-3">
              <div className="space-y-1">
                <SlideOverTitle>
                  {t("forms.proxy.addTitle")}
                </SlideOverTitle>
                <p className="text-sm text-gray-500 dark:text-gray-200">
                  {t("forms.proxy.addDescription")}
                </p>
              </div>
              <div className="h-7 flex items-center">
                <button
                  type="button"
                  className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-hidden focus:ring-2 focus:ring-blue-500"
                  onClick={toggle}
                >
                  <span className="sr-only">{t("forms.proxy.closePanel")}</span>
                  <XMarkIcon className="h-6 w-6" aria-hidden="true" />
                </button>
              </div>
            </div>
          </div>

          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            <TextFieldWide name="name" label={t("forms.proxy.name")} defaultValue="" required={true} />
            <SwitchGroupWide name="enabled" label={t("forms.proxy.enabled")} />
            <SelectFieldBasic
              name="type"
              label={t("forms.proxy.proxyType")}
              options={ProxyTypeOptions}
              tooltip={<span>{t("forms.proxy.proxyTypeTooltip")}</span>}
              help={t("forms.proxy.proxyTypeHelp")}
            />
            <TextFieldWide name="addr" label={t("forms.proxy.addr")} required={true} help={t("forms.proxy.addrHelp")} autoComplete="off"/>
          </div>

          <div>
            <TextFieldWide name="user" label={t("forms.proxy.user")} help={t("forms.proxy.userHelp")} autoComplete="off" />
            <PasswordFieldWide name="pass" label={t("forms.proxy.pass")} help={t("forms.proxy.passHelp")} autoComplete="off"/>
          </div>

          <DEBUG values={values}/>
        </div>

        <div className="shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-5 sm:px-6">
          <div className="space-x-3 flex justify-end">
            <button
              type="button"
              className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
              onClick={() => testProxy(values)}
            >
              {t("forms.proxy.test")}
            </button>
            <button
              type="button"
              className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
              onClick={toggle}
            >
              {t("forms.proxy.cancel")}
            </button>
            <button
              type="submit"
              className="inline-flex justify-center py-2 px-4 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
            >
              {t("forms.proxy.save")}
            </button>
          </div>
        </div>
      </form>
    </form.AppForm>
  );
}


interface UpdateFormProps<T> {
  isOpen: boolean;
  toggle: () => void;
  data: T;
}

export function ProxyUpdateForm({ isOpen, toggle, data }: UpdateFormProps<Proxy>) {
  const { t } = useTranslation("settings");
  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: (req: Proxy) => APIClient.proxy.update(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ProxyKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.proxy.updated", { name: data.name })} t={toastInstance} />);
      toggle();
    },
    onError: () => {
      toast.custom((toastInstance) => <Toast type="error" body={t("forms.proxy.updateFailed")} t={toastInstance} />);
    }
  });

  const onSubmit = (formData: Proxy) => {
    updateMutation.mutate(formData);
  }

  const deleteMutation = useMutation({
    mutationFn: (proxyId: number) => APIClient.proxy.delete(proxyId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ProxyKeys.lists() });
      queryClient.invalidateQueries({ queryKey: IndexerKeys.lists() });
      queryClient.invalidateQueries({ queryKey: IrcKeys.lists() });
      queryClient.invalidateQueries({ queryKey: FeedKeys.lists() });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.proxy.deleted", { name: data.name })} t={toastInstance}/>);
    }
  });

  const deleteFn = () => deleteMutation.mutate(data.id);

  const { data: usage } = useQuery(ProxyUsageQueryOptions(data.id, isOpen));

  const testMutation = useMutation({
    mutationFn: (data: Proxy) => APIClient.proxy.test(data),
    onError: (err) => {
      console.error(err);
    }
  });

  const testProxy = (data: unknown) => testMutation.mutate(data as Proxy);

  const initialValues: Proxy = {
    id: data.id,
    enabled: data.enabled,
    name: data.name,
    type: data.type,
    addr: data.addr,
    user: data.user,
    pass: data.pass,
  }

  return (
    <SlideOver<Proxy>
      title={t("forms.proxy.title")}
      initialValues={initialValues}
      onSubmit={onSubmit}
      deleteAction={deleteFn}
      deleteWarning={<ProxyUsageWarning usage={usage} />}
      testFn={testProxy}
      isOpen={isOpen}
      toggle={toggle}
      type="UPDATE"
    >
      {() => (
        <div>
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            <TextFieldWide name="name" label={t("forms.proxy.name")} defaultValue="" required={true}/>
            <SwitchGroupWide name="enabled" label={t("forms.proxy.enabled")}/>
            <SelectFieldBasic
              name="type"
              label={t("forms.proxy.proxyType")}
              required={true}
              options={ProxyTypeOptions}
              tooltip={<span>{t("forms.proxy.proxyTypeTooltip")}</span>}
              help={t("forms.proxy.proxyTypeHelp")}
            />
            <TextFieldWide name="addr" label={t("forms.proxy.addr")} required={true} help={t("forms.proxy.addrHelp")} autoComplete="off"/>
          </div>

          <div>
            <TextFieldWide name="user" label={t("forms.proxy.user")} help={t("forms.proxy.userHelp")} autoComplete="off"/>
            <PasswordFieldWide name="pass" label={t("forms.proxy.pass")} help={t("forms.proxy.passHelp")} autoComplete="off"/>
          </div>
        </div>
      )}
    </SlideOver>
  );
}

interface ProxyUsageWarningProps {
  usage?: ProxyUsage;
}

function ProxyUsageWarning({ usage }: ProxyUsageWarningProps) {
  const { t } = useTranslation("settings");

  if (!usage) {
    return null;
  }

  const groups = [
    { label: t("forms.proxy.usageIndexers"), items: usage.indexers },
    { label: t("forms.proxy.usageIrcNetworks"), items: usage.irc_networks },
    { label: t("forms.proxy.usageFeeds"), items: usage.feeds }
  ].filter((group) => group.items.length > 0);

  if (!groups.length) {
    return null;
  }

  return (
    <div className="mt-4 rounded-md border border-amber-300 dark:border-amber-500/40 bg-amber-50 dark:bg-amber-400/10 px-3 py-2 text-sm">
      <p className="font-medium text-amber-800 dark:text-amber-300">{t("forms.proxy.usageTitle")}</p>
      <p className="mt-1 text-amber-700 dark:text-amber-200">{t("forms.proxy.usageText")}</p>
      {groups.map((group) => (
        <div key={group.label} className="mt-2">
          <span className="text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-400">{group.label}</span>
          <ul className="mt-1 flex flex-wrap gap-1">
            {group.items.map((item) => (
              <li key={item.id} className="inline-flex items-center rounded-md bg-gray-100 dark:bg-gray-400/10 px-2 py-1 text-xs font-medium text-gray-700 dark:text-gray-300">
                {item.name}
              </li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  );
}
