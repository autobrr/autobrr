type IrcNetworkStatus = {
  enabled: boolean;
  healthy: boolean;
};

/** Reports enabled backend health failures, including those on connected IRC sockets. */
export function isUnhealthyIrcNetwork(network: IrcNetworkStatus): boolean {
  return network.enabled && !network.healthy;
}

/** Confirms networks reported unhealthy by consecutive health polls. */
export function evaluateIrcHealthPoll<T extends IrcNetworkStatus & { id: number }>(
  networks: readonly T[],
  previousUnhealthyIds: ReadonlySet<number>,
): { currentUnhealthyIds: Set<number>; confirmedNetworks: T[] } {
  const unhealthyNetworks = networks.filter(isUnhealthyIrcNetwork);

  return {
    currentUnhealthyIds: new Set(unhealthyNetworks.map(network => network.id)),
    confirmedNetworks: unhealthyNetworks.filter(network => previousUnhealthyIds.has(network.id)),
  };
}
