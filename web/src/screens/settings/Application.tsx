/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQuery } from "@tanstack/react-query";
import { getRouteApi } from "@tanstack/react-router";

import { APIClient } from "@api/APIClient";
import { ConfigQueryOptions, UpdatesQueryOptions } from "@api/queries";
import { SettingsKeys } from "@api/query_keys";
import { SettingsContext } from "@utils/Context";
import { Checkbox } from "@components/Checkbox";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { ExternalLink } from "@components/ExternalLink";

import { Section, RowItem } from "./_components";

function ApplicationSettings() {
  const [settings, setSettings] = SettingsContext.use();

  const settingsIndexRoute = getRouteApi("/auth/authenticated-routes/settings/");
  const { queryClient } =  settingsIndexRoute.useRouteContext();

  const { isError:isConfigError, error: configError, data } = useQuery(ConfigQueryOptions());
  if (isConfigError) {
    console.log(configError);
  }

  const { isError, error, data: updateData } = useQuery(UpdatesQueryOptions(data?.check_for_updates === true));
  if (isError) {
    console.log(error);
  }

  const checkUpdateMutation = useMutation({
    mutationFn: APIClient.updates.check,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: SettingsKeys.updates() });
    }
  });

  const toggleCheckUpdateMutation = useMutation({
    mutationFn: (value: boolean) => APIClient.config.update({ check_for_updates: value }).then(() => value),
    onSuccess: (_, value: boolean) => {
      toast.custom(t => <Toast type="success" body={`${value ? "You will now be notified of new updates." : "You will no longer be notified of new updates."}`} t={t} />);
      queryClient.invalidateQueries({ queryKey: SettingsKeys.config() });
      checkUpdateMutation.mutate();
    }
  });

  return (
    <Section
      title="Application"
      description="Application settings. Change in config.toml and restart to take effect."
    >
      <div className="-mx-4 divide-y divide-gray-150 dark:divide-gray-750">
        <form className="mt-6 mb-4" action="#" method="POST">
          {data && (
            <div className="grid grid-cols-12 gap-2 sm:gap-6 px-4 sm:px-6">
              <div className="col-span-12 sm:col-span-4">
                <label htmlFor="host" className="block ml-px text-xs font-bold text-gray-700 dark:text-white uppercase tracking-wide">
                  Host
                </label>
                <input
                  type="text"
                  name="host"
                  id="host"
                  value={data.host}
                  disabled={true}
                  className="mt-1 block w-full sm:text-sm rounded-md border-gray-300 dark:border-gray-750 bg-gray-100 dark:bg-gray-825 dark:text-gray-100"
                />
              </div>

              <div className="col-span-12 sm:col-span-4">
                <label htmlFor="port" className="block ml-px text-xs font-bold text-gray-700 dark:text-white uppercase tracking-wide">
                  Port
                </label>
                <input
                  type="text"
                  name="port"
                  id="port"
                  value={data.port}
                  disabled={true}
                  className="mt-1 block w-full sm:text-sm rounded-md border-gray-300 dark:border-gray-750 bg-gray-100 dark:bg-gray-825 dark:text-gray-100"
                />
              </div>

              <div className="col-span-12 sm:col-span-4">
                <label htmlFor="base_url" className="block ml-px text-xs font-bold text-gray-700 dark:text-white uppercase tracking-wide">
                  Base url
                </label>
                <input
                  type="text"
                  name="base_url"
                  id="base_url"
                  value={data.base_url}
                  disabled={true}
                  className="mt-1 block w-full sm:text-sm rounded-md border-gray-300 dark:border-gray-750 bg-gray-100 dark:bg-gray-825 dark:text-gray-100"
                />
              </div>
            </div>
          )}
        </form>

        <RowItem
          label="Version"
          value={data?.version}
          rightSide={
            updateData && updateData.html_url ? (
              <ExternalLink
                href={updateData.html_url}
                className="ml-2 inline-flex items-center rounded-md bg-green-100 px-2.5 py-0.5 text-sm font-medium text-green-800"
              >
                {updateData.name} available!
              </ExternalLink>
            ) : null
          }
        />
        {data?.commit && <RowItem label="Commit" value={data.commit} />}
        {data?.date && <RowItem label="Build date" value={data.date} />}
        <RowItem label="Application" value={data?.application} />
        <RowItem label="Config path" value={data?.config_dir} />
        <RowItem label="Database" value={data?.database} />
        <div className="py-0.5">
          <Checkbox
            name="debug"
            label="WebUI Debug mode"
            value={settings.debug}
            className="p-4 sm:px-6"
            setValue={
              (newValue: boolean) => setSettings((prevState) => ({
                ...prevState,
                debug: newValue
              }))
            }
          />
        </div>
        <Checkbox
          name="check_for_updates"
          label="Check for updates"
          description="Get notified of new updates."
          value={data?.check_for_updates ?? true}
          className="p-4 sm:px-6"
          setValue={(newValue: boolean) => {
            toggleCheckUpdateMutation.mutate(newValue);
          }}
        />
        <Checkbox
          name="darkTheme"
          label="Dark theme"
          description="Switch between dark and light theme."
          value={settings.darkTheme}
          className="p-4 sm:px-6"
          setValue={
            (newValue: boolean) => setSettings((prevState) => ({
              ...prevState,
              darkTheme: newValue
            }))
          }
        />
      </div>
    </Section>
  );
}

export default ApplicationSettings;
