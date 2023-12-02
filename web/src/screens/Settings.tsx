/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Suspense } from "react";
import { NavLink, Outlet, useLocation } from "react-router-dom";
import {
  BellIcon,
  ChatBubbleLeftRightIcon,
  CogIcon,
  FolderArrowDownIcon,
  KeyIcon,
  RectangleStackIcon,
  RssIcon,
  Square3Stack3DIcon,
  UserCircleIcon
} from "@heroicons/react/24/outline";

import { classNames } from "@utils";
import { SectionLoader } from "@components/SectionLoader";

interface NavTabType {
  name: string;
  href: string;
  icon: typeof CogIcon;
}

const subNavigation: NavTabType[] = [
  { name: "Application", href: "", icon: CogIcon },
  { name: "Logs", href: "logs", icon: Square3Stack3DIcon },
  { name: "Indexers", href: "indexers", icon: KeyIcon },
  { name: "IRC", href: "irc", icon: ChatBubbleLeftRightIcon },
  { name: "Feeds", href: "feeds", icon: RssIcon },
  { name: "Clients", href: "clients", icon: FolderArrowDownIcon },
  { name: "Notifications", href: "notifications", icon: BellIcon },
  { name: "API keys", href: "api-keys", icon: KeyIcon },
  { name: "Releases", href: "releases", icon: RectangleStackIcon },
  { name: "Profile", href: "profile", icon: UserCircleIcon }
  // {name: 'Regex Playground', href: 'regex-playground', icon: CogIcon, current: false}
  // {name: 'Rules', href: 'rules', icon: ClipboardCheckIcon, current: false},
];

interface NavLinkProps {
  item: NavTabType;
}

function SubNavLink({ item }: NavLinkProps) {
  const { pathname } = useLocation();
  const splitLocation = pathname.split("/");

  // we need to clean the / if it's a base root path
  return (
    <NavLink
      key={item.name}
      to={item.href}
      end
      className={({ isActive }) => classNames(
        "transition group border-l-4 px-3 py-2 flex items-center text-sm font-medium",
        isActive
          ? "font-bold bg-blue-100 dark:bg-gray-700 border-sky-500 dark:border-blue-500 text-sky-700 dark:text-gray-200 hover:bg-blue-200 dark:hover:bg-gray-600 hover:text-sky-900 dark:hover:text-white"
          : "border-transparent text-gray-900 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 hover:text-gray-900 dark:hover:text-gray-300"
      )}
      aria-current={splitLocation[2] === item.href ? "page" : undefined}
    >
      <item.icon
        className="text-gray-500 dark:text-gray-400 group-hover:text-gray-600 dark:group-hover:text-gray-300 flex-shrink-0 -ml-1 mr-3 h-6 w-6"
        aria-hidden="true"
      />
      <span className="truncate">{item.name}</span>
    </NavLink>
  );
}

interface SidebarNavProps {
  subNavigation: NavTabType[];
}

function SidebarNav({ subNavigation }: SidebarNavProps) {
  return (
    <aside className="py-2 lg:col-span-3">
      <nav className="space-y-1">
        {subNavigation.map((item) => (
          <SubNavLink item={item} key={item.href} />
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
          <div className="divide-y divide-gray-150 dark:divide-gray-725 lg:grid lg:grid-cols-12 lg:divide-y-0 lg:divide-x">
            <SidebarNav subNavigation={subNavigation}/>
            <Suspense
              fallback={
                <div className="flex items-center justify-center lg:col-span-9">
                  <SectionLoader $size="large" />
                </div>
              }
            >
              <Outlet />
            </Suspense>
          </div>
        </div>
      </div>
    </main>
  );
}
