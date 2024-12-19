/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useSuspenseQuery } from "@tanstack/react-query";
import { getRouteApi } from "@tanstack/react-router";
import { toast } from "react-hot-toast";
import Select from "react-select";

import { APIClient } from "@api/APIClient";
import { ConfigQueryOptions } from "@api/queries";
import { SettingsKeys } from "@api/query_keys";
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

const SelectWrapper = ({ id, value, onChange, options }: SelectWrapperProps) => (
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
    placeholder="Choose a type"
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

function LogSettings() {
  const settingsLogRoute = getRouteApi("/auth/authenticated-routes/settings/logs");
  const { queryClient} =  settingsLogRoute.useRouteContext();

  const configQuery = useSuspenseQuery(ConfigQueryOptions())

  const config = configQuery.data

  const setLogLevelUpdateMutation = useMutation({
    mutationFn: (value: string) => APIClient.config.update({ log_level: value }),
    onSuccess: () => {
      toast.custom((t) => <Toast type="success" body={"Config successfully updated!"} t={t} />);

      queryClient.invalidateQueries({ queryKey: SettingsKeys.config() });
    }
  });

  return (
    <Section
      title="Logs"
      description="Configure log level, log size rotation, etc. You can download your old log files below."
    >
      <div className="-mx-4 lg:col-span-9">
        <div className="divide-y divide-gray-200 dark:divide-gray-750">
          {!configQuery.isLoading && config && (
            <form className="divide-y divide-gray-200 dark:divide-gray-750" action="#" method="POST">
              <RowItem label="Path" value={config?.log_path} title="Set in config.toml" emptyText="Not set!"/>
              <RowItem
                className="sm:col-span-1"
                label="Level"
                title="Log level"
                value={
                  <SelectWrapper
                    id="log_level"
                    value={config?.log_level}
                    options={LogLevelOptions}
                    onChange={(value: SelectOption) => setLogLevelUpdateMutation.mutate(value.value)}
                  />
                }
              />
              <RowItem label="Max Size" value={config?.log_max_size} title="Set in config.toml" rightSide="MB"/>
              <RowItem label="Max Backups" value={config?.log_max_backups} title="Set in config.toml"/>
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
