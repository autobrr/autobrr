interface NavItem {
  name: string;
  path: string;
}

export interface RightNavProps {
  logoutMutation: () => void;
}

export const NAV_ROUTES: Array<NavItem> = [
  { name: "Dashboard", path: "/" },
  { name: "Filters", path: "/filters" },
  { name: "Releases", path: "/releases" },
  { name: "Settings", path: "/settings" },
  { name: "Logs", path: "/logs" }
];
