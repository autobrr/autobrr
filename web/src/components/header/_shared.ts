/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

interface NavItem {
  labelKey: string;
  path: string;
  exact?: boolean;
}

export interface RightNavProps {
  logoutMutation: () => void;
}

export const NAV_ROUTES: Array<NavItem> = [
  { labelKey: "nav.dashboard", path: "/", exact: true },
  { labelKey: "nav.filters", path: "/filters" },
  { labelKey: "nav.releases", path: "/releases" },
  { labelKey: "nav.settings", path: "/settings" },
  { labelKey: "nav.logs", path: "/logs" }
];
