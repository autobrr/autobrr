import { ExclamationCircleIcon } from "@heroicons/react/24/solid";

import { GetChannelKey } from "./Shared";
import type { ChannelProps } from "./Shared";
import { classNames, simplifyDate } from "@utils";

export const Channel = ({
  channel,
  network,
  selectedChannel,
  setSelectedChannel
}: ChannelProps) => {
  const channelKey = GetChannelKey(channel, network);
  const isSelected = channelKey === selectedChannel.key;
  return (
    <div
      className={classNames(
        isSelected ? "bg-sky-600 dark:bg-sky-800 border-sky-500" : "border-transparent hover:bg-white dark:hover:bg-gray-900",
        network.enabled && channel.enabled ? "cursor-pointer" : "cursor-not-allowed",
        "border flex items-center justify-between p-2 w-full rounded-lg transition",
      )}
      onClick={(e) => {
        e.preventDefault();
        setSelectedChannel({
          key: channelKey,
          channel: channel,
          network: network,
        });
      }}
    >
      <span className={classNames(
        "ml-2",
        network.enabled && channel.enabled ? (
          isSelected ? "text-white" : "text-gray-700 dark:text-gray-400"
        ) : (
          "text-gray-700 dark:text-gray-400 line-through"
        )
      )}>
        {channel.name}
      </span>
      {channel.enabled ? (
        network.healthy ? (
          <span
            className="flex h-3 w-3 relative"
            title={`Connected since: ${simplifyDate(network.connected_since)}`}
          >
            <span className="animate-ping inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
            <span className="inline-flex absolute rounded-full h-3 w-3 bg-green-500" />
          </span>
        ) : (
          <span
            className="flex items-center"
            title={network.connection_errors.toString()}
          >
            <ExclamationCircleIcon className="h-5 w-5 text-amber-400 dark:text-yellow-400" />
          </span>
        )
      ) : (
        <span className="flex h-3 w-3 rounded-full opacity-75 bg-gray-500" />
      )}
    </div>
  );
}
