/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import "./index.css";
import "react-tooltip/dist/react-tooltip.css";

import { App } from "./App";
import { InitializeGlobalContext } from "./utils/Context";

declare global {
    interface Window { APP: APP; }
}

window.APP = window.APP || {};

// Initializes auth and theme contexts
InitializeGlobalContext();

// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
const root = createRoot(document.getElementById("root")!);
root.render(
  <StrictMode>
    <App />
  </StrictMode>
);