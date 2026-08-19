import assert from "node:assert/strict";
import { test } from "node:test";

import { isUnhealthyIrcNetwork } from "../src/components/header/ircStatus.ts";

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
