import { useToggle } from "../../hooks/hooks";
import { Switch } from "@headlessui/react";
import { useMutation, useQuery, useQueryClient } from "react-query";
import { classNames } from "../../utils";
import { DownloadClientAddForm, DownloadClientUpdateForm } from "../../forms";
import { EmptySimple } from "../../components/emptystates";
import { APIClient } from "../../api/APIClient";
import { DownloadClientTypeNameMap } from "../../domain/constants";
import toast from "react-hot-toast";
import Toast from "../../components/notifications/Toast";

interface DLSettingsItemProps {
    client: DownloadClient;
    idx: number;
}

function DownloadClientSettingsListItem({ client, idx }: DLSettingsItemProps) {
  const [updateClientIsOpen, toggleUpdateClient] = useToggle(false);

  const queryClient = useQueryClient();
  const mutation = useMutation(
    (client: DownloadClient) => APIClient.download_clients.update(client),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(["downloadClients"]);
        toast.custom((t) => <Toast type="success" body={`${client.name} was updated successfully`} t={t}/>);
      }
    }
  );

  const onToggleMutation = (newState: boolean) => {
    mutation.mutate({
      ...client,
      enabled: newState
    });
  };

  return (
    <li key={client.name}>
      <div className="grid grid-cols-12 gap-2 lg:gap-4 items-center py-2">
        <DownloadClientUpdateForm
          client={client}
          isOpen={updateClientIsOpen}
          toggle={toggleUpdateClient}
        />
          <div className="col-span-3 sm:col-span-2 px-4 sm:px-6 py-4 whitespace-nowrap text-sm text-gray-500">
            <Switch
              checked={client.enabled}
              onChange={onToggleMutation}
              className={classNames(
                client.enabled ? "bg-blue-500" : "bg-gray-200 dark:bg-gray-600",
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
          </div>
          <div className="col-span-7 sm:col-span-3 px-1 sm:px-0 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white truncate" title={client.name}>{client.name}</div>
          <div className="hidden sm:block col-span-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 truncate" title={client.host}>{client.host}</div>
          <div className="hidden sm:block col-span-2 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">{DownloadClientTypeNameMap[client.type]}</div>
          <div className="col-span-1 whitespace-nowrap text-center text-sm font-medium">
            <span className="text-blue-600 dark:text-gray-300 hover:text-blue-900 cursor-pointer" onClick={toggleUpdateClient}>
              Edit
            </span>
        </div>
      </div>
    </li>
  );
}

function DownloadClientSettings() {
  const [addClientIsOpen, toggleAddClient] = useToggle(false);

  const { error, data } = useQuery(
    "downloadClients",
    () => APIClient.download_clients.getAll(),
    { refetchOnWindowFocus: false }
  );

  if (error) {
    return <p>Failed to fetch download clients</p>;
  }

  return (
    <div className="lg:col-span-9">

      <DownloadClientAddForm isOpen={addClientIsOpen} toggle={toggleAddClient} />

      <div className="py-6 px-2 lg:pb-8">
        <div className="px-4 -ml-4 -mt-4 flex justify-between items-center flex-wrap sm:flex-nowrap">
          <div className="ml-4 mt-4">
            <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">Clients</h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Manage download clients.
            </p>
          </div>
          <div className="ml-4 mt-4 flex-shrink-0">
            <button
              type="button"
              className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              onClick={toggleAddClient}
            >
              Add new
            </button>
          </div>
        </div>

        <div className="flex flex-col mt-6 px-4">
          {data && data.length > 0 ?
            <section className="light:bg-white dark:bg-gray-800 light:shadow sm:rounded-md">
              <ol className="min-w-full relative">
                <li className="grid grid-cols-12 gap-4 border-b border-gray-200 dark:border-gray-700">
                  <div className="col-span-3 sm:col-span-2 px-2 sm:px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Enabled
                  </div>
                  <div className="col-span-6 sm:col-span-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Name
                  </div>
                  <div className="hidden sm:block col-span-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Host
                  </div>
                  <div className="hidden sm:block col-span-2 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Type
                  </div>
                </li>
                {data && data.map((client, idx) => (
                  <DownloadClientSettingsListItem client={client} idx={idx} key={idx} />
                ))}
              </ol>
            </section>
            : <EmptySimple title="No download clients" subtitle="Add a new client" buttonText="New client" buttonAction={toggleAddClient} />
          }
        </div>
      </div>
    </div>

  );
}

export default DownloadClientSettings;