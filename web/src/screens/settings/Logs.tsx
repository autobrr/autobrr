import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "react-hot-toast";
import Select, { components, ControlProps, InputProps, MenuProps, OptionProps } from "react-select";

import { APIClient } from "../../api/APIClient";
import { GithubRelease } from "../../types/Update";
import Toast from "../../components/notifications/Toast";
import { LogLevelOptions, SelectOption } from "../../domain/constants";
import { LogFiles } from "../Logs";

interface RowItemProps {
  label: string;
  value?: string;
  title?: string;
  emptyText?: string;
  newUpdate?: GithubRelease;
}

const RowItem = ({ label, value, title, emptyText }: RowItemProps) => {
  return (
    <div className="py-4 sm:py-5 sm:grid sm:grid-cols-4 sm:gap-4 sm:px-6">
      <dt className="font-medium text-gray-500 dark:text-white" title={title}>{label}:</dt>
      <dd className="mt-1 text-gray-900 dark:text-white sm:mt-0 sm:col-span-2 break-all">
        <span className="px-1 py-0.5 bg-gray-200 dark:bg-gray-700 rounded shadow">{value ? value : emptyText}</span>
      </dd>
    </div>
  );
};

interface RowItemNumberProps {
  label: string;
  value?: string | number;
  title?: string;
  unit?: string;
}

const RowItemNumber = ({ label, value, title, unit }: RowItemNumberProps) => {
  return (
    <div className="py-4 sm:py-5 sm:grid sm:grid-cols-4 sm:gap-4 sm:px-6">
      <dt className="font-medium text-gray-500 dark:text-white" title={title}>{label}:</dt>
      <dd className="mt-1 text-gray-900 dark:text-white sm:mt-0 sm:col-span-2 break-all">
        <span className="px-1 py-0.5 bg-gray-200 dark:bg-gray-700 rounded shadow">{value}</span>
        {unit &&
          <span className="ml-1 text-sm text-gray-700 dark:text-gray-400">{unit}</span>
        }
      </dd>
    </div>
  );
};

const Input = (props: InputProps) => {
  return (
    <components.Input
      {...props}
      inputClassName="outline-none border-none shadow-none focus:ring-transparent"
      className="text-gray-400 dark:text-gray-100"
      children={props.children}
    />
  );
};

const Control = (props: ControlProps) => {
  return (
    <components.Control
      {...props}
      className="p-1 block w-full dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:text-gray-100 sm:text-sm"
      children={props.children}
    />
  );
};

const Menu = (props: MenuProps) => {
  return (
    <components.Menu
      {...props}
      className="dark:bg-gray-800 border border-gray-300 dark:border-gray-700 dark:text-gray-400 rounded-md shadow-sm"
      children={props.children}
    />
  );
};

const Option = (props: OptionProps) => {
  return (
    <components.Option
      {...props}
      className="dark:text-gray-400 dark:bg-gray-800 dark:hover:bg-gray-900 dark:focus:bg-gray-900"
      children={props.children}
    />
  );
};

const RowItemSelect = ({ id, title, label, value, options, onChange }: any) => {
  return (
    <div className="py-4 sm:py-5 sm:grid sm:grid-cols-4 sm:gap-4 sm:px-6">
      <dt className="font-medium text-gray-500 dark:text-white" title={title}>{label}:</dt>
      <dd className="mt-1 text-gray-900 dark:text-white sm:mt-0 sm:col-span-2 break-all">
        <Select
          id={id}
          components={{ Input, Control, Menu, Option }}
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
      </dd>
    </div>
  );
};

function LogSettings() {
  const { isLoading, data } = useQuery({
    queryKey: ["config"],
    queryFn: APIClient.config.get,
    retry: false,
    refetchOnWindowFocus: false,
    onError: err => console.log(err)
  });

  const queryClient = useQueryClient();

  const setLogLevelUpdateMutation = useMutation({
    mutationFn: (value: string) => APIClient.config.update({ log_level: value }),
    onSuccess: () => {
      toast.custom((t) => <Toast type="success" body={"Config successfully updated!"} t={t}/>);

      queryClient.invalidateQueries({ queryKey: ["config"] });
    }
  });

  return (
    <div className="divide-y divide-gray-200 dark:divide-gray-700 lg:col-span-9">
      <div className="py-6 px-4 sm:p-6 lg:pb-8">
        <div>
          <h2 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">Logs</h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Set level, size etc.
          </p>
        </div>

      </div>

      <div className="divide-y divide-gray-200 dark:divide-gray-700">
        <div className="px-4 py-5 sm:p-0">
          <form className="divide-y divide-gray-200 dark:divide-gray-700 lg:col-span-9" action="#" method="POST">
            {!isLoading && data && (
              <dl className="sm:divide-y divide-gray-200 dark:divide-gray-700">
                <RowItem label="Path" value={data?.log_path} title="Set in config.toml" emptyText="Not set!" />
                <RowItemSelect id="log_level" label="Level" value={data?.log_level} title="Log level" options={LogLevelOptions} onChange={(value: SelectOption) => setLogLevelUpdateMutation.mutate(value.value)} />
                <RowItemNumber label="Max Size" value={data?.log_max_size} title="Set in config.toml" unit="MB" />
                <RowItemNumber label="Max Backups" value={data?.log_max_backups} title="Set in config.toml" />
              </dl>
            )}
          </form>
        </div>

        <div className="mt-4 flex flex-col py-4 px-4 sm:px-6">
          <LogFiles />
        </div>

        {/*<div className="mt-4 flex justify-end py-4 px-4 sm:px-6">*/}
        {/*  <button*/}
        {/*    type="button"*/}
        {/*    className="inline-flex justify-center rounded-md border border-gray-300 bg-white py-2 px-4 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"*/}
        {/*  >*/}
        {/*    Cancel*/}
        {/*  </button>*/}
        {/*  <button*/}
        {/*    type="submit"*/}
        {/*    className="ml-5 inline-flex justify-center rounded-md border border-transparent bg-blue-700 py-2 px-4 text-sm font-medium text-white shadow-sm hover:bg-blue-800 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"*/}
        {/*  >*/}
        {/*    Save*/}
        {/*  </button>*/}
        {/*</div>*/}
      </div>
    </div>
  );
}

export default LogSettings;