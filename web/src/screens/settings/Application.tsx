import { useQuery } from "react-query";

import { APIClient } from "../../api/APIClient";
import { Checkbox } from "../../components/Checkbox";
import { SettingsContext } from "../../utils/Context";

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

  return (
    <form className="divide-y divide-gray-200 dark:divide-gray-700 lg:col-span-9" action="#" method="POST">
      <div className="py-6 px-4 sm:p-6 lg:pb-8">
        <div>
          <h2 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">Application</h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Application settings. Change in config.toml and restart to take effect.
          </p>
        </div>

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
      </div>

      <div className="pb-6 divide-y divide-gray-200 dark:divide-gray-700">
        <div className="px-4 py-5 sm:p-0">
          <dl className="sm:divide-y divide-gray-200 dark:divide-gray-700">
            {data?.version ? (
              <div className="py-4 sm:py-5 sm:grid sm:grid-cols-4 sm:gap-4 sm:px-6">
                <dt className="font-medium text-gray-500 dark:text-white">Version:</dt>
                <dd className="mt-1 text-gray-900 dark:text-white sm:mt-0 sm:col-span-2 break-all">
                  {data?.version}
                </dd>
              </div>
            ) : null}
            {data?.commit ? (
              <div className="py-4 sm:py-5 sm:grid sm:grid-cols-4 sm:gap-4 sm:px-6">
                <dt className="font-medium text-gray-500 dark:text-white">Commit:</dt>
                <dd className="mt-1 text-gray-900 dark:text-white sm:mt-0 sm:col-span-2 break-all">{data.commit}</dd>
              </div>
            ) : null}
            {data?.date ? (
              <div className="py-4 sm:py-5 sm:grid sm:grid-cols-4 sm:gap-4 sm:px-6">
                <dt className="font-medium text-gray-500 dark:text-white">Date:</dt>
                <dd className="mt-1 text-gray-900 dark:text-white sm:mt-0 sm:col-span-2 break-all">{data?.date}</dd>
              </div>
            ) : null}
          </dl>
        </div>
        <ul className="divide-y divide-gray-200 dark:divide-gray-700">
          <div className="px-4 sm:px-6 py-1">
            <Checkbox
              label="Debug"
              description="Enable debug mode to get more logs."
              value={settings.debug}
              setValue={(newValue: boolean) => setSettings({
                ...settings,
                debug: newValue
              })}
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
    </form>
  );
}

export default ApplicationSettings;