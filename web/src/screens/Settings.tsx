import { NavLink, Outlet, useLocation } from "react-router-dom";
import {
  BellIcon,
  ChatAlt2Icon,
  CogIcon,
  CollectionIcon,
  DownloadIcon,
  KeyIcon,
  RssIcon
} from "@heroicons/react/outline";

import { classNames } from "../utils";

interface NavTabType {
  name: string;
  href: string;
  icon: typeof CogIcon;
}

const subNavigation: NavTabType[] = [
  { name: "Application", href: "", icon: CogIcon },
  { name: "Indexers", href: "indexers", icon: KeyIcon },
  { name: "IRC", href: "irc", icon: ChatAlt2Icon },
  { name: "Feeds", href: "feeds", icon: RssIcon },
  { name: "Clients", href: "clients", icon: DownloadIcon },
  { name: "Notifications", href: "notifications", icon: BellIcon },
  { name: "Releases", href: "releases", icon: CollectionIcon }
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
        "border-transparent text-gray-900 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 hover:text-gray-900 dark:hover:text-gray-300 group border-l-4 px-3 py-2 flex items-center text-sm font-medium",
        isActive ?
          "font-bold bg-teal-50 dark:bg-gray-700 border-teal-500 dark:border-blue-500 text-teal-700 dark:text-white hover:bg-teal-50 dark:hover:bg-gray-500 hover:text-teal-700 dark:hover:text-gray-200" : ""
      )}
      aria-current={splitLocation[2] === item.href ? "page" : undefined}
    >
      <item.icon
        className="text-gray-400 group-hover:text-gray-500 dark:group-hover:text-gray-300 flex-shrink-0 -ml-1 mr-3 h-6 w-6"
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
          <SubNavLink item={item} key={item.href}/>
        ))}
      </nav>
    </aside>
  );
}

export default function Settings() {
  return (
    <main>
      <header className="py-10">
        <div className="max-w-screen-xl mx-auto px-4 sm:px-6 lg:px-8">
          <h1 className="text-3xl font-bold text-black dark:text-white">Settings</h1>
        </div>
      </header>

      <div className="max-w-screen-xl mx-auto pb-6 px-4 sm:px-6 lg:pb-16 lg:px-8">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg">
          <div className="divide-y divide-gray-200 dark:divide-gray-700 lg:grid lg:grid-cols-12 lg:divide-y-0 lg:divide-x">
            <SidebarNav subNavigation={subNavigation}/>
            <Outlet />
          </div>
        </div>
      </div>
    </main>
  );
}

