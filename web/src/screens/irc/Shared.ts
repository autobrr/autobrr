export type SelectedChannel = {
  key: string;
  network?: IrcNetworkWithHealth;
  channel?: IrcChannelWithHealth;
};

export interface NetworkProps {
  network: IrcNetworkWithHealth;
  selectedChannel: SelectedChannel;
  setSelectedChannel: (newSelection: SelectedChannel) => void;
}

export interface ChannelProps extends NetworkProps {
  channel: IrcChannelWithHealth;
}

export const GetChannelKey = (channel: IrcChannelWithHealth, network: IrcNetworkWithHealth) =>
  window.btoa(`${network.id}${channel.name.toLowerCase()}`)
      .replaceAll("+", "-")
      .replaceAll("/", "_")
      .replaceAll("=", "");

export const IRC_KEYS = {
  all: ["irc_networks"] as const,
  lists: () => [...IRC_KEYS.all, "list"] as const,
  details: () => [...IRC_KEYS.all, "detail"] as const,
  detail: (id: number) => [...IRC_KEYS.details(), id] as const
};
