import { useToggle } from "../../hooks/hooks";
import { Switch } from "@headlessui/react";
import { useQuery } from "react-query";
import { classNames } from "../../utils";
import { DownloadClientAddForm, DownloadClientUpdateForm } from "../../forms";
import { EmptySimple } from "../../components/emptystates";
import { APIClient } from "../../api/APIClient";
import { DownloadClientTypeNameMap } from "../../domain/constants";

interface DLSettingsItemProps {
    client: DownloadClient;
    idx: number;
}

function DownloadClientSettingsListItem({ client, idx }: DLSettingsItemProps) {
  const [updateClientIsOpen, toggleUpdateClient] = useToggle(false);

  return (
    <tr key={client.name} className={idx % 2 === 0 ? "light:bg-white" : "light:bg-gray-50"}>
      <DownloadClientUpdateForm client={client} isOpen={updateClientIsOpen} toggle={toggleUpdateClient} />
            
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
        <Switch
          checked={client.enabled}
          onChange={toggleUpdateClient}
          className={classNames(
            client.enabled ? "bg-teal-500 dark:bg-blue-500" : "bg-gray-200 dark:bg-gray-600",
            "relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          )}
        >
          <span className="sr-only">Use setting</span>
          <span
            aria-hidden="true"
            className={classNames(
              client.enabled ? "translate-x-5" : "translate-x-0",
              "inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200"
            )}
          />
        </Switch>
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">{client.name}</td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">{client.host}</td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">{DownloadClientTypeNameMap[client.type]}</td>
      <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
        <span className="text-indigo-600 dark:text-gray-300 hover:text-indigo-900 cursor-pointer" onClick={toggleUpdateClient}>
                    Edit
        </span>
      </td>
    </tr>
  );
}

function DownloadClientSettings() {
  const [addClientIsOpen, toggleAddClient] = useToggle(false);

  const { error, data } = useQuery(
    "downloadClients",
    APIClient.download_clients.getAll,
    { refetchOnWindowFocus: false }
  );

  if (error)
    return (<p>An error has occurred: </p>);

  return (
    <div className="divide-y divide-gray-200 lg:col-span-9">

      <DownloadClientAddForm isOpen={addClientIsOpen} toggle={toggleAddClient} />

      <div className="py-6 px-4 sm:p-6 lg:pb-8">
        <div className="-ml-4 -mt-4 flex justify-between items-center flex-wrap sm:flex-nowrap">
          <div className="ml-4 mt-4">
            <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">Clients</h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                            Manage download clients.
            </p>
          </div>
          <div className="ml-4 mt-4 flex-shrink-0">
            <button
              type="button"
              className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 dark:bg-blue-600 hover:bg-indigo-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
              onClick={toggleAddClient}
            >
                            Add new
            </button>
          </div>
        </div>

        <div className="flex flex-col mt-6">
          {data && data.length > 0 ?
            <div className="-my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
              <div className="py-2 align-middle inline-block min-w-full sm:px-6 lg:px-8">
                <div className="light:shadow overflow-hidden light:border-b light:border-gray-200 sm:rounded-lg">
                  <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                    <thead className="light:bg-gray-50">
                      <tr>
                        <th
                          scope="col"
                          className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
                        >
                                                    Enabled
                        </th>
                        <th
                          scope="col"
                          className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
                        >
                                                    Name
                        </th>
                        <th
                          scope="col"
                          className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
                        >
                                                    Host
                        </th>
                        <th
                          scope="col"
                          className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
                        >
                                                    Type
                        </th>
                        <th scope="col" className="relative px-6 py-3">
                          <span className="sr-only">Edit</span>
                        </th>
                      </tr>
                    </thead>
                    <tbody className="light:bg-white divide-y divide-gray-200 dark:divide-gray-700">
                      {data && data.map((client, idx) => (
                        <DownloadClientSettingsListItem client={client} idx={idx} key={idx} />
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
            : <EmptySimple title="No download clients" subtitle="Add a new client" buttonText="New client" buttonAction={toggleAddClient} />
          }
        </div>
      </div>
    </div>

  );
}

export default DownloadClientSettings;