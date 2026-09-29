/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { lazy, Suspense, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Squares2X2Icon } from "@heroicons/react/24/outline";
import { EyeIcon as SolidEyeIcon, EyeSlashIcon as SolidEyeSlashIcon } from "@heroicons/react/24/solid";

import { DashboardConfigContext, SettingsContext } from "@utils/Context";
import { resolveLayout } from "./widgets";
import type { WidgetLayout } from "./widgets";
import { WidgetBoundary } from "./WidgetBoundary";

// dnd-kit is only needed once the user starts customizing, so the editor is its own chunk.
const loadEditor = () => import("./DashboardGridEditor");
const DashboardGridEditor = lazy(() => loadEditor().then((m) => ({ default: m.DashboardGridEditor })));

const StaticGrid = ({ layout }: { layout: WidgetLayout[] }) => (
  <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-2 sm:gap-5 mt-5">
    {layout.map(({ def, hidden }) =>
      !hidden && (
        <div key={def.id} data-widget-id={def.id} className={`${def.spanClass} h-full min-w-0`}>
          <WidgetBoundary def={def} />
        </div>
      )
    )}
  </div>
);

export const DashboardGrid = () => {
  const { t } = useTranslation("common");
  const config = DashboardConfigContext.useValue();
  const [settings, setSettings] = SettingsContext.use();
  const [editing, setEditing] = useState(false);

  const layout = useMemo(() => resolveLayout(config), [config]);

  const persist = (next: WidgetLayout[]) => {
    DashboardConfigContext.set({
      version: 1,
      widgets: next.map(({ def, hidden }) => ({ id: def.id, hidden }))
    });
  };

  return (
    <div>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-3xl font-bold text-black dark:text-white">
          {t("dashboardStats.title")}
        </h1>
        <div className="flex flex-wrap items-center justify-end gap-2">
          <button
            type="button"
            onClick={() => setSettings((current) => ({ ...current, incognitoMode: !current.incognitoMode }))}
            className="inline-flex size-9 items-center justify-center rounded-md text-gray-500 transition-colors duration-200 ease-in-out hover:bg-gray-100 hover:text-gray-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500 dark:hover:bg-gray-800 dark:hover:text-gray-200 cursor-pointer"
            aria-label={t("releaseTable.goIncognito")}
            title={t("releaseTable.goIncognito")}
          >
            {settings.incognitoMode ? (
              <SolidEyeIcon className="size-4" aria-hidden="true" />
            ) : (
              <SolidEyeSlashIcon className="size-4" aria-hidden="true" />
            )}
          </button>
          {editing ? (
            <>
              <button
                type="button"
                onClick={() => DashboardConfigContext.set({ version: 1, widgets: [] })}
                className="min-h-9 rounded-md px-2.5 py-1.5 text-sm text-gray-500 transition-colors duration-200 ease-in-out hover:bg-gray-100 hover:text-gray-700 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500 dark:hover:bg-gray-800 dark:hover:text-gray-300 cursor-pointer"
              >
                {t("dashboardCustomize.reset")}
              </button>
              <button
                type="button"
                onClick={() => setEditing(false)}
                className="min-h-9 rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white transition-colors duration-200 ease-in-out hover:bg-blue-700 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500 cursor-pointer"
              >
                {t("dashboardCustomize.done")}
              </button>
            </>
          ) : (
            <button
              type="button"
              onClick={() => setEditing(true)}
              onMouseEnter={() => void loadEditor()}
              onFocus={() => void loadEditor()}
              className="flex min-h-9 items-center gap-1.5 rounded-md px-2.5 py-1.5 text-sm text-gray-500 transition-colors duration-200 ease-in-out hover:bg-gray-100 hover:text-gray-700 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500 dark:hover:bg-gray-800 dark:hover:text-gray-300 cursor-pointer"
            >
              <Squares2X2Icon className="h-4 w-4" aria-hidden="true" />
              {t("dashboardCustomize.customize")}
            </button>
          )}
        </div>
      </div>
      {editing && (
        <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {t("dashboardCustomize.hint")}
        </p>
      )}

      {editing ? (
        <Suspense fallback={<StaticGrid layout={layout} />}>
          <DashboardGridEditor layout={layout} persist={persist} />
        </Suspense>
      ) : (
        <StaticGrid layout={layout} />
      )}
    </div>
  );
};
