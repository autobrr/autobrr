import { useMutation, useQuery } from "react-query";
import { APIClient } from "../../api/APIClient";
import { Checkbox } from "../../components/Checkbox";
import { SettingsContext } from "../../utils/Context";
import { GithubRelease } from "../../types/Update";
import { toast } from "react-hot-toast";
import Toast from "../../components/notifications/Toast";
import { queryClient } from "../../App";

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
        {value ? <span className="px-1 py-0.5 bg-gray-200 dark:bg-gray-700 rounded shadow">{value}</span> : emptyText}
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
        <span className="px-1 py-0.5 bg-gray-700 rounded shadow">{value}</span>
        {unit &&
          <span className="ml-1 text-sm text-gray-800 dark:text-gray-400">{unit}</span>
        }
      </dd>
    </div>
  );
};

const RowItemVersion = ({ label, value, title, newUpdate }: RowItemProps) => {
  if (!value)
    return null;

  return (
    <div className="py-4 sm:py-5 sm:grid sm:grid-cols-4 sm:gap-4 sm:px-6">
      <dt className="font-medium text-gray-500 dark:text-white" title={title}>{label}:</dt>
      <dd className="mt-1 text-gray-900 dark:text-white sm:mt-0 sm:col-span-2 break-all">
        <span className="px-1 py-0.5 bg-gray-200 dark:bg-gray-700 rounded shadow">{value}</span>
        {newUpdate && newUpdate.html_url && (
          <span>
            <a href={newUpdate.html_url} target="_blank"><span className="ml-2 inline-flex items-center rounded-md bg-green-100 px-2.5 py-0.5 text-sm font-medium text-green-800">{newUpdate.name} available!</span></a>
          </span>
        )}
      </dd>
    </div>
  );
};

function ApplicationSettings() {
  const [settings, setSettings] = SettingsContext.use();

  const { isLoading, data } = useQuery(
    ["config"],
    () => APIClient.config.get(),
    {
      retry: false,
      refetchOnWindowFocus: false,
      onError: err => console.log(err)
    }
  );

  const { data: updateData } = useQuery(
    ["updates"],
    () => APIClient.updates.getLatestRelease(),
    {
      retry: false,
      refetchOnWindowFocus: false,
      onError: err => console.log(err)
    }
  );

  const checkUpdateMutation = useMutation(
    () => APIClient.updates.check(),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(["updates"]);
      }
    }
  );

  const toggleCheckUpdateMutation = useMutation(
    (value: boolean) => APIClient.config.update({ check_for_updates: value }),
    {
      onSuccess: () => {
        toast.custom((t) => <Toast type="success" body={"Config successfully updated!"} t={t}/>);

        queryClient.invalidateQueries(["config"]);

        checkUpdateMutation.mutate();
      }
    }
  );

  return (
    <div className="divide-y divide-gray-200 dark:divide-gray-700 lg:col-span-9">
      <div className="py-6 px-4 sm:p-6 lg:pb-8">
        <div>
          <h2 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">Application</h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Application settings. Change in config.toml and restart to take effect.
          </p>
        </div>

        <form className="divide-y divide-gray-200 dark:divide-gray-700 lg:col-span-9" action="#" method="POST">
          {!isLoading && data && (
            <div className="mt-6 grid grid-cols-12 gap-6">
              <div className="col-span-6 sm:col-span-4">
                <label htmlFor="host" className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
                  Host
                </label>
                <input
                  type="text"
                  name="host"
                  id="host"
                  value={data.host}
                  disabled={true}
                  className="mt-2 block w-full dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:text-gray-100 sm:text-sm"
                />
              </div>

              <div className="col-span-6 sm:col-span-4">
                <label htmlFor="port" className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
                  Port
                </label>
                <input
                  type="text"
                  name="port"
                  id="port"
                  value={data.port}
                  disabled={true}
                  className="mt-2 block w-full dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:text-gray-100 sm:text-sm"
                />
              </div>

              <div className="col-span-6 sm:col-span-4">
                <label htmlFor="base_url" className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
                  Base url
                </label>
                <input
                  type="text"
                  name="base_url"
                  id="base_url"
                  value={data.base_url}
                  disabled={true}
                  className="mt-2 block w-full dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:text-gray-100 sm:text-sm"
                />
              </div>
            </div>
          )}
        </form>
      </div>

      <div className="divide-y divide-gray-200 dark:divide-gray-700">
        <div className="px-4 py-5 sm:p-0">
          <dl className="sm:divide-y divide-gray-200 dark:divide-gray-700">
            <RowItemVersion label="Version" value={data?.version} newUpdate={updateData ?? undefined} />
            <RowItem label="Commit" value={data?.commit} />
            <RowItem label="Build date" value={data?.date} />
          </dl>
        </div>
        <ul className="divide-y divide-gray-200 dark:divide-gray-700">
          <div className="px-4 sm:px-6 py-1">
            <Checkbox
              label="WebUI Debug mode"
              value={settings.debug}
              setValue={(newValue: boolean) => setSettings({
                ...settings,
                debug: newValue
              })}
            />
          </div>
          <div className="px-4 sm:px-6 py-1">
            <Checkbox
              label="Check for updates"
              description="Get notified of new updates."
              value={data?.check_for_updates ?? true}
              setValue={(newValue: boolean) => {
                toggleCheckUpdateMutation.mutate(newValue);
              }}
            />
          </div>
          <div className="px-4 sm:px-6 py-1">
            <Checkbox
              label="Dark theme"
              description="Switch between dark and light theme."
              value={settings.darkTheme}
              setValue={(newValue: boolean) => setSettings({
                ...settings,
                darkTheme: newValue
              })}
            />
          </div>
        </ul>
      </div>

    </div>
  );
}

export default ApplicationSettings;
