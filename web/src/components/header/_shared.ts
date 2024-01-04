/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import {Auth} from "@app/App.tsx";

interface NavItem {
  name: string;
  path: string;
}

export interface RightNavProps {
  logoutMutation: () => void;
  auth: Auth
}

export const NAV_ROUTES: Array<NavItem> = [
  { name: "Dashboard", path: "/" },
  { name: "Filters", path: "/filters" },
  { name: "Releases", path: "/releases" },
  { name: "Settings", path: "/settings" },
  { name: "Logs", path: "/logs" }
];
