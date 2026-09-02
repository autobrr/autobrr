/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useSuspenseQuery } from "@tanstack/react-query";
import { getRouteApi } from "@tanstack/react-router";
import Select from "react-select";
import { useTranslation } from "react-i18next";

import { APIClient } from "@api/APIClient";
import { ConfigQueryOptions } from "@api/queries";
import { SettingsKeys } from "@api/query_keys";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { LogLevelOptions, SelectOption } from "@domain/constants";

import { Section, RowItem } from "./_components";
import * as common from "@components/inputs/common";
import { LogFiles } from "@screens/Logs";

type SelectWrapperProps = {
  id: string;
  value: unknown;
  onChange: any;
  options: unknown[];
};

const SelectWrapper = ({ id, value, onChange, options }: SelectWrapperProps) => {
  const { t } = useTranslation("settings");

  return (
    <Select
      id={id}
      components={{
        Input: common.SelectInput,
        Control: common.SelectControl,
        Menu: common.SelectMenu,
        Option: common.SelectOption,
        IndicatorSeparator: common.IndicatorSeparator,
        DropdownIndicator: common.DropdownIndicator
      }}
      placeholder={t("logsPage.chooseType")}
      styles={{
        singleValue: (base) => ({
          ...base,
          color: "unset"
        })
      }}
      theme={(theme) => ({
        ...theme,
        spacing: {
          ...theme.spacing,
          controlHeight: 30,
          baseUnit: 2
        }
      })}
      value={value && options.find((o: any) => o.value == value)}
      onChange={onChange}
      options={options}
    />
  );
};

function LogSettings() {
  const { t } = useTranslation("settings");
  const settingsLogRoute = getRouteApi("/auth/authenticated-routes/settings/logs");
  const { queryClient} =  settingsLogRoute.useRouteContext();

  const configQuery = useSuspenseQuery(ConfigQueryOptions())

  const config = configQuery.data

  const setLogLevelUpdateMutation = useMutation({
    mutationFn: (value: string) => APIClient.config.update({ log_level: value }),
    onSuccess: () => {
      toast.custom((toastInstance) => <Toast type="success" body={t("logsPage.updated")} t={toastInstance} />);

      queryClient.invalidateQueries({ queryKey: SettingsKeys.config() });
    }
  });

  return (
    <Section
      title={t("logsPage.title")}
      description={t("logsPage.description")}
    >
      <div className="-mx-4 lg:col-span-9">
        <div className="divide-y divide-gray-200 dark:divide-gray-750">
          {!configQuery.isLoading && config && (
            <form className="divide-y divide-gray-200 dark:divide-gray-750" action="#" method="POST">
              <RowItem label={t("logsPage.path")} value={config?.log_path} title={t("logsPage.setInConfig")} emptyText={t("logsPage.notSet")}/>
              <RowItem
                className="sm:col-span-1"
                label={t("logsPage.level")}
                title={t("logsPage.logLevel")}
                value={
                  <SelectWrapper
                    id="log_level"
                    value={config?.log_level}
                    options={LogLevelOptions}
                    onChange={(value: SelectOption) => setLogLevelUpdateMutation.mutate(value.value)}
                  />
                }
              />
              <RowItem label={t("logsPage.maxSize")} value={config?.log_max_size} title={t("logsPage.setInConfig")} rightSide="MB"/>
              <RowItem label={t("logsPage.maxBackups")} value={config?.log_max_backups} title={t("logsPage.setInConfig")}/>
            </form>
          )}

          <div className="px-6 pt-4">
            <LogFiles/>
          </div>
        </div>
      </div>

    </Section>
  );
}

export default LogSettings;
