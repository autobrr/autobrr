import {CogIcon, CollectionIcon, DownloadIcon, KeyIcon} from '@heroicons/react/outline'
import {NavLink, Route, Switch as RouteSwitch, useLocation, useRouteMatch} from "react-router-dom";

import { classNames } from "../utils";
import IndexerSettings from "./settings/Indexer";
import { IrcSettings } from "./settings/Irc";
import ApplicationSettings from "./settings/Application";
import DownloadClientSettings from "./settings/DownloadClient";
import { RegexPlayground } from './settings/RegexPlayground';
import ReleaseSettings from "./settings/Releases";

const subNavigation = [
    {name: 'Application', href: '', icon: CogIcon, current: true},
    {name: 'Indexers', href: 'indexers', icon: KeyIcon, current: false},
    {name: 'IRC', href: 'irc', icon: KeyIcon, current: false},
    {name: 'Clients', href: 'clients', icon: DownloadIcon, current: false},
    {name: 'Releases', href: 'releases', icon: CollectionIcon, current: false},
    // {name: 'Regex Playground', href: 'regex-playground', icon: CogIcon, current: false}
    // {name: 'Actions', href: 'actions', icon: PlayIcon, current: false},
    // {name: 'Rules', href: 'rules', icon: ClipboardCheckIcon, current: false},
    // {name: 'Notifications', href: 'notifications', icon: BellIcon, current: false},
]

function SubNavLink({item, url}: any) {
    const location = useLocation();
    const { pathname } = location;

    const splitLocation = pathname.split("/");

    // we need to clean the / if it's a base root path
    const too = item.href ? `${url}/${item.href}` : url
    return (
        <NavLink
            key={item.name}
            to={too}
            exact={true}
            activeClassName="bg-teal-50 dark:bg-gray-700 border-teal-500 dark:border-blue-500 text-teal-700 dark:text-white hover:bg-teal-50 dark:hover:bg-gray-500 hover:text-teal-700 dark:hover:text-gray-200"
            className={classNames(
                'border-transparent text-gray-900 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 hover:text-gray-900 dark:hover:text-gray-300 group border-l-4 px-3 py-2 flex items-center text-sm font-medium'
            )}
            aria-current={splitLocation[2] === item.href ? 'page' : undefined}
        >
            <item.icon
                className="text-gray-400 group-hover:text-gray-500 dark:group-hover:text-gray-300 flex-shrink-0 -ml-1 mr-3 h-6 w-6"
                aria-hidden="true"
            />
            <span className="truncate">{item.name}</span>
        </NavLink>
    )
}

function SidebarNav({subNavigation, url}: any) {
    return (
        <aside className="py-6 lg:col-span-3">
            <nav className="space-y-1">
                {subNavigation.map((item: any) => (
                    <SubNavLink item={item} url={url} key={item.href}/>
                ))}
            </nav>
        </aside>
    )
}

export default function Settings() {
    const { url } = useRouteMatch();
    return (
        <main>
            <header className="py-10">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <h1 className="text-3xl font-bold text-black dark:text-white capitalize">Settings</h1>
                </div>
            </header>

            <div className="max-w-screen-xl mx-auto pb-6 px-4 sm:px-6 lg:pb-16 lg:px-8">
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg overflow-hidden">
                    <div className="divide-y divide-gray-200 dark:divide-gray-700 lg:grid lg:grid-cols-12 lg:divide-y-0 lg:divide-x">
                        <SidebarNav url={url} subNavigation={subNavigation}/>

                        <RouteSwitch>
                            <Route exact path={url}>
                                <ApplicationSettings/>
                            </Route>

                            <Route path={`${url}/indexers`}>
                                <IndexerSettings/>
                            </Route>

                            <Route path={`${url}/irc`}>
                                <IrcSettings/>
                            </Route>

                            <Route path={`${url}/clients`}>
                                <DownloadClientSettings/>
                            </Route>

                            <Route path={`${url}/releases`}>
                                <ReleaseSettings/>
                            </Route>

                            {/*<Route path={`${url}/actions`}>
                                <ActionSettings/>
                            </Route>*/}

                            <Route path={`${url}/regex-playground`}>
                                <RegexPlayground />
                            </Route>
                        </RouteSwitch>
                    </div>
                </div>
            </div>
        </main>
    )
}

