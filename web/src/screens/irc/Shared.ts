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

