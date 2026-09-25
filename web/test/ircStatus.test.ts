import assert from "node:assert/strict";
import { test } from "node:test";

import {
  evaluateIrcHealthPoll,
  isUnhealthyIrcNetwork,
} from "../src/components/header/ircStatus.ts";

test("identifies IRC networks that need a status warning", () => {
  const cases = [
    {
      name: "enabled healthy network",
      network: { enabled: true, connected: true, healthy: true },
      want: false,
    },
    {
      name: "enabled disconnected unhealthy network",
      network: { enabled: true, connected: false, healthy: false },
      want: true,
    },
    {
      name: "enabled connected unhealthy network",
      network: { enabled: true, connected: true, healthy: false },
      want: true,
    },
    {
      name: "disabled unhealthy network",
      network: { enabled: false, connected: false, healthy: false },
      want: false,
    },
  ];

  for (const testCase of cases) {
    assert.equal(
      isUnhealthyIrcNetwork(testCase.network),
      testCase.want,
      testCase.name,
    );
  }
});

test("requires two consecutive unhealthy IRC polls", () => {
  const firstPoll = evaluateIrcHealthPoll([
    { id: 1, enabled: true, healthy: false },
    { id: 2, enabled: true, healthy: true },
  ], new Set());
  assert.deepEqual(firstPoll.confirmedNetworks, []);

  const secondPoll = evaluateIrcHealthPoll([
    { id: 1, enabled: true, healthy: false },
    { id: 2, enabled: true, healthy: false },
  ], firstPoll.currentUnhealthyIds);
  assert.deepEqual(secondPoll.confirmedNetworks.map(network => network.id), [1]);

  const thirdPoll = evaluateIrcHealthPoll([
    { id: 1, enabled: true, healthy: true },
    { id: 2, enabled: true, healthy: false },
  ], secondPoll.currentUnhealthyIds);
  assert.deepEqual(thirdPoll.confirmedNetworks.map(network => network.id), [2]);
});
