import { BrowserRouter, Routes, Route } from "react-router-dom";

import { Login } from "../screens/auth/login";
import { Logout } from "../screens/auth/logout";
import { Onboarding } from "../screens/auth/onboarding";
import Base from "../screens/Base";
import { Dashboard } from "../screens/dashboard";
import { FilterDetails, Filters } from "../screens/filters";
import { Logs } from "../screens/Logs";
import { Releases } from "../screens/releases";
import Settings from "../screens/Settings";
import ApplicationSettings from "../screens/settings/Application";
import DownloadClientSettings from "../screens/settings/DownloadClient";
import FeedSettings from "../screens/settings/Feed";
import IndexerSettings from "../screens/settings/Indexer";
import { IrcSettings } from "../screens/settings/Irc";
import NotificationSettings from "../screens/settings/Notifications";
import { RegexPlayground } from "../screens/settings/RegexPlayground";
import ReleaseSettings from "../screens/settings/Releases";

import { baseUrl } from "../utils";

export const LocalRouter = ({ isLoggedIn }: { isLoggedIn: boolean }) => (
  <BrowserRouter basename={baseUrl()}>
    {isLoggedIn ? (
      <Routes>
        <Route path="/logout" element={<Logout />} />
        <Route element={<Base />}>
          <Route index element={<Dashboard />} />
          <Route path="logs" element={<Logs />} />
          <Route path="releases" element={<Releases />} />
          <Route path="filters">
            <Route index element={<Filters />} />
            <Route path=":filterId/*" element={<FilterDetails />} />
          </Route>
          <Route path="settings" element={<Settings />}>
            <Route index element={<ApplicationSettings />} />
            <Route path="indexers" element={<IndexerSettings />} />
            <Route path="feeds" element={<FeedSettings />} />
            <Route path="irc" element={<IrcSettings />} />
            <Route path="clients" element={<DownloadClientSettings />} />
            <Route path="notifications" element={<NotificationSettings />} />
            <Route path="releases" element={<ReleaseSettings />} />
            <Route path="regex-playground" element={<RegexPlayground />} />
          </Route>
        </Route>
      </Routes>
    ) : (
      <Routes>
        <Route path="/onboard" element={<Onboarding />} />
        <Route path="*" element={<Login />} />
      </Routes>
    )}
  </BrowserRouter>
);
