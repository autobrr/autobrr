import { useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowPathIcon, PaperAirplaneIcon, PencilIcon } from "@heroicons/react/24/solid";
import toast from "react-hot-toast";

import { useToggle } from "@hooks/hooks";
import { APIClient } from "@api/APIClient";
import { classNames, simplifyDate } from "@utils";
import { SettingsContext } from "@utils/Context";
import { IrcNetworkUpdateForm } from "@forms/settings/IrcForms";
import Toast from "@components/notifications/Toast";

import { Network } from "./Network";
import { IRC_KEYS } from "./Shared";
import type { SelectedChannel } from "./Shared";
import { ConfigurationDropdown } from "./ConfigurationDropdown";

type IrcEvent = {
  channel: string;
  nick: string;
  msg: string;
  time: string;
};

const NetworkFilterPredicate = (network: IrcNetworkWithHealth, filter: string) => {
  if (!filter.length) {
    return true;
  }

  if (network.name.toLowerCase().includes(filter)) {
    return true;
  }

  for (let i = 0; i < network.channels.length; ++i) {
    if (network.channels[i].name.toLowerCase().includes(filter)) {
      return true;
    }
  }

  return false;
};

export const Panel = () => {
  const commandBarRef = useRef<HTMLInputElement>(null);

  const [filter, setFilter] = useState("");
  const [inEdit, toggleInEdit] = useToggle(false);

  const [settings,] = SettingsContext.use();
  const [selectedChannel, setSelectedChannel] = useState<SelectedChannel>({
    key: "",
    channel: undefined,
    network: undefined,
  });
  
  const { data: ircNetworks } = useQuery({
    queryKey: IRC_KEYS.lists(),
    queryFn: APIClient.irc.getNetworks,
    refetchOnWindowFocus: false,
    refetchInterval: 3000 // Refetch every 3 seconds
  });

  const queryClient = useQueryClient();
  const restartMutation = useMutation({
    mutationFn: (network: IrcNetworkWithHealth) => APIClient.irc.restartNetwork(network.id),
    onSuccess: (_, network) => {
      queryClient.invalidateQueries({ queryKey: IRC_KEYS.lists() });
      queryClient.invalidateQueries({ queryKey: IRC_KEYS.detail(network.id) });

      toast.custom((t) => <Toast type="success" body={`${network.name} was successfully restarted`} t={t}/>);
    }
  });

  const logs: IrcEvent[] = [
    {
      "channel": "#announces",
      "nick": "_AnnounceBot_",
      "msg": "1",
      "time": "2023-09-13T18:48:54.814124152+02:00"
    },
    {
      "channel": "#announces",
      "nick": "_AnnounceBot_",
      "msg": "2",
      "time": "2023-09-13T18:48:55.450742193+02:00"
    },
    {
      "channel": "#announces",
      "nick": "_AnnounceBot_",
      "msg": "3",
      "time": "2023-09-13T18:48:56.10952651+02:00"
    },
    {
      "channel": "#announces",
      "nick": "_AnnounceBot_",
      "msg": "4",
      "time": "2023-09-13T18:48:56.654417467+02:00"
    },
    {
      "channel": "#announces",
      "nick": "_AnnounceBot_",
      "msg": "5",
      "time": "2023-09-13T18:48:57.285460485+02:00"
    },
    {
      "channel": "#announces",
      "nick": "_AnnounceBot_",
      "msg": "6",
      "time": "2023-09-13T18:48:57.965797077+02:00"
    },
    {
      "channel": "#announces",
      "nick": "_AnnounceBot_",
      "msg": "7",
      "time": "2023-09-13T18:48:58.545520683+02:00"
    },
    {
      "channel": "#announces",
      "nick": "_AnnounceBot_",
      "msg": "8",
      "time": "2023-09-13T18:48:59.571392357+02:00"
    },
    {
      "channel": "#announces",
      "nick": "_AnnounceBot_",
      "msg": "9",
      "time": "2023-09-13T18:49:00.436493858+02:00"
    },
    {
      "channel": "#announces",
      "nick": "_AnnounceBot_",
      "msg": "10",
      "time": "2023-09-13T18:49:02.714897171+02:00"
    },
    {
      "channel": "#announces",
      "nick": "_AnnounceBot_",
      "msg": "11",
      "time": "2023-09-13T18:49:03.619499004+02:00"
    },
    {
      "channel": "#announces",
      "nick": "_AnnounceBot_",
      "msg": "12",
      "time": "2023-09-13T18:49:04.351389875+02:00"
    }
  ];

  for (let i = 0; i < 10; ++i) {
    for (let j = 0; j < 2; ++j) {
      logs.push(logs[i]);
    }
  }

  /*
  // TODO: Keep this in a map keyed by unique channel key?
  useEffect(() => {
    // Following RFC4648
    let es: EventSource | undefined = undefined;
    if (selectedChannel.length) {
      if (typeof(es) == "object") {
        // Close previous event source
        (es as EventSource).close();
      }

      es = APIClient.irc.events(selectedChannel);
      es.onmessage = (event) => {
        const newData = JSON.parse(event.data) as IrcEvent;
        // Keep last 50 events from previous state
        setLogs((prevState) => [...(prevState.slice(-50)), newData]);
      };
    }

    return () => es?.close();
  }, [selectedChannel]);*/
  const networkFilterFn = (network: IrcNetworkWithHealth) =>
    NetworkFilterPredicate(network, filter);

  return (
    <div className="flex flex-col md:flex-row min-h-[50vh] h-full lg:max-h-[80vh] md:max-w-[80vw] mx-auto my-4 rounded-lg bg-white dark:bg-gray-800 border border-gray-400 dark:border-gray-700">
      {selectedChannel?.network ? (
        <IrcNetworkUpdateForm
          isOpen={inEdit}
          toggle={toggleInEdit}
          network={selectedChannel.network}
        />
      ) : null}
      <div className="flex flex-col max-w-md py-3 rounded-l-lg">
        <div className="sticky top-0 pl-2">
          <div className="flex items-center">
            <input
              className="mr-1 py-1.5 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-400 dark:border-gray-700 block w-full bg-gray-100 dark:bg-gray-900 shadow-sm dark:text-gray-100 sm:text-sm rounded-md"
              placeholder="Filter networks/channels"
              onChange={(e) => {
                e.preventDefault();
                setFilter(e.currentTarget.value.trim().toLowerCase());
              }}
            />
            <ConfigurationDropdown />
          </div>
        </div>
        <div className="px-2 overflow-y-auto">
          {ircNetworks?.filter(networkFilterFn).map((network) =>
            <Network
              key={`${network.name}-${network.id}`}
              network={network}
              selectedChannel={selectedChannel}
              setSelectedChannel={setSelectedChannel}
            />
          )}
        </div>
      </div>
      <div className="flex flex-grow flex-col p-2 rounded-r-lg">
        {selectedChannel.channel && selectedChannel.network ? (
          <div className="flex items-center justify-between mb-1.5">
            <div className="flex flex-col">
              <p className="text-sm ml-1 text-gray-700 dark:text-gray-400">
                Monitoring since: {simplifyDate(selectedChannel.channel.monitoring_since)}
              </p>
              <p className="text-sm ml-1 text-gray-700 dark:text-gray-400">
                Last announce: {simplifyDate(selectedChannel.channel.last_announce)}
              </p>
            </div>
            <div className="flex flex-row gap-2">
              <button
                className="flex items-center text-sm text-gray-800 dark:text-gray-200 p-1 px-2 rounded shadow transition border border-amber-500 dark:border-amber-600 bg-amber-300 dark:bg-amber-800 hover:bg-amber-400 dark:hover:bg-amber-700"
                onClick={(e) => {
                  e.preventDefault();
                  restartMutation.mutate(selectedChannel.network!);
                }}
              >
                <span className="flex items-center"><ArrowPathIcon className="mr-2 w-4 h-4" />Restart network</span>
              </button>
              <button
                className="flex items-center text-sm text-gray-800 dark:text-gray-300 p-1 px-2 rounded shadow transition border border-gray-500 bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600"
                onClick={(e) => {
                  e.preventDefault();
                  toggleInEdit();
                }}
              >
                <span className="flex items-center"><PencilIcon className="mr-2 w-4 h-4" />Manage network</span>
              </button>
            </div>
          </div>
        ) : null}
        <div
          className="flex-grow px-2 py-1 overflow-auto rounded-lg min-w-full border border-gray-400 dark:border-gray-700 bg-gray-100 dark:bg-gray-900"
        >
          {selectedChannel ? (
            <>
              {logs.map((entry, idx) => (
                <div
                  key={idx}
                  className={classNames(
                    settings.indentLogLines ? "grid justify-start grid-flow-col" : "",
                    settings.hideWrappedText ? "truncate hover:text-ellipsis hover:whitespace-normal" : ""
                  )}
                >
                  <span className="font-mono text-gray-900 dark:text-gray-200 mr-1">
                    <span className="text-amber-700 dark:text-amber-400">
                      <span className="text-gray-500 dark:text-gray-700">[{simplifyDate(entry.time)}]</span>
                      {" "}{entry.nick}:
                    </span>
                    {" "}{entry.msg}
                  </span>
                </div>
              ))}
            </>
          ) : (
            <div className="w-full h-full flex items-center justify-center">
              <p className="text-2xl dark:text-white">Please select a channel from the sidebar</p>
            </div>
          )}
        </div>
        {selectedChannel ? (
          <div className="mt-2 flex items-center">
            <input
              className="block w-full shadow-sm sm:text-sm rounded-md border py-2.5 focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-400 dark:border-gray-700 bg-gray-100 dark:bg-gray-900 dark:text-gray-100"
              placeholder="Type a command to execute..."
              ref={commandBarRef}
            />
            <button className="flex items-center ml-2 px-3 py-1.5 transition rounded-md shadow border border-sky-500 bg-sky-300 hover:bg-sky-400 dark:bg-sky-900 dark:hover:bg-sky-700">
              <PaperAirplaneIcon
                className="h-4 w-4 text-gray-900 dark:text-gray-300"
                aria-hidden="true"
              />
              <span className="ml-2 text-black dark:text-white">Execute</span>
            </button>
          </div>
        ) : null}
      </div>
    </div>
  )
};
