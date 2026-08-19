type IrcNetworkStatus = {
  enabled: boolean;
  healthy: boolean;
};

/** Reports enabled backend health failures, including those on connected IRC sockets. */
export function isUnhealthyIrcNetwork(network: IrcNetworkStatus): boolean {
  return network.enabled && !network.healthy;
}
