/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { Buffer } from "buffer";

import "./index.css";

import { App } from "./App";
import { InitializeGlobalContext } from "./utils/Context";

declare global {
    interface Window { APP: APP; }
}

window.APP = window.APP || {};
// Apparently Stacktracey requires this for some weird reason
// (at least in local dev env)
window.Buffer = Buffer;

// Initializes auth and theme contexts
InitializeGlobalContext();

const root = createRoot(document.getElementById("root")!);
root.render(
  <StrictMode>
    <App />
  </StrictMode>
);
