/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { ExclamationTriangleIcon, MegaphoneIcon, XMarkIcon } from "@heroicons/react/24/outline";
import { useTranslation } from "react-i18next";

import { AlertsQueryOptions } from "@api/queries";
import { ExternalLink } from "@components/ExternalLink";
import { SettingsContext } from "@utils/Context";

export const AlertBanner = () => {
  const { t } = useTranslation("common");

  const { data: alerts } = useQuery(AlertsQueryOptions());
  const dismissedUpdate = SettingsContext.useSelector((s) => s.dismissedUpdate);

  const update = alerts?.find((alert) => alert.kind === "APP_UPDATE");
  const ircNetworks = alerts?.filter((alert) => alert.kind === "IRC_UNHEALTHY") ?? [];
  const lists = alerts?.filter((alert) => alert.kind === "LIST_REFRESH_ERROR") ?? [];

  const dismissUpdate = (version: string) => {
    SettingsContext.set((prev) => ({ ...prev, dismissedUpdate: version }));
  };

  return (
    <>
      {update && update.subject !== dismissedUpdate && (
        <div className="flex mt-4 py-2 bg-blue-500 rounded-sm items-center">
          <ExternalLink href={update.url ?? ""} className="flex flex-1 justify-center">
            <MegaphoneIcon className="h-6 w-6 text-blue-100" />
            <span className="text-blue-100 font-medium mx-3">{t("header.newUpdateAvailable")}</span>
            <AlertChip variant="info">{update.subject}</AlertChip>
          </ExternalLink>
          <button
            type="button"
            onClick={() => dismissUpdate(update.subject)}
            className="mr-2 p-1 rounded-md text-blue-100 hover:bg-blue-600 cursor-pointer"
          >
            <span className="sr-only">{t("header.dismiss")}</span>
            <XMarkIcon className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>
      )}

      {ircNetworks.length > 0 && (
        <ErrorBanner
          to="/settings/irc"
          label={ircNetworks.length === 1 ? t("header.ircNetworkUnhealthy") : t("header.multipleIrcNetworksUnhealthy")}
          alerts={ircNetworks}
        />
      )}

      {lists.length > 0 && (
        <ErrorBanner
          to="/settings/lists"
          label={lists.length === 1 ? t("header.listRefreshFailed") : t("header.multipleListRefreshesFailed")}
          alerts={lists}
        />
      )}
    </>
  );
};

interface ErrorBannerProps {
  to: "/settings/irc" | "/settings/lists";
  label: string;
  alerts: Alert[];
}

const ErrorBanner = ({ to, label, alerts }: ErrorBannerProps) => (
  <Link to={to} className="flex flex-wrap mt-4 py-2 px-3 gap-y-1 bg-red-500 rounded-sm justify-center items-center cursor-pointer">
    <ExclamationTriangleIcon className="h-6 w-6 text-red-100" />
    <span className="text-red-100 font-medium mx-3">{label}</span>
    <span className="flex flex-wrap justify-center gap-1">
      {alerts.map((alert) => (
        <AlertChip key={`${alert.kind}-${alert.subject_id}`} variant="error" title={alert.message}>
          {alert.subject}
        </AlertChip>
      ))}
    </span>
  </Link>
);

interface AlertChipProps {
  variant: "info" | "error";
  title?: string;
  children: string;
}

const AlertChip = ({ variant, title, children }: AlertChipProps) => (
  <span
    title={title}
    className={variant === "info"
      ? "inline-flex items-center rounded-md bg-blue-100 px-2.5 py-0.5 text-sm font-medium text-blue-800"
      : "inline-flex items-center rounded-md bg-red-100 px-2.5 py-0.5 text-sm font-medium text-red-800"}
  >
    {children}
  </span>
);
