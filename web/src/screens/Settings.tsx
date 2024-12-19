/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import {
  BellIcon,
  ChatBubbleLeftRightIcon,
  CogIcon,
  FolderArrowDownIcon,
  GlobeAltIcon,
  KeyIcon,
  RectangleStackIcon,
  RssIcon,
  Square3Stack3DIcon,
  UserCircleIcon
} from "@heroicons/react/24/outline";
import { Link, Outlet } from "@tanstack/react-router";

import { classNames } from "@utils";

interface NavTabType {
  name: string;
  href: string;
  icon: typeof CogIcon;
  exact?: boolean;
}

const subNavigation: NavTabType[] = [
  { name: "Application", href: "/settings", icon: CogIcon, exact: true },
  { name: "Logs", href: "/settings/logs", icon: Square3Stack3DIcon },
  { name: "Indexers", href: "/settings/indexers", icon: KeyIcon },
  { name: "IRC", href: "/settings/irc", icon: ChatBubbleLeftRightIcon },
  { name: "Feeds", href: "/settings/feeds", icon: RssIcon },
  { name: "Lists", href: "/settings/lists", icon: RssIcon },
  { name: "Clients", href: "/settings/clients", icon: FolderArrowDownIcon },
  { name: "Notifications", href: "/settings/notifications", icon: BellIcon },
  { name: "API keys", href: "/settings/api", icon: KeyIcon },
  { name: "Proxies", href: "/settings/proxies", icon: GlobeAltIcon },
  { name: "Releases", href: "/settings/releases", icon: RectangleStackIcon },
  { name: "Account", href: "/settings/account", icon: UserCircleIcon }
  // {name: 'Regex Playground', href: 'regex-playground', icon: CogIcon, current: false}
  // {name: 'Rules', href: 'rules', icon: ClipboardCheckIcon, current: false},
];

interface NavLinkProps {
  item: NavTabType;
}

function SubNavLink({ item }: NavLinkProps) {
  // const { pathname } = useLocation();
  // const splitLocation = pathname.split("/");

  // we need to clean the / if it's a base root path
  return (
    <Link
      key={item.href}
      to={item.href}
      activeOptions={{ exact: item.exact }}
      search={{}}
      params={{}}
      // aria-current={splitLocation[2] === item.href ? "page" : undefined}
    >
      {({ isActive }) => {
        return (
          <span className={
            classNames(
              "transition group border-l-4 px-3 py-2 flex items-center text-sm font-medium",
              isActive
                ? "font-bold bg-blue-100 dark:bg-gray-700 border-sky-500 dark:border-blue-500 text-sky-700 dark:text-gray-200 hover:bg-blue-200 dark:hover:bg-gray-600 hover:text-sky-900 dark:hover:text-white"
                : "border-transparent text-gray-900 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 hover:text-gray-900 dark:hover:text-gray-300"
            )
          }>
            <item.icon
              className="text-gray-500 dark:text-gray-400 group-hover:text-gray-600 dark:group-hover:text-gray-300 flex-shrink-0 -ml-1 mr-3 h-6 w-6"
              aria-hidden="true"
            />
            <span className="truncate">{item.name}</span>
          </span>
        )
      }}
    </Link>
  );
}

interface SidebarNavProps {
  subNavigation: NavTabType[];
}

function SidebarNav({ subNavigation }: SidebarNavProps) {
  return (
    <aside className="py-2 lg:col-span-3 border-b lg:border-b-0 lg:border-r border-gray-150 dark:border-gray-725">
      <nav className="space-y-1">
        {subNavigation.map((item) => (
          <SubNavLink key={item.href} item={item} />
        ))}
      </nav>
    </aside>
  );
}

export function Settings() {
  return (
    <main>
      <div className="my-6 max-w-screen-xl mx-auto px-4 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold text-black dark:text-white">Settings</h1>
      </div>

      <div className="max-w-screen-xl mx-auto pb-6 px-2 sm:px-6 lg:pb-16 lg:px-8">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-table border border-gray-250 dark:border-gray-775">
          <div className="lg:grid lg:grid-cols-12">
            <SidebarNav subNavigation={subNavigation}/>
              <Outlet />
          </div>
        </div>
      </div>
    </main>
  );
}
